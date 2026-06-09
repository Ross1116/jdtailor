package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	factStatusApproved    = "approved"
	factStatusNeedsReview = "needs_review"
	factStatusRejected    = "rejected"
)

type CandidateProfile struct {
	Contact CandidateContact         `json:"contact"`
	Records []CandidateProfileRecord `json:"records"`
}

type CandidateContact struct {
	FullName  string   `json:"full_name"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Location  string   `json:"location"`
	LinkedIn  string   `json:"linkedin"`
	GitHub    string   `json:"github"`
	Portfolio string   `json:"portfolio"`
	Links     []string `json:"links"`
	Verified  bool     `json:"verified"`
	UpdatedAt string   `json:"updated_at"`
}

type CandidateProfileRecord struct {
	ID           int64  `json:"id"`
	RecordType   string `json:"record_type"`
	Label        string `json:"label"`
	Organization string `json:"organization"`
	Role         string `json:"role"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Value        string `json:"value"`
	Verified     bool   `json:"verified"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CandidateSource struct {
	ID         int64  `json:"id"`
	SourceType string `json:"source_type"`
	Title      string `json:"title"`
	RawText    string `json:"raw_text"`
	FilePath   string `json:"file_path"`
	ImportedAt string `json:"imported_at"`
	UpdatedAt  string `json:"updated_at"`
}

type CreateCandidateSourceInput struct {
	SourceType string `json:"source_type"`
	Title      string `json:"title"`
	RawText    string `json:"raw_text"`
}

type ImportCandidateSourceFileInput struct {
	Path       string `json:"path"`
	SourceType string `json:"source_type"`
	Title      string `json:"title"`
}

type DeleteInput struct {
	ID int64 `json:"id"`
}

type SourceSection struct {
	ID          int64  `json:"id"`
	SourceID    int64  `json:"source_id"`
	Heading     string `json:"heading"`
	SectionType string `json:"section_type"`
	Content     string `json:"content"`
	SortOrder   int    `json:"sort_order"`
	StartChar   int    `json:"start_char"`
	EndChar     int    `json:"end_char"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateSourceSectionInput struct {
	ID          int64  `json:"id"`
	Heading     string `json:"heading"`
	SectionType string `json:"section_type"`
	Content     string `json:"content"`
}

type ExtractEvidenceFactsInput struct {
	SourceID  int64 `json:"source_id"`
	SectionID int64 `json:"section_id"`
}

type EvidenceFact struct {
	ID            int64    `json:"id"`
	SourceID      int64    `json:"source_id"`
	SectionID     int64    `json:"section_id"`
	FactText      string   `json:"fact_text"`
	EvidenceQuote string   `json:"evidence_quote"`
	Technologies  []string `json:"technologies"`
	Confidence    string   `json:"confidence"`
	RiskFlags     []string `json:"risk_flags"`
	OriginHeading string   `json:"origin_heading"`
	OriginType    string   `json:"origin_type"`
	Context       []string `json:"context"`
	Status        string   `json:"status"`
	AutoApproved  bool     `json:"auto_approved"`
	ReviewNote    string   `json:"review_note"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type UpdateEvidenceFactReviewInput struct {
	ID            int64    `json:"id"`
	FactText      string   `json:"fact_text"`
	EvidenceQuote string   `json:"evidence_quote"`
	Technologies  []string `json:"technologies"`
	Confidence    string   `json:"confidence"`
	RiskFlags     []string `json:"risk_flags"`
	Status        string   `json:"status"`
	ReviewNote    string   `json:"review_note"`
}

type extractedFactsResponse struct {
	Facts []extractedFact `json:"facts"`
}

type extractedFact struct {
	FactText      string   `json:"fact_text"`
	EvidenceQuote string   `json:"evidence_quote"`
	Technologies  []string `json:"technologies"`
	Confidence    string   `json:"confidence"`
	RiskFlags     []string `json:"risk_flags"`
	Context       []string `json:"context"`
}

func (s *Store) GetCandidateProfile() (CandidateProfile, error) {
	var profile CandidateProfile
	var linksJSON string
	var contactVerified int
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT full_name, email, phone, location, linkedin, github, portfolio, links_json, verified, updated_at
		FROM candidate_profile WHERE id = 1`,
	).Scan(
		&profile.Contact.FullName,
		&profile.Contact.Email,
		&profile.Contact.Phone,
		&profile.Contact.Location,
		&profile.Contact.LinkedIn,
		&profile.Contact.GitHub,
		&profile.Contact.Portfolio,
		&linksJSON,
		&contactVerified,
		&profile.Contact.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().UTC().Format(time.RFC3339)
		if _, insertErr := s.db.ExecContext(context.Background(), `INSERT INTO candidate_profile (id, updated_at) VALUES (1, ?)`, now); insertErr != nil {
			return CandidateProfile{}, insertErr
		}
		profile.Contact.UpdatedAt = now
	} else if err != nil {
		return CandidateProfile{}, err
	}
	profile.Contact.Links = decodeStringList(linksJSON)
	profile.Contact.Verified = intToBool(contactVerified)

	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, record_type, label, organization, role, start_date, end_date, value, verified, created_at, updated_at
		FROM candidate_profile_records ORDER BY record_type, id`,
	)
	if err != nil {
		return CandidateProfile{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var record CandidateProfileRecord
		var recordVerified int
		if err := rows.Scan(
			&record.ID,
			&record.RecordType,
			&record.Label,
			&record.Organization,
			&record.Role,
			&record.StartDate,
			&record.EndDate,
			&record.Value,
			&recordVerified,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return CandidateProfile{}, err
		}
		record.Verified = intToBool(recordVerified)
		profile.Records = append(profile.Records, record)
	}
	return profile, rows.Err()
}

func (s *Store) SaveCandidateProfile(input CandidateProfile) (CandidateProfile, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	linksJSON, err := encodeStringList(input.Contact.Links)
	if err != nil {
		return CandidateProfile{}, err
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return CandidateProfile{}, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		context.Background(),
		`INSERT INTO candidate_profile
			(id, full_name, email, phone, location, linkedin, github, portfolio, links_json, verified, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			full_name = excluded.full_name,
			email = excluded.email,
			phone = excluded.phone,
			location = excluded.location,
			linkedin = excluded.linkedin,
			github = excluded.github,
			portfolio = excluded.portfolio,
			links_json = excluded.links_json,
			verified = excluded.verified,
			updated_at = excluded.updated_at`,
		strings.TrimSpace(input.Contact.FullName),
		strings.TrimSpace(input.Contact.Email),
		strings.TrimSpace(input.Contact.Phone),
		strings.TrimSpace(input.Contact.Location),
		strings.TrimSpace(input.Contact.LinkedIn),
		strings.TrimSpace(input.Contact.GitHub),
		strings.TrimSpace(input.Contact.Portfolio),
		linksJSON,
		boolToInt(input.Contact.Verified),
		now,
	)
	if err != nil {
		return CandidateProfile{}, err
	}

	if _, err := tx.ExecContext(context.Background(), `DELETE FROM candidate_profile_records`); err != nil {
		return CandidateProfile{}, err
	}
	for _, record := range input.Records {
		recordType := normalizeProfileRecordType(record.RecordType)
		if recordType == "" && strings.TrimSpace(record.Value+record.Label+record.Organization+record.Role) == "" {
			continue
		}
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO candidate_profile_records
				(record_type, label, organization, role, start_date, end_date, value, verified, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			recordType,
			strings.TrimSpace(record.Label),
			strings.TrimSpace(record.Organization),
			strings.TrimSpace(record.Role),
			strings.TrimSpace(record.StartDate),
			strings.TrimSpace(record.EndDate),
			strings.TrimSpace(record.Value),
			boolToInt(record.Verified),
			now,
			now,
		); err != nil {
			return CandidateProfile{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return CandidateProfile{}, err
	}
	_ = s.LogEvent("info", "candidate profile saved")
	return s.GetCandidateProfile()
}

func (s *Store) ListCandidateSources() ([]CandidateSource, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_type, title, raw_text, file_path, imported_at, updated_at
		FROM candidate_sources ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidateSources(rows)
}

func (s *Store) CreateCandidateSource(input CreateCandidateSourceInput) (CandidateSource, error) {
	rawText := normalizeRawSourceText(input.RawText)
	if rawText == "" {
		return CandidateSource{}, errors.New("source text is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Untitled source"
	}
	sourceType := normalizeSourceType(input.SourceType)
	now := time.Now().UTC().Format(time.RFC3339)

	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO candidate_sources (source_type, title, raw_text, file_path, imported_at, updated_at)
		VALUES (?, ?, ?, '', ?, ?)`,
		sourceType,
		title,
		rawText,
		now,
		now,
	)
	if err != nil {
		return CandidateSource{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return CandidateSource{}, err
	}
	_ = s.LogEvent("info", "candidate source imported")
	return s.getCandidateSource(id)
}

func (s *Store) ImportCandidateSourceFile(input ImportCandidateSourceFileInput) (CandidateSource, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return CandidateSource{}, errors.New("file path is required")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".txt" && ext != ".md" && ext != ".markdown" && ext != ".tex" && ext != ".latex" {
		return CandidateSource{}, errors.New("only .txt, .md, .markdown, .tex, and .latex source files are supported")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return CandidateSource{}, err
	}
	source := CreateCandidateSourceInput{
		SourceType: input.SourceType,
		Title:      strings.TrimSpace(input.Title),
		RawText:    string(content),
	}
	if source.Title == "" {
		source.Title = strings.TrimSuffix(filepath.Base(path), ext)
	}
	created, err := s.CreateCandidateSource(source)
	if err != nil {
		return CandidateSource{}, err
	}
	_, err = s.db.ExecContext(
		context.Background(),
		`UPDATE candidate_sources SET file_path = ?, updated_at = ? WHERE id = ?`,
		path,
		time.Now().UTC().Format(time.RFC3339),
		created.ID,
	)
	if err != nil {
		return CandidateSource{}, err
	}
	return s.getCandidateSource(created.ID)
}

func (s *Store) DeleteCandidateSource(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("source id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM candidate_sources WHERE id = ?`, input.ID)
	if err != nil {
		return err
	}
	_ = s.LogEvent("info", "candidate source deleted")
	return nil
}

func (s *Store) ListSourceSections(sourceID int64) ([]SourceSection, error) {
	query := `SELECT id, source_id, heading, section_type, content, sort_order, start_char, end_char, created_at, updated_at FROM source_sections`
	args := []any{}
	if sourceID > 0 {
		query += ` WHERE source_id = ?`
		args = append(args, sourceID)
	}
	query += ` ORDER BY source_id DESC, sort_order, id`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSourceSections(rows)
}

func (s *Store) DetectSourceSections(sourceID int64) ([]SourceSection, error) {
	source, err := s.getCandidateSource(sourceID)
	if err != nil {
		return nil, err
	}
	sections := detectSections(source.RawText)
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM source_sections WHERE source_id = ?`, sourceID); err != nil {
		return nil, err
	}
	for i, section := range sections {
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO source_sections
				(source_id, heading, section_type, content, sort_order, start_char, end_char, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sourceID,
			section.Heading,
			section.SectionType,
			section.Content,
			i,
			section.StartChar,
			section.EndChar,
			now,
			now,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "source sections detected")
	return s.ListSourceSections(sourceID)
}

func (s *Store) UpdateSourceSection(input UpdateSourceSectionInput) (SourceSection, error) {
	if input.ID <= 0 {
		return SourceSection{}, errors.New("section id is required")
	}
	heading := strings.TrimSpace(input.Heading)
	if heading == "" {
		heading = "Untitled section"
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return SourceSection{}, errors.New("section content is required")
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE source_sections
		SET heading = ?, section_type = ?, content = ?, updated_at = ?
		WHERE id = ?`,
		heading,
		normalizeSectionType(input.SectionType, heading),
		content,
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return SourceSection{}, err
	}
	return s.getSourceSection(input.ID)
}

func (s *Store) DeleteSourceSection(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("section id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM source_sections WHERE id = ?`, input.ID)
	if err != nil {
		return err
	}
	_ = s.LogEvent("info", "source section deleted")
	return nil
}

func (s *Store) ExtractEvidenceFacts(ctx context.Context, input ExtractEvidenceFactsInput, client *http.Client) ([]EvidenceFact, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	section, err := s.getSourceSection(input.SectionID)
	if err != nil {
		return nil, err
	}
	if input.SourceID > 0 && section.SourceID != input.SourceID {
		return nil, errors.New("section does not belong to source")
	}
	system := `You are JD Tailor's evidence extraction engine. Return strict JSON only.
Your job is to convert messy resume/source text into small truthful evidence atoms that can be safely reused later.`
	user := fmt.Sprintf(`# Task
Extract atomic, evidence-backed candidate facts from one source section.

# Output JSON schema
{"facts":[{"fact_text":"","evidence_quote":"","technologies":[],"confidence":"high|medium|low","risk_flags":[],"context":[]}]}

# Fact contract
- `+"`evidence_quote`"+` must be an exact quote from <section_content>.
- `+"`fact_text`"+` must be compact key=value atoms, not prose bullets.
- `+"`context`"+` must include useful origin atoms available at extraction time, such as origin_heading, section_type, organization, role, project, dates, or location.
- Split compound bullets into independent facts: core work, scope, environment, tools, metrics, outcomes.
- Prefer hard facts useful for future resume generation: actions, artifacts, tools, domains, scale, audience, metrics, outcomes.
- Ignore headings, company/role/date-only lines, formatting artifacts, and claims not supported by the quote.
- Fill `+"`technologies`"+` only with explicit tools/languages/frameworks/databases/cloud products in the quote.
- Use confidence=high for explicit action/tool/metric/outcome; medium for light inference; low for ambiguous ownership or vague scope.
- Add risk_flags from this controlled set when applicable: unclear_ownership, unclear_metric, ambiguous_technology, leadership_implication, production_vs_project_ambiguity.

# Good fact_text examples
- actions=added; artifact=load-test coverage; tools=Locust
- scope=login, project listing, booking creation; tools=Locust
- metric=25%%; outcome=reduced manual workload

# Good context examples
- origin_heading=Sitespace - Backend Engineer (Founding Team)
- section_type=experience
- organization=Sitespace
- role=Backend Engineer (Founding Team)

# Bad outputs
- Full sentence resume bullets.
- Fact text identical to evidence_quote.
- Technologies inferred from the project title rather than the quote.

<section heading="%s" type="%s">
%s
</section>`, section.Heading, section.SectionType, compactPromptText(section.Content, 12000))

	text, err := s.GenerateLLMText(ctx, client, system, user, 1600)
	if err != nil {
		parsed := fallbackFactsFromSection(section)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM request failed and no local facts could be extracted: %w", err)
		}
		_ = s.LogEvent("warning", "fact extraction used local fallback after LLM request failure: "+err.Error())
		inserted, insertErr := s.insertExtractedFacts(section, refineExtractedFacts(section, parsed))
		if insertErr != nil {
			return nil, insertErr
		}
		_ = s.LogEvent("info", "evidence facts extracted")
		return inserted, nil
	}
	parsed, err := parseExtractedFacts(text)
	if err != nil {
		parsed = fallbackFactsFromSection(section)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM returned unusable fact JSON and no local facts could be extracted: %w", err)
		}
		_ = s.LogEvent("warning", "fact extraction used local fallback: "+err.Error())
	}
	parsed = refineExtractedFacts(section, parsed)
	inserted, err := s.insertExtractedFacts(section, parsed)
	if err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "evidence facts extracted")
	return inserted, nil
}

