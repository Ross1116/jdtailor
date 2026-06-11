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
	ID                int64    `json:"id"`
	JobID             int64    `json:"job_id"`
	RequirementID     int64    `json:"requirement_id"`
	FactIDs           []int64  `json:"fact_ids"`
	ClaimIDs          []int64  `json:"claim_ids"`
	OriginHeading     string   `json:"origin_heading"`
	OriginType        string   `json:"origin_type"`
	DraftText         string   `json:"draft_text"`
	Rationale         string   `json:"rationale"`
	Status            string   `json:"status"`
	RiskFlags         []string `json:"risk_flags"`
	SelectionScore    float64  `json:"selection_score"`
	SelectedForResume bool     `json:"selected_for_resume"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type UpdateTailoredBulletDraftInput struct {
	ID        int64    `json:"id"`
	DraftText string   `json:"draft_text"`
	Rationale string   `json:"rationale"`
	Status    string   `json:"status"`
	RiskFlags []string `json:"risk_flags"`
}

type SelectTailoredBulletDraftInput struct {
	ID       int64 `json:"id"`
	Selected bool  `json:"selected"`
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
	OriginHeading string `json:"origin_heading"`
	OriginType    string `json:"origin_type"`

	RequirementID  int64   `json:"requirement_id"`
	RequirementIDs []int64 `json:"requirement_ids,omitempty"`

	FactIDs  []int64 `json:"fact_ids"`
	ClaimIDs []int64 `json:"claim_ids"`

	DraftText string   `json:"draft_text"`
	Rationale string   `json:"rationale"`
	RiskFlags []string `json:"risk_flags"`
}

type claimPromptContext struct {
	ID               int64    `json:"id"`
	Label            string   `json:"label"`
	Status           string   `json:"status"`
	ClaimType        string   `json:"claim_type"`
	Strength         string   `json:"strength"`
	EvidenceStrength string   `json:"evidence_strength"`
	SourceFactIDs    []int64  `json:"source_fact_ids"`
	Actions          []string `json:"actions"`
	Capabilities     []string `json:"capabilities"`
	Objects          []string `json:"objects"`
	Technologies     []string `json:"technologies"`
	Domains          []string `json:"domains"`
	Artifacts        []string `json:"artifacts"`
	Scope            []string `json:"scope"`
	Metrics          []string `json:"metrics"`
	Outcomes         []string `json:"outcomes"`
	ProfileContext   []string `json:"profile_context"`
	AllowedUse       []string `json:"allowed_use"`
	AllowedContexts  []string `json:"allowed_contexts"`
	BlockedContexts  []string `json:"blocked_contexts"`
	OriginHeading    string   `json:"origin_heading"`
	OriginType       string   `json:"origin_type"`
	RiskFlags        []string `json:"risk_flags"`
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
	SectionType    string   `json:"section_type"`
	Context        []string `json:"context"`
}

type bulletOriginGroup struct {
	OriginHeading string
	OriginType    string

	Requirements []JobRequirement
	Matches      []JobFactMatch
	Facts        []factPromptContext
	Claims       []CandidateClaim
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
- Recruiter/poster profile headlines, role title rows, company mission paragraphs, and employer product blurbs.
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
		requirements, replaceErr := s.replaceJobRequirements(job, parsed)
		if replaceErr != nil {
			return nil, replaceErr
		}
		_ = s.replaceJobAnalysis(buildJobAnalysis(job, requirements))
		return requirements, nil
	}
	parsed, err := parseJobRequirements(text)
	if err != nil {
		parsed = fallbackJobRequirements(job.RawText)
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM returned unusable requirement JSON and no local requirements could be extracted: %w", err)
		}
		_ = s.LogEvent("warning", "job requirements used local fallback: "+err.Error())
	}
	requirements, err := s.replaceJobRequirements(job, parsed)
	if err != nil {
		return nil, err
	}
	_ = s.replaceJobAnalysis(buildJobAnalysis(job, requirements))
	return requirements, nil
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
	claims, err := s.listApprovedClaimPromptContexts()
	if err != nil {
		return nil, err
	}
	requirementsJSON, _ := json.Marshal(requirements)
	promptFacts := selectFactsForRequirements(requirements, facts, 90)
	factsJSON, _ := json.Marshal(promptFacts)
	promptClaims := selectClaimsForRequirements(requirements, claims, 90)
	claimsJSON, _ := json.Marshal(promptClaims)
	system := `You are JD Tailor's evidence matcher. Return strict JSON only.
Match job requirements to approved candidate profile-bank atoms, then map support back to source fact IDs.`
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
- Use only IDs from <requirements_json>, <candidate_claims_json>, and <candidate_facts_json>.
- Prefer candidate_claims_json. Each claim is an approved atom-bank entry with source_fact_ids.
- Output fact_id values only from a matched claim's source_fact_ids.
- If no approved claim supports a requirement, omit it.
- Use claim atom fields to distinguish action, capability, tool, scope, domain, origin, and blocked contexts.
- Do not match on generic words alone: engineer, software, application, team, role, business, agile, communication.
- Rationale must explain the overlap and the missing caveat in one sentence.
- Score range: strong 0.75-1.0, partial 0.45-0.74, weak 0.2-0.44.

<job company="%s" title="%s"/>
<requirements_json>
%s
</requirements_json>
<candidate_claims_json>
%s
</candidate_claims_json>
<candidate_facts_json>
%s
</candidate_facts_json>`, job.Company, job.Title, string(requirementsJSON), string(claimsJSON), string(factsJSON))
	text, err := s.GenerateLLMText(ctx, client, system, user, 1600)
	if err != nil {
		parsed := fallbackJobMatchesFromClaims(requirements, claims)
		if len(parsed) == 0 {
			parsed = fallbackJobMatches(requirements, facts)
		}
		if len(parsed) == 0 {
			return nil, fmt.Errorf("LLM request failed and no local matches could be built: %w", err)
		}
		_ = s.LogEvent("warning", "job match map used local fallback after LLM request failure: "+err.Error())
		return s.replaceJobMatches(jobID, parsed, requirements, facts)
	}
	parsed, err := parseJobMatches(text)
	if err != nil {
		parsed = fallbackJobMatchesFromClaims(requirements, claims)
		if len(parsed) == 0 {
			parsed = fallbackJobMatches(requirements, facts)
		}
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
	if len(requirements) == 0 {
		return nil, errors.New("parse job requirements before generating bullet drafts")
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
	if len(facts) == 0 {
		return nil, errors.New("extract evidence facts before generating bullet drafts")
	}

	claims, err := s.listApprovedClaimsForDrafts()
	if err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		return nil, errors.New("approve profile-bank claims before generating bullet drafts")
	}

	styleRules := s.promptRuleDigest("resume", "validation")

	groups := buildBulletOriginGroups(requirements, matches, facts, claims)
	if len(groups) == 0 {
		return nil, errors.New("no same-origin evidence groups available for bullet drafting")
	}

	parsedAll := []parsedBulletDraft{}

	for _, group := range groups {
		if len(group.Requirements) == 0 || len(group.Facts) == 0 || len(group.Claims) == 0 {
			continue
		}

		parsed, err := s.generateBulletDraftsForOrigin(ctx, client, job, group, styleRules)
		if err != nil {
			parsed = fallbackBulletDraftsForOrigin(group)
			if len(parsed) == 0 {
				_ = s.LogEvent("warning", "origin bullet drafting failed for "+group.OriginHeading+": "+err.Error())
				continue
			}
			_ = s.LogEvent("warning", "origin bullet drafts used local fallback for "+group.OriginHeading+": "+err.Error())
		}

		parsedAll = append(parsedAll, parsed...)
	}

	if len(parsedAll) == 0 {
		return nil, errors.New("no usable same-origin bullet drafts generated")
	}

	return s.replaceBulletDrafts(jobID, parsedAll, requirements, facts, claims)
}

