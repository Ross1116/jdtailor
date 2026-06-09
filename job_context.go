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
	details := inferJobDetails(rawText)
	company := strings.TrimSpace(input.Company)
	if company == "" {
		company = details.Company
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = details.Title
	}
	if title == "" {
		title = "Untitled job"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO job_descriptions (company, title, url, raw_text, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		company,
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
	details := inferJobDetails(rawText)
	company := strings.TrimSpace(input.Company)
	if company == "" {
		company = details.Company
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = details.Title
	}
	if title == "" {
		title = "Untitled job"
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE job_descriptions SET company = ?, title = ?, url = ?, raw_text = ?, updated_at = ? WHERE id = ?`,
		company,
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
	system := `You are JD Tailor's job-description parser. Return strict JSON only.
Extract only requirements that can change how a resume is tailored.`
	user := fmt.Sprintf(`# Task
Parse the pasted job description into resume-tailoring requirements.

# Output JSON schema
{"requirements":[{"category":"must_have|responsibility|nice_to_have|domain|seniority","requirement_text":"","keywords":[],"priority":"high|medium|low","source_quote":""}]}

# Keep
- Technical skills, tools, languages, frameworks, databases, cloud platforms.
- Architecture and delivery expectations: distributed systems, security, observability, resilience, DevSecOps, testing, data models.
- Concrete responsibilities that a resume bullet can support.
- Seniority signals only when they imply experience depth, leadership, mentoring, influence, or years.
- Domain requirements only when the JD explicitly asks for domain experience/background/knowledge.

# Reject
- LinkedIn/page chrome: logos, promoted text, profile match banners, Premium upsells, alumni/people panels, "helpful" prompts.
- Company/about-us blurbs, generic mission statements, benefits, salary, location, contract duration, application instructions.
- Role-title-only metadata such as "Software Engineer".
- Generic soft skills unless concrete and evidence-backed: stakeholder management, mentoring, leadership, cross-functional delivery.

# Quality bar
- Return 8 to 16 highest-value requirements at most.
- Make each requirement atomic: one skill/responsibility cluster per item.
- `+"`source_quote`"+` must be an exact quote from <job_description>.
- `+"`keywords`"+` should contain 2 to 8 matching terms, favoring tools and domain-specific nouns.
- Do not invent requirements not stated in the JD.

<job company="%s" title="%s">
<job_description>
%s
</job_description>
</job>`, job.Company, job.Title, compactPromptText(job.RawText, 18000))
	text, err := s.GenerateLLMText(ctx, client, system, user, 1200)
	if err != nil {
		parsed := fallbackJobRequirements(job.RawText)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM request failed and no local requirements could be extracted: %w", err)
		}
		_ = s.LogEvent("warning", "job requirements used local fallback after LLM request failure: "+err.Error())
		return s.replaceJobRequirements(job, parsed)
	}
	parsed, err := parseJobRequirements(text)
	if err != nil {
		parsed = fallbackJobRequirements(job.RawText)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM returned unusable requirement JSON and no local requirements could be extracted: %w", err)
		}
		_ = s.LogEvent("warning", "job requirements used local fallback: "+err.Error())
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
	promptFacts := selectFactsForRequirements(requirements, facts, 90)
	factsJSON, _ := json.Marshal(promptFacts)
	system := `You are JD Tailor's evidence matcher. Return strict JSON only.
Match job requirements to candidate facts only when the evidence genuinely supports the requirement.`
	user := fmt.Sprintf(`# Task
Build an evidence-backed match map for this job.

# Output JSON schema
{"matches":[{"requirement_id":0,"fact_id":0,"score":0.0,"rationale":"","coverage_status":"strong|partial|weak"}]}

# Matching rubric
- strong: direct support for the core requirement/tool/responsibility.
- partial: supports an adjacent part of the requirement but misses an important tool, platform, scale, or domain.
- weak: only broad transferable evidence; still real evidence, not a gap.
- Do not output gaps. If no fact supports a requirement, omit it.

# Rules
- Use only IDs from <requirements_json> and <candidate_facts_json>.
- Prefer approved facts. You may use needs_review or rejected facts only when highly relevant; mention that risk in rationale.
- Do not match on generic words alone: engineer, software, application, team, role, business, agile, communication.
- Rationale must explain the overlap and the missing caveat in one sentence.
- Score range: strong 0.75-1.0, partial 0.45-0.74, weak 0.2-0.44.

<job company="%s" title="%s"/>
<requirements_json>
%s
</requirements_json>
<candidate_facts_json>
%s
</candidate_facts_json>`, job.Company, job.Title, string(requirementsJSON), string(factsJSON))
	text, err := s.GenerateLLMText(ctx, client, system, user, 1600)
	if err != nil {
		parsed := fallbackJobMatches(requirements, facts)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM request failed and no local matches could be built: %w", err)
		}
		_ = s.LogEvent("warning", "job match map used local fallback after LLM request failure: "+err.Error())
		return s.replaceJobMatches(jobID, parsed, requirements, facts)
	}
	parsed, err := parseJobMatches(text)
	if err != nil {
		parsed = fallbackJobMatches(requirements, facts)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM returned unusable match JSON and no local matches could be built: %w", err)
		}
		_ = s.LogEvent("warning", "job match map used local fallback: "+err.Error())
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
	promptFacts := selectFactsForMatches(matches, facts)
	factJSON, _ := json.Marshal(promptFacts)
	system := `You are JD Tailor's resume bullet drafter. Return strict JSON only.
Write tailored bullet suggestions from provided evidence only; never invent source truth.`
	user := fmt.Sprintf(`# Task
Generate saved resume bullet suggestions for this job from the matched evidence.

# Output JSON schema
{"drafts":[{"requirement_id":0,"fact_ids":[0],"draft_text":"","rationale":"","risk_flags":[]}]}

# Bullet style
- Start each draft_text with a strong past-tense action verb, no leading hyphen.
- Make the bullet specific to the JD language when supported by facts.
- Combine compatible facts only when they share a believable project/context.
- Prefer concrete artifacts, technologies, scale, users/teams, metrics, and outcomes.
- Keep each bullet 22 to 34 words.

# Evidence rules
- Every fact_id must exist in <candidate_facts_json>.
- Do not introduce unsupported metrics, tools, cloud platforms, leadership, security, production scope, or business impact.
- If evidence is weak/partial, phrase conservatively; do not overclaim the exact JD requirement.
- Include risk_flags for needs_review facts, rejected facts, low confidence, ambiguous metric/scope, or inferred tailoring.
- Drafts are suggestions only and must not mutate locked profile/source truth.

# Rationale
- Explain which requirement is targeted and why the facts support the wording.
- Mention any missing JD detail that prevented stronger tailoring.

<job company="%s" title="%s"/>
<requirements_json>
%s
</requirements_json>
<matches_json>
%s
</matches_json>
<candidate_facts_json>
%s
</candidate_facts_json>`, job.Company, job.Title, string(reqJSON), string(matchJSON), string(factJSON))
	text, err := s.GenerateLLMText(ctx, client, system, user, 1600)
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
		if strings.EqualFold(match.CoverageStatus, "gap") || match.Score <= 0 {
			continue
		}
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if count == 0 {
		_ = s.LogEvent("warning", "job match map built with no evidence-backed matches")
	} else {
		_ = s.LogEvent("info", "job match map built")
	}
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
	filtered := []parsedJobRequirement{}
	for _, req := range parsed.Requirements {
		if strings.TrimSpace(req.RequirementText) == "" || strings.TrimSpace(req.SourceQuote) == "" {
			return nil, errors.New("every requirement must include requirement_text and source_quote")
		}
		req.RequirementText = strings.TrimSpace(req.RequirementText)
		req.SourceQuote = strings.TrimSpace(req.SourceQuote)
		if isIrrelevantJobRequirement(req) {
			continue
		}
		filtered = append(filtered, req)
	}
	if len(filtered) == 0 {
		return nil, errors.New("LLM returned no relevant resume-tailoring requirements")
	}
	if len(filtered) > 16 {
		filtered = filtered[:16]
	}
	return filtered, nil
}

func parseJobMatches(text string) ([]parsedJobMatch, error) {
	var parsed parsedJobMatchesResponse
	if err := parseJSONObject(text, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Matches) == 0 {
		return nil, errors.New("LLM returned no matches")
	}
	filtered := []parsedJobMatch{}
	for _, match := range parsed.Matches {
		if strings.EqualFold(match.CoverageStatus, "gap") || match.Score <= 0 {
			continue
		}
		filtered = append(filtered, match)
	}
	if len(filtered) == 0 {
		return nil, errors.New("LLM returned no evidence-backed matches")
	}
	return filtered, nil
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
	if text == "" {
		return errors.New("LLM returned empty JSON response")
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
		return errors.New("LLM response did not contain a JSON object")
	}
	if err := json.Unmarshal([]byte(text), target); err != nil {
		return fmt.Errorf("LLM JSON could not be parsed: %w", err)
	}
	return nil
}

func isIrrelevantJobRequirement(req parsedJobRequirement) bool {
	text := strings.ToLower(strings.Join([]string{
		req.Category,
		req.RequirementText,
		req.SourceQuote,
		strings.Join(req.Keywords, " "),
	}, " "))
	if text == "" {
		return true
	}
	irrelevantMarkers := []string{
		"12-month", "12 month", "contract", "max term", "fixed term", "salary", "compensation", "benefit", "leave", "hybrid", "remote", "location", "office",
		"how to apply", "application process", "submit your application", "recruit", "hiring", "interview", "equal opportunity", "diversity", "background check", "sponsorship",
		"leading personal injury", "class actions law firm", "about us", "about the company", "company is", "we are a", "we're a", "our client",
		"logo", "linkedin", "promoted by", "responses managed", "profile matches", "is this information helpful", "personalized tips", "top applicant", "retry premium", "people you can reach out", "school alumni", "clicked apply",
	}
	for _, marker := range irrelevantMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	if isJobHeadingOrMetadata(req.RequirementText) || isJobHeadingOrMetadata(req.SourceQuote) {
		return true
	}
	if !hasTailorableRequirementSignal(req.RequirementText, req.Keywords) {
		return true
	}
	if strings.EqualFold(req.Category, "domain") && !strings.Contains(text, "experience") && !strings.Contains(text, "knowledge") && !strings.Contains(text, "background") {
		return true
	}
	if strings.EqualFold(req.Category, "seniority") && !strings.Contains(text, "senior") && !strings.Contains(text, "lead") && !strings.Contains(text, "mentor") && !strings.Contains(text, "years") {
		return true
	}
	if len(jobMatchTerms(req.RequirementText, req.Keywords)) == 0 {
		return true
	}
	return false
}

func fallbackJobRequirements(raw string) []parsedJobRequirement {
	lines := splitJobRequirementCandidates(raw)
	requirements := []parsedJobRequirement{}
	seen := map[string]bool{}
	for _, line := range lines {
		line = cleanJobDetailLine(line)
		if len(line) < 12 {
			continue
		}
		if isJobHeadingOrMetadata(line) || !hasTailorableRequirementSignal(line, nil) {
			continue
		}
		keywords := extractJobKeywords(line)
		req := parsedJobRequirement{
			Category:        inferJobRequirementCategory(line),
			RequirementText: strings.TrimSuffix(line, "."),
			Keywords:        keywords,
			Priority:        inferJobRequirementPriority(line, len(requirements)),
			SourceQuote:     line,
		}
		if isIrrelevantJobRequirement(req) || len(req.Keywords) == 0 {
			continue
		}
		key := strings.ToLower(req.RequirementText)
		if seen[key] {
			continue
		}
		seen[key] = true
		requirements = append(requirements, req)
		if len(requirements) >= 16 {
			break
		}
	}
	return requirements
}

func isJobHeadingOrMetadata(line string) bool {
	cleaned := strings.TrimSpace(strings.ToLower(strings.Trim(line, ".:?!")))
	if cleaned == "" {
		return true
	}
	exact := map[string]bool{
		"about": true, "about the job": true, "what are we looking for": true, "responsibilities": true, "requirements": true, "key responsibilities": true,
		"software engineer": true, "slater and gordon lawyers": true, "slater and gordon lawyers logo": true,
	}
	if exact[cleaned] {
		return true
	}
	if strings.Contains(cleaned, "·") || strings.Contains(cleaned, " clicked apply") || strings.HasSuffix(cleaned, " logo") {
		return true
	}
	if len(strings.Fields(cleaned)) <= 4 && looksLikeRoleTitle(cleaned) {
		return true
	}
	return false
}

func hasTailorableRequirementSignal(text string, keywords []string) bool {
	lower := strings.ToLower(strings.Join(append([]string{text}, keywords...), " "))
	signals := []string{
		"experience", "hands-on", "strong", "deep", "knowledge", "understanding", "programming", "skills", "design", "build", "develop", "deliver", "modernis", "test", "support",
		"architecture", "cloud", "serverless", "event-driven", "distributed", "scalable", "resilience", "observability", "security", "networking", "identity", "database", "nosql",
		"devsecops", "agile", "solid", "containers", "messaging", "queues", "topics", "stakeholder", "mentor", "engineering practices", "technical excellence",
	}
	for _, signal := range signals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	for _, tech := range []string{"fastapi", "postgresql", "react", "typescript", "javascript", "python", "golang", "java", "spring", "mysql", "node.js", "node", "azure", "cosmos db", "aws", "gcp", "docker", "kubernetes", "terraform", "redis"} {
		if strings.Contains(lower, tech) {
			return true
		}
	}
	return false
}

func splitJobRequirementCandidates(raw string) []string {
	candidates := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, part := range strings.FieldsFunc(line, func(r rune) bool {
			return r == ';' || r == '•'
		}) {
			part = strings.TrimSpace(part)
			if part != "" {
				candidates = append(candidates, part)
			}
		}
	}
	return candidates
}

func inferJobRequirementCategory(line string) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "nice") || strings.Contains(lower, "preferred") || strings.Contains(lower, "bonus") || strings.Contains(lower, "highly regarded") {
		return "nice_to_have"
	}
	if strings.Contains(lower, "senior") || strings.Contains(lower, "years") || strings.Contains(lower, "mentor") || strings.Contains(lower, "lead ") {
		return "seniority"
	}
	if strings.Contains(lower, "domain") || strings.Contains(lower, "industry") || strings.Contains(lower, "legal") || strings.Contains(lower, "finance") {
		return "domain"
	}
	if strings.Contains(lower, "design") || strings.Contains(lower, "build") || strings.Contains(lower, "deliver") || strings.Contains(lower, "collaborate") || strings.Contains(lower, "champion") || strings.Contains(lower, "apply") {
		return "responsibility"
	}
	return "must_have"
}

func inferJobRequirementPriority(line string, index int) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "nice") || strings.Contains(lower, "preferred") || strings.Contains(lower, "bonus") || strings.Contains(lower, "highly regarded") {
		return "low"
	}
	if index < 8 || strings.Contains(lower, "strong") || strings.Contains(lower, "deep") || strings.Contains(lower, "hands-on") || strings.Contains(lower, "must") {
		return "high"
	}
	return "medium"
}

func extractJobKeywords(text string) []string {
	stop := map[string]bool{}
	for _, word := range []string{"the", "and", "for", "with", "that", "this", "you", "your", "our", "are", "will", "have", "has", "from", "into", "work", "role", "team", "need", "needs", "required", "requirement", "requirements", "experience", "strong", "deep", "hands", "excellent", "ability", "using", "including"} {
		stop[word] = true
	}
	keywords := []string{}
	for _, tech := range []string{"Node.js", "React.js", "Azure", "Cosmos DB", "NoSQL", "DevSecOps", "Agile", "SOLID", "serverless", "event-driven", "containers", "messaging", "queues", "topics", "cloud", "observability", "security", "networking", "resilience", "identity", "distributed", "scalable"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(tech)) {
			keywords = append(keywords, tech)
		}
	}
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '#' && r != '+'
	}) {
		lower := strings.ToLower(strings.TrimSpace(token))
		if len(lower) < 4 || stop[lower] {
			continue
		}
		keywords = append(keywords, token)
	}
	if len(keywords) > 8 {
		keywords = keywords[:8]
	}
	return normalizeStringList(keywords)
}

type inferredJobDetails struct {
	Company string
	Title   string
}

func inferJobDetails(raw string) inferredJobDetails {
	lines := meaningfulJobLines(raw, 12)
	details := inferredJobDetails{}
	for _, line := range lines {
		cleaned := cleanJobDetailLine(line)
		lower := strings.ToLower(cleaned)
		for _, prefix := range []string{"company:", "employer:", "organisation:", "organization:"} {
			if strings.HasPrefix(lower, prefix) {
				details.Company = strings.TrimSpace(cleaned[len(prefix):])
			}
		}
		for _, prefix := range []string{"job title:", "role:", "title:", "position:"} {
			if strings.HasPrefix(lower, prefix) {
				details.Title = strings.TrimSpace(cleaned[len(prefix):])
			}
		}
	}
	if details.Title == "" {
		for _, line := range lines {
			cleaned := cleanJobDetailLine(line)
			if looksLikeRoleTitle(cleaned) {
				details.Title = cleaned
				break
			}
		}
	}
	if details.Company == "" {
		for _, line := range lines {
			cleaned := cleanJobDetailLine(line)
			lower := strings.ToLower(cleaned)
			if cleaned == "" || cleaned == details.Title || looksLikeRoleTitle(cleaned) || strings.Contains(lower, "responsibilities") || strings.Contains(lower, "requirements") || strings.Contains(lower, "about the role") {
				continue
			}
			if len(strings.Fields(cleaned)) <= 5 {
				details.Company = strings.Trim(cleaned, "|- ")
				break
			}
		}
	}
	if strings.Contains(details.Title, " at ") && details.Company == "" {
		before, after, ok := strings.Cut(details.Title, " at ")
		if ok {
			details.Title = strings.TrimSpace(before)
			details.Company = strings.TrimSpace(after)
		}
	}
	return details
}

func meaningfulJobLines(raw string, limit int) []string {
	lines := []string{}
	for _, line := range strings.Split(raw, "\n") {
		cleaned := cleanJobDetailLine(line)
		if cleaned == "" || strings.HasPrefix(strings.ToLower(cleaned), "http") {
			continue
		}
		lines = append(lines, cleaned)
		if len(lines) >= limit {
			break
		}
	}
	return lines
}

func cleanJobDetailLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "•")
	line = strings.TrimSpace(strings.Trim(line, "#*_`"))
	return strings.Join(strings.Fields(line), " ")
}

func looksLikeRoleTitle(line string) bool {
	lower := strings.ToLower(line)
	if len(strings.Fields(line)) > 9 {
		return false
	}
	roleTerms := []string{"engineer", "developer", "manager", "analyst", "designer", "architect", "consultant", "specialist", "lead", "intern", "graduate", "backend", "frontend", "full stack", "software", "data", "devops", "platform"}
	for _, term := range roleTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func fallbackJobMatches(requirements []JobRequirement, facts []factPromptContext) []parsedJobMatch {
	matches := []parsedJobMatch{}
	for _, req := range requirements {
		reqTerms := jobMatchTerms(req.RequirementText, req.Keywords)
		if len(reqTerms) == 0 {
			continue
		}
		candidates := []parsedJobMatch{}
		for _, fact := range facts {
			factText := strings.ToLower(strings.Join([]string{
				fact.FactText,
				fact.EvidenceQuote,
				strings.Join(fact.Technologies, " "),
				fact.SectionHeading,
			}, " "))
			overlap := []string{}
			for _, term := range reqTerms {
				if strings.Contains(factText, term) {
					overlap = append(overlap, term)
				}
			}
			overlap = normalizeStringList(overlap)
			if len(overlap) == 0 {
				continue
			}
			score := 0.35 + float64(len(overlap))*0.16
			status := "partial"
			if len(overlap) >= 3 || score >= 0.75 {
				status = "strong"
			}
			if len(overlap) == 1 {
				status = "weak"
			}
			candidates = append(candidates, parsedJobMatch{
				RequirementID:  req.ID,
				FactID:         fact.ID,
				Score:          clampScore(score),
				Rationale:      "Local keyword overlap: " + strings.Join(overlap, ", "),
				CoverageStatus: status,
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			leftStatus := factStatusRank(factStatusForID(candidates[i].FactID, facts))
			rightStatus := factStatusRank(factStatusForID(candidates[j].FactID, facts))
			if leftStatus != rightStatus {
				return leftStatus < rightStatus
			}
			return candidates[i].Score > candidates[j].Score
		})
		if len(candidates) > 5 {
			candidates = candidates[:5]
		}
		matches = append(matches, candidates...)
	}
	return matches
}

func selectFactsForRequirements(requirements []JobRequirement, facts []factPromptContext, limit int) []factPromptContext {
	type scoredFact struct {
		fact  factPromptContext
		score int
	}
	terms := []string{}
	for _, req := range requirements {
		terms = append(terms, jobMatchTerms(req.RequirementText, req.Keywords)...)
	}
	terms = normalizeStringList(terms)
	scored := []scoredFact{}
	for _, fact := range facts {
		text := strings.ToLower(strings.Join([]string{
			fact.FactText,
			fact.EvidenceQuote,
			strings.Join(fact.Technologies, " "),
			fact.SectionHeading,
		}, " "))
		score := 0
		for _, term := range terms {
			if strings.Contains(text, term) {
				score += 3
			}
		}
		score += factPromptStatusWeight(fact.Status)
		if len(fact.Technologies) > 0 {
			score++
		}
		if score > 0 {
			scored = append(scored, scoredFact{fact: compactFactPromptContext(fact), score: score})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].fact.ID > scored[j].fact.ID
	})
	selected := []factPromptContext{}
	seenStatus := map[string]bool{}
	for _, item := range scored {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, item.fact)
		seenStatus[item.fact.Status] = true
	}
	for _, fact := range facts {
		if len(selected) >= limit {
			break
		}
		if seenStatus[fact.Status] {
			continue
		}
		selected = append(selected, compactFactPromptContext(fact))
		seenStatus[fact.Status] = true
	}
	return selected
}

func selectFactsForMatches(matches []JobFactMatch, facts []factPromptContext) []factPromptContext {
	ids := map[int64]bool{}
	for _, match := range matches {
		ids[match.FactID] = true
	}
	selected := []factPromptContext{}
	for _, fact := range facts {
		if ids[fact.ID] {
			selected = append(selected, compactFactPromptContext(fact))
		}
	}
	return selected
}

func compactFactPromptContext(fact factPromptContext) factPromptContext {
	fact.FactText = compactPromptText(fact.FactText, 320)
	fact.EvidenceQuote = compactPromptText(fact.EvidenceQuote, 360)
	if len(fact.RiskFlags) > 5 {
		fact.RiskFlags = fact.RiskFlags[:5]
	}
	if len(fact.Technologies) > 8 {
		fact.Technologies = fact.Technologies[:8]
	}
	return fact
}

func compactPromptText(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func factPromptStatusWeight(status string) int {
	switch status {
	case factStatusApproved:
		return 3
	case factStatusNeedsReview:
		return 2
	case factStatusRejected:
		return 1
	default:
		return 0
	}
}

func factStatusForID(id int64, facts []factPromptContext) string {
	for _, fact := range facts {
		if fact.ID == id {
			return fact.Status
		}
	}
	return ""
}

func factStatusRank(status string) int {
	switch status {
	case factStatusApproved:
		return 0
	case factStatusNeedsReview:
		return 1
	case factStatusRejected:
		return 2
	default:
		return 3
	}
}

func jobMatchTerms(text string, keywords []string) []string {
	terms := []string{}
	for _, keyword := range keywords {
		terms = append(terms, strings.ToLower(strings.TrimSpace(keyword)))
	}
	for _, token := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '#'
	}) {
		if len(token) >= 3 && !isJobStopWord(token) {
			terms = append(terms, token)
		}
	}
	return normalizeStringList(terms)
}

func isJobStopWord(token string) bool {
	switch token {
	case "the", "and", "for", "with", "that", "this", "you", "your", "our", "are", "will", "have", "has", "from", "into", "work", "role", "team", "need", "needs", "required", "requirement", "requirements", "experience", "responsibilities", "ability":
		return true
	default:
		return false
	}
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
