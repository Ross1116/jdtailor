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
	defaultProvider = "openai"
	defaultModel    = "gpt-5-mini"
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

	if provider := strings.TrimSpace(values[settingProvider]); provider != "" {
		settings.Provider = provider
	}
	settings.Model = strings.TrimSpace(values[settingModel])
	settings.APIKeyConfigured = s.APIKeyConfigured()
	return settings, nil
}

func (s *Store) SaveSettings(input SaveSettingsInput) (Settings, error) {
	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = defaultProvider
	}
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
		Model:            model,
		APIKeyConfigured: s.APIKeyConfigured(),
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

	if err := upsertSettingTx(tx, settingProvider, defaultProvider); err != nil {
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