func (s *Store) ListEvidenceFacts(status string) ([]EvidenceFact, error) {
	query := `SELECT id, source_id, section_id, fact_text, evidence_quote, technologies_json, confidence, risk_flags_json, origin_heading, origin_type, context_json, status, auto_approved, review_note, created_at, updated_at FROM evidence_facts`
	args := []any{}
	if strings.TrimSpace(status) != "" && status != "all" {
		query += ` WHERE status = ?`
		args = append(args, normalizeFactStatus(status))
	}
	query += ` ORDER BY CASE status WHEN 'needs_review' THEN 0 WHEN 'approved' THEN 1 ELSE 2 END, id DESC`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvidenceFacts(rows)
}

func (s *Store) UpdateEvidenceFactReview(input UpdateEvidenceFactReviewInput) (EvidenceFact, error) {
	if input.ID <= 0 {
		return EvidenceFact{}, errors.New("fact id is required")
	}
	factText := strings.TrimSpace(input.FactText)
	evidenceQuote := strings.TrimSpace(input.EvidenceQuote)
	if factText == "" || evidenceQuote == "" {
		return EvidenceFact{}, errors.New("fact text and evidence quote are required")
	}
	techJSON, err := encodeStringList(input.Technologies)
	if err != nil {
		return EvidenceFact{}, err
	}
	riskJSON, err := encodeStringList(input.RiskFlags)
	if err != nil {
		return EvidenceFact{}, err
	}
	_, err = s.db.ExecContext(
		context.Background(),
		`UPDATE evidence_facts
		SET fact_text = ?, evidence_quote = ?, technologies_json = ?, confidence = ?, risk_flags_json = ?,
			status = ?, auto_approved = 0, review_note = ?, updated_at = ?
		WHERE id = ?`,
		factText,
		evidenceQuote,
		techJSON,
		normalizeConfidence(input.Confidence),
		riskJSON,
		normalizeFactStatus(input.Status),
		strings.TrimSpace(input.ReviewNote),
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return EvidenceFact{}, err
	}
	return s.getEvidenceFact(input.ID)
}