func originCanHaveResumeBullet(originType string) bool {
	switch strings.TrimSpace(strings.ToLower(originType)) {
	case "experience", "project":
		return true
	default:
		return false
	}
}

func buildBulletOriginGroups(requirements []JobRequirement, matches []JobFactMatch, facts []factPromptContext, claims []CandidateClaim) []bulletOriginGroup {
	reqsByID := map[int64]JobRequirement{}
	for _, req := range requirements {
		reqsByID[req.ID] = req
	}

	factsByID := map[int64]factPromptContext{}
	for _, fact := range facts {
		factsByID[fact.ID] = fact
	}

	claimsByFactID := map[int64][]CandidateClaim{}
	for _, claim := range claims {
		if !claimAllowedForDrafts(claim) {
			continue
		}
		for _, factID := range claim.SourceFactIDs {
			claimsByFactID[factID] = append(claimsByFactID[factID], claim)
		}
	}

	groupsByOrigin := map[string]*bulletOriginGroup{}

	for _, match := range matches {
		req, ok := reqsByID[match.RequirementID]
		if !ok {
			continue
		}

		if !requirementCanDriveBullet(req) {
			continue
		}

		if !matchCanDriveBullet(match) {
			continue
		}

		fact, ok := factsByID[match.FactID]
		if !ok {
			continue
		}

		originHeading := strings.TrimSpace(fact.SectionHeading)
		originType := strings.TrimSpace(fact.SectionType)
		if originHeading == "" || originType == "" {
			continue
		}

		if !originCanHaveResumeBullet(originType) {
			continue
		}

		key := originKey(originHeading, originType)

		group := groupsByOrigin[key]
		if group == nil {
			group = &bulletOriginGroup{
				OriginHeading: originHeading,
				OriginType:    originType,
			}
			groupsByOrigin[key] = group
		}

		group.Requirements = appendUniqueRequirement(group.Requirements, req)
		group.Matches = append(group.Matches, match)
		group.Facts = appendUniqueFact(group.Facts, fact)

		for _, claim := range claimsByFactID[match.FactID] {
			if !claimAllowedForOriginDraft(claim, originHeading, originType, factsByID) {
				continue
			}

			group.Claims = appendUniqueClaim(group.Claims, claim)

			for _, sourceFactID := range claim.SourceFactIDs {
				sourceFact, ok := factsByID[sourceFactID]
				if !ok {
					continue
				}

				if !sameOrigin(sourceFact.SectionHeading, sourceFact.SectionType, originHeading, originType) {
					continue
				}

				group.Facts = appendUniqueFact(group.Facts, sourceFact)
			}
		}
	}

	groups := []bulletOriginGroup{}
	for _, group := range groupsByOrigin {
		if len(group.Requirements) == 0 || len(group.Facts) == 0 || len(group.Claims) == 0 {
			continue
		}
		groups = append(groups, *group)
	}

	sort.Slice(groups, func(i, j int) bool {
		leftBudget := bulletBudgetForSectionType(groups[i].OriginType)
		rightBudget := bulletBudgetForSectionType(groups[j].OriginType)
		if leftBudget != rightBudget {
			return leftBudget > rightBudget
		}
		return groups[i].OriginHeading < groups[j].OriginHeading
	})

	return groups
}

func matchCanDriveBullet(match JobFactMatch) bool {
	status := strings.ToLower(strings.TrimSpace(match.CoverageStatus))

	if status == "weak" {
		return false
	}

	if match.Score < 0.72 {
		return false
	}

	return status == "strong" || status == "partial"
}

