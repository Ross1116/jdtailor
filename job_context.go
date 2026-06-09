package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type JobDescription struct {
	ID        int64  `json:"id"`
	Company   string `json:"company"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	RawText   string `json:"raw_text"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateJobDescriptionInput struct {
	Company string `json:"company"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	RawText string `json:"raw_text"`
}

type UpdateJobDescriptionInput struct {
	ID      int64  `json:"id"`
	Company string `json:"company"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	RawText string `json:"raw_text"`
}

type JobRequirement struct {
	ID              int64    `json:"id"`
	JobID           int64    `json:"job_id"`
	Category        string   `json:"category"`
	RequirementText string   `json:"requirement_text"`
	Keywords        []string `json:"keywords"`
	Priority        string   `json:"priority"`
	SourceQuote     string   `json:"source_quote"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type JobFactMatch struct {
	ID             int64    `json:"id"`
	JobID          int64    `json:"job_id"`
	RequirementID  int64    `json:"requirement_id"`
	FactID         int64    `json:"fact_id"`
	Score          float64  `json:"score"`
	Rationale      string   `json:"rationale"`
	CoverageStatus string   `json:"coverage_status"`
	FactStatus     string   `json:"fact_status"`
	FactText       string   `json:"fact_text"`
	EvidenceQuote  string   `json:"evidence_quote"`
	RiskFlags      []string `json:"risk_flags"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type TailoredBulletDraft struct {
	ID            int64    `json:"id"`
	JobID         int64    `json:"job_id"`
	RequirementID int64    `json:"requirement_id"`
	FactIDs       []int64  `json:"fact_ids"`
	DraftText     string   `json:"draft_text"`
	Rationale     string   `json:"rationale"`
	Status        string   `json:"status"`
	RiskFlags     []string `json:"risk_flags"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type UpdateTailoredBulletDraftInput struct {
	ID        int64    `json:"id"`
	DraftText string   `json:"draft_text"`
	Rationale string   `json:"rationale"`
	Status    string   `json:"status"`
	RiskFlags []string `json:"risk_flags"`
}

type parsedJobRequirementsResponse struct {
	Requirements []parsedJobRequirement `json:"requirements"`
}

type parsedJobRequirement struct {
	Category        string   `json:"category"`
	RequirementText string   `json:"requirement_text"`
	Keywords        []string `json:"keywords"`
	Priority        string   `json:"priority"`
	SourceQuote     string   `json:"source_quote"`
}

type parsedJobMatchesResponse struct {
	Matches []parsedJobMatch `json:"matches"`
}

type parsedJobMatch struct {
	RequirementID  int64   `json:"requirement_id"`
	FactID         int64   `json:"fact_id"`
	Score          float64 `json:"score"`
	Rationale      string  `json:"rationale"`
	CoverageStatus string  `json:"coverage_status"`
}

type parsedBulletDraftsResponse struct {
	Drafts []parsedBulletDraft `json:"drafts"`
}

type parsedBulletDraft struct {
	RequirementID int64    `json:"requirement_id"`
	FactIDs       []int64  `json:"fact_ids"`
	DraftText     string   `json:"draft_text"`
	Rationale     string   `json:"rationale"`
	RiskFlags     []string `json:"risk_flags"`
}

type factPromptContext struct {
	ID             int64    `json:"id"`
	Status         string   `json:"status"`
	Confidence     string   `json:"confidence"`
	RiskFlags      []string `json:"risk_flags"`
	FactText       string   `json:"fact_text"`
	EvidenceQuote  string   `json:"evidence_quote"`
	Technologies   []string `json:"technologies"`
	SectionHeading string   `json:"section_heading"`
}

func (s *Store) ListJobDescriptions() ([]JobDescription, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, company, title, url, raw_text, created_at, updated_at FROM job_descriptions ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobDescriptions(rows)
}

func (s *Store) CreateJobDescription(input CreateJobDescriptionInput) (JobDescription, error) {
	rawText := strings.TrimSpace(input.RawText)
	if rawText == "" {
		return JobDescription{}, errors.New("job description text is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Untitled job"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO job_descriptions (company, title, url, raw_text, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Company),
		title,
		strings.TrimSpace(input.URL),
		rawText,
		now,
		now,
	)
	if err != nil {
		return JobDescription{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return JobDescription{}, err
	}
	_ = s.LogEvent("info", "job description saved")
	return s.getJobDescription(id)
}

func (s *Store) UpdateJobDescription(input UpdateJobDescriptionInput) (JobDescription, error) {
	if input.ID <= 0 {
		return JobDescription{}, errors.New("job id is required")
	}
	rawText := strings.TrimSpace(input.RawText)
	if rawText == "" {
		return JobDescription{}, errors.New("job description text is required")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Untitled job"
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE job_descriptions SET company = ?, title = ?, url = ?, raw_text = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.Company),
		title,
		strings.TrimSpace(input.URL),
		rawText,
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return JobDescription{}, err
	}
	return s.getJobDescription(input.ID)
}

func (s *Store) DeleteJobDescription(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("job id is required")
	}
	if _, err := s.db.ExecContext(context.Background(), `DELETE FROM job_descriptions WHERE id = ?`, input.ID); err != nil {
		return err
	}
	_ = s.LogEvent("info", "job description deleted")
	return nil
}

func (s *Store) ParseJobDescription(ctx context.Context, jobID int64, client *http.Client) ([]JobRequirement, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	job, err := s.getJobDescription(jobID)
	if err != nil {
		return nil, err
	}
	system := "You parse job descriptions into requirements. Return strict JSON only."
	user := fmt.Sprintf(`Parse this job description into distinct requirements.

Rules:
- Return JSON shaped as {"requirements":[{"category":"must_have|responsibility|nice_to_have|domain|seniority","requirement_text":"","keywords":[],"priority":"high|medium|low","source_quote":""}]}
- source_quote must be an exact quote from the job description.
- Keep requirements atomic and useful for matching resume evidence.
- No markdown, no commentary.

Company: %s
Title: %s
Job description:
%s`, job.Company, job.Title, job.RawText)
	text, err := s.GenerateLLMText(ctx, client, system, user, 2000)
	if err != nil {
		return nil, err
	}
	parsed, err := parseJobRequirements(text)
	if err != nil {
		return nil, err
	}
	return s.replaceJobRequirements(job, parsed)
}

func (s *Store) BuildJobMatchMap(ctx context.Context, jobID int64, client *http.Client) ([]JobFactMatch, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	job, err := s.getJobDescription(jobID)
	if err != nil {
		return nil, err
	}
	requirements, err := s.ListJobRequirements(jobID)
	if err != nil {
		return nil, err
	}
	if len(requirements) == 0 {
		return nil, errors.New("parse job requirements before matching")
	}
	facts, err := s.listFactPromptContext()
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return nil, errors.New("extract evidence facts before matching")
	}
	requirementsJSON, _ := json.Marshal(requirements)
	factsJSON, _ := json.Marshal(facts)
	system := "You match job requirements to candidate evidence facts. Return strict JSON only."
	user := fmt.Sprintf(`Build a match map from job requirements to candidate facts.

Rules:
- Return JSON shaped as {"matches":[{"requirement_id":0,"fact_id":0,"score":0.0,"rationale":"","coverage_status":"strong|partial|weak|gap"}]}
- Use only requirement_id values and fact_id values provided below.
- Facts include approved, needs_review, and rejected status; include rejected/needs_review only when relevant and explain the risk.
- No markdown, no commentary.

Job: %s at %s
Requirements:
%s

Candidate facts:
%s`, job.Title, job.Company, string(requirementsJSON), string(factsJSON))
	text, err := s.GenerateLLMText(ctx, client, system, user, 2400)
	if err != nil {
		return nil, err
	}
	parsed, err := parseJobMatches(text)
	if err != nil {
		return nil, err
	}
	return s.replaceJobMatches(jobID, parsed, requirements, facts)
}

func (s *Store) GenerateTailoredBulletDrafts(ctx context.Context, jobID int64, client *http.Client) ([]TailoredBulletDraft, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	job, err := s.getJobDescription(jobID)
	if err != nil {
		return nil, err
	}
	requirements, err := s.ListJobRequirements(jobID)
	if err != nil {
		return nil, err
	}
	matches, err := s.ListJobFactMatches(jobID)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, errors.New("build match map before generating bullet drafts")
	}
	facts, err := s.listFactPromptContext()
	if err != nil {
		return nil, err
	}
	reqJSON, _ := json.Marshal(requirements)
	matchJSON, _ := json.Marshal(matches)
	factJSON, _ := json.Marshal(facts)
	system := "You draft tailored resume bullets from evidence only. Return strict JSON only."
	user := fmt.Sprintf(`Generate tailored bullet suggestions for this job.

Rules:
- Return JSON shaped as {"drafts":[{"requirement_id":0,"fact_ids":[0],"draft_text":"","rationale":"","risk_flags":[]}]}
- Every fact_id must come from the provided candidate facts.
- Do not introduce unsupported claims, metrics, tools, leadership, or production scope.
- Include risk_flags when supporting facts are needs_review, rejected, low confidence, or ambiguous.
- These are suggestions only, not source truth.
- No markdown, no commentary.

Job: %s at %s
Requirements:
%s

Matches:
%s

Candidate facts:
%s`, job.Title, job.Company, string(reqJSON), string(matchJSON), string(factJSON))
	text, err := s.GenerateLLMText(ctx, client, system, user, 2400)
	if err != nil {
		return nil, err
	}
	parsed, err := parseBulletDrafts(text)
	if err != nil {
		return nil, err
	}
	return s.replaceBulletDrafts(jobID, parsed, requirements, facts)
}

func (s *Store) ListJobRequirements(jobID int64) ([]JobRequirement, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, category, requirement_text, keywords_json, priority, source_quote, created_at, updated_at
		FROM job_requirements WHERE job_id = ? ORDER BY CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, id`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobRequirements(rows)
}

func (s *Store) ListJobFactMatches(jobID int64) ([]JobFactMatch, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT m.id, m.job_id, m.requirement_id, m.fact_id, m.score, m.rationale, m.coverage_status,
			f.status, f.fact_text, f.evidence_quote, f.risk_flags_json, m.created_at, m.updated_at
		FROM job_fact_matches m
		JOIN evidence_facts f ON f.id = m.fact_id
		WHERE m.job_id = ?
		ORDER BY m.requirement_id, m.score DESC, m.id`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobFactMatches(rows)
}

func (s *Store) ListTailoredBulletDrafts(jobID int64) ([]TailoredBulletDraft, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at
		FROM tailored_bullet_drafts WHERE job_id = ?
		ORDER BY CASE status WHEN 'needs_review' THEN 0 WHEN 'accepted' THEN 1 ELSE 2 END, id DESC`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTailoredBulletDrafts(rows)
}

func (s *Store) UpdateTailoredBulletDraft(input UpdateTailoredBulletDraftInput) (TailoredBulletDraft, error) {
	if input.ID <= 0 {
		return TailoredBulletDraft{}, errors.New("draft id is required")
	}
	draftText := strings.TrimSpace(input.DraftText)
	if draftText == "" {
		return TailoredBulletDraft{}, errors.New("draft text is required")
	}
	riskJSON, err := encodeStringList(input.RiskFlags)
	if err != nil {
		return TailoredBulletDraft{}, err
	}
	_, err = s.db.ExecContext(
		context.Background(),
		`UPDATE tailored_bullet_drafts SET draft_text = ?, rationale = ?, status = ?, risk_flags_json = ?, updated_at = ? WHERE id = ?`,
		draftText,
		strings.TrimSpace(input.Rationale),
		normalizeDraftStatus(input.Status),
		riskJSON,
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return TailoredBulletDraft{}, err
	}
	return s.getTailoredBulletDraft(input.ID)
}

func (s *Store) DeleteTailoredBulletDraft(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("draft id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM tailored_bullet_drafts WHERE id = ?`, input.ID)
	return err
}

func (s *Store) replaceJobRequirements(job JobDescription, requirements []parsedJobRequirement) ([]JobRequirement, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM tailored_bullet_drafts WHERE job_id = ?`, job.ID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM job_fact_matches WHERE job_id = ?`, job.ID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM job_requirements WHERE job_id = ?`, job.ID); err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, req := range requirements {
		req.RequirementText = strings.TrimSpace(req.RequirementText)
		req.SourceQuote = strings.TrimSpace(req.SourceQuote)
		if req.RequirementText == "" || req.SourceQuote == "" || !strings.Contains(job.RawText, req.SourceQuote) {
			continue
		}
		keywordsJSON, err := encodeStringList(req.Keywords)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO job_requirements (job_id, category, requirement_text, keywords_json, priority, source_quote, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID,
			normalizeRequirementCategory(req.Category),
			req.RequirementText,
			keywordsJSON,
			normalizePriority(req.Priority),
			req.SourceQuote,
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
	if len(ids) == 0 {
		return nil, errors.New("LLM returned no usable requirements")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "job requirements parsed")
	return s.ListJobRequirements(job.ID)
}

func (s *Store) replaceJobMatches(jobID int64, matches []parsedJobMatch, requirements []JobRequirement, facts []factPromptContext) ([]JobFactMatch, error) {
	reqIDs := map[int64]bool{}
	for _, req := range requirements {
		reqIDs[req.ID] = true
	}
	factIDs := map[int64]bool{}
	for _, fact := range facts {
		factIDs[fact.ID] = true
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM job_fact_matches WHERE job_id = ?`, jobID); err != nil {
		return nil, err
	}
	count := 0
	for _, match := range matches {
		if !reqIDs[match.RequirementID] || !factIDs[match.FactID] {
			continue
		}
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO job_fact_matches (job_id, requirement_id, fact_id, score, rationale, coverage_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			jobID,
			match.RequirementID,
			match.FactID,
			clampScore(match.Score),
			strings.TrimSpace(match.Rationale),
			normalizeCoverageStatus(match.CoverageStatus),
			now,
			now,
		); err != nil {
			return nil, err
		}
		count++
	}
	if count == 0 {
		return nil, errors.New("LLM returned no usable matches")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "job match map built")
	return s.ListJobFactMatches(jobID)
}

func (s *Store) replaceBulletDrafts(jobID int64, drafts []parsedBulletDraft, requirements []JobRequirement, facts []factPromptContext) ([]TailoredBulletDraft, error) {
	reqIDs := map[int64]bool{}
	for _, req := range requirements {
		reqIDs[req.ID] = true
	}
	factIDs := map[int64]bool{}
	for _, fact := range facts {
		factIDs[fact.ID] = true
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM tailored_bullet_drafts WHERE job_id = ?`, jobID); err != nil {
		return nil, err
	}
	count := 0
	for _, draft := range drafts {
		draft.DraftText = strings.TrimSpace(draft.DraftText)
		if draft.DraftText == "" || !reqIDs[draft.RequirementID] {
			continue
		}
		validFactIDs := []int64{}
		for _, factID := range draft.FactIDs {
			if factIDs[factID] {
				validFactIDs = append(validFactIDs, factID)
			}
		}
		if len(validFactIDs) == 0 {
			continue
		}
		factIDsJSON, err := encodeInt64List(validFactIDs)
		if err != nil {
			return nil, err
		}
		riskJSON, err := encodeStringList(draft.RiskFlags)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO tailored_bullet_drafts (job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 'needs_review', ?, ?, ?)`,
			jobID,
			draft.RequirementID,
			factIDsJSON,
			draft.DraftText,
			strings.TrimSpace(draft.Rationale),
			riskJSON,
			now,
			now,
		); err != nil {
			return nil, err
		}
		count++
	}
	if count == 0 {
		return nil, errors.New("LLM returned no usable bullet drafts")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "tailored bullet drafts generated")
	return s.ListTailoredBulletDrafts(jobID)
}

func parseJobRequirements(text string) ([]parsedJobRequirement, error) {
	var parsed parsedJobRequirementsResponse
	if err := parseJSONObject(text, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Requirements) == 0 {
		return nil, errors.New("LLM returned no requirements")
	}
	for _, req := range parsed.Requirements {
		if strings.TrimSpace(req.RequirementText) == "" || strings.TrimSpace(req.SourceQuote) == "" {
			return nil, errors.New("every requirement must include requirement_text and source_quote")
		}
	}
	return parsed.Requirements, nil
}

func parseJobMatches(text string) ([]parsedJobMatch, error) {
	var parsed parsedJobMatchesResponse
	if err := parseJSONObject(text, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Matches) == 0 {
		return nil, errors.New("LLM returned no matches")
	}
	return parsed.Matches, nil
}

func parseBulletDrafts(text string) ([]parsedBulletDraft, error) {
	var parsed parsedBulletDraftsResponse
	if err := parseJSONObject(text, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Drafts) == 0 {
		return nil, errors.New("LLM returned no drafts")
	}
	for _, draft := range parsed.Drafts {
		if strings.TrimSpace(draft.DraftText) == "" || len(draft.FactIDs) == 0 {
			return nil, errors.New("every draft must include draft_text and fact_ids")
		}
	}
	return parsed.Drafts, nil
}

func parseJSONObject(text string, target any) error {
	text = strings.TrimSpace(text)
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
	return json.Unmarshal([]byte(text), target)
}

func (s *Store) listFactPromptContext() ([]factPromptContext, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT f.id, f.status, f.confidence, f.risk_flags_json, f.fact_text, f.evidence_quote, f.technologies_json, s.heading
		FROM evidence_facts f
		JOIN source_sections s ON s.id = f.section_id
		ORDER BY f.id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	facts := []factPromptContext{}
	for rows.Next() {
		var fact factPromptContext
		var riskJSON string
		var techJSON string
		if err := rows.Scan(&fact.ID, &fact.Status, &fact.Confidence, &riskJSON, &fact.FactText, &fact.EvidenceQuote, &techJSON, &fact.SectionHeading); err != nil {
			return nil, err
		}
		fact.RiskFlags = decodeStringList(riskJSON)
		fact.Technologies = decodeStringList(techJSON)
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func (s *Store) getJobDescription(id int64) (JobDescription, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, company, title, url, raw_text, created_at, updated_at FROM job_descriptions WHERE id = ?`, id)
	if err != nil {
		return JobDescription{}, err
	}
	defer rows.Close()
	jobs, err := scanJobDescriptions(rows)
	if err != nil {
		return JobDescription{}, err
	}
	if len(jobs) == 0 {
		return JobDescription{}, sql.ErrNoRows
	}
	return jobs[0], nil
}

func (s *Store) getTailoredBulletDraft(id int64) (TailoredBulletDraft, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at
		FROM tailored_bullet_drafts WHERE id = ?`,
		id,
	)
	if err != nil {
		return TailoredBulletDraft{}, err
	}
	defer rows.Close()
	drafts, err := scanTailoredBulletDrafts(rows)
	if err != nil {
		return TailoredBulletDraft{}, err
	}
	if len(drafts) == 0 {
		return TailoredBulletDraft{}, sql.ErrNoRows
	}
	return drafts[0], nil
}

func scanJobDescriptions(rows *sql.Rows) ([]JobDescription, error) {
	jobs := []JobDescription{}
	for rows.Next() {
		var job JobDescription
		if err := rows.Scan(&job.ID, &job.Company, &job.Title, &job.URL, &job.RawText, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func scanJobRequirements(rows *sql.Rows) ([]JobRequirement, error) {
	requirements := []JobRequirement{}
	for rows.Next() {
		var req JobRequirement
		var keywordsJSON string
		if err := rows.Scan(&req.ID, &req.JobID, &req.Category, &req.RequirementText, &keywordsJSON, &req.Priority, &req.SourceQuote, &req.CreatedAt, &req.UpdatedAt); err != nil {
			return nil, err
		}
		req.Keywords = decodeStringList(keywordsJSON)
		requirements = append(requirements, req)
	}
	return requirements, rows.Err()
}

func scanJobFactMatches(rows *sql.Rows) ([]JobFactMatch, error) {
	matches := []JobFactMatch{}
	for rows.Next() {
		var match JobFactMatch
		var riskJSON string
		if err := rows.Scan(
			&match.ID,
			&match.JobID,
			&match.RequirementID,
			&match.FactID,
			&match.Score,
			&match.Rationale,
			&match.CoverageStatus,
			&match.FactStatus,
			&match.FactText,
			&match.EvidenceQuote,
			&riskJSON,
			&match.CreatedAt,
			&match.UpdatedAt,
		); err != nil {
			return nil, err
		}
		match.RiskFlags = decodeStringList(riskJSON)
		matches = append(matches, match)
	}
	return matches, rows.Err()
}

func scanTailoredBulletDrafts(rows *sql.Rows) ([]TailoredBulletDraft, error) {
	drafts := []TailoredBulletDraft{}
	for rows.Next() {
		var draft TailoredBulletDraft
		var factIDsJSON string
		var riskJSON string
		if err := rows.Scan(&draft.ID, &draft.JobID, &draft.RequirementID, &factIDsJSON, &draft.DraftText, &draft.Rationale, &draft.Status, &riskJSON, &draft.CreatedAt, &draft.UpdatedAt); err != nil {
			return nil, err
		}
		draft.FactIDs = decodeInt64List(factIDsJSON)
		draft.RiskFlags = decodeStringList(riskJSON)
		drafts = append(drafts, draft)
	}
	return drafts, rows.Err()
}

func encodeInt64List(values []int64) (string, error) {
	normalized := []int64{}
	seen := map[int64]bool{}
	for _, value := range values {
		if value <= 0 || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeInt64List(raw string) []int64 {
	values := []int64{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []int64{}
	}
	return values
}

func normalizeRequirementCategory(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "must_have", "responsibility", "nice_to_have", "domain", "seniority":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "responsibility"
	}
}

func normalizePriority(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "high", "medium", "low":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "medium"
	}
}

func normalizeCoverageStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "strong", "partial", "weak", "gap":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "partial"
	}
}

func normalizeDraftStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "needs_review", "accepted", "rejected":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "needs_review"
	}
}

func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