func (s *Store) DeleteEvidenceFact(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("fact id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM evidence_facts WHERE id = ?`, input.ID)
	if err != nil {
		return err
	}
	_ = s.LogEvent("info", "evidence fact deleted")
	return nil
}

func (s *Store) DeleteAllEvidenceFacts() error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, statement := range []string{
		`DELETE FROM tailored_bullet_drafts`,
		`DELETE FROM job_fact_matches`,
		`DELETE FROM job_fit_analyses`,
		`DELETE FROM application_strategies`,
		`DELETE FROM evidence_facts`,
	} {
		if _, err := tx.ExecContext(context.Background(), statement); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = s.LogEvent("info", "all evidence facts deleted")
	return nil
}

func (s *Store) DraftCandidateProfileFromSource(sourceID int64) (CandidateProfile, error) {
	source, err := s.getCandidateSource(sourceID)
	if err != nil {
		return CandidateProfile{}, err
	}
	profile := draftCandidateProfile(source.RawText)
	_ = s.LogEvent("info", "candidate profile drafted from source")
	return profile, nil
}

func (s *Store) insertExtractedFacts(section SourceSection, facts []extractedFact) ([]EvidenceFact, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ids := make([]int64, 0, len(facts))
	for _, fact := range facts {
		fact = enrichExtractedFact(section, fact)
		fact.FactText = strings.TrimSpace(fact.FactText)
		fact.EvidenceQuote = strings.TrimSpace(fact.EvidenceQuote)
		if fact.FactText == "" || fact.EvidenceQuote == "" {
			continue
		}
		if !strings.Contains(section.Content, fact.EvidenceQuote) {
			repairedQuote := closestEvidenceQuote(section.Content, fact.EvidenceQuote)
			if repairedQuote == "" {
				return nil, fmt.Errorf("evidence quote is not present in source section: %q", fact.EvidenceQuote)
			}
			fact.EvidenceQuote = repairedQuote
		}
		confidence := normalizeConfidence(fact.Confidence)
		riskFlags := normalizeStringList(fact.RiskFlags)
		status, autoApproved := reviewStatus(confidence, riskFlags)
		techJSON, err := encodeStringList(fact.Technologies)
		if err != nil {
			return nil, err
		}
		riskJSON, err := encodeStringList(riskFlags)
		if err != nil {
			return nil, err
		}
		contextAtoms := factContextAtoms(section, fact)
		contextJSON, err := encodeStringList(contextAtoms)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO evidence_facts
				(source_id, section_id, fact_text, evidence_quote, technologies_json, confidence, risk_flags_json, origin_heading, origin_type, context_json, status, auto_approved, review_note, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?)`,
			section.SourceID,
			section.ID,
			fact.FactText,
			fact.EvidenceQuote,
			techJSON,
			confidence,
			riskJSON,
			strings.TrimSpace(section.Heading),
			normalizeSectionType(section.SectionType, section.Heading),
			contextJSON,
			status,
			boolToInt(autoApproved),
			now,
			now,
		)
		if err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	inserted := make([]EvidenceFact, 0, len(ids))
	for _, id := range ids {
		fact, err := s.getEvidenceFact(id)
		if err != nil {
			return nil, err
		}
		inserted = append(inserted, fact)
	}
	return inserted, nil
}

func parseExtractedFacts(text string) ([]extractedFact, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("LLM returned empty fact response")
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}
	if !strings.HasPrefix(strings.TrimSpace(text), "{") {
		return nil, errors.New("LLM fact response did not contain a JSON object")
	}
	var parsed extractedFactsResponse
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("LLM fact JSON could not be parsed: %w", err)
	}
	if len(parsed.Facts) == 0 {
		return nil, errors.New("LLM returned no facts")
	}
	for _, fact := range parsed.Facts {
		if strings.TrimSpace(fact.FactText) == "" || strings.TrimSpace(fact.EvidenceQuote) == "" {
			return nil, errors.New("every fact must include fact_text and evidence_quote")
		}
	}
	return parsed.Facts, nil
}

func refineExtractedFacts(section SourceSection, facts []extractedFact) []extractedFact {
	filtered := make([]extractedFact, 0, len(facts))
	for _, fact := range facts {
		fact = enrichExtractedFact(section, fact)
		if isTrivialExtractedFact(section, fact) {
			continue
		}
		filtered = append(filtered, fact)
	}
	if len(filtered) > 0 {
		return filtered
	}
	return fallbackFactsFromSection(section)
}

func isTrivialExtractedFact(section SourceSection, fact extractedFact) bool {
	content := strings.TrimSpace(section.Content)
	factText := strings.TrimSpace(fact.FactText)
	quote := strings.TrimSpace(fact.EvidenceQuote)
	if quote == "" || factText == "" {
		return true
	}
	if strings.EqualFold(quote, content) || strings.EqualFold(factText, content) {
		return true
	}
	for _, line := range metadataLines(section) {
		if strings.EqualFold(quote, line) || strings.EqualFold(factText, line) {
			return true
		}
	}
	return false
}

func fallbackFactsFromSection(section SourceSection) []extractedFact {
	lines := nonEmptyLines(section.Content)
	metadataCount := metadataContentLineCount(section)
	facts := []extractedFact{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isKnownSectionHeading(trimmed) || i < metadataCount {
			continue
		}
		factText := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if len(factText) < 12 {
			continue
		}
		facts = append(facts, atomicFactsFromLine(section, factText, trimmed)...)
	}
	return facts
}

func atomicFactsFromLine(section SourceSection, factText string, evidenceQuote string) []extractedFact {
	technologies := extractTechnologies(evidenceQuote)
	riskFlags := inferEvidenceRiskFlags(evidenceQuote)
	confidence := inferEvidenceConfidence(evidenceQuote, technologies, riskFlags)
	base := extractedFact{
		EvidenceQuote: evidenceQuote,
		Technologies:  technologies,
		Confidence:    confidence,
		RiskFlags:     riskFlags,
	}
	facts := []extractedFact{}
	if core := compactCoreEvidenceFact(factText); core != "" {
		item := base
		item.FactText = core
		facts = append(facts, item)
	}
	if scope := inferScope(factText); scope != "" {
		item := base
		item.FactText = keyValueFact("scope", scope, "tools", strings.Join(technologies, ", "))
		facts = append(facts, item)
	}
	if environment := inferEnvironment(factText); environment != "" {
		item := base
		item.FactText = keyValueFact("environment", environment, "tools", strings.Join(technologies, ", "))
		facts = append(facts, item)
	}
	if outcome := inferOutcome(factText); outcome != "" || len(extractFigures(factText)) > 0 {
		item := base
		item.FactText = keyValueFact("metric", strings.Join(extractFigures(factText), ", "), "outcome", outcome)
		facts = append(facts, item)
	}
	if len(facts) == 0 {
		item := base
		item.FactText = "evidence=" + strings.Trim(strings.TrimSpace(factText), ".")
		facts = append(facts, item)
	}
	return dedupeExtractedFacts(facts)
}

func enrichExtractedFact(section SourceSection, fact extractedFact) extractedFact {
	fact.EvidenceQuote = strings.TrimSpace(fact.EvidenceQuote)
	fact.FactText = strings.TrimSpace(fact.FactText)
	quoteWithoutBullet := strings.TrimSpace(strings.TrimPrefix(fact.EvidenceQuote, "- "))
	if fact.FactText == "" || strings.EqualFold(fact.FactText, fact.EvidenceQuote) || strings.EqualFold(fact.FactText, quoteWithoutBullet) || looksLikeSentence(fact.FactText) {
		source := firstNonEmpty(quoteWithoutBullet, fact.FactText)
		fact.FactText = compactEvidenceFact(source, section)
	}
	if len(fact.Technologies) == 0 {
		fact.Technologies = extractTechnologies(firstNonEmpty(fact.EvidenceQuote, fact.FactText))
	}
	fact.RiskFlags = normalizeStringList(append(fact.RiskFlags, inferEvidenceRiskFlags(firstNonEmpty(fact.EvidenceQuote, fact.FactText))...))
	if strings.TrimSpace(fact.Confidence) == "" || strings.EqualFold(fact.Confidence, "medium") {
		fact.Confidence = inferEvidenceConfidence(firstNonEmpty(fact.EvidenceQuote, fact.FactText), fact.Technologies, fact.RiskFlags)
	}
	fact.Context = factContextAtoms(section, fact)
	return fact
}

func factContextAtoms(section SourceSection, fact extractedFact) []string {
	atoms := []string{}
	if heading := strings.TrimSpace(section.Heading); heading != "" {
		atoms = append(atoms, "origin_heading="+heading)
	}
	sectionType := normalizeSectionType(section.SectionType, section.Heading)
	if sectionType != "" {
		atoms = append(atoms, "section_type="+sectionType)
	}
	atoms = append(atoms, sectionMetadataContext(section)...)
	atoms = append(atoms, fact.Context...)
	return normalizeStringList(atoms)
}

func sectionMetadataContext(section SourceSection) []string {
	lines := metadataLines(section)
	if len(lines) == 0 {
		return nil
	}
	sectionType := normalizeSectionType(section.SectionType, section.Heading)
	atoms := []string{}
	firstParts := splitPipeLine(lines[0])
	secondParts := []string{}
	if len(lines) > 1 {
		secondParts = splitPipeLine(lines[1])
	}
	switch sectionType {
	case "experience":
		if len(firstParts) > 0 {
			atoms = append(atoms, "organization="+firstParts[0])
		}
		if len(firstParts) > 1 {
			atoms = append(atoms, "location="+strings.Join(firstParts[1:], " | "))
		}
		if len(secondParts) > 0 {
			atoms = append(atoms, "role="+secondParts[0])
		}
		if len(secondParts) > 1 {
			atoms = append(atoms, "dates="+secondParts[len(secondParts)-1])
		}
	case "project":
		if len(firstParts) > 0 {
			atoms = append(atoms, "project="+firstParts[0])
		}
		if len(firstParts) > 1 {
			atoms = append(atoms, "project_context="+strings.Join(firstParts[1:], " | "))
		}
	case "education":
		if len(firstParts) > 0 {
			atoms = append(atoms, "organization="+firstParts[0])
		}
		if len(secondParts) > 0 {
			atoms = append(atoms, "credential="+secondParts[0])
		}
		if len(secondParts) > 1 {
			atoms = append(atoms, "dates="+secondParts[len(secondParts)-1])
		}
	default:
		if len(firstParts) > 0 && strings.Contains(lines[0], "|") {
			atoms = append(atoms, "context_line="+strings.Join(firstParts, " | "))
		}
	}
	return atoms
}

func closestEvidenceQuote(sectionContent string, quote string) string {
	quoteTerms := evidenceQuoteTerms(quote)
	if len(quoteTerms) == 0 {
		return ""
	}
	bestLine := ""
	bestScore := 0
	for _, line := range strings.Split(sectionContent, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" || isKnownSectionHeading(candidate) {
			continue
		}
		score := 0
		candidateLower := strings.ToLower(candidate)
		for _, term := range quoteTerms {
			if strings.Contains(candidateLower, term) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestLine = candidate
		}
	}
	if bestScore >= minInt(4, len(quoteTerms)) || bestScore*2 >= len(quoteTerms) {
		return bestLine
	}
	return ""
}

func evidenceQuoteTerms(value string) []string {
	stop := map[string]bool{}
	for _, word := range []string{"the", "and", "for", "with", "from", "that", "this", "into", "core", "logic"} {
		stop[word] = true
	}
	terms := []string{}
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '#'
	}) {
		if len(token) < 4 || stop[token] {
			continue
		}
		terms = append(terms, token)
	}
	return normalizeStringList(terms)
}

func looksLikeSentence(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, ";") || strings.Contains(value, ":") {
		return false
	}
	return strings.HasSuffix(value, ".") || len(strings.Fields(value)) > 9
}

func compactEvidenceFact(text string, section SourceSection) string {
	atoms := atomicFactsFromLine(section, text, firstNonEmpty(text))
	if len(atoms) > 0 {
		return atoms[0].FactText
	}
	return compactCoreEvidenceFact(text)
}

func compactCoreEvidenceFact(text string) string {
	text = strings.Trim(strings.TrimSpace(strings.TrimPrefix(text, "- ")), ".")
	technologies := extractTechnologies(text)
	pairs := []string{}
	if actions := extractActions(text); len(actions) > 0 {
		pairs = append(pairs, "actions="+strings.Join(actions, ", "))
	}
	if artifact := inferArtifact(text, technologies); artifact != "" {
		pairs = append(pairs, "artifact="+artifact)
	}
	if len(technologies) > 0 {
		pairs = append(pairs, "tools="+strings.Join(technologies, ", "))
	}
	if len(pairs) == 0 {
		pairs = append(pairs, "evidence="+text)
	}
	return strings.Join(pairs, "; ")
}

func inferAction(text string) string {
	actions := extractActions(text)
	if len(actions) == 0 {
		return ""
	}
	return actions[0]
}

func extractActions(text string) []string {
	lower := strings.ToLower(text)
	actions := []string{"built", "shipped", "added", "implemented", "designed", "developed", "created", "improved", "reduced", "migrated", "automated", "supported", "tested", "integrated", "delivered", "optimized", "deployed"}
	found := []string{}
	for _, action := range actions {
		if strings.Contains(lower, action+" ") || strings.HasPrefix(lower, action) {
			found = append(found, action)
		}
	}
	return normalizeStringList(found)
}

func inferArtifact(text string, technologies []string) string {
	cleaned := strings.Trim(strings.TrimSpace(strings.TrimPrefix(text, "- ")), ".")
	lower := strings.ToLower(cleaned)
	actions := extractActions(cleaned)
	if len(actions) > 0 {
		actionIndex := strings.Index(lower, actions[0])
		if actionIndex >= 0 {
			cleaned = strings.TrimSpace(cleaned[actionIndex+len(actions[0]):])
		}
	}
	for strings.HasPrefix(strings.ToLower(cleaned), "and ") || strings.HasPrefix(strings.ToLower(cleaned), "shipped ") || strings.HasPrefix(strings.ToLower(cleaned), "built ") {
		_, rest, ok := strings.Cut(cleaned, " ")
		if !ok {
			break
		}
		cleaned = strings.TrimSpace(rest)
	}
	for _, stop := range []string{" for ", " across ", " against ", " using ", " with ", " by ", " through ", " to ", ", reducing ", ", improving "} {
		if index := strings.Index(strings.ToLower(cleaned), stop); index > 0 {
			cleaned = cleaned[:index]
		}
	}
	for _, tech := range technologies {
		cleaned = regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(tech)+`\b`).ReplaceAllString(cleaned, "")
	}
	cleaned = regexp.MustCompile(`[ /]+`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

func inferScope(text string) string {
	cleaned := strings.Trim(strings.TrimSpace(strings.TrimPrefix(text, "- ")), ".")
	lower := strings.ToLower(cleaned)
	for _, marker := range []string{" for ", " across ", " covering ", " coverage for "} {
		if index := strings.Index(lower, marker); index >= 0 {
			scope := cleaned[index+len(marker):]
			for _, stop := range []string{" against ", " using ", " with ", " by ", " through ", " to "} {
				if stopIndex := strings.Index(strings.ToLower(scope), stop); stopIndex > 0 {
					scope = scope[:stopIndex]
				}
			}
			return strings.TrimSpace(scope)
		}
	}
	return ""
}

func inferEnvironment(text string) string {
	cleaned := strings.Trim(strings.TrimSpace(strings.TrimPrefix(text, "- ")), ".")
	lower := strings.ToLower(cleaned)
	for _, marker := range []string{" against ", " in ", " on "} {
		index := strings.Index(lower, marker)
		if index < 0 {
			continue
		}
		candidate := strings.TrimSpace(cleaned[index+len(marker):])
		candidateLower := strings.ToLower(candidate)
		if strings.Contains(candidateLower, "workflow") || strings.Contains(candidateLower, "production") || strings.Contains(candidateLower, "staging") || strings.Contains(candidateLower, "linux") || strings.Contains(candidateLower, "internal system") {
			return candidate
		}
	}
	return ""
}

func inferOutcome(text string) string {
	cleaned := strings.Trim(strings.TrimSpace(strings.TrimPrefix(text, "- ")), ".")
	lower := strings.ToLower(cleaned)
	for _, marker := range []string{" reducing ", " reduced ", " improving ", " improved ", " delivering ", " delivered ", " enabling ", " enabled "} {
		if index := strings.Index(lower, marker); index >= 0 {
			return strings.TrimSpace(cleaned[index+1:])
		}
	}
	if strings.Contains(lower, "coverage") {
		return "test coverage"
	}
	if strings.Contains(lower, "production") {
		return "production delivery"
	}
	return ""
}

func keyValueFact(parts ...string) string {
	pairs := []string{}
	for i := 0; i+1 < len(parts); i += 2 {
		key := strings.TrimSpace(parts[i])
		value := strings.Trim(strings.TrimSpace(parts[i+1]), ".")
		if key == "" || value == "" {
			continue
		}
		pairs = append(pairs, key+"="+value)
	}
	return strings.Join(pairs, "; ")
}

func dedupeExtractedFacts(facts []extractedFact) []extractedFact {
	seen := map[string]bool{}
	next := []extractedFact{}
	for _, fact := range facts {
		key := strings.ToLower(strings.TrimSpace(fact.FactText))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		next = append(next, fact)
	}
	return next
}

func extractFigures(text string) []string {
	matches := regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*(?:%|percent|ms|s|sec|seconds|min|minutes|x|k|m|hours?|days?|weeks?|months?|years?)\b`).FindAllString(text, -1)
	return normalizeStringList(matches)
}

func extractTechnologies(text string) []string {
	known := []string{
		"FastAPI", "PostgreSQL", "React", "TypeScript", "JavaScript", "Python", "Go", "Golang", "Java", "C#", "C++", "Node", "Node.js", "Express",
		"Linux", "Docker", "Kubernetes", "AWS", "Azure", "GCP", "Terraform", "Postgres", "SQLite", "MySQL", "Redis", "Locust", "Playwright",
		"Tailwind", "Vite", "Wails", "GitHub Actions", "CI/CD", "REST", "GraphQL", "RBAC", "SQL", "NoSQL", "MongoDB", "DynamoDB",
	}
	found := []string{}
	lower := strings.ToLower(text)
	for _, tech := range known {
		if strings.Contains(lower, strings.ToLower(tech)) {
			if tech == "Golang" {
				tech = "Go"
			}
			if tech == "Postgres" {
				tech = "PostgreSQL"
			}
			found = append(found, tech)
		}
	}
	return normalizeStringList(found)
}

func inferEvidenceRiskFlags(text string) []string {
	lower := strings.ToLower(text)
	flags := []string{}
	if strings.Contains(lower, "approximately") || strings.Contains(lower, "around ") || strings.Contains(lower, "~") {
		flags = append(flags, "unclear_metric")
	}
	if strings.Contains(lower, "supported") && !strings.Contains(lower, "built") && !strings.Contains(lower, "implemented") {
		flags = append(flags, "unclear_ownership")
	}
	if strings.Contains(lower, "staging-style") || strings.Contains(lower, "prototype") {
		flags = append(flags, "production_vs_project_ambiguity")
	}
	return normalizeStringList(flags)
}

func inferEvidenceConfidence(text string, technologies []string, riskFlags []string) string {
	if len(riskFlags) > 0 {
		return "medium"
	}
	if len(technologies) > 0 || len(extractFigures(text)) > 0 {
		return "high"
	}
	if strings.Contains(strings.ToLower(text), "built") || strings.Contains(strings.ToLower(text), "shipped") || strings.Contains(strings.ToLower(text), "implemented") {
		return "high"
	}
	return "medium"
}

func metadataLines(section SourceSection) []string {
	lines := nonEmptyLines(section.Content)
	count := metadataContentLineCount(section)
	metadata := make([]string, 0, count+1)
	if section.Heading != "" {
		metadata = append(metadata, strings.TrimSpace(section.Heading))
	}
	for i := 0; i < count; i++ {
		metadata = append(metadata, strings.TrimSpace(lines[i]))
	}
	return metadata
}

func metadataContentLineCount(section SourceSection) int {
	lines := nonEmptyLines(section.Content)
	switch section.SectionType {
	case "experience", "education":
		return minInt(2, len(lines))
	case "project":
		return minInt(1, len(lines))
	default:
		return 0
	}
}

func normalizeRawSourceText(rawText string) string {
	text := strings.TrimSpace(rawText)
	if text == "" {
		return ""
	}
	if looksLikeLatex(text) {
		return cleanLatexText(text)
	}
	return text
}

func looksLikeLatex(text string) bool {
	return strings.Contains(text, `\documentclass`) ||
		strings.Contains(text, `\begin{document}`) ||
		strings.Contains(text, `\section{`) ||
		strings.Contains(text, `\resumeItem`)
}

func cleanLatexText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if start := strings.Index(text, `\begin{document}`); start >= 0 {
		text = text[start+len(`\begin{document}`):]
	}
	if end := strings.Index(text, `\end{document}`); end >= 0 {
		text = text[:end]
	}
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%") {
			continue
		}
		lines = append(lines, stripLatexComment(line))
	}
	text = strings.Join(lines, "\n")

	text = expandLatexCommand(text, "section", func(args []string) string {
		return "\n" + args[0] + "\n"
	})
	text = expandLatexCommand(text, "resumeSubheading", func(args []string) string {
		return fmt.Sprintf("\n%s | %s\n%s | %s\n", args[0], args[1], args[2], args[3])
	})
	text = expandLatexCommand(text, "resumeProjectHeading", func(args []string) string {
		return fmt.Sprintf("\n%s | %s\n", args[0], args[1])
	})
	text = expandLatexCommand(text, "resumeItem", func(args []string) string {
		return "- " + args[0] + "\n"
	})
	text = expandLatexCommand(text, "href", func(args []string) string {
		return args[1]
	})
	for _, command := range []string{"textbf", "textit", "emph", "small", "scshape"} {
		text = expandLatexCommand(text, command, func(args []string) string {
			return strings.Join(args, "")
		})
	}

	text = regexp.MustCompile(`\\begin\{[^}]+\}(\[[^]]+\])?`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\\end\{[^}]+\}`).ReplaceAllString(text, "")
	replacements := map[string]string{
		`$|$`:                        " | ",
		`\\`:                         "\n",
		`\&`:                         "&",
		`\%`:                         "%",
		`\_`:                         "_",
		`\#`:                         "#",
		`\$`:                         "$",
		`\vspace{-4pt}`:              "",
		`\vspace{-5pt}`:              "",
		`\vspace{-6pt}`:              "",
		`\vspace{1pt}`:               "",
		`\vspace{3pt}`:               "",
		`\vspace{4pt}`:               "",
		`\resumeSubHeadingListStart`: "",
		`\resumeSubHeadingListEnd`:   "",
		`\resumeItemListStart`:       "",
		`\resumeItemListEnd`:         "",
		`\begin{center}`:             "",
		`\end{center}`:               "",
		`\begin{itemize}`:            "",
		`\end{itemize}`:              "",
	}
	for old, replacement := range replacements {
		text = strings.ReplaceAll(text, old, replacement)
	}
	text = regexp.MustCompile(`(?m)^[ \t]*\[[^]\n]*(?:leftmargin|label)[^]\n]*\][ \t]*$`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\\[a-zA-Z]+(\[[^]]+\])?`).ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`(?m)^[ \t]+`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	text = ensureKnownSectionHeadingsOnLines(text)
	return strings.TrimSpace(text)
}

func stripLatexComment(line string) string {
	for i, r := range line {
		if r == '%' && (i == 0 || line[i-1] != '\\') {
			return line[:i]
		}
	}
	return line
}

func expandLatexCommand(text string, command string, format func([]string) string) string {
	searchStart := 0
	for {
		relativeIndex := strings.Index(text[searchStart:], `\`+command)
		if relativeIndex < 0 {
			return text
		}
		index := searchStart + relativeIndex
		commandEnd := index + len(command) + 1
		if commandEnd < len(text) && isLatexCommandLetter(text[commandEnd]) {
			searchStart = commandEnd
			continue
		}
		args, end, ok := readLatexCommandArgs(text, index+len(command)+1)
		if !ok || len(args) == 0 {
			return text[:index] + text[index+len(command)+1:]
		}
		text = text[:index] + format(args) + text[end:]
		searchStart = index
	}
}

func isLatexCommandLetter(char byte) bool {
	return (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
}

func readLatexCommandArgs(text string, index int) ([]string, int, bool) {
	args := []string{}
	for {
		for index < len(text) && (text[index] == ' ' || text[index] == '\n' || text[index] == '\t') {
			index++
		}
		if index >= len(text) || text[index] != '{' {
			break
		}
		depth := 0
		start := index + 1
		for index < len(text) {
			switch text[index] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					args = append(args, cleanLatexTextFragment(text[start:index]))
					index++
					goto nextArg
				}
			}
			index++
		}
		return args, index, false
	nextArg:
	}
	return args, index, len(args) > 0
}

func cleanLatexTextFragment(text string) string {
	text = expandLatexCommand(text, "href", func(args []string) string {
		return args[1]
	})
	for _, command := range []string{"textbf", "textit", "emph", "small"} {
		text = expandLatexCommand(text, command, func(args []string) string {
			return strings.Join(args, "")
		})
	}
	text = strings.ReplaceAll(text, `$|$`, " | ")
	text = strings.ReplaceAll(text, `\&`, "&")
	text = strings.ReplaceAll(text, `\%`, "%")
	text = regexp.MustCompile(`\\[a-zA-Z]+`).ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func ensureKnownSectionHeadingsOnLines(text string) string {
	headings := []string{
		"Professional Summary",
		"Technical Skills",
		"Work Experience",
		"Other Projects",
		"Experience",
		"Projects",
		"Education",
		"Certifications",
	}
	lines := strings.Split(text, "\n")
	nextLines := make([]string, 0, len(lines))
	for _, line := range lines {
		remaining := strings.TrimSpace(line)
		if remaining == "" {
			nextLines = append(nextLines, "")
			continue
		}
		for remaining != "" {
			matchedHeading := ""
			matchedIndex := -1
			for _, heading := range headings {
				index := strings.Index(remaining, heading)
				if index < 0 || !headingBoundary(remaining, index, heading) {
					continue
				}
				if matchedIndex < 0 || index < matchedIndex || (index == matchedIndex && len(heading) > len(matchedHeading)) {
					matchedHeading = heading
					matchedIndex = index
				}
			}
			if matchedIndex < 0 {
				nextLines = append(nextLines, remaining)
				break
			}
			prefix := strings.TrimSpace(remaining[:matchedIndex])
			if prefix != "" {
				nextLines = append(nextLines, prefix)
			}
			nextLines = append(nextLines, matchedHeading)
			remaining = strings.TrimSpace(remaining[matchedIndex+len(matchedHeading):])
		}
	}
	text = strings.Join(nextLines, "\n")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func headingBoundary(text string, index int, heading string) bool {
	afterIndex := index + len(heading)
	if afterIndex >= len(text) {
		return true
	}
	after := text[afterIndex : afterIndex+1]
	return after == ":" || after == "." || after == " " || after == "-" || regexp.MustCompile(`[A-Z0-9]`).MatchString(after)
}

func draftCandidateProfile(text string) CandidateProfile {
	text = normalizeRawSourceText(text)
	lines := nonEmptyLines(text)
	profile := CandidateProfile{
		Contact: CandidateContact{Verified: false},
		Records: []CandidateProfileRecord{},
	}
	if len(lines) > 0 && !isKnownSectionHeading(lines[0]) {
		profile.Contact.FullName = strings.TrimSpace(lines[0])
	}
	profile.Contact.Email = firstRegex(text, `[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	profile.Contact.Phone = firstRegex(text, `(?:\+\d{1,3}\s*)?(?:\(?\d{2,4}\)?[\s-]*)?\d{3}[\s-]*\d{3}[\s-]*\d{3,4}`)
	profile.Contact.LinkedIn = firstRegex(text, `(?:https?://)?(?:www\.)?linkedin\.com/in/[A-Za-z0-9_\-./]+`)
	profile.Contact.GitHub = firstRegex(text, `(?:https?://)?(?:www\.)?github\.com/[A-Za-z0-9_\-./]+`)
	if location := inferLocation(lines); location != "" {
		profile.Contact.Location = location
	}

	profile.Records = append(profile.Records, draftRecordsFromSection(text, "Experience", "employment")...)
	profile.Records = append(profile.Records, draftRecordsFromSection(text, "Projects", "project")...)
	profile.Records = append(profile.Records, draftRecordsFromSection(text, "Education", "education")...)
	return profile
}