func (s *Store) generateBulletDraftsForOrigin(ctx context.Context, client *http.Client, job JobDescription, group bulletOriginGroup, styleRules string) ([]parsedBulletDraft, error) {
	reqJSON, _ := json.Marshal(group.Requirements)
	matchJSON, _ := json.Marshal(group.Matches)

	factJSON, _ := json.Marshal(group.Facts)

	promptClaims := []claimPromptContext{}
	for _, claim := range group.Claims {
		promptClaims = append(promptClaims, claimPromptContextFromCandidate(claim))
	}
	claimJSON, _ := json.Marshal(promptClaims)

	system := `You are JD Tailor's section-scoped resume bullet drafter. Return strict JSON only.
Write bullet suggestions for one resume origin only. Never use facts from another origin.`

	user := fmt.Sprintf(`# Task
Generate the strongest resume-worthy bullet suggestions for this origin only.

The JD requirements are prioritization context, not a one-to-one checklist.
Do not create a bullet just because a requirement has keyword overlap.
Only write bullets that would be valuable on a real software engineering resume.

# Locked origin
origin_heading: %s
origin_type: %s

# Output JSON schema
{"drafts":[{"origin_heading":"","origin_type":"","requirement_id":0,"requirement_ids":[0],"claim_ids":[0],"fact_ids":[0],"draft_text":"","rationale":"","risk_flags":[]}]}

# Section-scope rules
- Every draft must use origin_heading="%s" and origin_type="%s".
- Every claim_id must exist in <candidate_claims_json>.
- Every fact_id must exist in <candidate_facts_json>.
- Do not use any claim or fact from another origin.
- Do not combine this origin with other roles, employers, projects, education, or skills-only claims.
- Cross-origin evidence may influence resume-level positioning later, but not this origin's bullet text.
- JD language may influence wording only; it must not add unsupported tools, metrics, ownership, scale, domain, production scope, or responsibilities.
- If the JD asks for a tool, domain, production scope, or responsibility not supported by this origin, omit it or phrase the match conservatively.

# Resume-value rules
- Prefer bullets that show concrete engineering output, technical ownership, system design, production reliability, security, observability, or measurable scope.
- Prefer real artifacts over generic skills: APIs, backend services, data models, ingestion pipelines, auth systems, tracing, background jobs, schedulers, distributed systems, UI surfaces, packaging systems.
- Prefer supported outcomes: improved debugging, reduced manual work, reliability, validation, auditability, recovery, processing quality, or delivery breadth.
- Prefer bullets with clear action + artifact + technical detail + supported outcome/scope.
- Do not write low-value bullets only to cover missing JD terms.
- Do not force bullets for soft skills, degree requirements, team culture, collaboration, generic ownership, or domain language unless same-origin evidence directly supports it.
- If a JD requirement is only weakly supported, do not write a bullet for it unless the underlying evidence is still strong and resume-worthy.

# Requirement usage
- requirement_id should identify the main JD requirement this bullet best supports.
- requirement_ids may include secondary requirements.
- A bullet does not need to perfectly satisfy a high-priority requirement if it is still one of the strongest same-origin resume bullets.
- Do not create one bullet per requirement. Create only the strongest bullets for this origin.

# Bullet style
- Start each draft_text with a strong past-tense action verb, no leading hyphen.
- Keep each bullet 22 to 34 words.
- Use concrete artifacts, technologies, scope, and outcomes only when present in the facts.
- Write like a practical engineer, not a marketing page.
- Avoid inflated resume cliches.
- Avoid generic tool-list bullets.
- Avoid vague phrases like "worked on", "contributed to", "helped with", "various", "business outcomes", or "leveraged".
- Mention the origin in the rationale so reviewers know where the bullet belongs.
- Respect a one-page resume budget: max 5 bullets per experience origin, max 2 bullets per project origin, max 1 per education/certification origin.

# Evidence rules
- Every claim_id must exist in <candidate_claims_json>.
- Every fact_id must exist in <candidate_facts_json> and support the listed claim_id.
- Do not introduce unsupported metrics, tools, cloud platforms, leadership, security, production scope, domain experience, or business impact.
- If evidence is weak/partial, phrase conservatively; do not overclaim the exact JD requirement.
- Include risk_flags for approved-restricted claims, weak evidence, ambiguous metric/scope, or inferred tailoring.
- Do not include style risk flags such as style_too_long or style_buzzword; those are computed by code.
- Drafts are suggestions only and must not mutate locked profile/source truth.

# Rationale format
Each rationale must use this exact structure:
Resume value: explain why this is worth a resume bullet.
JD relevance: explain which JD requirement(s) it supports and how.
Missing/unsupported: mention any important JD detail that was not supported and therefore omitted.

# Human style rules
%s

<job company="%s" title="%s"/>
<requirements_json>
%s
</requirements_json>
<matches_json>
%s
</matches_json>
<candidate_claims_json>
%s
</candidate_claims_json>
<candidate_facts_json>
%s
</candidate_facts_json>`,
		group.OriginHeading,
		group.OriginType,
		group.OriginHeading,
		group.OriginType,
		firstNonEmpty(styleRules, "Use plain, specific, evidence-backed resume language."),
		job.Company,
		job.Title,
		string(reqJSON),
		string(matchJSON),
		string(claimJSON),
		string(factJSON),
	)

	text, err := s.GenerateLLMText(ctx, client, system, user, 1200)
	if err != nil {
		return nil, err
	}

	return parseBulletDrafts(text)
}

func appendUniqueRequirement(list []JobRequirement, req JobRequirement) []JobRequirement {
	for _, existing := range list {
		if existing.ID == req.ID {
			return list
		}
	}
	return append(list, req)
}

func appendUniqueFact(list []factPromptContext, fact factPromptContext) []factPromptContext {
	for _, existing := range list {
		if existing.ID == fact.ID {
			return list
		}
	}
	return append(list, fact)
}

func appendUniqueClaim(list []CandidateClaim, claim CandidateClaim) []CandidateClaim {
	for _, existing := range list {
		if existing.ID == claim.ID {
			return list
		}
	}
	return append(list, claim)
}

func requirementCanDriveBullet(req JobRequirement) bool {
	text := strings.ToLower(strings.Join([]string{
		req.Category,
		req.RequirementText,
		strings.Join(req.Keywords, " "),
	}, " "))

	// Education/credential checks belong in fit analysis, not bullet drafting.
	if strings.Contains(text, "degree") ||
		strings.Contains(text, "computer science") ||
		strings.Contains(text, "equivalent practical experience") {
		return false
	}

	// Pure soft-skill/team-culture requirements should not create technical bullets.
	softOnlyMarkers := []string{
		"collaborate with cross-functional",
		"shared outcomes",
		"communication skills",
		"team-first",
		"team first",
		"high-performing team culture",
		"proactive collaboration",
		"prioritise high-impact",
		"seek clarity",
		"ownership mentality",
		"seeing work through to completion",
	}

	for _, marker := range softOnlyMarkers {
		if strings.Contains(text, marker) {
			return false
		}
	}

	// Domain requirements should only drive bullets if the candidate has direct domain evidence.
	// For now, do not create bullets from these directly.
	if strings.EqualFold(req.Category, "domain") {
		return false
	}

	return true
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

func claimResumeValueScore(claim CandidateClaim) float64 {
	score := 0.0

	for _, action := range claim.Actions {
		switch strings.ToLower(strings.TrimSpace(action)) {
		case "built", "shipped", "designed", "implemented", "refactored", "modeled", "developed":
			score += 0.12
		case "maintained", "supported", "integrated":
			score += 0.07
		}
	}

	if len(claim.Artifacts) > 0 || len(claim.Objects) > 0 || len(claim.Capabilities) > 0 {
		score += 0.15
	}

	if len(claim.Technologies) > 0 {
		score += 0.08
	}

	if len(claim.Scope) > 0 {
		score += 0.10
	}

	if len(claim.Metrics) > 0 {
		score += 0.16
	}

	if len(claim.Outcomes) > 0 {
		score += 0.18
	}

	switch claim.Strength {
	case "strong":
		score += 0.16
	case "moderate":
		score += 0.09
	case "project-level":
		score += 0.04
	}

	if claim.EvidenceStrength == "direct" {
		score += 0.10
	}

	if claim.Status == claimStatusApprovedRestricted {
		score -= 0.10
	}

	for _, flag := range claim.RiskFlags {
		switch flag {
		case "ambiguous technology", "ambiguous_technology", "weak_evidence", "inferred_tailoring":
			score -= 0.08
		}
	}

	return clampScore(score)
}

func (s *Store) ListTailoredBulletDrafts(jobID int64) ([]TailoredBulletDraft, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at,
			claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume
		FROM tailored_bullet_drafts WHERE job_id = ?
		ORDER BY selected_for_resume DESC, selection_score DESC, CASE status WHEN 'needs_review' THEN 0 WHEN 'accepted' THEN 1 ELSE 2 END, id DESC`,
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

func (s *Store) SelectTailoredBulletDraft(input SelectTailoredBulletDraftInput) (TailoredBulletDraft, error) {
	if input.ID <= 0 {
		return TailoredBulletDraft{}, errors.New("draft id is required")
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE tailored_bullet_drafts SET selected_for_resume = ?, updated_at = ? WHERE id = ?`,
		boolToInt(input.Selected),
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return TailoredBulletDraft{}, err
	}
	return s.getTailoredBulletDraft(input.ID)
}

