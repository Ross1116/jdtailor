package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultProvider = "openrouter"
	defaultModel    = "deepseek/deepseek-v4-flash"
	settingProvider = "llm_provider"
	settingModel    = "llm_model"
)

type Health struct {
	Version       string `json:"version"`
	StorageStatus string `json:"storage_status"`
	DBPath        string `json:"db_path"`
	LogPath       string `json:"log_path"`
	GeneratedPath string `json:"generated_path"`
	PDFRenderer   string `json:"pdf_renderer"`
}

type Settings struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	APIKeyConfigured bool   `json:"api_key_configured"`
}

type SaveSettingsInput struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type AppEvent struct {
	ID        int64  `json:"id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type Store struct {
	root          string
	dataPath      string
	dbPath        string
	logPath       string
	generatedPath string
	db            *sql.DB
	logFile       *os.File
	logger        *slog.Logger
}

func NewStore(root string) (*Store, error) {
	paths, err := ensureLocalPaths(root)
	if err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(paths.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	db, err := sql.Open("sqlite", paths.dbPath)
	if err != nil {
		_ = logFile.Close()
		return nil, err
	}

	store := &Store{
		root:          paths.root,
		dataPath:      paths.dataPath,
		dbPath:        paths.dbPath,
		logPath:       paths.logPath,
		generatedPath: paths.generatedPath,
		db:            db,
		logFile:       logFile,
		logger:        logger,
	}

	if err := store.migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Logger() *slog.Logger {
	if s.logger == nil {
		return slog.Default()
	}
	return s.logger
}

func (s *Store) Close() error {
	var err error
	if s.db != nil {
		err = errors.Join(err, s.db.Close())
	}
	if s.logFile != nil {
		err = errors.Join(err, s.logFile.Close())
	}
	return err
}

func (s *Store) Health(version string) Health {
	return Health{
		Version:       version,
		StorageStatus: "ready",
		DBPath:        s.dbPath,
		LogPath:       s.logPath,
		GeneratedPath: s.generatedPath,
		PDFRenderer:   s.TectonicStatus().Status,
	}
}

func (s *Store) GetSettings() (Settings, error) {
	settings := Settings{
		Provider: defaultProvider,
	}

	values, err := s.getSettingsMap(context.Background())
	if err != nil {
		return Settings{}, err
	}

	if provider := configuredProvider(values[settingProvider]); provider != "" {
		settings.Provider = provider
	}
	settings.Model = configuredModel(settings.Provider, values[settingModel])
	settings.APIKeyConfigured = s.APIKeyConfigured(settings.Provider)
	return settings, nil
}

func (s *Store) SaveSettings(input SaveSettingsInput) (Settings, error) {
	provider := configuredProvider(input.Provider)
	model := strings.TrimSpace(input.Model)

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return Settings{}, err
	}
	defer tx.Rollback()

	if err := upsertSettingTx(tx, settingProvider, provider); err != nil {
		return Settings{}, err
	}
	if err := upsertSettingTx(tx, settingModel, model); err != nil {
		return Settings{}, err
	}
	if err := tx.Commit(); err != nil {
		return Settings{}, err
	}

	if err := s.LogEvent("info", "settings saved"); err != nil {
		return Settings{}, err
	}

	return Settings{
		Provider:         provider,
		Model:            configuredModel(provider, model),
		APIKeyConfigured: s.APIKeyConfigured(provider),
	}, nil
}

func (s *Store) LogEvent(level string, message string) error {
	level = strings.TrimSpace(strings.ToLower(level))
	if level == "" {
		level = "info"
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "event"
	}

	s.Logger().Info(message, "level", level)
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO app_events (level, message, created_at) VALUES (?, ?, ?)`,
		level,
		message,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetRecentEvents(limit int) ([]AppEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, level, message, created_at FROM app_events ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]AppEvent, 0, limit)
	for rows.Next() {
		var event AppEvent
		if err := rows.Scan(&event.ID, &event.Level, &event.Message, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (s *Store) getSettingsMap(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);
	`); err != nil {
		return err
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
				CREATE TABLE IF NOT EXISTS settings (
					key TEXT PRIMARY KEY,
					value TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS app_events (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					level TEXT NOT NULL,
					message TEXT NOT NULL,
					created_at TEXT NOT NULL
				);
			`,
		},
		{
			version: 2,
			sql: `
				UPDATE settings
				SET value = 'openrouter', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
				WHERE key = 'llm_provider' AND value = 'openai';

				UPDATE settings
				SET value = '', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
				WHERE key = 'llm_model' AND value IN ('gpt-5-mini', 'gpt-5.4-mini');
			`,
		},
		{
			version: 3,
			sql: `
				CREATE TABLE IF NOT EXISTS candidate_profile (
					id INTEGER PRIMARY KEY CHECK (id = 1),
					full_name TEXT NOT NULL DEFAULT '',
					email TEXT NOT NULL DEFAULT '',
					phone TEXT NOT NULL DEFAULT '',
					location TEXT NOT NULL DEFAULT '',
					linkedin TEXT NOT NULL DEFAULT '',
					github TEXT NOT NULL DEFAULT '',
					portfolio TEXT NOT NULL DEFAULT '',
					links_json TEXT NOT NULL DEFAULT '[]',
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS candidate_profile_records (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					record_type TEXT NOT NULL,
					label TEXT NOT NULL DEFAULT '',
					organization TEXT NOT NULL DEFAULT '',
					role TEXT NOT NULL DEFAULT '',
					start_date TEXT NOT NULL DEFAULT '',
					end_date TEXT NOT NULL DEFAULT '',
					value TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS candidate_sources (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					source_type TEXT NOT NULL,
					title TEXT NOT NULL,
					raw_text TEXT NOT NULL,
					file_path TEXT NOT NULL DEFAULT '',
					imported_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS source_sections (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					source_id INTEGER NOT NULL REFERENCES candidate_sources(id) ON DELETE CASCADE,
					heading TEXT NOT NULL,
					section_type TEXT NOT NULL,
					content TEXT NOT NULL,
					sort_order INTEGER NOT NULL,
					start_char INTEGER NOT NULL DEFAULT 0,
					end_char INTEGER NOT NULL DEFAULT 0,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS evidence_facts (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					source_id INTEGER NOT NULL REFERENCES candidate_sources(id) ON DELETE CASCADE,
					section_id INTEGER NOT NULL REFERENCES source_sections(id) ON DELETE CASCADE,
					fact_text TEXT NOT NULL,
					evidence_quote TEXT NOT NULL,
					technologies_json TEXT NOT NULL DEFAULT '[]',
					confidence TEXT NOT NULL,
					risk_flags_json TEXT NOT NULL DEFAULT '[]',
					status TEXT NOT NULL,
					auto_approved INTEGER NOT NULL DEFAULT 0,
					review_note TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				INSERT INTO candidate_profile (id, updated_at)
				VALUES (1, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
				ON CONFLICT(id) DO NOTHING;
			`,
		},
		{
			version: 4,
			sql: `
				ALTER TABLE candidate_profile ADD COLUMN verified INTEGER NOT NULL DEFAULT 0;
				ALTER TABLE candidate_profile_records ADD COLUMN verified INTEGER NOT NULL DEFAULT 0;
			`,
		},
		{
			version: 5,
			sql: `
				CREATE TABLE IF NOT EXISTS job_descriptions (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					company TEXT NOT NULL DEFAULT '',
					title TEXT NOT NULL,
					url TEXT NOT NULL DEFAULT '',
					raw_text TEXT NOT NULL,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS job_requirements (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					job_id INTEGER NOT NULL REFERENCES job_descriptions(id) ON DELETE CASCADE,
					category TEXT NOT NULL,
					requirement_text TEXT NOT NULL,
					keywords_json TEXT NOT NULL DEFAULT '[]',
					priority TEXT NOT NULL,
					source_quote TEXT NOT NULL,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS job_fact_matches (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					job_id INTEGER NOT NULL REFERENCES job_descriptions(id) ON DELETE CASCADE,
					requirement_id INTEGER NOT NULL REFERENCES job_requirements(id) ON DELETE CASCADE,
					fact_id INTEGER NOT NULL REFERENCES evidence_facts(id) ON DELETE CASCADE,
					score REAL NOT NULL DEFAULT 0,
					rationale TEXT NOT NULL DEFAULT '',
					coverage_status TEXT NOT NULL,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS tailored_bullet_drafts (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					job_id INTEGER NOT NULL REFERENCES job_descriptions(id) ON DELETE CASCADE,
					requirement_id INTEGER NOT NULL REFERENCES job_requirements(id) ON DELETE CASCADE,
					fact_ids_json TEXT NOT NULL DEFAULT '[]',
					draft_text TEXT NOT NULL,
					rationale TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL,
					risk_flags_json TEXT NOT NULL DEFAULT '[]',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);
			`,
		},
		{
			version: 6,
			sql: `
				CREATE TABLE IF NOT EXISTS prompt_rules (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					rule_key TEXT NOT NULL UNIQUE,
					category TEXT NOT NULL,
					title TEXT NOT NULL,
					content TEXT NOT NULL,
					enabled INTEGER NOT NULL DEFAULT 1,
					version INTEGER NOT NULL DEFAULT 1,
					source TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS prompt_research_sources (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					source_type TEXT NOT NULL,
					trust_tier TEXT NOT NULL,
					title TEXT NOT NULL,
					url TEXT NOT NULL,
					extracted_pattern TEXT NOT NULL,
					app_adaptation TEXT NOT NULL,
					accessed_at TEXT NOT NULL,
					created_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS job_analyses (
					job_id INTEGER PRIMARY KEY REFERENCES job_descriptions(id) ON DELETE CASCADE,
					company TEXT NOT NULL DEFAULT '',
					role_title TEXT NOT NULL DEFAULT '',
					location TEXT NOT NULL DEFAULT '',
					work_arrangement TEXT NOT NULL DEFAULT '',
					salary TEXT NOT NULL DEFAULT '',
					top_pain_points_json TEXT NOT NULL DEFAULT '[]',
					required_skills_json TEXT NOT NULL DEFAULT '[]',
					preferred_skills_json TEXT NOT NULL DEFAULT '[]',
					responsibilities_json TEXT NOT NULL DEFAULT '[]',
					seniority_level TEXT NOT NULL DEFAULT '',
					role_archetype TEXT NOT NULL DEFAULT '',
					keywords_json TEXT NOT NULL DEFAULT '[]',
					risk_flags_json TEXT NOT NULL DEFAULT '[]',
					job_poster TEXT NOT NULL DEFAULT '',
					company_url TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS job_fit_analyses (
					job_id INTEGER PRIMARY KEY REFERENCES job_descriptions(id) ON DELETE CASCADE,
					overall_score INTEGER NOT NULL DEFAULT 0,
					recommendation TEXT NOT NULL DEFAULT '',
					strengths_json TEXT NOT NULL DEFAULT '[]',
					critical_gaps_json TEXT NOT NULL DEFAULT '[]',
					reality_check TEXT NOT NULL DEFAULT '',
					analysis_json TEXT NOT NULL DEFAULT '[]',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS application_strategies (
					job_id INTEGER PRIMARY KEY REFERENCES job_descriptions(id) ON DELETE CASCADE,
					approved_fact_ids_json TEXT NOT NULL DEFAULT '[]',
					rejected_fact_ids_json TEXT NOT NULL DEFAULT '[]',
					weak_or_missing_json TEXT NOT NULL DEFAULT '[]',
					resume_headline TEXT NOT NULL DEFAULT '',
					experience_titles_json TEXT NOT NULL DEFAULT '{}',
					positioning_strategy TEXT NOT NULL DEFAULT '',
					keywords_json TEXT NOT NULL DEFAULT '[]',
					do_not_overclaim_json TEXT NOT NULL DEFAULT '[]',
					fit_summary TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);
			`,
		},
		{
			version: 7,
			sql: `
				ALTER TABLE evidence_facts ADD COLUMN origin_heading TEXT NOT NULL DEFAULT '';
				ALTER TABLE evidence_facts ADD COLUMN origin_type TEXT NOT NULL DEFAULT '';
				ALTER TABLE evidence_facts ADD COLUMN context_json TEXT NOT NULL DEFAULT '[]';

				UPDATE evidence_facts
				SET origin_heading = COALESCE((SELECT heading FROM source_sections WHERE source_sections.id = evidence_facts.section_id), ''),
					origin_type = COALESCE((SELECT section_type FROM source_sections WHERE source_sections.id = evidence_facts.section_id), '')
				WHERE origin_heading = '' OR origin_type = '';
			`,
		},
		{
			version: 8,
			sql: `
				CREATE TABLE IF NOT EXISTS candidate_claims (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					claim_text TEXT NOT NULL,
					claim_type TEXT NOT NULL DEFAULT 'experience',
					source_fact_ids_json TEXT NOT NULL DEFAULT '[]',
					evidence_quotes_json TEXT NOT NULL DEFAULT '[]',
					technologies_json TEXT NOT NULL DEFAULT '[]',
					strength TEXT NOT NULL DEFAULT 'moderate',
					allowed_use_json TEXT NOT NULL DEFAULT '[]',
					allowed_contexts_json TEXT NOT NULL DEFAULT '[]',
					blocked_contexts_json TEXT NOT NULL DEFAULT '[]',
					safe_phrasings_json TEXT NOT NULL DEFAULT '[]',
					unsafe_phrasings_json TEXT NOT NULL DEFAULT '[]',
					origin_heading TEXT NOT NULL DEFAULT '',
					origin_type TEXT NOT NULL DEFAULT '',
					status TEXT NOT NULL DEFAULT 'needs_review',
					risk_flags_json TEXT NOT NULL DEFAULT '[]',
					review_note TEXT NOT NULL DEFAULT '',
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);

				CREATE TABLE IF NOT EXISTS blocked_claims (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					pattern TEXT NOT NULL UNIQUE,
					reason TEXT NOT NULL DEFAULT '',
					severity TEXT NOT NULL DEFAULT 'medium',
					source TEXT NOT NULL DEFAULT '',
					enabled INTEGER NOT NULL DEFAULT 1,
					created_at TEXT NOT NULL,
					updated_at TEXT NOT NULL
				);
			`,
		},
		{
			version: 9,
			sql: `
				ALTER TABLE candidate_claims ADD COLUMN actions_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN capabilities_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN objects_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN domains_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN artifacts_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN scope_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN metrics_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN outcomes_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN profile_context_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE candidate_claims ADD COLUMN evidence_strength TEXT NOT NULL DEFAULT 'direct';

				ALTER TABLE tailored_bullet_drafts ADD COLUMN claim_ids_json TEXT NOT NULL DEFAULT '[]';
				ALTER TABLE tailored_bullet_drafts ADD COLUMN origin_heading TEXT NOT NULL DEFAULT '';
				ALTER TABLE tailored_bullet_drafts ADD COLUMN origin_type TEXT NOT NULL DEFAULT '';
				ALTER TABLE tailored_bullet_drafts ADD COLUMN selection_score REAL NOT NULL DEFAULT 0;
				ALTER TABLE tailored_bullet_drafts ADD COLUMN selected_for_resume INTEGER NOT NULL DEFAULT 0;
			`,
		},
		{
			version: 10,
			sql: `
				ALTER TABLE candidate_sources ADD COLUMN trust_tier TEXT NOT NULL DEFAULT 'unverified_ai';

				ALTER TABLE evidence_facts ADD COLUMN similarity_key TEXT NOT NULL DEFAULT '';
				ALTER TABLE evidence_facts ADD COLUMN similarity_score REAL NOT NULL DEFAULT 1;
				ALTER TABLE evidence_facts ADD COLUMN duplicate_of_id INTEGER NOT NULL DEFAULT 0;

				ALTER TABLE candidate_claims ADD COLUMN similarity_key TEXT NOT NULL DEFAULT '';
				ALTER TABLE candidate_claims ADD COLUMN similarity_score REAL NOT NULL DEFAULT 1;
				ALTER TABLE candidate_claims ADD COLUMN duplicate_of_id INTEGER NOT NULL DEFAULT 0;
			`,
		},
	}

	for _, migration := range migrations {
		applied, err := s.migrationApplied(ctx, migration.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			migration.version,
			time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	if err := s.ensureDefaultSettings(ctx); err != nil {
		return err
	}
	if err := s.seedPromptDefaults(ctx); err != nil {
		return err
	}
	if err := s.seedBlockedClaimDefaults(ctx); err != nil {
		return err
	}

	return s.LogEvent("info", "database migrated")
}

func (s *Store) migrationApplied(ctx context.Context, version int) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM schema_migrations WHERE version = ?`,
		version,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return exists == 1, nil
}

func (s *Store) ensureDefaultSettings(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := ensureSettingTx(tx, settingProvider, defaultProvider); err != nil {
		return err
	}
	if err := ensureSettingTx(tx, settingModel, ""); err != nil {
		return err
	}

	return tx.Commit()
}

func ensureSettingTx(tx *sql.Tx, key string, value string) error {
	_, err := tx.Exec(
		`INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO NOTHING`,
		key,
		value,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func upsertSettingTx(tx *sql.Tx, key string, value string) error {
	_, err := tx.Exec(
		`INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at`,
		key,
		value,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

type localPaths struct {
	root          string
	dataPath      string
	dbPath        string
	logPath       string
	generatedPath string
}

func ensureLocalPaths(root string) (localPaths, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return localPaths{}, err
	}

	paths := localPaths{
		root:          absRoot,
		dataPath:      filepath.Join(absRoot, "data"),
		dbPath:        filepath.Join(absRoot, "data", "app.db"),
		logPath:       filepath.Join(absRoot, "logs", "app.log"),
		generatedPath: filepath.Join(absRoot, "generated"),
	}

	for _, path := range []string{
		paths.dataPath,
		filepath.Dir(paths.logPath),
		paths.generatedPath,
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return localPaths{}, err
		}
	}

	return paths, nil
}