func nonEmptyLines(text string) []string {
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func firstRegex(text string, pattern string) string {
	match := regexp.MustCompile(pattern).FindString(text)
	return strings.Trim(strings.TrimSpace(match), ".,;")
}

func inferLocation(lines []string) string {
	for _, line := range lines[:minInt(len(lines), 8)] {
		if strings.Contains(strings.ToLower(line), "melbourne") || strings.Contains(strings.ToLower(line), "remote") {
			parts := strings.Split(line, "|")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return ""
}

func draftRecordsFromSection(text string, heading string, recordType string) []CandidateProfileRecord {
	section := extractNamedSection(text, heading)
	if section == "" {
		return nil
	}
	lines := nonEmptyLines(section)
	records := []CandidateProfileRecord{}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "- ") || isKnownSectionHeading(line) {
			continue
		}
		if !strings.Contains(line, "|") && recordType != "education" {
			continue
		}
		record := CandidateProfileRecord{RecordType: recordType, Verified: false}
		parts := splitPipeLine(line)
		if len(parts) > 0 {
			record.Organization = parts[0]
			record.Label = parts[0]
		}
		if len(parts) > 1 {
			record.Value = strings.Join(parts[1:], " | ")
		}
		if i+1 < len(lines) && strings.Contains(lines[i+1], "|") && !strings.HasPrefix(lines[i+1], "- ") {
			detailParts := splitPipeLine(lines[i+1])
			if len(detailParts) > 0 {
				record.Role = detailParts[0]
			}
			if len(detailParts) > 1 {
				record.StartDate, record.EndDate = splitDateRange(detailParts[len(detailParts)-1])
			}
			i++
		}
		if recordType == "education" {
			if len(parts) > 1 {
				record.Role = parts[1]
			}
			if i+1 < len(lines) && strings.Contains(lines[i+1], "|") {
				detailParts := splitPipeLine(lines[i+1])
				if len(detailParts) > 0 {
					record.Role = detailParts[0]
				}
				if len(detailParts) > 1 {
					record.StartDate, record.EndDate = splitDateRange(detailParts[len(detailParts)-1])
				}
				i++
			}
		}
		if record.Organization != "" || record.Role != "" || record.Value != "" {
			records = append(records, record)
		}
	}
	return records
}

func extractNamedSection(text string, heading string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), heading) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if isKnownSectionHeading(strings.TrimSpace(lines[i])) {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func isKnownSectionHeading(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch lower {
	case "professional summary", "summary", "technical skills", "skills", "experience", "projects", "education":
		return true
	default:
		return false
	}
}

func splitPipeLine(line string) []string {
	parts := []string{}
	for _, part := range strings.Split(line, "|") {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitDateRange(value string) (string, string) {
	parts := strings.Split(value, "--")
	if len(parts) == 1 {
		parts = strings.Split(value, "-")
	}
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", strings.TrimSpace(value)
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func reviewStatus(confidence string, riskFlags []string) (string, bool) {
	if confidence == "high" && len(riskFlags) == 0 {
		return factStatusApproved, true
	}
	return factStatusNeedsReview, false
}

func detectSections(rawText string) []SourceSection {
	text := normalizeRawSourceText(rawText)
	text = ensureKnownSectionHeadingsOnLines(text)
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	type marker struct {
		lineIndex int
		heading   string
	}
	markers := []marker{}
	offset := 0
	lineOffsets := make([]int, len(lines))
	for i, line := range lines {
		lineOffsets[i] = offset
		trimmed := strings.TrimSpace(line)
		if isSectionHeading(trimmed) {
			markers = append(markers, marker{lineIndex: i, heading: cleanHeading(trimmed)})
		}
		offset += len(line) + 1
	}
	if len(markers) == 0 {
		return []SourceSection{{
			Heading:     "Imported source",
			SectionType: "misc",
			Content:     text,
			SortOrder:   0,
			StartChar:   0,
			EndChar:     len(text),
		}}
	}
	sections := make([]SourceSection, 0, len(markers))
	for i, current := range markers {
		startLine := current.lineIndex + 1
		endLine := len(lines)
		if i+1 < len(markers) {
			endLine = markers[i+1].lineIndex
		}
		content := strings.TrimSpace(strings.Join(lines[startLine:endLine], "\n"))
		if content == "" {
			content = current.heading
		}
		startChar := 0
		if startLine < len(lineOffsets) {
			startChar = lineOffsets[startLine]
		}
		endChar := len(text)
		if endLine < len(lineOffsets) {
			endChar = lineOffsets[endLine]
		}
		section := SourceSection{
			Heading:     current.heading,
			SectionType: normalizeSectionType("", current.heading),
			Content:     content,
			SortOrder:   i,
			StartChar:   startChar,
			EndChar:     endChar,
		}
		sections = append(sections, splitResumeEntrySections(section)...)
	}
	for i := range sections {
		sections[i].SortOrder = i
	}
	return sections
}

func splitResumeEntrySections(section SourceSection) []SourceSection {
	if section.SectionType != "experience" && section.SectionType != "project" && section.SectionType != "education" {
		return []SourceSection{section}
	}
	lines := nonEmptyLines(section.Content)
	if len(lines) == 0 {
		return []SourceSection{section}
	}
	sections := []SourceSection{}
	for i := 0; i < len(lines); {
		for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") {
			i++
		}
		if i >= len(lines) {
			break
		}
		start := i
		i++
		if section.SectionType != "project" && i < len(lines) && isEntryHeaderLine(lines[i]) {
			i++
		}
		for i < len(lines) && !isResumeEntryStart(lines, i, section.SectionType) {
			i++
		}
		entryLines := lines[start:i]
		entryContent := strings.TrimSpace(strings.Join(entryLines, "\n"))
		if entryContent == "" {
			continue
		}
		entry := section
		entry.Heading = resumeEntryHeading(section.SectionType, entryLines, section.Heading)
		entry.Content = entryContent
		sections = append(sections, entry)
	}
	if len(sections) == 0 {
		return []SourceSection{section}
	}
	return sections
}

func isResumeEntryStart(lines []string, index int, sectionType string) bool {
	line := strings.TrimSpace(lines[index])
	if line == "" || strings.HasPrefix(line, "- ") || isKnownSectionHeading(line) {
		return false
	}
	if !isEntryHeaderLine(line) {
		return false
	}
	if sectionType == "project" {
		return true
	}
	return index+1 < len(lines) && isEntryHeaderLine(lines[index+1]) && !strings.HasPrefix(strings.TrimSpace(lines[index+1]), "- ")
}

func isEntryHeaderLine(line string) bool {
	line = strings.TrimSpace(line)
	return line != "" && !strings.HasPrefix(line, "- ") && strings.Contains(line, "|")
}

func resumeEntryHeading(sectionType string, lines []string, fallback string) string {
	if len(lines) == 0 {
		return fallback
	}
	firstParts := splitPipeLine(lines[0])
	name := strings.TrimSpace(lines[0])
	if len(firstParts) > 0 {
		name = firstParts[0]
	}
	if sectionType == "project" {
		return firstNonEmpty(name, fallback)
	}
	role := ""
	if len(lines) > 1 {
		detailParts := splitPipeLine(lines[1])
		if len(detailParts) > 0 {
			role = detailParts[0]
		}
	}
	if role != "" && name != "" {
		return name + " - " + role
	}
	return firstNonEmpty(name, fallback)
}

func isSectionHeading(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	trimmed := strings.TrimSuffix(value, ":")
	lower := strings.ToLower(trimmed)
	common := map[string]bool{
		"summary": true, "professional summary": true, "experience": true, "work experience": true,
		"employment": true, "projects": true, "technical skills": true, "skills": true,
		"education": true, "certifications": true, "other projects": true, "notes": true,
	}
	if common[lower] {
		return true
	}
	letters := regexp.MustCompile(`[A-Za-z]`).FindAllString(trimmed, -1)
	return len(letters) >= 3 && strings.ToUpper(trimmed) == trimmed
}

func cleanHeading(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "# ")
	value = strings.TrimSuffix(value, ":")
	if value == "" {
		return "Untitled section"
	}
	return value
}

func normalizeSectionType(sectionType string, heading string) string {
	sectionType = strings.TrimSpace(strings.ToLower(sectionType))
	switch sectionType {
	case "summary", "skills", "experience", "project", "education", "certification", "misc":
		return sectionType
	}
	lower := strings.ToLower(heading)
	switch {
	case strings.Contains(lower, "summary"):
		return "summary"
	case strings.Contains(lower, "skill"):
		return "skills"
	case strings.Contains(lower, "education"), strings.Contains(lower, "degree"):
		return "education"
	case strings.Contains(lower, "cert"):
		return "certification"
	case strings.Contains(lower, "project"):
		return "project"
	case strings.Contains(lower, "experience"), strings.Contains(lower, "employment"), strings.Contains(lower, "work"):
		return "experience"
	default:
		return "misc"
	}
}

func normalizeSourceType(sourceType string) string {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	switch sourceType {
	case "current_resume", "extended_resume", "old_resume", "project_notes", "readme", "architecture_notes", "interview_notes", "manual_notes":
		return sourceType
	default:
		return "manual_notes"
	}
}

func normalizeProfileRecordType(recordType string) string {
	recordType = strings.TrimSpace(strings.ToLower(recordType))
	switch recordType {
	case "education", "employment", "project", "allowed_alias", "blocked_alias":
		return recordType
	default:
		return "project"
	}
}

func normalizeConfidence(confidence string) string {
	confidence = strings.TrimSpace(strings.ToLower(confidence))
	switch confidence {
	case "high", "medium", "low":
		return confidence
	default:
		return "medium"
	}
}

func normalizeFactStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case factStatusApproved, factStatusNeedsReview, factStatusRejected:
		return status
	default:
		return factStatusNeedsReview
	}
}