func (s *Store) AutoSelectResumeBullets(jobID int64) ([]TailoredBulletDraft, error) {
	drafts, err := s.ListTailoredBulletDrafts(jobID)
	if err != nil {
		return nil, err
	}
	selectedIDs := map[int64]bool{}
	originCounts := map[string]int{}
	for _, draft := range drafts {
		if draft.Status == "rejected" || draftHasUnsupportedRisk(draft.RiskFlags) {
			continue
		}

		if strings.TrimSpace(draft.OriginHeading) == "" || strings.TrimSpace(draft.OriginType) == "" {
			continue
		}

		originKey := strings.TrimSpace(draft.OriginType) + "|" + strings.TrimSpace(draft.OriginHeading)
		budget := bulletBudgetForSectionType(draft.OriginType)
		if budget <= 0 {
			budget = 2
		}

		if originCounts[originKey] >= budget {
			continue
		}

		selectedIDs[draft.ID] = true
		originCounts[originKey]++
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(context.Background(), `UPDATE tailored_bullet_drafts SET selected_for_resume = 0, updated_at = ? WHERE job_id = ?`, now, jobID); err != nil {
		return nil, err
	}
	for id := range selectedIDs {
		if _, err := tx.ExecContext(context.Background(), `UPDATE tailored_bullet_drafts SET selected_for_resume = 1, updated_at = ? WHERE id = ?`, now, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListTailoredBulletDrafts(jobID)
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
	factsByID := map[int64]factPromptContext{}
	for _, fact := range facts {
		factIDs[fact.ID] = true
		factsByID[fact.ID] = fact
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

func (s *Store) replaceBulletDrafts(jobID int64, drafts []parsedBulletDraft, requirements []JobRequirement, facts []factPromptContext, claims []CandidateClaim) ([]TailoredBulletDraft, error) {
	reqIDs := map[int64]bool{}
	reqsByID := map[int64]JobRequirement{}
	for _, req := range requirements {
		reqIDs[req.ID] = true
		reqsByID[req.ID] = req
	}

	factsByID := map[int64]factPromptContext{}
	for _, fact := range facts {
		factsByID[fact.ID] = fact
	}

	claimsByID := map[int64]CandidateClaim{}
	for _, claim := range claims {
		if !claimAllowedForDrafts(claim) {
			continue
		}
		claimsByID[claim.ID] = claim
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
	originCounts := map[string]int{}

	for _, draft := range drafts {
		draft.DraftText = strings.TrimSpace(draft.DraftText)

		if draft.RequirementID == 0 && len(draft.RequirementIDs) > 0 {
			draft.RequirementID = draft.RequirementIDs[0]
		}

		originHeading := strings.TrimSpace(draft.OriginHeading)
		originType := strings.TrimSpace(draft.OriginType)

		if draft.DraftText == "" || !reqIDs[draft.RequirementID] || originHeading == "" || originType == "" {
			continue
		}

		validClaimIDs := []int64{}
		for _, claimID := range draft.ClaimIDs {
			claim, ok := claimsByID[claimID]
			if !ok {
				continue
			}

			if !claimAllowedForOriginDraft(claim, originHeading, originType, factsByID) {
				continue
			}

			validClaimIDs = append(validClaimIDs, claimID)
		}
		validClaimIDs = uniqueInt64s(validClaimIDs)

		if len(validClaimIDs) == 0 {
			continue
		}

		claimSourceFactIDs := map[int64]bool{}
		for _, claimID := range validClaimIDs {
			claim := claimsByID[claimID]
			for _, factID := range claim.SourceFactIDs {
				claimSourceFactIDs[factID] = true
			}
		}

		validFactIDs := []int64{}
		for _, factID := range draft.FactIDs {
			fact, ok := factsByID[factID]
			if !ok {
				continue
			}

			if !claimSourceFactIDs[factID] {
				continue
			}

			if !sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
				continue
			}

			validFactIDs = append(validFactIDs, factID)
		}
		validFactIDs = uniqueInt64s(validFactIDs)

		if len(validFactIDs) == 0 {
			continue
		}

		originKey, originBudget := draftOriginBudgetFromOrigin(originHeading, originType)
		if originBudget > 0 && originCounts[originKey] >= originBudget {
			continue
		}

		factIDsJSON, err := encodeInt64List(validFactIDs)
		if err != nil {
			return nil, err
		}

		claimIDsJSON, err := encodeInt64List(validClaimIDs)
		if err != nil {
			return nil, err
		}

		draft.RiskFlags = normalizeStringList(append(
			draft.RiskFlags,
			styleRiskFlags(draft.DraftText)...,
		))

		draft.RiskFlags = normalizeStringList(append(
			draft.RiskFlags,
			sectionScopeRiskFlags(draft, validClaimIDs, claimsByID, validFactIDs, factsByID)...,
		))

		if draftHasUnsupportedRisk(draft.RiskFlags) {
			continue
		}

		riskJSON, err := encodeStringList(draft.RiskFlags)
		if err != nil {
			return nil, err
		}

		if lowResumeValueDraft(draft, validClaimIDs, claimsByID) {
			continue
		}

		selectionScore := draftSelectionScore(draft, reqsByID[draft.RequirementID], validClaimIDs, claimsByID)

		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO tailored_bullet_drafts
				(job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at,
				claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume)
			VALUES (?, ?, ?, ?, ?, 'needs_review', ?, ?, ?, ?, ?, ?, ?, 0)`,
			jobID,
			draft.RequirementID,
			factIDsJSON,
			draft.DraftText,
			strings.TrimSpace(draft.Rationale),
			riskJSON,
			now,
			now,
			claimIDsJSON,
			originHeading,
			originType,
			selectionScore,
		); err != nil {
			return nil, err
		}

		count++

		if originBudget > 0 {
			originCounts[originKey]++
		}
	}

	if count == 0 {
		return nil, errors.New("LLM returned no usable same-origin bullet drafts")
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	_ = s.LogEvent("info", "tailored bullet drafts generated")
	return s.ListTailoredBulletDrafts(jobID)
}

func lowResumeValueDraft(draft parsedBulletDraft, claimIDs []int64, claimsByID map[int64]CandidateClaim) bool {
	if draftHasUnsupportedRisk(draft.RiskFlags) {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(draft.DraftText))

	// Reject bullets that are just generic statements.
	genericPhrases := []string{
		"worked on",
		"helped with",
		"used various",
		"contributed to team",
		"collaborated with",
		"supported development",
		"participated in",
	}

	for _, phrase := range genericPhrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}

	valueScore := 0.0
	for _, claimID := range claimIDs {
		valueScore += claimResumeValueScore(claimsByID[claimID])
	}

	if len(claimIDs) > 0 {
		valueScore = valueScore / float64(len(claimIDs))
	}

	return valueScore < 0.35
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

func draftOriginBudgetFromOrigin(originHeading string, originType string) (string, int) {
	if strings.TrimSpace(originHeading) == "" {
		originHeading = "unknown"
	}

	if strings.TrimSpace(originType) == "" {
		originType = "unknown"
	}

	key := originKey(originHeading, originType)
	return key, bulletBudgetForSectionType(originType)
}

func sameOrigin(aHeading, aType, bHeading, bType string) bool {
	return normalizeOriginPart(aHeading) == normalizeOriginPart(bHeading) &&
		normalizeOriginPart(aType) == normalizeOriginPart(bType)
}

func normalizeOriginPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func originKey(heading string, typ string) string {
	return normalizeOriginPart(typ) + "|" + normalizeOriginPart(heading)
}

func claimAllowedForOriginDraft(claim CandidateClaim, originHeading string, originType string, factsByID map[int64]factPromptContext) bool {
	if !claimAllowedForDrafts(claim) {
		return false
	}

	claimOriginHeading := strings.TrimSpace(claim.OriginHeading)
	claimOriginType := strings.TrimSpace(claim.OriginType)

	if claimOriginHeading != "" || claimOriginType != "" {
		if !sameOrigin(claimOriginHeading, claimOriginType, originHeading, originType) {
			return false
		}
	}

	for _, factID := range claim.SourceFactIDs {
		fact, ok := factsByID[factID]
		if !ok {
			return false
		}

		if !sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
			return false
		}
	}

	return true
}

func sectionScopeRiskFlags(
	draft parsedBulletDraft,
	claimIDs []int64,
	claimsByID map[int64]CandidateClaim,
	factIDs []int64,
	factsByID map[int64]factPromptContext,
) []string {
	flags := []string{}

	originHeading := strings.TrimSpace(draft.OriginHeading)
	originType := strings.TrimSpace(draft.OriginType)

	if originHeading == "" || originType == "" {
		return []string{"missing_origin"}
	}

	for _, claimID := range claimIDs {
		claim, ok := claimsByID[claimID]
		if !ok {
			flags = append(flags, "unknown_claim")
			continue
		}

		claimOriginHeading := strings.TrimSpace(claim.OriginHeading)
		claimOriginType := strings.TrimSpace(claim.OriginType)

		if claimOriginHeading != "" || claimOriginType != "" {
			if !sameOrigin(claimOriginHeading, claimOriginType, originHeading, originType) {
				flags = append(flags, "cross_origin_claim")
			}
		}
	}

	for _, factID := range factIDs {
		fact, ok := factsByID[factID]
		if !ok {
			flags = append(flags, "unknown_fact")
			continue
		}

		if !sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
			flags = append(flags, "cross_origin_fact")
		}
	}

	return normalizeStringList(flags)
}

func draftOriginFromClaims(claimIDs []int64, claimsByID map[int64]CandidateClaim, factIDs []int64, factsByID map[int64]factPromptContext) (string, string) {
	for _, claimID := range claimIDs {
		claim, ok := claimsByID[claimID]
		if !ok {
			continue
		}
		if claim.OriginHeading != "" || claim.OriginType != "" {
			return claim.OriginHeading, claim.OriginType
		}
	}
	for _, factID := range factIDs {
		fact, ok := factsByID[factID]
		if ok {
			return fact.SectionHeading, fact.SectionType
		}
	}
	return "", ""
}

func draftHasUnsupportedRisk(flags []string) bool {
	for _, flag := range flags {
		switch flag {
		case "blocked_claim",
			"blocked_context",
			"unsupported_metric",
			"unsupported_tool",
			"unsupported_seniority",
			"unsupported_ownership",
			"cross_origin_claim",
			"cross_origin_fact",
			"missing_origin",
			"unknown_claim",
			"unknown_fact",
			"summary_only_claim_used_as_bullet",
			"project_claim_used_as_experience":
			return true
		}
	}

	return false
}

func draftSelectionScore(draft parsedBulletDraft, req JobRequirement, claimIDs []int64, claimsByID map[int64]CandidateClaim) float64 {
	score := 0.25

	// 1. Resume value is primary.
	valueTotal := 0.0
	for _, claimID := range claimIDs {
		claim := claimsByID[claimID]
		valueTotal += claimResumeValueScore(claim)
	}
	if len(claimIDs) > 0 {
		score += (valueTotal / float64(len(claimIDs))) * 0.45
	}

	// 2. JD relevance is secondary.
	switch req.Priority {
	case "high":
		score += 0.12
	case "medium":
		score += 0.07
	case "low":
		score += 0.02
	}

	// 3. Prefer real work experience over projects when the JD asks for production.
	reqText := strings.ToLower(req.RequirementText)
	if draft.OriginType == "experience" {
		score += 0.08
	}

	if draft.OriginType == "project" && strings.Contains(reqText, "production") {
		score -= 0.12
	}

	// 4. Penalize weak or inferred tailoring.
	for _, flag := range draft.RiskFlags {
		switch flag {
		case "weak_evidence":
			score -= 0.10
		case "inferred_tailoring":
			score -= 0.08
		case "ambiguous technology", "ambiguous_technology":
			score -= 0.06
		default:
			score -= 0.03
		}
	}

	if looksLikeToolListBullet(draft.DraftText) {
		score -= 0.12
	}

	return clampScore(score)
}

func looksLikeToolListBullet(text string) bool {
	lower := strings.ToLower(text)

	toolCount := 0
	for _, tool := range []string{
		"fastapi", "postgresql", "sql", "typescript", "react", "node.js", "node",
		"java", "spring boot", "go", "python", "docker", "aws", "sqlite",
	} {
		if strings.Contains(lower, tool) {
			toolCount++
		}
	}

	hasOutcomeVerb := false
	for _, word := range []string{
		"improving", "reducing", "accelerating", "supporting", "enabling",
		"reliable", "reliability", "observability", "debugging", "validation",
		"recovery", "audit", "scheduling", "processing",
	} {
		if strings.Contains(lower, word) {
			hasOutcomeVerb = true
			break
		}
	}

	return toolCount >= 4 && !hasOutcomeVerb
}

func bulletBudgetForSectionType(sectionType string) int {
	switch strings.TrimSpace(sectionType) {
	case "experience":
		return 5
	case "project":
		return 2
	case "education", "certification":
		return 1
	default:
		return 2
	}
}

func styleRiskFlags(text string) []string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return nil
	}
	flags := []string{}
	buzzwords := []string{
		"leveraged", "spearheaded", "empowered", "utilized", "cutting-edge", "game-changer", "innovative", "seamless", "dynamic", "pivotal",
		"deep dive", "in-depth", "unlock", "drive growth", "enhance efficiency", "business outcomes", "synergy", "transformative",
	}
	for _, word := range buzzwords {
		if strings.Contains(lower, word) {
			flags = append(flags, "style_buzzword")
			break
		}
	}
	if strings.Contains(lower, " as a result") || strings.Contains(lower, " in order to") || strings.Contains(lower, " it is important") {
		flags = append(flags, "style_filler")
	}
	if strings.Contains(text, "*") || strings.Contains(text, "•") {
		flags = append(flags, "style_formatting")
	}
	words := strings.Fields(text)
	if len(words) > 34 {
		flags = append(flags, "style_too_long")
	}
	if len(words) < 8 {
		flags = append(flags, "style_too_thin")
	}
	if strings.HasPrefix(strings.TrimSpace(text), "-") {
		flags = append(flags, "style_leading_dash")
	}
	return normalizeStringList(flags)
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
		if strings.TrimSpace(draft.OriginHeading) == "" ||
			strings.TrimSpace(draft.OriginType) == "" ||
			strings.TrimSpace(draft.DraftText) == "" ||
			len(draft.ClaimIDs) == 0 ||
			len(draft.FactIDs) == 0 {
			return nil, errors.New("every draft must include origin_heading, origin_type, draft_text, claim_ids, and fact_ids")
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
	requirementText := strings.TrimSpace(req.RequirementText)
	sourceQuote := strings.TrimSpace(req.SourceQuote)
	text := strings.ToLower(strings.Join([]string{
		req.Category,
		requirementText,
		sourceQuote,
		strings.Join(req.Keywords, " "),
	}, " "))
	if text == "" {
		return true
	}
	if isJobBoilerplateLine(requirementText) || isJobBoilerplateLine(sourceQuote) {
		return true
	}
	irrelevantMarkers := []string{
		"12-month", "12 month", "contract", "max term", "fixed term", "salary", "compensation", "benefit", "leave", "hybrid", "remote", "location", "office",
		"how to apply", "application process", "submit your application", "recruit", "hiring", "interview", "equal opportunity", "diversity", "background check", "sponsorship",
		"leading personal injury", "class actions law firm", "about us", "about the company", "company is", "we are a", "we're a", "our client", "we believe", "deserves to feel", "redefining", "platform provides", "dedicated team", "immediate care", "critical situations",
		"logo", "linkedin", "promoted by", "responses managed", "profile matches", "is this information helpful", "personalized tips", "top applicant", "retry premium", "people you can reach out", "school alumni", "clicked apply",
	}
	for _, marker := range irrelevantMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	if isJobHeadingOrMetadata(requirementText) || isJobHeadingOrMetadata(sourceQuote) {
		return true
	}
	if !hasStrictTailorableRequirementSignal(requirementText, req.Keywords) {
		return true
	}
	if strings.EqualFold(req.Category, "domain") && !strings.Contains(text, "experience") && !strings.Contains(text, "knowledge") && !strings.Contains(text, "background") {
		return true
	}
	if strings.EqualFold(req.Category, "seniority") && !strings.Contains(text, "senior") && !strings.Contains(text, "lead") && !strings.Contains(text, "mentor") && !strings.Contains(text, "years") {
		return true
	}
	if len(jobMatchTerms(requirementText, req.Keywords)) == 0 {
		return true
	}
	return false
}

func hasStrictTailorableRequirementSignal(text string, keywords []string) bool {
	if isJobBoilerplateLine(text) {
		return false
	}
	lower := strings.ToLower(strings.Join(append([]string{text}, keywords...), " "))
	for _, signal := range []string{
		"experience", "hands-on", "strong", "deep", "knowledge", "understanding", "programming", "skills",
		"design ", "design and", "build ", "build and", "develop", "deliver", "modernis", "test", "support ",
		"architecture", "api", "apis", "data ingestion", "high-concurrency", "dashboard", "dashboards",
		"workflow", "workflows", "rag", "retrieval", "chunking", "metadata filtering", "model context protocol", "evaluation framework",
		"cloud", "serverless", "event-driven", "distributed", "scalable", "resilience", "observability", "security",
		"networking", "identity", "database", "nosql", "data model", "data models", "schema",
		"devsecops", "agile", "solid", "containers", "messaging", "queues", "topics", "stakeholder", "mentor",
		"engineering practices", "technical excellence",
	} {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	for _, tech := range []string{"fastapi", "postgresql", "react", "next.js", "typescript", "javascript", "python", "golang", "java", "spring", "mysql", "node.js", "node", "azure", "cosmos db", "aws", "gcp", "docker", "kubernetes", "terraform", "redis", "snowflake", "langgraph", "crewai", "mcp", "langsmith", "arize", "phoenix", "cursor", "claudecode", "copilot"} {
		if strings.Contains(lower, tech) {
			return true
		}
	}
	return false
}

func isJobBoilerplateLine(line string) bool {
	cleaned := strings.TrimSpace(line)
	lower := strings.ToLower(cleaned)
	if cleaned == "" {
		return true
	}
	if strings.Count(cleaned, "|") >= 2 {
		for _, marker := range []string{"acting cto", "vp of", "founder", "recruiter", "talent", "hiring", "ex-", " at "} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	if looksLikeRoleTitle(cleaned) && !hasConcreteRequirementVerb(cleaned) && len(strings.Fields(cleaned)) <= 10 {
		return true
	}
	for _, marker := range []string{"we believe", "our mission", "deserves to feel", "redefining", "platform provides", "dedicated team", "members receive", "immediate care", "critical situations"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func hasConcreteRequirementVerb(line string) bool {
	lower := strings.ToLower(line)
	for _, verb := range []string{"build ", "build and", "design ", "design and", "develop", "ship ", "implement", "maintain", "optimize", "architect", "own ", "write ", "review ", "triage ", "debug", "deploy"} {
		if strings.Contains(lower, verb) || strings.HasPrefix(lower, strings.TrimSpace(verb)+" ") {
			return true
		}
	}
	return false
}

func fallbackBulletDraftsForOrigin(group bulletOriginGroup) []parsedBulletDraft {
	reqsByID := map[int64]JobRequirement{}
	for _, req := range group.Requirements {
		reqsByID[req.ID] = req
	}

	claimsByFactID := map[int64][]CandidateClaim{}
	for _, claim := range group.Claims {
		if !claimAllowedForDrafts(claim) {
			continue
		}

		for _, factID := range claim.SourceFactIDs {
			claimsByFactID[factID] = append(claimsByFactID[factID], claim)
		}
	}

	drafts := []parsedBulletDraft{}
	seen := map[string]bool{}

	for _, match := range group.Matches {
		req, ok := reqsByID[match.RequirementID]
		if !ok {
			continue
		}

		for _, claim := range claimsByFactID[match.FactID] {
			text := fallbackBulletTextFromClaim(claim, req)
			if text == "" {
				continue
			}

			key := fmt.Sprintf("%d|%d|%s", req.ID, claim.ID, strings.ToLower(text))
			if seen[key] {
				continue
			}
			seen[key] = true

			drafts = append(drafts, parsedBulletDraft{
				OriginHeading: group.OriginHeading,
				OriginType:    group.OriginType,
				RequirementID: req.ID,
				ClaimIDs:      []int64{claim.ID},
				FactIDs:       []int64{match.FactID},
				DraftText:     text,
				Rationale:     "Local same-origin atom-bank draft for " + req.RequirementText + " from " + group.OriginHeading,
				RiskFlags:     claim.RiskFlags,
			})
		}
	}

	return drafts
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
		if isJobHeadingOrMetadata(line) || !hasStrictTailorableRequirementSignal(line, nil) {
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

func fallbackJobMatchesFromClaims(requirements []JobRequirement, claims []claimPromptContext) []parsedJobMatch {
	matches := []parsedJobMatch{}
	for _, req := range requirements {
		reqTerms := jobMatchTerms(req.RequirementText, req.Keywords)
		if len(reqTerms) == 0 {
			continue
		}
		candidates := []parsedJobMatch{}
		for _, claim := range claims {
			if len(claim.SourceFactIDs) == 0 {
				continue
			}
			claimText := strings.ToLower(claimSearchText(claim))
			overlap := []string{}
			for _, term := range reqTerms {
				if strings.Contains(claimText, term) {
					overlap = append(overlap, term)
				}
			}
			overlap = normalizeStringList(overlap)
			if len(overlap) == 0 {
				continue
			}
			score := 0.42 + float64(len(overlap))*0.17 + claimStrengthWeight(claim)
			status := "partial"
			if len(overlap) >= 3 || score >= 0.75 {
				status = "strong"
			}
			if len(overlap) == 1 {
				status = "weak"
			}
			candidates = append(candidates, parsedJobMatch{
				RequirementID:  req.ID,
				FactID:         claim.SourceFactIDs[0],
				Score:          clampScore(score),
				Rationale:      "Atom-bank overlap: " + strings.Join(overlap, ", ") + " via " + claim.Label,
				CoverageStatus: status,
			})
		}
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Score > candidates[j].Score
		})
		if len(candidates) > 5 {
			candidates = candidates[:5]
		}
		matches = append(matches, candidates...)
	}
	return matches
}

func fallbackBulletDraftsFromClaims(requirements []JobRequirement, matches []JobFactMatch, claims []CandidateClaim) []parsedBulletDraft {
	requirementsByID := map[int64]JobRequirement{}
	for _, req := range requirements {
		requirementsByID[req.ID] = req
	}
	claimsByFactID := map[int64][]CandidateClaim{}
	for _, claim := range claims {
		for _, factID := range claim.SourceFactIDs {
			claimsByFactID[factID] = append(claimsByFactID[factID], claim)
		}
	}
	drafts := []parsedBulletDraft{}
	seen := map[string]bool{}
	for _, match := range matches {
		req, ok := requirementsByID[match.RequirementID]
		if !ok {
			continue
		}
		for _, claim := range claimsByFactID[match.FactID] {
			if !claimAllowedForDrafts(claim) {
				continue
			}
			text := fallbackBulletTextFromClaim(claim, req)
			if text == "" {
				continue
			}
			key := fmt.Sprintf("%d|%d|%s", req.ID, claim.ID, strings.ToLower(text))
			if seen[key] {
				continue
			}
			seen[key] = true
			drafts = append(drafts, parsedBulletDraft{
				RequirementID: req.ID,
				ClaimIDs:      []int64{claim.ID},
				FactIDs:       claim.SourceFactIDs,
				DraftText:     text,
				Rationale:     "Local atom-bank draft for " + req.RequirementText + " from " + claim.OriginHeading,
				RiskFlags:     claim.RiskFlags,
			})
		}
	}
	return drafts
}

func fallbackBulletTextFromClaim(claim CandidateClaim, req JobRequirement) string {
	action := firstNonEmpty(firstString(claim.Actions), "Built")
	action = strings.TrimSuffix(strings.Title(strings.ToLower(action)), "ed")
	if strings.EqualFold(action, "built") {
		action = "Built"
	}
	capability := firstNonEmpty(firstString(claim.Capabilities), firstString(claim.Objects), firstString(claim.Artifacts), claim.ClaimText)
	tools := strings.Join(limitStrings(claim.Technologies, 3), "/")
	scope := strings.Join(limitStrings(firstNonEmptyStringList(claim.Scope, claim.Objects, claim.Artifacts), 3), ", ")
	outcome := strings.Join(limitStrings(claim.Outcomes, 2), ", ")
	parts := []string{action}
	if capability != "" {
		parts = append(parts, capability)
	}
	if tools != "" {
		parts = append(parts, "using "+tools)
	}
	if scope != "" && !strings.Contains(strings.ToLower(strings.Join(parts, " ")), strings.ToLower(scope)) {
		parts = append(parts, "across "+scope)
	}
	if outcome != "" {
		parts = append(parts, "to support "+outcome)
	}
	bullet := strings.TrimSpace(strings.Join(parts, " "))
	if bullet == "" || strings.EqualFold(bullet, claim.ClaimText) {
		bullet = strings.TrimSpace(claim.ClaimText)
	}
	bullet = strings.TrimSuffix(bullet, ".")
	if bullet == "" || isIrrelevantJobRequirement(parsedJobRequirement{RequirementText: req.RequirementText, SourceQuote: req.SourceQuote, Keywords: req.Keywords, Category: req.Category}) {
		return bullet
	}
	return bullet + "."
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

func (s *Store) listApprovedClaimPromptContexts() ([]claimPromptContext, error) {
	claims, err := s.listApprovedClaimsForDrafts()
	if err != nil {
		return nil, err
	}
	result := []claimPromptContext{}
	for _, claim := range claims {
		result = append(result, claimPromptContextFromCandidate(claim))
	}
	return result, nil
}

func (s *Store) listApprovedClaimsForDrafts() ([]CandidateClaim, error) {
	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return nil, err
	}
	result := []CandidateClaim{}
	for _, claim := range claims {
		if claimAllowedForDrafts(claim) {
			result = append(result, claim)
		}
	}
	return result, nil
}

func claimAllowedForDrafts(claim CandidateClaim) bool {
	if claim.Status != claimStatusApproved && claim.Status != claimStatusApprovedRestricted {
		return false
	}

	for _, flag := range claim.RiskFlags {
		switch flag {
		case "blocked_claim",
			"blocked_context",
			"unsupported_metric",
			"unsupported_tool",
			"cross_origin_summary_only",
			"summary_only_claim":
			return false
		}
	}

	return len(claim.SourceFactIDs) > 0
}

func claimPromptContextFromCandidate(claim CandidateClaim) claimPromptContext {
	return claimPromptContext{
		ID:               claim.ID,
		Label:            claim.ClaimText,
		Status:           claim.Status,
		ClaimType:        claim.ClaimType,
		Strength:         claim.Strength,
		EvidenceStrength: claim.EvidenceStrength,
		SourceFactIDs:    claim.SourceFactIDs,
		Actions:          claim.Actions,
		Capabilities:     claim.Capabilities,
		Objects:          claim.Objects,
		Technologies:     claim.Technologies,
		Domains:          claim.Domains,
		Artifacts:        claim.Artifacts,
		Scope:            claim.Scope,
		Metrics:          claim.Metrics,
		Outcomes:         claim.Outcomes,
		ProfileContext:   claim.ProfileContext,
		AllowedUse:       claim.AllowedUse,
		AllowedContexts:  claim.AllowedContexts,
		BlockedContexts:  claim.BlockedContexts,
		OriginHeading:    claim.OriginHeading,
		OriginType:       claim.OriginType,
		RiskFlags:        claim.RiskFlags,
	}
}

func selectClaimsForRequirements(requirements []JobRequirement, claims []claimPromptContext, limit int) []claimPromptContext {
	terms := []string{}
	for _, req := range requirements {
		terms = append(terms, jobMatchTerms(req.RequirementText, req.Keywords)...)
	}
	terms = normalizeStringList(terms)
	type scoredClaim struct {
		claim claimPromptContext
		score int
	}
	scored := []scoredClaim{}
	for _, claim := range claims {
		text := strings.ToLower(claimSearchText(claim))
		score := 0
		for _, term := range terms {
			if strings.Contains(text, term) {
				score += 3
			}
		}
		if claim.Strength == "strong" {
			score += 2
		}
		if claim.EvidenceStrength == "direct" {
			score++
		}
		if len(claim.Technologies) > 0 {
			score++
		}
		if score > 0 {
			scored = append(scored, scoredClaim{claim: compactClaimPromptContext(claim), score: score})
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].claim.ID > scored[j].claim.ID
	})
	selected := []claimPromptContext{}
	for _, item := range scored {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, item.claim)
	}
	return selected
}

func selectClaimsForMatches(matches []JobFactMatch, claims []CandidateClaim) []claimPromptContext {
	matchedFactIDs := map[int64]bool{}
	for _, match := range matches {
		matchedFactIDs[match.FactID] = true
	}
	selected := []claimPromptContext{}
	for _, claim := range claims {
		for _, factID := range claim.SourceFactIDs {
			if matchedFactIDs[factID] {
				selected = append(selected, compactClaimPromptContext(claimPromptContextFromCandidate(claim)))
				break
			}
		}
	}
	return selected
}

func compactClaimPromptContext(claim claimPromptContext) claimPromptContext {
	claim.SourceFactIDs = limitInt64s(claim.SourceFactIDs, 6)
	claim.Actions = limitStrings(claim.Actions, 6)
	claim.Capabilities = limitStrings(claim.Capabilities, 8)
	claim.Objects = limitStrings(claim.Objects, 8)
	claim.Technologies = limitStrings(claim.Technologies, 10)
	claim.Domains = limitStrings(claim.Domains, 6)
	claim.Artifacts = limitStrings(claim.Artifacts, 8)
	claim.Scope = limitStrings(claim.Scope, 8)
	claim.Metrics = limitStrings(claim.Metrics, 5)
	claim.Outcomes = limitStrings(claim.Outcomes, 5)
	claim.ProfileContext = limitStrings(claim.ProfileContext, 8)
	claim.AllowedContexts = limitStrings(claim.AllowedContexts, 8)
	claim.BlockedContexts = limitStrings(claim.BlockedContexts, 8)
	claim.RiskFlags = limitStrings(claim.RiskFlags, 6)
	return claim
}

func claimSearchText(claim claimPromptContext) string {
	return strings.Join([]string{
		claim.Label,
		strings.Join(claim.Actions, " "),
		strings.Join(claim.Capabilities, " "),
		strings.Join(claim.Objects, " "),
		strings.Join(claim.Technologies, " "),
		strings.Join(claim.Domains, " "),
		strings.Join(claim.Artifacts, " "),
		strings.Join(claim.Scope, " "),
		strings.Join(claim.Metrics, " "),
		strings.Join(claim.Outcomes, " "),
		strings.Join(claim.ProfileContext, " "),
		strings.Join(claim.AllowedContexts, " "),
		claim.OriginHeading,
		claim.OriginType,
	}, " ")
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
	if len(fact.Context) > 8 {
		fact.Context = fact.Context[:8]
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

func claimStrengthWeight(claim claimPromptContext) float64 {
	score := 0.0
	switch claim.Strength {
	case "strong":
		score += 0.08
	case "moderate":
		score += 0.04
	}
	if claim.EvidenceStrength == "direct" {
		score += 0.04
	}
	if claim.Status == claimStatusApprovedRestricted {
		score -= 0.05
	}
	return score
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyStringList(lists ...[]string) []string {
	for _, values := range lists {
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func limitInt64s(values []int64, limit int) []int64 {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func uniqueInt64s(values []int64) []int64 {
	result := []int64{}
	seen := map[int64]bool{}
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
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
	case "the", "and", "for", "with", "that", "this", "you", "your", "our", "are",
		"will", "have", "has", "from", "into", "work", "role", "need", "needs", "required", "requirement",
		"requirements", "experience", "responsibilities", "ability", "it", "development", "tools", "platform",
		"build", "built", "deliver", "design", "own", "ownership", "support", "across", "shared", "outcomes",
		"health", "health-tech", "team", "collaborate", "cross", "group", "completion", "workflow", "workflows":
		return true
	default:
		return false
	}
}

func (s *Store) listFactPromptContext() ([]factPromptContext, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT f.id, f.status, f.confidence, f.risk_flags_json, f.fact_text, f.evidence_quote, f.technologies_json,
			COALESCE(NULLIF(f.origin_heading, ''), s.heading),
			COALESCE(NULLIF(f.origin_type, ''), s.section_type),
			f.context_json
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
		var contextJSON string
		if err := rows.Scan(&fact.ID, &fact.Status, &fact.Confidence, &riskJSON, &fact.FactText, &fact.EvidenceQuote, &techJSON, &fact.SectionHeading, &fact.SectionType, &contextJSON); err != nil {
			return nil, err
		}
		fact.RiskFlags = decodeStringList(riskJSON)
		fact.Technologies = decodeStringList(techJSON)
		fact.Context = decodeStringList(contextJSON)
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
		`SELECT id, job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at,
			claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume
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
		var claimIDsJSON string
		var selected int
		if err := rows.Scan(
			&draft.ID,
			&draft.JobID,
			&draft.RequirementID,
			&factIDsJSON,
			&draft.DraftText,
			&draft.Rationale,
			&draft.Status,
			&riskJSON,
			&draft.CreatedAt,
			&draft.UpdatedAt,
			&claimIDsJSON,
			&draft.OriginHeading,
			&draft.OriginType,
			&draft.SelectionScore,
			&selected,
		); err != nil {
			return nil, err
		}
		draft.FactIDs = decodeInt64List(factIDsJSON)
		draft.ClaimIDs = decodeInt64List(claimIDsJSON)
		draft.RiskFlags = decodeStringList(riskJSON)
		draft.SelectedForResume = intToBool(selected)
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