func normalizeStringList(values []string) []string {
	cleaned := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	sort.Strings(cleaned)
	return cleaned
}

func encodeStringList(values []string) (string, error) {
	data, err := json.Marshal(normalizeStringList(values))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeStringList(value string) []string {
	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return []string{}
	}
	return normalizeStringList(values)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value != 0
}

func (s *Store) getCandidateSource(id int64) (CandidateSource, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_type, title, raw_text, file_path, imported_at, updated_at
		FROM candidate_sources WHERE id = ?`,
		id,
	)
	if err != nil {
		return CandidateSource{}, err
	}
	defer rows.Close()
	sources, err := scanCandidateSources(rows)
	if err != nil {
		return CandidateSource{}, err
	}
	if len(sources) == 0 {
		return CandidateSource{}, sql.ErrNoRows
	}
	return sources[0], nil
}

func (s *Store) getSourceSection(id int64) (SourceSection, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_id, heading, section_type, content, sort_order, start_char, end_char, created_at, updated_at
		FROM source_sections WHERE id = ?`,
		id,
	)
	if err != nil {
		return SourceSection{}, err
	}
	defer rows.Close()
	sections, err := scanSourceSections(rows)
	if err != nil {
		return SourceSection{}, err
	}
	if len(sections) == 0 {
		return SourceSection{}, sql.ErrNoRows
	}
	return sections[0], nil
}

func (s *Store) getEvidenceFact(id int64) (EvidenceFact, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_id, section_id, fact_text, evidence_quote, technologies_json, confidence, risk_flags_json, origin_heading, origin_type, context_json, status, auto_approved, review_note, created_at, updated_at
		FROM evidence_facts WHERE id = ?`,
		id,
	)
	if err != nil {
		return EvidenceFact{}, err
	}
	defer rows.Close()
	facts, err := scanEvidenceFacts(rows)
	if err != nil {
		return EvidenceFact{}, err
	}
	if len(facts) == 0 {
		return EvidenceFact{}, sql.ErrNoRows
	}
	return facts[0], nil
}

func scanCandidateSources(rows *sql.Rows) ([]CandidateSource, error) {
	sources := []CandidateSource{}
	for rows.Next() {
		var source CandidateSource
		if err := rows.Scan(&source.ID, &source.SourceType, &source.Title, &source.RawText, &source.FilePath, &source.ImportedAt, &source.UpdatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func scanSourceSections(rows *sql.Rows) ([]SourceSection, error) {
	sections := []SourceSection{}
	for rows.Next() {
		var section SourceSection
		if err := rows.Scan(
			&section.ID,
			&section.SourceID,
			&section.Heading,
			&section.SectionType,
			&section.Content,
			&section.SortOrder,
			&section.StartChar,
			&section.EndChar,
			&section.CreatedAt,
			&section.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, rows.Err()
}

func scanEvidenceFacts(rows *sql.Rows) ([]EvidenceFact, error) {
	facts := []EvidenceFact{}
	for rows.Next() {
		var fact EvidenceFact
		var technologiesJSON string
		var riskFlagsJSON string
		var contextJSON string
		var autoApproved int
		if err := rows.Scan(
			&fact.ID,
			&fact.SourceID,
			&fact.SectionID,
			&fact.FactText,
			&fact.EvidenceQuote,
			&technologiesJSON,
			&fact.Confidence,
			&riskFlagsJSON,
			&fact.OriginHeading,
			&fact.OriginType,
			&contextJSON,
			&fact.Status,
			&autoApproved,
			&fact.ReviewNote,
			&fact.CreatedAt,
			&fact.UpdatedAt,
		); err != nil {
			return nil, err
		}
		fact.Technologies = decodeStringList(technologiesJSON)
		fact.RiskFlags = decodeStringList(riskFlagsJSON)
		fact.Context = decodeStringList(contextJSON)
		fact.AutoApproved = intToBool(autoApproved)
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}
