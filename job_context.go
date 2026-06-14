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
	ID                 int64    `json:"id"`
	JobID              int64    `json:"job_id"`
	RequirementID      int64    `json:"requirement_id"`
	FactIDs            []int64  `json:"fact_ids"`
	ClaimIDs           []int64  `json:"claim_ids"`
	OriginHeading      string   `json:"origin_heading"`
	OriginType         string   `json:"origin_type"`
	ValueTheme         string   `json:"value_theme"`
	DraftText          string   `json:"draft_text"`
	Rationale          string   `json:"rationale"`
	Status             string   `json:"status"`
	RiskFlags          []string `json:"risk_flags"`
	SelectionScore     float64  `json:"selection_score"`
	ResumeValueScore   float64  `json:"resume_value_score"`
	JDRelevanceScore   float64  `json:"jd_relevance_score"`
	OriginWeight       float64  `json:"origin_weight"`
	RiskPenalty        float64  `json:"risk_penalty"`
	UnsupportedPenalty float64  `json:"unsupported_context_penalty"`
	SelectionReason    string   `json:"selection_reason"`
	DisplayOrder       int      `json:"display_order"`
	SelectedForResume  bool     `json:"selected_for_resume"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type BulletGenerationEvent struct {
	ID            int64  `json:"id"`
	JobID         int64  `json:"job_id"`
	OriginHeading string `json:"origin_heading"`
	Stage         string `json:"stage"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	DraftText     string `json:"draft_text"`
	CreatedAt     string `json:"created_at"`
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
	ValueTheme    string `json:"value_theme"`

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

type evidencePacket struct {
	ID             int64                `json:"id"`
	OriginHeading  string               `json:"origin_heading"`
	OriginType     string               `json:"origin_type"`
	ValueTheme     string               `json:"value_theme"`
	RequirementIDs []int64              `json:"requirement_ids"`
	Facts          []factPromptContext  `json:"facts"`
	Claims         []claimPromptContext `json:"claims"`
	Atoms          map[string][]string  `json:"atoms"`
	SupportText    string               `json:"support_text"`
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
	rawText := normalizePastedText(input.RawText)
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
	rawText := normalizePastedText(input.RawText)
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
- Technical skills, tools, languages, frameworks, databases, cloud platforms, and required architecture patterns.
- Concrete engineering responsibilities that a resume bullet can support: build, design, migrate, test, own, troubleshoot, mentor, integrate.
- Seniority signals only when they imply experience depth, technical leadership, mentoring, influence, or years.
- Domain requirements only when the JD explicitly asks for domain experience/background/knowledge.

# Reject
- LinkedIn/page chrome: logos, promoted text, profile match banners, Premium upsells, alumni/people panels, "helpful" prompts.
- Company/about-us blurbs, generic mission statements, benefits, salary, location, contract duration, application instructions.
- Recruiter/poster profile headlines, role title rows, company mission paragraphs, employer product blurbs, and "why join us" sections.
- Role-title-only metadata such as "Software Engineer".
- Generic soft skills unless tied to concrete engineering work: communication, passion, mindset, openness, team player, curiosity.
- Customer/product mindset and global-culture statements unless they are tied to a concrete engineering artifact or delivery responsibility.
- Success timeline sections such as "in 6 months" or "in 12 months" unless they introduce a new hard requirement.
- Optional stack lists should be category "nice_to_have" and priority "low", not "must_have".

# Quality bar
- Return 6 to 12 highest-value requirements at most.
- Make each requirement atomic: one skill/responsibility cluster per item.
- Remove decorative heading prefixes. For "Live Data Stream Processing: Dive into real-time processing using Kafka", output "Real-time data processing using Kafka".
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
	parsed = mergeParsedJobRequirements(parsed, fallbackJobRequirements(job.RawText))
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
	parsed = mergeParsedJobMatches(parsed, fallbackJobMatchesFromClaims(requirements, claims))
	parsed = mergeParsedJobMatches(parsed, fallbackJobMatches(requirements, facts))
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
	bulletClient := llmClientWithMinTimeout(client, 120*time.Second)

	_ = s.clearBulletGenerationEvents(jobID)

	groups := buildBulletOriginGroups(requirements, matches, facts, claims)
	if len(groups) == 0 {
		return nil, errors.New("no same-origin evidence groups available for bullet drafting")
	}

	parsedAll := []parsedBulletDraft{}

	for _, group := range groups {
		_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "origin_grouped", "ok", fmt.Sprintf("reqs=%d facts=%d claims=%d", len(group.Requirements), len(group.Facts), len(group.Claims)), "")
		if len(group.Requirements) == 0 || len(group.Facts) == 0 || len(group.Claims) == 0 {
			_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "origin_grouped", "skipped", "missing requirements, facts, or claims", "")
			_ = s.LogEvent(
				"warning",
				fmt.Sprintf(
					"origin skipped before drafting: %s reqs=%d facts=%d claims=%d",
					group.OriginHeading,
					len(group.Requirements),
					len(group.Facts),
					len(group.Claims),
				),
			)
			continue
		}
		packets := buildEvidencePackets(group)
		_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "packets_built", "ok", fmt.Sprintf("packets=%d", len(packets)), "")
		if len(packets) == 0 {
			_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "packets_built", "skipped", "no same-origin evidence packets", "")
			continue
		}

		parsed, err := s.generateBulletDraftsForOrigin(ctx, bulletClient, job, group, packets, styleRules)
		if err != nil {
			_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "llm_failed", "failed", err.Error(), "")
			_ = s.LogEvent("warning", "origin bullet drafting failed for "+group.OriginHeading+": "+err.Error())
			continue
		}
		_ = s.recordBulletGenerationEvent(jobID, group.OriginHeading, "llm_returned", "ok", fmt.Sprintf("drafts=%d", len(parsed)), "")

		_ = s.LogEvent(
			"info",
			fmt.Sprintf(
				"origin bullet drafting returned drafts: %s drafts=%d reqs=%d facts=%d claims=%d",
				group.OriginHeading,
				len(parsed),
				len(group.Requirements),
				len(group.Facts),
				len(group.Claims),
			),
		)

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

	enrichBulletOriginGroupsWithOriginClaims(groupsByOrigin, claims, factsByID)

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

func enrichBulletOriginGroupsWithOriginClaims(groupsByOrigin map[string]*bulletOriginGroup, claims []CandidateClaim, factsByID map[int64]factPromptContext) {
	for _, group := range groupsByOrigin {
		for _, claim := range claims {
			if !claimAllowedForDrafts(claim) || len(claim.SourceFactIDs) == 0 {
				continue
			}
			if !claimAllowedForOriginDraft(claim, group.OriginHeading, group.OriginType, factsByID) {
				continue
			}
			group.Claims = appendUniqueClaim(group.Claims, claim)
			for _, factID := range claim.SourceFactIDs {
				fact, ok := factsByID[factID]
				if !ok || !sameOrigin(fact.SectionHeading, fact.SectionType, group.OriginHeading, group.OriginType) {
					continue
				}
				group.Facts = appendUniqueFact(group.Facts, fact)
			}
		}
	}
}

func buildEvidencePackets(group bulletOriginGroup) []evidencePacket {
	reqIDs := []int64{}
	for _, req := range group.Requirements {
		reqIDs = append(reqIDs, req.ID)
	}
	factsByID := map[int64]factPromptContext{}
	for _, fact := range group.Facts {
		factsByID[fact.ID] = fact
	}
	packetsByTheme := map[string]*evidencePacket{}
	for _, claim := range group.Claims {
		if len(claim.SourceFactIDs) == 0 {
			continue
		}
		for _, theme := range inferClaimValueThemes(claim) {
			packet := packetsByTheme[theme]
			if packet == nil {
				packet = &evidencePacket{
					ID:             int64(len(packetsByTheme) + 1),
					OriginHeading:  group.OriginHeading,
					OriginType:     group.OriginType,
					ValueTheme:     theme,
					RequirementIDs: reqIDs,
					Atoms:          map[string][]string{},
				}
				packetsByTheme[theme] = packet
			}
			packet.Claims = append(packet.Claims, claimPromptContextFromCandidate(claim))
			appendPacketAtoms(packet, claim)
			for _, factID := range claim.SourceFactIDs {
				fact, ok := factsByID[factID]
				if !ok {
					continue
				}
				packet.Facts = appendUniqueFact(packet.Facts, fact)
			}
		}
	}
	packets := []evidencePacket{}
	for _, packet := range packetsByTheme {
		if len(packet.Facts) == 0 || len(packet.Claims) == 0 {
			continue
		}
		packet.Facts = limitFactPromptContext(packet.Facts, 4)
		if len(packet.Claims) > 4 {
			packet.Claims = packet.Claims[:4]
		}
		packet.SupportText = packetSupportText(*packet)
		packets = append(packets, *packet)
	}
	sort.SliceStable(packets, func(i, j int) bool {
		return valueThemeOrder(packets[i].ValueTheme) < valueThemeOrder(packets[j].ValueTheme)
	})
	return packets
}

func appendPacketAtoms(packet *evidencePacket, claim CandidateClaim) {
	add := func(key string, values []string) {
		if len(values) == 0 {
			return
		}
		packet.Atoms[key] = normalizeStringList(append(packet.Atoms[key], values...))
	}
	add("actions", claim.Actions)
	add("capabilities", claim.Capabilities)
	add("objects", claim.Objects)
	add("technologies", claim.Technologies)
	add("domains", claim.Domains)
	add("artifacts", claim.Artifacts)
	add("scope", claim.Scope)
	add("metrics", claim.Metrics)
	add("outcomes", claim.Outcomes)
}

func inferClaimValueTheme(claim CandidateClaim) string {
	themes := inferClaimValueThemes(claim)
	if len(themes) == 0 {
		return "engineering_delivery"
	}
	return themes[0]
}

func inferClaimValueThemes(claim CandidateClaim) []string {
	text := strings.ToLower(strings.Join([]string{
		claim.ClaimText,
		strings.Join(claim.Actions, " "),
		strings.Join(claim.Capabilities, " "),
		strings.Join(claim.Objects, " "),
		strings.Join(claim.Artifacts, " "),
		strings.Join(claim.Scope, " "),
		strings.Join(claim.Outcomes, " "),
	}, " "))
	themes := []string{}
	if strings.Contains(text, "booking") || strings.Contains(text, "scheduling") || strings.Contains(text, "workflow") || strings.Contains(text, "api") || strings.Contains(text, "backend") || strings.Contains(text, "platform") || strings.Contains(text, "shipped") {
		themes = append(themes, "product_platform_delivery")
	}
	if strings.Contains(text, "architecture") || strings.Contains(text, "design") || strings.Contains(text, "data model") || strings.Contains(text, "maintainable") || strings.Contains(text, "service") {
		themes = append(themes, "technical_design")
	}
	if strings.Contains(text, "audit") || strings.Contains(text, "rbac") || strings.Contains(text, "access") || strings.Contains(text, "security") || strings.Contains(text, "integrity") || strings.Contains(text, "traceability") {
		themes = append(themes, "security_traceability")
	}
	if strings.Contains(text, "debug") || strings.Contains(text, "reliability") || strings.Contains(text, "observability") || strings.Contains(text, "validation") || strings.Contains(text, "recovery") {
		themes = append(themes, "reliability_quality")
	}
	if strings.Contains(text, "ai") || strings.Contains(text, "llm") || strings.Contains(text, "extract") || strings.Contains(text, "token") || strings.Contains(text, "automation") {
		themes = append(themes, "automation_ai")
	}
	if strings.Contains(text, "react") || strings.Contains(text, "ui") || strings.Contains(text, "frontend") || strings.Contains(text, "dashboard") {
		themes = append(themes, "frontend_product")
	}
	if len(themes) == 0 {
		themes = append(themes, "engineering_delivery")
	}
	return normalizeStringList(themes)
}

func valueThemeOrder(theme string) int {
	switch theme {
	case "product_platform_delivery":
		return 10
	case "technical_design":
		return 20
	case "reliability_quality", "security_traceability":
		return 30
	case "automation_ai":
		return 40
	case "frontend_product":
		return 50
	default:
		return 60
	}
}

func limitFactPromptContext(facts []factPromptContext, limit int) []factPromptContext {
	if len(facts) <= limit {
		return facts
	}
	return facts[:limit]
}

func packetSupportText(packet evidencePacket) string {
	parts := []string{packet.ValueTheme}
	for key, values := range packet.Atoms {
		if len(values) == 0 {
			continue
		}
		parts = append(parts, key+"="+strings.Join(limitStrings(values, 5), ", "))
	}
	for _, fact := range packet.Facts {
		parts = append(parts, fact.FactText)
	}
	return strings.Join(parts, " | ")
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

func (s *Store) generateBulletDraftsForOrigin(ctx context.Context, client *http.Client, job JobDescription, group bulletOriginGroup, packets []evidencePacket, styleRules string) ([]parsedBulletDraft, error) {
	attempts := []struct {
		maxDrafts int
		compact   bool
	}{
		{maxDrafts: 3},
		{maxDrafts: 2, compact: true},
		{maxDrafts: 1, compact: true},
	}

	var lastErr error
	for index, attempt := range attempts {
		_ = s.recordBulletGenerationEvent(job.ID, group.OriginHeading, "llm_started", "attempt", fmt.Sprintf("attempt=%d max_drafts=%d compact=%t", index+1, attempt.maxDrafts, attempt.compact), "")
		parsed, err := s.generateBulletDraftsForOriginAttempt(ctx, client, job, group, packets, styleRules, attempt.maxDrafts, attempt.compact)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
		_ = s.recordBulletGenerationEvent(job.ID, group.OriginHeading, "parse_failed", "retry", err.Error(), "")
	}
	return nil, lastErr
}

func (s *Store) generateBulletDraftsForOriginAttempt(ctx context.Context, client *http.Client, job JobDescription, group bulletOriginGroup, packets []evidencePacket, styleRules string, maxDrafts int, compact bool) ([]parsedBulletDraft, error) {
	reqJSON, _ := json.Marshal(group.Requirements)
	matchJSON, _ := json.Marshal(group.Matches)
	packetJSON, _ := json.Marshal(packets)

	system := `You are JD Tailor's section-scoped resume bullet drafter. Return strict JSON only.
Write bullet suggestions for one resume origin only. Never use facts from another origin.`

	resumeValueRules := `- Prefer bullets that show concrete engineering output, technical ownership, system design, production reliability, security, observability, or measurable scope.
- Prefer real artifacts over generic skills: APIs, backend services, data models, ingestion pipelines, auth systems, tracing, background jobs, schedulers, distributed systems, UI surfaces, packaging systems.
- Prefer supported outcomes: improved debugging, reduced manual work, reliability, validation, auditability, recovery, processing quality, or delivery breadth.
- Prefer bullets with clear action + artifact + technical detail + supported outcome/scope.
- Do not write low-value bullets only to cover missing JD terms.
- Do not force bullets for soft skills, degree requirements, team culture, collaboration, generic ownership, or domain language unless same-origin evidence directly supports it.
- If a JD requirement is only weakly supported, do not write a bullet for it unless the underlying evidence is still strong and resume-worthy.`
	if compact {
		resumeValueRules = `- Return only the best evidence-backed bullets for this origin.
- Prefer concrete software artifacts, production reliability, architecture, data, security, observability, or shipped UI surfaces.
- Skip degree, soft-skill, culture, generic collaboration, and weak keyword-overlap requirements.`
	}

	user := fmt.Sprintf(`# Task
Generate the strongest resume-worthy bullet suggestions for this origin only.

The JD requirements are prioritization context, not a one-to-one checklist.
Do not create a bullet just because a requirement has keyword overlap.
Only write bullets that would be valuable on a real software engineering resume.
The final set should read like a coherent resume section, not repeated variants of the closest JD requirement.

# Locked origin
origin_heading: %s
origin_type: %s

# Output JSON schema
{"drafts":[{"origin_heading":"","origin_type":"","value_theme":"","requirement_id":0,"requirement_ids":[0],"claim_ids":[0],"fact_ids":[0],"draft_text":"","rationale":"","risk_flags":[]}]}

# Draft budget
Return at most %d drafts.

# Section-scope rules
- Every draft must use origin_heading="%s" and origin_type="%s".
- Every claim_id must exist in one <evidence_packets_json> packet's claims.
- Every fact_id must exist in the same <evidence_packets_json> packet's facts.
- Do not use any claim or fact from another origin.
- Do not combine this origin with other roles, employers, projects, education, or skills-only claims.
- Cross-origin evidence may influence resume-level positioning later, but not this origin's bullet text.
- JD language may influence wording only; it must not add unsupported tools, metrics, ownership, scale, domain, production scope, or responsibilities.
- If the JD asks for a tool, domain, production scope, or responsibility not supported by this origin, omit it or phrase the match conservatively.

# Resume-value rules
%s

# Requirement usage
- requirement_id should identify the main JD requirement this bullet best supports.
- requirement_ids may include secondary requirements.
- value_theme must match one packet's value_theme.
- Prefer one bullet per evidence packet; skip packets that do not produce a strong human resume insight.
- A bullet does not need to perfectly satisfy a high-priority requirement if it is still one of the strongest same-origin resume bullets.
- Do not create one bullet per requirement. Create only the strongest bullets for this origin.
- Prefer story diversity over exact JD mirroring: one product/platform bullet, one security/reliability/quality bullet, one automation/frontend/architecture bullet when evidence supports those lanes.

# Bullet style
- Start each draft_text with a strong past-tense action verb, no leading hyphen.
- Keep each bullet 22 to 34 words.
- Use concrete artifacts, technologies, scope, and outcomes only when present in the facts.
- Treat packet atoms as atomic building blocks. Combine 2 to 4 same-origin atoms only when they form one coherent value insight.
- Use JD keywords only as prioritization and wording cues; the bullet must be built from candidate claim/fact atoms.
- Each bullet should make clear what real value was created: reliability, traceability, validation, delivery breadth, reduced manual work, safer access, faster debugging, or another supported outcome.
- Do not stuff unrelated facts together. If atoms do not naturally support one insight, write a narrower bullet.
- Do not repeat the same primary technology, artifact, or topic across drafts. If one draft mentions FastAPI/PostgreSQL/backend APIs, other drafts should focus on different evidenced value such as auditability, access control, maintainability, AI workflow, frontend work, or architecture.
- Write like a practical engineer, not a marketing page.
- Avoid inflated resume cliches.
- Avoid generic tool-list bullets.
- Avoid vague phrases like "worked on", "contributed to", "helped with", "various", "business outcomes", or "leveraged".
- Mention the origin in the rationale so reviewers know where the bullet belongs.
- Respect a one-page resume budget: max 5 bullets per experience origin, max 2 bullets per project origin, max 1 per education/certification origin.

# Evidence rules
- Every claim_id must exist in <evidence_packets_json>.
- Every fact_id must exist in <evidence_packets_json> and support the listed claim_id.
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
<evidence_packets_json>
%s
</evidence_packets_json>`,
		group.OriginHeading,
		group.OriginType,
		maxDrafts,
		group.OriginHeading,
		group.OriginType,
		resumeValueRules,
		firstNonEmpty(styleRules, "Use plain, specific, evidence-backed resume language."),
		job.Company,
		job.Title,
		string(reqJSON),
		string(matchJSON),
		string(packetJSON),
	)

	text, err := s.GenerateLLMText(ctx, client, system, user, 1200)
	if err != nil {
		return nil, err
	}

	parsed, err := parseBulletDrafts(text)
	if err != nil {
		return nil, err
	}
	if maxDrafts > 0 && len(parsed) > maxDrafts {
		parsed = parsed[:maxDrafts]
	}
	return parsed, nil
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
			claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume,
			resume_value_score, jd_relevance_score, origin_weight, risk_penalty, unsupported_context_penalty, selection_reason,
			value_theme, display_order
		FROM tailored_bullet_drafts WHERE job_id = ?
		ORDER BY selected_for_resume DESC, display_order ASC, selection_score DESC, CASE status WHEN 'needs_review' THEN 0 WHEN 'accepted' THEN 1 ELSE 2 END, id DESC`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTailoredBulletDrafts(rows)
}

func (s *Store) ListBulletGenerationEvents(jobID int64) ([]BulletGenerationEvent, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, origin_heading, stage, status, reason, draft_text, created_at
		FROM bullet_generation_events WHERE job_id = ?
		ORDER BY id DESC LIMIT 200`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBulletGenerationEvents(rows)
}

func (s *Store) clearBulletGenerationEvents(jobID int64) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM bullet_generation_events WHERE job_id = ?`, jobID)
	return err
}

func (s *Store) recordBulletGenerationEvent(jobID int64, originHeading, stage, status, reason, draftText string) error {
	return s.recordBulletGenerationEventWith(s.db, jobID, originHeading, stage, status, reason, draftText)
}

func (s *Store) recordBulletGenerationEventTx(tx *sql.Tx, jobID int64, originHeading, stage, status, reason, draftText string) error {
	return s.recordBulletGenerationEventWith(tx, jobID, originHeading, stage, status, reason, draftText)
}

func (s *Store) recordBulletGenerationEventWith(exec sqlExecutor, jobID int64, originHeading, stage, status, reason, draftText string) error {
	if jobID <= 0 {
		return nil
	}
	_, err := exec.ExecContext(
		context.Background(),
		`INSERT INTO bullet_generation_events (job_id, origin_heading, stage, status, reason, draft_text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		jobID,
		strings.TrimSpace(originHeading),
		strings.TrimSpace(stage),
		strings.TrimSpace(status),
		strings.TrimSpace(reason),
		strings.TrimSpace(draftText),
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
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
	originThemeCounts := map[string]map[string]bool{}
	originHighReqCounts := map[string]map[int64]bool{}
	selectedDrafts := []TailoredBulletDraft{}
	candidates := []TailoredBulletDraft{}
	for _, draft := range drafts {
		if draft.Status == "rejected" || draftHasUnsupportedRisk(draft.RiskFlags) {
			continue
		}

		if strings.TrimSpace(draft.OriginHeading) == "" || strings.TrimSpace(draft.OriginType) == "" {
			continue
		}

		candidates = append(candidates, draft)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].SelectionScore > candidates[j].SelectionScore
	})

	// First pass: guarantee at least one good bullet from each major experience origin.
	coveredExperienceOrigins := map[string]bool{}
	for _, draft := range candidates {
		if normalizeOriginPart(draft.OriginType) != "experience" {
			continue
		}
		originKey := originKey(draft.OriginHeading, draft.OriginType)
		if coveredExperienceOrigins[originKey] {
			continue
		}
		if !autoSelectCanAdd(draft, originCounts) {
			continue
		}
		if s.autoSelectDuplicate(jobID, draft, selectedDrafts) {
			continue
		}
		selectedIDs[draft.ID] = true
		originCounts[originKey]++
		trackSelectionDiversity(draft, originThemeCounts, originHighReqCounts)
		selectedDrafts = append(selectedDrafts, draft)
		coveredExperienceOrigins[originKey] = true
	}

	// Second pass: fill by score, subject to origin budgets and project penalties.
	for _, draft := range candidates {
		if selectedIDs[draft.ID] {
			continue
		}
		if !autoSelectCanAdd(draft, originCounts) {
			continue
		}
		originKey := originKey(draft.OriginHeading, draft.OriginType)
		if !autoSelectCanAddDiverse(draft, originCounts, originThemeCounts, originHighReqCounts) {
			continue
		}
		if s.autoSelectDuplicate(jobID, draft, selectedDrafts) {
			continue
		}
		selectedIDs[draft.ID] = true
		originCounts[originKey]++
		trackSelectionDiversity(draft, originThemeCounts, originHighReqCounts)
		selectedDrafts = append(selectedDrafts, draft)
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
		displayOrder := selectedDisplayOrder(id, selectedDrafts)
		if _, err := tx.ExecContext(context.Background(), `UPDATE tailored_bullet_drafts SET selected_for_resume = 1, display_order = ?, updated_at = ? WHERE id = ?`, displayOrder, now, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	for _, draft := range candidates {
		if selectedIDs[draft.ID] {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "selected", "ok", draft.SelectionReason, draft.DraftText)
		}
	}
	return s.ListTailoredBulletDrafts(jobID)
}

func autoSelectCanAdd(draft TailoredBulletDraft, originCounts map[string]int) bool {
	key := originKey(draft.OriginHeading, draft.OriginType)
	budget := bulletBudgetForSectionType(draft.OriginType)
	if budget <= 0 {
		budget = 2
	}
	return originCounts[key] < budget
}

func autoSelectCanAddDiverse(draft TailoredBulletDraft, originCounts map[string]int, originThemeCounts map[string]map[string]bool, originHighReqCounts map[string]map[int64]bool) bool {
	key := originKey(draft.OriginHeading, draft.OriginType)
	count := originCounts[key]
	typ := normalizeOriginPart(draft.OriginType)
	if typ == "experience" && count < 3 {
		return true
	}
	if typ == "project" && count < 2 {
		return true
	}
	theme := normalizeValueTheme(draft.ValueTheme)
	if theme == "" || originThemeCounts[key][theme] {
		return false
	}
	if draft.JDRelevanceScore < 0.12 {
		return false
	}
	if originHighReqCounts[key][draft.RequirementID] {
		return false
	}
	return true
}

func trackSelectionDiversity(draft TailoredBulletDraft, originThemeCounts map[string]map[string]bool, originHighReqCounts map[string]map[int64]bool) {
	key := originKey(draft.OriginHeading, draft.OriginType)
	if originThemeCounts[key] == nil {
		originThemeCounts[key] = map[string]bool{}
	}
	if originHighReqCounts[key] == nil {
		originHighReqCounts[key] = map[int64]bool{}
	}
	originThemeCounts[key][normalizeValueTheme(draft.ValueTheme)] = true
	if draft.JDRelevanceScore >= 0.12 {
		originHighReqCounts[key][draft.RequirementID] = true
	}
}

func selectedDisplayOrder(id int64, drafts []TailoredBulletDraft) int {
	for _, draft := range drafts {
		if draft.ID == id {
			return valueThemeOrder(draft.ValueTheme)
		}
	}
	return 999
}

func (s *Store) autoSelectDuplicate(jobID int64, draft TailoredBulletDraft, selected []TailoredBulletDraft) bool {
	for _, existing := range selected {
		if !sameOrigin(draft.OriginHeading, draft.OriginType, existing.OriginHeading, existing.OriginType) {
			continue
		}
		local := jaccardScore(similarityTokens(draft.DraftText), similarityTokens(existing.DraftText))
		if normalizeValueTheme(draft.ValueTheme) == normalizeValueTheme(existing.ValueTheme) && local >= 0.55 {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "selection_skipped", "duplicate", fmt.Sprintf("same theme similarity %.0f%%", local*100), draft.DraftText)
			return true
		}
		if storyFamiliesTooSimilar(bulletStoryFamilies(draft.DraftText), bulletStoryFamilies(existing.DraftText)) && local >= 0.34 {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "selection_skipped", "duplicate", "same story family", draft.DraftText)
			return true
		}
		if local >= 0.68 {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "selection_skipped", "duplicate", fmt.Sprintf("local similarity %.0f%%", local*100), draft.DraftText)
			return true
		}
		similarity, err := s.draftEmbeddingSimilarity(context.Background(), draft, existing)
		if err != nil {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "embedding_fallback", "warning", err.Error(), draft.DraftText)
			continue
		}
		if similarity >= 0.88 {
			_ = s.recordBulletGenerationEvent(jobID, draft.OriginHeading, "selection_skipped", "duplicate", fmt.Sprintf("embedding similarity %.0f%%", similarity*100), draft.DraftText)
			return true
		}
	}
	return false
}

func (s *Store) draftEmbeddingSimilarity(ctx context.Context, left TailoredBulletDraft, right TailoredBulletDraft) (float64, error) {
	leftText := strings.Join([]string{left.ValueTheme, left.DraftText}, " ")
	rightText := strings.Join([]string{right.ValueTheme, right.DraftText}, " ")
	leftVector, err := s.embeddingForEntity(ctx, nil, "tailored_bullet_draft", left.ID, leftText)
	if err != nil {
		return 0, err
	}
	rightVector, err := s.embeddingForEntity(ctx, nil, "tailored_bullet_draft", right.ID, rightText)
	if err != nil {
		return 0, err
	}
	return cosineSimilarity(leftVector, rightVector), nil
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
	reqsByID := map[int64]JobRequirement{}
	for _, req := range requirements {
		reqIDs[req.ID] = true
		reqsByID[req.ID] = req
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
	sanitizedMatches := []parsedJobMatch{}
	count := 0
	for _, match := range matches {
		if strings.EqualFold(match.CoverageStatus, "gap") || match.Score <= 0 {
			continue
		}
		if !reqIDs[match.RequirementID] || !factIDs[match.FactID] {
			continue
		}
		req := reqsByID[match.RequirementID]
		fact := factsByID[match.FactID]
		match = sanitizeParsedJobMatch(match, req, fact)
		if strings.EqualFold(match.CoverageStatus, "gap") || match.Score <= 0 {
			continue
		}
		sanitizedMatches = append(sanitizedMatches, match)
	}
	sanitizedMatches = selectStoredJobMatches(sanitizedMatches, 5)
	for _, match := range sanitizedMatches {
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

func selectStoredJobMatches(matches []parsedJobMatch, maxPerRequirement int) []parsedJobMatch {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].RequirementID != matches[j].RequirementID {
			return matches[i].RequirementID < matches[j].RequirementID
		}
		if coverageRank(matches[i].CoverageStatus) != coverageRank(matches[j].CoverageStatus) {
			return coverageRank(matches[i].CoverageStatus) > coverageRank(matches[j].CoverageStatus)
		}
		return matches[i].Score > matches[j].Score
	})
	selected := []parsedJobMatch{}
	counts := map[int64]int{}
	weakCounts := map[int64]int{}
	seen := map[string]bool{}
	for _, match := range matches {
		key := fmt.Sprintf("%d|%d", match.RequirementID, match.FactID)
		if seen[key] {
			continue
		}
		if counts[match.RequirementID] >= maxPerRequirement {
			continue
		}
		if strings.EqualFold(match.CoverageStatus, "weak") {
			if weakCounts[match.RequirementID] >= 2 {
				continue
			}
			weakCounts[match.RequirementID]++
		}
		seen[key] = true
		counts[match.RequirementID]++
		selected = append(selected, match)
	}
	return selected
}

func coverageRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "strong":
		return 3
	case "partial":
		return 2
	case "weak":
		return 1
	default:
		return 0
	}
}

func sanitizeParsedJobMatch(match parsedJobMatch, req JobRequirement, fact factPromptContext) parsedJobMatch {
	evidenceText := strings.ToLower(strings.Join([]string{
		fact.FactText,
		fact.EvidenceQuote,
		strings.Join(fact.Technologies, " "),
		fact.SectionHeading,
		strings.Join(fact.Context, " "),
	}, " "))
	if strings.TrimSpace(evidenceText) == "" {
		return match
	}
	overlap := []string{}
	for _, term := range jobMatchTerms(req.RequirementText, req.Keywords) {
		if strings.Contains(evidenceText, term) {
			overlap = append(overlap, term)
		}
	}
	overlap = normalizeStringList(overlap)
	if !jobMatchOverlapAllowed(req, overlap, evidenceText) {
		if transferableScore, theme := transferableJobMatchScore(req, evidenceText); transferableScore > 0 {
			if match.Score > transferableScore || match.Score <= 0 {
				match.Score = transferableScore
			}
			match.CoverageStatus = matchCoverageForScore(match.Score)
			match.Rationale = "Transferable evidence: " + theme
			return match
		}
		adjacentScore := adjacentJobMatchScore(req, overlap, evidenceText)
		if adjacentScore <= 0 {
			match.Score = 0
			match.CoverageStatus = "gap"
			match.Rationale = "Rejected generic or missing required technical overlap."
			return match
		}
		if match.Score > adjacentScore {
			match.Score = adjacentScore
		}
		match.CoverageStatus = "weak"
		match.Rationale = "Adjacent evidence only: " + strings.Join(meaningfulJobOverlapTerms(overlap), ", ")
		return match
	}
	required := requirementRequiredMatchTerms(req)
	if len(required) > 0 {
		sanitizedScore := jobMatchScore(req, overlap, evidenceText, 0.32)
		if match.Score > sanitizedScore {
			match.Score = sanitizedScore
		}
	} else {
		match.Score = clampScore(match.Score)
	}
	switch {
	case match.Score >= 0.75 && len(meaningfulJobOverlapTerms(overlap)) >= 1:
		match.CoverageStatus = "strong"
	case match.Score >= 0.45:
		match.CoverageStatus = "partial"
	default:
		match.CoverageStatus = "weak"
	}
	if strings.TrimSpace(match.Rationale) == "" || strings.Contains(strings.ToLower(match.Rationale), "data, time") {
		match.Rationale = "Evidence overlap: " + strings.Join(meaningfulJobOverlapTerms(overlap), ", ")
	}
	return match
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
	insertedDraftRefs := []draftSemanticRef{}

	for _, draft := range drafts {
		draft.DraftText = strings.TrimSpace(draft.DraftText)

		if draft.RequirementID == 0 && len(draft.RequirementIDs) > 0 {
			draft.RequirementID = draft.RequirementIDs[0]
		}

		originHeading := strings.TrimSpace(draft.OriginHeading)
		originType := strings.TrimSpace(draft.OriginType)
		if originHeading == "" || originType == "" {
			originHeading, originType = inferDraftOrigin(draft, factsByID, claimsByID)
			draft.OriginHeading = originHeading
			draft.OriginType = originType
		}
		draft.ValueTheme = normalizeValueTheme(draft.ValueTheme)

		if draft.DraftText == "" || !reqIDs[draft.RequirementID] || originHeading == "" || originType == "" {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "missing text, requirement, or origin", draft.DraftText)
			_ = s.logEventTx(tx, "warning", "bullet draft rejected: missing text, requirement, or origin")
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
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "no valid same-origin claims", draft.DraftText)
			_ = s.logEventTx(tx, "warning", "bullet draft rejected: no valid same-origin claims for "+originHeading+" text="+draft.DraftText)
			continue
		}
		if draft.ValueTheme == "" {
			draft.ValueTheme = inferDraftValueTheme(validClaimIDs, claimsByID)
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

			if strings.TrimSpace(fact.SectionHeading) != "" && strings.TrimSpace(fact.SectionType) != "" &&
				!sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
				continue
			}

			validFactIDs = append(validFactIDs, factID)
		}
		validFactIDs = uniqueInt64s(validFactIDs)

		if len(validFactIDs) == 0 {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "no valid same-origin supporting facts", draft.DraftText)
			_ = s.logEventTx(tx, "warning", "bullet draft rejected: no valid same-origin supporting facts for "+originHeading+" text="+draft.DraftText)
			continue
		}
		originKey, originBudget := draftOriginBudgetFromOrigin(originHeading, originType)
		if originBudget > 0 && originCounts[originKey] >= originBudget {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "origin budget reached", draft.DraftText)
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
			filterLLMRiskFlags(draft.RiskFlags),
			styleRiskFlags(draft.DraftText)...,
		))
		draft.RiskFlags = normalizeStringList(append(draft.RiskFlags, semanticBulletRiskFlags(draft, validClaimIDs, claimsByID, validFactIDs, factsByID)...))

		draft.RiskFlags = normalizeStringList(append(
			draft.RiskFlags,
			sectionScopeRiskFlags(draft, validClaimIDs, claimsByID, validFactIDs, factsByID)...,
		))

		if draftHasUnsupportedRisk(draft.RiskFlags) {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "unsupported risk: "+strings.Join(draft.RiskFlags, ","), draft.DraftText)
			_ = s.logEventTx(tx, "warning", "bullet draft rejected: unsupported risk for "+originHeading+" flags="+strings.Join(draft.RiskFlags, ",")+" text="+draft.DraftText)
			continue
		}

		riskJSON, err := encodeStringList(draft.RiskFlags)
		if err != nil {
			return nil, err
		}

		if lowResumeValueDraft(draft, validClaimIDs, claimsByID) {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "low resume value", draft.DraftText)
			_ = s.logEventTx(tx, "warning", "bullet draft rejected: low resume value for "+originHeading+" text="+draft.DraftText)
			continue
		}
		if duplicateAcceptedDraft(draft, validClaimIDs, validFactIDs, insertedDraftRefs) {
			_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "validation_rejected", "rejected", "semantic duplicate", draft.DraftText)
			continue
		}

		score := draftSelectionScoreDetail(draft, reqsByID[draft.RequirementID], validClaimIDs, claimsByID)
		displayOrder := valueThemeOrder(draft.ValueTheme)

		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO tailored_bullet_drafts
				(job_id, requirement_id, fact_ids_json, draft_text, rationale, status, risk_flags_json, created_at, updated_at,
				claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume,
				resume_value_score, jd_relevance_score, origin_weight, risk_penalty, unsupported_context_penalty, selection_reason,
				value_theme, display_order)
			VALUES (?, ?, ?, ?, ?, 'needs_review', ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
			score.Final,
			score.ResumeValue,
			score.JDRelevance,
			score.OriginWeight,
			score.RiskPenalty,
			score.UnsupportedPenalty,
			score.Reason,
			draft.ValueTheme,
			displayOrder,
		); err != nil {
			return nil, err
		}

		_ = s.recordBulletGenerationEventTx(tx, jobID, originHeading, "inserted", "ok", score.Reason, draft.DraftText)
		insertedDraftRefs = append(insertedDraftRefs, draftRefFromParsed(draft, validClaimIDs, validFactIDs))
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

	return text == ""
}

type draftSemanticRef struct {
	OriginHeading string
	OriginType    string
	ValueTheme    string
	StoryFamilies []string
	Text          string
	ClaimIDs      []int64
	FactIDs       []int64
}

func draftRefFromParsed(draft parsedBulletDraft, claimIDs []int64, factIDs []int64) draftSemanticRef {
	return draftSemanticRef{
		OriginHeading: draft.OriginHeading,
		OriginType:    draft.OriginType,
		ValueTheme:    normalizeValueTheme(draft.ValueTheme),
		StoryFamilies: bulletStoryFamilies(draft.DraftText),
		Text:          draft.DraftText,
		ClaimIDs:      uniqueInt64s(claimIDs),
		FactIDs:       uniqueInt64s(factIDs),
	}
}

func duplicateAcceptedDraft(draft parsedBulletDraft, claimIDs []int64, factIDs []int64, existing []draftSemanticRef) bool {
	ref := draftRefFromParsed(draft, claimIDs, factIDs)
	refTokens := similarityTokens(strings.Join([]string{ref.ValueTheme, ref.Text}, " "))
	for _, other := range existing {
		if !sameOrigin(ref.OriginHeading, ref.OriginType, other.OriginHeading, other.OriginType) {
			continue
		}
		overlap := jaccardScore(refTokens, similarityTokens(strings.Join([]string{other.ValueTheme, other.Text}, " ")))
		if normalizeValueTheme(ref.ValueTheme) == normalizeValueTheme(other.ValueTheme) {
			if overlap >= 0.62 && (int64OverlapCount(ref.ClaimIDs, other.ClaimIDs) > 0 || int64OverlapCount(ref.FactIDs, other.FactIDs) > 0) {
				return true
			}
		}
		if storyFamiliesTooSimilar(ref.StoryFamilies, other.StoryFamilies) && overlap >= 0.38 {
			return true
		}
		if overlap >= 0.72 && (int64OverlapCount(ref.ClaimIDs, other.ClaimIDs) > 0 || int64OverlapCount(ref.FactIDs, other.FactIDs) > 0) {
			return true
		}
	}
	return false
}

func bulletStoryFamilies(text string) []string {
	lower := strings.ToLower(text)
	families := []string{}
	if strings.Contains(lower, "fastapi") || strings.Contains(lower, "postgresql") || strings.Contains(lower, "backend") || strings.Contains(lower, "api") || strings.Contains(lower, "apis") {
		families = append(families, "backend_api")
	}
	if strings.Contains(lower, "booking") || strings.Contains(lower, "scheduling") || strings.Contains(lower, "planning") || strings.Contains(lower, "programme") || strings.Contains(lower, "upload") {
		families = append(families, "workflow_delivery")
	}
	if strings.Contains(lower, "rbac") || strings.Contains(lower, "access control") || strings.Contains(lower, "permission") {
		families = append(families, "access_control")
	}
	if strings.Contains(lower, "audit") || strings.Contains(lower, "traceability") || strings.Contains(lower, "data changes") || strings.Contains(lower, "integrity") {
		families = append(families, "audit_traceability")
	}
	if strings.Contains(lower, "react") || strings.Contains(lower, "typescript") || strings.Contains(lower, "frontend") || strings.Contains(lower, "ui") {
		families = append(families, "frontend_ui")
	}
	if strings.Contains(lower, "ai") || strings.Contains(lower, "llm") || strings.Contains(lower, "extraction") || strings.Contains(lower, "token") {
		families = append(families, "ai_workflows")
	}
	if strings.Contains(lower, "architecture") || strings.Contains(lower, "data model") || strings.Contains(lower, "service design") || strings.Contains(lower, "maintainable") {
		families = append(families, "architecture_quality")
	}
	return normalizeStringList(families)
}

func storyFamiliesTooSimilar(left []string, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	overlap := intStringOverlapCount(left, right)
	if overlap == 0 {
		return false
	}
	if overlap >= minInt(len(left), len(right)) {
		return true
	}
	if containsString(left, "backend_api") && containsString(right, "backend_api") &&
		(containsString(left, "workflow_delivery") == containsString(right, "workflow_delivery") ||
			containsString(left, "access_control") == containsString(right, "access_control") ||
			containsString(left, "audit_traceability") == containsString(right, "audit_traceability")) {
		return true
	}
	return false
}

func semanticBulletRiskFlags(draft parsedBulletDraft, claimIDs []int64, claimsByID map[int64]CandidateClaim, factIDs []int64, factsByID map[int64]factPromptContext) []string {
	flags := []string{}
	if normalizeValueTheme(draft.ValueTheme) == "" {
		flags = append(flags, "missing_value_theme")
	}
	if nonHumanResumeLanguage(draft.DraftText) {
		flags = append(flags, "style_non_human")
	}
	if unsupportedTermsInBullet(draft.DraftText, claimIDs, claimsByID, factIDs, factsByID) {
		flags = append(flags, "unsupported_term")
	}
	return normalizeStringList(flags)
}

func normalizeValueTheme(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func inferDraftValueTheme(claimIDs []int64, claimsByID map[int64]CandidateClaim) string {
	counts := map[string]int{}
	for _, claimID := range claimIDs {
		claim, ok := claimsByID[claimID]
		if !ok {
			continue
		}
		counts[inferClaimValueTheme(claim)]++
	}
	best := ""
	bestCount := 0
	for theme, count := range counts {
		if count > bestCount || (count == bestCount && valueThemeOrder(theme) < valueThemeOrder(best)) {
			best = theme
			bestCount = count
		}
	}
	return best
}

func nonHumanResumeLanguage(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range []string{
		"leveraged", "utilized", "seamless", "cutting-edge", "dynamic", "business outcomes", "synergy",
		"game-changer", "transformative", "unlock", "empowered", "innovative solutions", "various",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func unsupportedTermsInBullet(text string, claimIDs []int64, claimsByID map[int64]CandidateClaim, factIDs []int64, factsByID map[int64]factPromptContext) bool {
	lower := strings.ToLower(text)
	support := strings.ToLower(supportedDraftVocabulary(claimIDs, claimsByID, factIDs, factsByID))
	for _, term := range []string{"aws", "serverless", "container", "containers", "kubernetes", "health-tech", "healthcare", "medical", "regulated", "compliance", "enterprise", "scalable"} {
		if strings.Contains(lower, term) && !strings.Contains(support, term) {
			return true
		}
	}
	return false
}

func supportedDraftVocabulary(claimIDs []int64, claimsByID map[int64]CandidateClaim, factIDs []int64, factsByID map[int64]factPromptContext) string {
	parts := []string{}
	for _, claimID := range claimIDs {
		claim, ok := claimsByID[claimID]
		if !ok {
			continue
		}
		parts = append(parts,
			claim.ClaimText,
			strings.Join(claim.Actions, " "),
			strings.Join(claim.Capabilities, " "),
			strings.Join(claim.Objects, " "),
			strings.Join(claim.Technologies, " "),
			strings.Join(claim.Domains, " "),
			strings.Join(claim.Artifacts, " "),
			strings.Join(claim.Scope, " "),
			strings.Join(claim.Metrics, " "),
			strings.Join(claim.Outcomes, " "),
		)
	}
	for _, factID := range factIDs {
		fact, ok := factsByID[factID]
		if !ok {
			continue
		}
		parts = append(parts, fact.FactText, fact.EvidenceQuote, strings.Join(fact.Technologies, " "), strings.Join(fact.Context, " "))
	}
	return strings.Join(parts, " ")
}

func int64OverlapCount(left []int64, right []int64) int {
	seen := map[int64]bool{}
	for _, value := range left {
		seen[value] = true
	}
	count := 0
	for _, value := range right {
		if seen[value] {
			count++
		}
	}
	return count
}

func intStringOverlapCount(left []string, right []string) int {
	seen := map[string]bool{}
	for _, value := range left {
		seen[value] = true
	}
	count := 0
	for _, value := range right {
		if seen[value] {
			count++
		}
	}
	return count
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
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
		req = normalizeParsedJobRequirement(req)
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

func normalizeParsedJobRequirement(req parsedJobRequirement) parsedJobRequirement {
	req.RequirementText = normalizeRequirementSentence(req.RequirementText)
	req.SourceQuote = strings.TrimSpace(req.SourceQuote)
	req.Keywords = extractJobKeywords(strings.Join([]string{req.RequirementText, strings.Join(req.Keywords, " ")}, " "))
	lower := strings.ToLower(strings.Join([]string{req.RequirementText, req.SourceQuote}, " "))
	if strings.Contains(lower, "advantageous but not mandatory") ||
		strings.Contains(lower, "not mandatory") ||
		strings.Contains(lower, "nice to have") ||
		strings.Contains(lower, "preferred") ||
		strings.Contains(lower, "bonus") {
		req.Category = "nice_to_have"
		req.Priority = "low"
	}
	if strings.Contains(lower, "mentor") || strings.Contains(lower, "mentoring") || strings.Contains(lower, "knowledge sharing") {
		req.Category = "seniority"
		if strings.TrimSpace(req.Priority) == "" || req.Priority == "high" {
			req.Priority = "medium"
		}
	}
	return req
}

func normalizeRequirementSentence(text string) string {
	text = cleanJobDetailLine(normalizePastedText(text))
	text = strings.TrimSuffix(text, ".")
	if before, after, ok := strings.Cut(text, ":"); ok {
		prefix := strings.ToLower(strings.TrimSpace(before))
		if len(strings.Fields(prefix)) <= 6 && requirementHeadingPrefix(prefix) {
			text = strings.TrimSpace(after)
		}
	}
	replacements := []struct {
		old string
		new string
	}{
		{"Dive into the world of real-time data processing using", "Real-time data processing using"},
		{"Harness the power of the cloud for", "Cloud-based"},
		{"Join the mission to enhance", "Enhance"},
		{"Be a key driver in defining", "Define"},
		{"Contributing to", "Contribute to"},
		{"Participating in", "Participate in"},
		{"Conducting", "Conduct"},
	}
	for _, replacement := range replacements {
		if strings.HasPrefix(text, replacement.old) {
			text = replacement.new + strings.TrimPrefix(text, replacement.old)
		}
	}
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func requirementHeadingPrefix(prefix string) bool {
	for _, marker := range []string{
		"develop high-quality software",
		"technical design and architecture",
		"collaboration and communication",
		"problem solving and troubleshooting",
		"code reviews and quality",
		"mentoring and knowledge sharing",
		"continuous learning and skill development",
		"adherence to best practices and standards",
		"live data stream processing",
		"cloud-based data processing",
		"iot capabilities for enhanced experiences",
		"cloud platform migration",
		"integrate with firmware and hardware",
		"scaling",
		"ai",
	} {
		if prefix == marker {
			return true
		}
	}
	return false
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

func inferDraftOrigin(draft parsedBulletDraft, factsByID map[int64]factPromptContext, claimsByID map[int64]CandidateClaim) (string, string) {
	for _, factID := range draft.FactIDs {
		fact, ok := factsByID[factID]
		if ok && strings.TrimSpace(fact.SectionHeading) != "" && strings.TrimSpace(fact.SectionType) != "" {
			return strings.TrimSpace(fact.SectionHeading), strings.TrimSpace(fact.SectionType)
		}
	}
	for _, claimID := range draft.ClaimIDs {
		claim, ok := claimsByID[claimID]
		if ok && strings.TrimSpace(claim.OriginHeading) != "" && strings.TrimSpace(claim.OriginType) != "" {
			return strings.TrimSpace(claim.OriginHeading), strings.TrimSpace(claim.OriginType)
		}
	}
	return "", ""
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

	hasSourceFacts := false
	for _, factID := range claim.SourceFactIDs {
		fact, ok := factsByID[factID]
		if !ok {
			return false
		}
		hasSourceFacts = true

		if strings.TrimSpace(fact.SectionHeading) != "" && strings.TrimSpace(fact.SectionType) != "" &&
			!sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
			return false
		}
	}

	if hasSourceFacts {
		return true
	}

	claimOriginHeading := strings.TrimSpace(claim.OriginHeading)
	claimOriginType := strings.TrimSpace(claim.OriginType)
	if claimOriginHeading == "" && claimOriginType == "" {
		return true
	}
	return sameOrigin(claimOriginHeading, claimOriginType, originHeading, originType)
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
		hasSourceFacts := false
		sourceFactsSameOrigin := true
		for _, factID := range claim.SourceFactIDs {
			fact, ok := factsByID[factID]
			if !ok {
				continue
			}
			hasSourceFacts = true
			if strings.TrimSpace(fact.SectionHeading) != "" && strings.TrimSpace(fact.SectionType) != "" &&
				!sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
				sourceFactsSameOrigin = false
			}
		}
		if hasSourceFacts {
			if !sourceFactsSameOrigin {
				flags = append(flags, "cross_origin_claim")
			}
			continue
		}

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

		if strings.TrimSpace(fact.SectionHeading) != "" && strings.TrimSpace(fact.SectionType) != "" &&
			!sameOrigin(fact.SectionHeading, fact.SectionType, originHeading, originType) {
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
			"project_claim_used_as_experience",
			"missing_value_theme",
			"style_non_human",
			"style_buzzword",
			"style_filler",
			"style_formatting",
			"unsupported_term":
			return true
		}
	}

	return false
}

type draftScoreDetail struct {
	Final              float64
	ResumeValue        float64
	JDRelevance        float64
	OriginWeight       float64
	RiskPenalty        float64
	UnsupportedPenalty float64
	Reason             string
}

func draftSelectionScore(draft parsedBulletDraft, req JobRequirement, claimIDs []int64, claimsByID map[int64]CandidateClaim) float64 {
	return draftSelectionScoreDetail(draft, req, claimIDs, claimsByID).Final
}

func draftSelectionScoreDetail(draft parsedBulletDraft, req JobRequirement, claimIDs []int64, claimsByID map[int64]CandidateClaim) draftScoreDetail {
	detail := draftScoreDetail{
		Final:        0.25,
		OriginWeight: originResumeWeight(draft.OriginType, draft.OriginHeading),
	}

	// 1. Resume value is primary.
	valueTotal := 0.0
	for _, claimID := range claimIDs {
		claim := claimsByID[claimID]
		valueTotal += claimResumeValueScore(claim)
	}
	if len(claimIDs) > 0 {
		detail.ResumeValue = valueTotal / float64(len(claimIDs))
		detail.Final += detail.ResumeValue * 0.45
	}

	// 2. JD relevance is secondary.
	switch req.Priority {
	case "high":
		detail.JDRelevance = 0.12
	case "medium":
		detail.JDRelevance = 0.07
	case "low":
		detail.JDRelevance = 0.02
	}
	if req.Category == "must_have" || req.Category == "responsibility" {
		detail.JDRelevance += 0.03
	}
	detail.Final += detail.JDRelevance

	// 3. Prefer real work experience over projects when the JD asks for production.
	reqText := strings.ToLower(req.RequirementText)
	if draft.OriginType == "experience" {
		detail.Final += 0.08
	}

	if draft.OriginType == "project" && strings.Contains(reqText, "production") {
		detail.UnsupportedPenalty += 0.12
	}

	// 4. Penalize weak or inferred tailoring.
	for _, flag := range draft.RiskFlags {
		switch flag {
		case "weak_evidence":
			detail.RiskPenalty += 0.10
		case "inferred_tailoring":
			detail.RiskPenalty += 0.08
		case "ambiguous technology", "ambiguous_technology":
			detail.RiskPenalty += 0.06
		default:
			detail.RiskPenalty += 0.03
		}
	}

	if looksLikeToolListBullet(draft.DraftText) {
		detail.RiskPenalty += 0.12
	}
	if normalizeValueTheme(draft.ValueTheme) != "" {
		detail.Final += 0.04
	}

	detail.Final -= detail.RiskPenalty + detail.UnsupportedPenalty
	detail.Final *= detail.OriginWeight
	detail.Final = clampScore(detail.Final)
	detail.Reason = draftSelectionReason(draft, detail)
	return detail
}

func originResumeWeight(originType, originHeading string) float64 {
	typ := strings.ToLower(strings.TrimSpace(originType))
	heading := strings.ToLower(strings.TrimSpace(originHeading))

	switch typ {
	case "experience":
		if strings.Contains(heading, "sitespace") {
			return 1.18
		}
		if strings.Contains(heading, "tata") || strings.Contains(heading, "tcs") {
			return 1.08
		}
		return 1.0
	case "project":
		return 0.82
	default:
		return 0.65
	}
}

func draftSelectionReason(draft parsedBulletDraft, detail draftScoreDetail) string {
	parts := []string{
		fmt.Sprintf("resume %.0f%%", detail.ResumeValue*100),
		fmt.Sprintf("JD %.0f%%", detail.JDRelevance*100),
		fmt.Sprintf("origin x%.2f", detail.OriginWeight),
	}
	if detail.RiskPenalty > 0 {
		parts = append(parts, fmt.Sprintf("risk -%.0f%%", detail.RiskPenalty*100))
	}
	if detail.UnsupportedPenalty > 0 {
		parts = append(parts, fmt.Sprintf("unsupported -%.0f%%", detail.UnsupportedPenalty*100))
	}
	if draft.OriginType == "project" && detail.OriginWeight < 1 {
		parts = append(parts, "project evidence capped below experience evidence")
	}
	return strings.Join(parts, "; ")
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
		return 3
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
		if strings.TrimSpace(draft.DraftText) == "" ||
			len(draft.ClaimIDs) == 0 ||
			len(draft.FactIDs) == 0 {
			return nil, errors.New("every draft must include draft_text, claim_ids, and fact_ids")
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
	if looksLikeEmployerDescription(requirementText) || looksLikeEmployerDescription(sourceQuote) {
		return true
	}
	if isPureSoftSkillRequirement(requirementText) {
		return true
	}
	irrelevantMarkers := []string{
		"12-month", "12 month", "contract", "max term", "fixed term", "salary", "compensation", "benefit", "leave", "hybrid", "remote", "location", "office",
		"how to apply", "application process", "submit your application", "recruit", "hiring", "interview", "equal opportunity", "diversity", "background check", "sponsorship",
		"leading personal injury", "class actions law firm", "about us", "about the company", "company is", "we are a", "we're a", "our client", "we believe", "we are looking for", "based in", "we want people", "right now, your expertise", "in 6 months", "in 12 months", "what your success will look like", "why catapult", "deserves to feel", "redefining", "platform provides", "dedicated team", "immediate care", "critical situations",
		"favourite sports team", "favorite sports team", "team / department / company", "growth and development of the team", "one of those rare roles", "as close to the edge as you can get",
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

func looksLikeEmployerDescription(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return true
	}
	companyMarkers := []string{
		" is a ",
		" is building ",
		"we work with ",
		"we provide ",
		"we deliver ",
		"our solutions ",
		"with a mission ",
		"empowers professional teams",
		"every decision is an opportunity",
		"future of sports performance",
	}
	for _, marker := range companyMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if strings.HasPrefix(lower, "catapult ") && !hasCandidateRequirementCue(lower) {
		return true
	}
	return false
}

func hasCandidateRequirementCue(lower string) bool {
	for _, cue := range []string{"experience", "proficiency", "understanding", "track record", "build", "design", "develop", "own", "test", "mentor", "migrate", "integrate", "troubleshoot"} {
		if strings.Contains(lower, cue) {
			return true
		}
	}
	return false
}

func isPureSoftSkillRequirement(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return true
	}
	if strings.Contains(lower, "collaborative team player") ||
		strings.Contains(lower, "verbal and written communication") ||
		strings.Contains(lower, "customer and product mindset") ||
		strings.Contains(lower, "understanding of user needs") ||
		strings.Contains(lower, "multiple nationalities") ||
		strings.Contains(lower, "global awareness") ||
		strings.Contains(lower, "positive and curious mindset") ||
		strings.Contains(lower, "can-do attitude") ||
		strings.HasPrefix(lower, "a passion for educating") ||
		strings.HasPrefix(lower, "an appetite for using/learning") {
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
		"exposure", "regulated", "healthcare", "pharma", "medical devices",
		"track record", "owning", "own ", "full-stack", "full stack",
		"design ", "design and", "build ", "build and", "develop", "deliver", "modernis", "test", "testing", "support ",
		"architecture", "api", "apis", "data ingestion", "high-concurrency", "dashboard", "dashboards",
		"workflow", "workflows", "real-time", "low-latency", "stream processing", "rag", "retrieval", "chunking", "metadata filtering", "model context protocol", "evaluation framework",
		"cloud", "serverless", "event-driven", "distributed", "scalable", "resilience", "observability", "security", "firmware", "hardware",
		"networking", "identity", "database", "nosql", "data model", "data models", "schema",
		"devsecops", "agile", "solid", "containers", "messaging", "queues", "topics", "stakeholder", "mentor",
		"engineering practices", "technical excellence", "microservice", "domain-driven",
	} {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	for _, tech := range []string{"fastapi", "postgresql", "react", "next.js", "typescript", "javascript", "python", "golang", "go", "java", "spring", "mysql", "node.js", "node", "azure", "cosmos db", "aws", "gcp", "docker", "kubernetes", "terraform", "redis", "snowflake", "kafka", "kinesis", "iot", "iac", "rust", "c#", ".net", "c++", "langgraph", "crewai", "mcp", "langsmith", "arize", "phoenix", "cursor", "claudecode", "copilot"} {
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
	for _, marker := range []string{"experience with", "hands-on", "understanding of", "exposure to"} {
		if strings.Contains(lower, marker) {
			return false
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

func fallbackJobRequirements(raw string) []parsedJobRequirement {
	raw = normalizePastedText(raw)
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
		req = normalizeParsedJobRequirement(req)
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

func mergeParsedJobRequirements(primary []parsedJobRequirement, supplemental []parsedJobRequirement) []parsedJobRequirement {
	merged := append([]parsedJobRequirement{}, primary...)
	for _, candidate := range supplemental {
		if len(merged) >= 16 {
			break
		}
		if hasSimilarParsedRequirement(merged, candidate) {
			continue
		}
		merged = append(merged, candidate)
	}
	return merged
}

func hasSimilarParsedRequirement(existing []parsedJobRequirement, candidate parsedJobRequirement) bool {
	candidateText := normalizedRequirementText(candidate.RequirementText)
	candidateTerms := jobMatchTerms(candidate.RequirementText, candidate.Keywords)
	for _, current := range existing {
		currentText := normalizedRequirementText(current.RequirementText)
		if candidateText == currentText || strings.Contains(candidateText, currentText) || strings.Contains(currentText, candidateText) {
			return true
		}
		currentTerms := jobMatchTerms(current.RequirementText, current.Keywords)
		minTerms := minInt(len(candidateTerms), len(currentTerms))
		if minTerms >= 2 && requirementTermOverlap(candidateTerms, currentTerms) >= minInt(2, minTerms) {
			return true
		}
	}
	return false
}

func normalizedRequirementText(text string) string {
	text = strings.ToLower(cleanJobDetailLine(normalizePastedText(text)))
	return strings.Join(strings.Fields(text), " ")
}

func requirementTermOverlap(left []string, right []string) int {
	seen := map[string]bool{}
	for _, term := range left {
		seen[term] = true
	}
	overlap := 0
	for _, term := range right {
		if seen[term] {
			overlap++
		}
	}
	return overlap
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
	raw = normalizePastedText(raw)
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
	for _, word := range []string{"the", "and", "for", "with", "that", "this", "you", "your", "our", "are", "will", "have", "has", "from", "into", "work", "role", "team", "need", "needs", "required", "requirement", "requirements", "experience", "strong", "deep", "hands", "excellent", "ability", "using", "including", "specific", "technologies", "advantageous", "mandatory", "participating", "contributing", "conducting", "understanding", "develop", "build", "join", "dive", "harness"} {
		stop[word] = true
	}
	keywords := []string{}
	for _, tech := range []string{"TypeScript", "Node.js", "React", "React.js", "AWS", "Azure", "Cosmos DB", "NoSQL", "SQL", "Kafka", "Kinesis", "IoT", "IaC", "Go", "Rust", "C#", ".Net", "C++", "DDD", "domain-driven design", "microservice", "microservices", "DevOps", "DevSecOps", "Agile", "SOLID", "serverless", "event-driven", "containers", "networking", "databases", "messaging", "queues", "topics", "cloud", "observability", "security", "firmware", "hardware", "edge devices", "real-time", "low-latency", "stream processing", "healthcare", "pharma", "medical devices", "regulated industries", "resilience", "identity", "distributed", "scalable"} {
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
	line = strings.TrimPrefix(line, "\u2022")
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
			if fact.Status != factStatusApproved {
				continue
			}
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
			if !jobMatchOverlapAllowed(req, overlap, factText) {
				if score, theme := transferableJobMatchScore(req, factText); score > 0 {
					candidates = append(candidates, parsedJobMatch{
						RequirementID:  req.ID,
						FactID:         fact.ID,
						Score:          score,
						CoverageStatus: matchCoverageForScore(score),
						Rationale:      "Transferable evidence: " + theme,
					})
					continue
				}
				score := adjacentJobMatchScore(req, overlap, factText)
				if score <= 0 {
					continue
				}
				candidates = append(candidates, parsedJobMatch{
					RequirementID:  req.ID,
					FactID:         fact.ID,
					Score:          score,
					CoverageStatus: "weak",
					Rationale:      "Adjacent evidence only: " + strings.Join(meaningfulJobOverlapTerms(overlap), ", "),
				})
				continue
			}
			score := jobMatchScore(req, overlap, factText, 0.35)
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
			if !jobMatchOverlapAllowed(req, overlap, claimText) {
				if score, theme := transferableJobMatchScore(req, claimText); score > 0 {
					candidates = append(candidates, parsedJobMatch{
						RequirementID:  req.ID,
						FactID:         claim.SourceFactIDs[0],
						Score:          score,
						CoverageStatus: matchCoverageForScore(score),
						Rationale:      "Transferable atom-bank evidence: " + theme + " via " + claim.Label,
					})
					continue
				}
				score := adjacentJobMatchScore(req, overlap, claimText)
				if score <= 0 {
					continue
				}
				candidates = append(candidates, parsedJobMatch{
					RequirementID:  req.ID,
					FactID:         claim.SourceFactIDs[0],
					Score:          score,
					CoverageStatus: "weak",
					Rationale:      "Adjacent atom-bank overlap: " + strings.Join(meaningfulJobOverlapTerms(overlap), ", "),
				})
				continue
			}
			score := jobMatchScore(req, overlap, claimText, 0.42) + claimStrengthWeight(claim)
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

func mergeParsedJobMatches(primary []parsedJobMatch, fallback []parsedJobMatch) []parsedJobMatch {
	merged := append([]parsedJobMatch{}, primary...)
	indexByKey := map[string]int{}
	for index, match := range merged {
		key := fmt.Sprintf("%d|%d", match.RequirementID, match.FactID)
		indexByKey[key] = index
	}
	for _, match := range fallback {
		key := fmt.Sprintf("%d|%d", match.RequirementID, match.FactID)
		if existingIndex, ok := indexByKey[key]; ok {
			if match.Score > merged[existingIndex].Score {
				merged[existingIndex] = match
			}
			continue
		}
		indexByKey[key] = len(merged)
		merged = append(merged, match)
	}
	return merged
}

func jobMatchOverlapAllowed(req JobRequirement, overlap []string, evidenceText string) bool {
	if len(overlap) == 0 {
		return false
	}
	meaningful := meaningfulJobOverlapTerms(overlap)
	if len(meaningful) == 0 {
		return false
	}
	required := requirementRequiredMatchTerms(req)
	if len(required) == 0 {
		return len(meaningful) >= 1
	}
	evidence := strings.ToLower(evidenceText)
	requiredHits := 0
	for _, term := range required {
		if containsString(meaningful, term) || strings.Contains(evidence, term) {
			requiredHits++
		}
	}
	if len(required) >= 3 {
		if normalizeRequirementCategory(req.Category) == "nice_to_have" {
			return requiredHits >= 1
		}
		return requiredHits >= 2
	}
	return requiredHits >= 1
}

func transferableJobMatchScore(req JobRequirement, evidenceText string) (float64, string) {
	reqText := strings.ToLower(strings.Join([]string{req.RequirementText, strings.Join(req.Keywords, " "), req.Category}, " "))
	evidence := strings.ToLower(evidenceText)
	category := normalizeRequirementCategory(req.Category)

	if isStreamingRequirement(reqText) {
		switch {
		case containsAny(evidence, "kafka", "kinesis", "stream processing", "real-time", "low-latency"):
			return 0.72, "direct streaming or low-latency data-processing evidence"
		case containsAny(evidence, "sftp", "ftp", "ingestion", "pipeline", "processing flow", "data transfer"):
			return 0.50, "data ingestion/pipeline evidence, but not real-time Kafka/Kinesis"
		case containsAny(evidence, "aws", "docker", "github actions") && containsAny(evidence, "data", "database", "backend", "pipeline"):
			return 0.42, "cloud/backend data-processing evidence, but missing streaming tools"
		}
	}

	if isCloudPlatformRequirement(reqText) {
		switch {
		case containsAny(evidence, "aws", "cloud", "serverless", "kubernetes", "terraform", "iac"):
			return 0.58, "cloud platform evidence"
		case containsAny(evidence, "docker", "github actions", "compose", "linux", "postgresql", "database-backed", "backend"):
			return 0.48, "deployment/backend platform evidence, but not full cloud migration"
		}
	}

	if isArchitectureRequirement(reqText) {
		switch {
		case containsAny(evidence, "architecture", "architected", "designed", "system design"):
			return 0.70, "backend/system architecture evidence"
		case containsAny(evidence, "fastapi", "sqlalchemy", "pydantic", "crud", "service modules", "api", "apis", "microservices"):
			return 0.56, "API/service design evidence"
		}
	}

	if isCodeQualityRequirement(reqText) {
		switch {
		case containsAny(evidence, "unit test", "unit testing", "integration test", "integration testing", "code review", "code reviews"):
			return 0.70, "testing or code-review evidence"
		case containsAny(evidence, "refactored", "reliability", "audit logging", "rbac", "integration workflows", "production backend"):
			return 0.48, "quality/reliability engineering evidence, but not explicit tests or reviews"
		}
	}

	if isDatabaseRequirement(reqText) {
		switch {
		case containsAny(evidence, "postgresql", "mysql", "sqlite", "sql", "database", "repository layer", "sqlalchemy"):
			return 0.66, "relational database and querying evidence"
		case containsAny(evidence, "data model", "schema", "migration", "alembic"):
			return 0.56, "database modeling/migration evidence"
		}
	}

	if isProductRequirement(reqText) {
		switch {
		case containsAny(evidence, "production planning platform", "bookings", "asset scheduling", "planning workflows"):
			return 0.58, "product-facing workflow/platform delivery evidence"
		case containsAny(evidence, "login", "registration", "password", "user", "customer", "invited-user"):
			return 0.46, "user workflow delivery evidence"
		}
	}

	if isOwnershipRequirement(reqText) ||
		(category == "responsibility" && containsAny(reqText, "build", "develop", "deliver")) {
		switch {
		case containsAny(evidence, "built and shipped", "built", "shipped", "delivered", "end-to-end", "production backend"):
			return 0.62, "end-to-end feature or backend delivery evidence"
		case containsAny(evidence, "developed", "maintained", "implemented"):
			return 0.52, "software delivery evidence"
		}
	}

	if isTroubleshootingRequirement(reqText) {
		switch {
		case containsAny(evidence, "refactored", "failure handling", "troubleshoot", "debug", "reliability", "reduced manual intervention"):
			return 0.56, "troubleshooting/reliability improvement evidence"
		case containsAny(evidence, "linux automation", "automation scripts", "background workers", "integration workflows"):
			return 0.48, "automation/cross-system operations evidence"
		}
	}

	if isIoTRequirement(reqText) {
		switch {
		case containsAny(evidence, "iot", "device", "devices", "hardware", "firmware"):
			return 0.62, "device/IoT evidence"
		case containsAny(evidence, "linux", "automation", "monitoring", "configuration"):
			return 0.34, "systems automation evidence, but not IoT/device-health work"
		}
	}

	return 0, ""
}

func isStreamingRequirement(text string) bool {
	return containsAny(text, "kafka", "kinesis", "stream processing", "athlete data streams") ||
		(strings.Contains(text, "real-time") && strings.Contains(text, "data"))
}

func isCloudPlatformRequirement(text string) bool {
	return containsAny(text, "cloud-based", "cloud based", "large-scale data processing", "cloud platform", "migrate", "migration", "on-premise")
}

func isArchitectureRequirement(text string) bool {
	return containsAny(text, "technical design", "system architecture", "design patterns", "define system") ||
		(strings.Contains(text, "architecture") && !isDatabaseRequirement(text))
}

func isCodeQualityRequirement(text string) bool {
	return containsAny(text, "code reviews", "code quality", "coding standards", "unit testing", "integration testing")
}

func isDatabaseRequirement(text string) bool {
	return containsAny(text, "nosql", "relational", "database", "querying") ||
		(strings.Contains(text, "sql") && strings.Contains(text, "architecture"))
}

func isProductRequirement(text string) bool {
	return containsAny(text, "product mindset", "user needs", "customer and product", "delivering impactful")
}

func isOwnershipRequirement(text string) bool {
	return containsAny(text, "owning features", "end-to-end", "owning outcomes", "features & initiatives", "build applications", "build applications and services")
}

func isTroubleshootingRequirement(text string) bool {
	return containsAny(text, "troubleshooting", "solving complex", "cross-stack", "complex problems", "team productivity")
}

func isIoTRequirement(text string) bool {
	return containsAny(text, "iot", "device health", "health and configuration") ||
		(strings.Contains(text, "devices") && strings.Contains(text, "real-time"))
}

func matchCoverageForScore(score float64) string {
	switch {
	case score >= 0.75:
		return "strong"
	case score >= 0.45:
		return "partial"
	default:
		return "weak"
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func adjacentJobMatchScore(req JobRequirement, overlap []string, evidenceText string) float64 {
	meaningful := meaningfulJobOverlapTerms(overlap)
	if len(meaningful) == 0 {
		return 0
	}
	required := requirementRequiredMatchTerms(req)
	if len(required) == 0 {
		return 0
	}
	evidence := strings.ToLower(evidenceText)
	requiredHits := 0
	for _, term := range required {
		if containsString(meaningful, term) || strings.Contains(evidence, term) {
			requiredHits++
		}
	}
	if requiredHits == 0 {
		return 0
	}
	if normalizeRequirementCategory(req.Category) == "nice_to_have" {
		return 0.40
	}
	if req.Priority == "high" || normalizeRequirementCategory(req.Category) == "must_have" {
		return 0.34 + minFloat(float64(requiredHits)*0.03, 0.06)
	}
	return 0.38
}

func jobMatchScore(req JobRequirement, overlap []string, evidenceText string, base float64) float64 {
	meaningful := meaningfulJobOverlapTerms(overlap)
	score := base + float64(len(meaningful))*0.14
	required := requirementRequiredMatchTerms(req)
	if len(required) > 0 {
		hits := 0
		evidence := strings.ToLower(evidenceText)
		for _, term := range required {
			if containsString(meaningful, term) || strings.Contains(evidence, term) {
				hits++
			}
		}
		if hits == 0 {
			score -= 0.25
		} else {
			score += float64(hits) * 0.08
		}
	}
	if normalizeRequirementCategory(req.Category) == "nice_to_have" {
		score = minFloat(score, 0.72)
	}
	return clampScore(score)
}

func meaningfulJobOverlapTerms(overlap []string) []string {
	terms := []string{}
	for _, term := range overlap {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || isJobStopWord(term) || weakJobMatchTerm(term) {
			continue
		}
		terms = append(terms, term)
	}
	return normalizeStringList(terms)
}

func requirementRequiredMatchTerms(req JobRequirement) []string {
	lower := strings.ToLower(strings.Join([]string{req.RequirementText, strings.Join(req.Keywords, " ")}, " "))
	groups := [][]string{
		{"kafka", "kinesis", "aws", "edge devices", "real-time", "stream processing"},
		{"iot", "device", "devices", "real-time"},
		{"cloud", "migration", "on-premise", "aws"},
		{"microservice", "microservices", "aws", "domain-driven", "ddd"},
		{"nosql", "relational", "database", "querying", "performance"},
		{"code review", "code reviews", "code quality", "unit testing", "integration testing", "coding standards"},
		{"firmware", "hardware", "edge"},
	}
	if strings.Contains(lower, "specific technologies") ||
		strings.Contains(lower, "advantageous but not mandatory") ||
		strings.Contains(lower, "not mandatory") ||
		strings.Contains(lower, "our stack") {
		groups = append([][]string{{"go", "rust", "c#", ".net", "c++", "iac", "iot", "aws"}}, groups...)
	}
	for _, group := range groups {
		matches := []string{}
		for _, term := range group {
			if strings.Contains(lower, term) {
				matches = append(matches, term)
			}
		}
		if len(matches) > 0 {
			return normalizeStringList(matches)
		}
	}
	return nil
}

func weakJobMatchTerm(term string) bool {
	switch strings.ToLower(strings.TrimSpace(term)) {
	case "app", "apps", "application", "applications", "service", "services", "data", "time", "real", "live",
		"next", "solution", "solutions", "customer", "product", "mindset", "support", "configuration",
		"development", "technical", "capability", "high", "degree", "company", "department", "growth",
		"best", "practice", "practices", "lessons", "learned", "people", "rare", "roles", "closely",
		"favourite", "sports", "likely", "most", "based", "large", "paving", "leading", "market",
		"track", "record", "end", "features", "initiatives", "but", "net", "stack", "specific",
		"user", "users", "delivering", "impactful", "integration", "reliability", "coding", "quality",
		"ensure", "adherence", "participate", "conduct", "demonstrate", "consistently", "complex",
		"problems", "solving", "cross", "generation", "migrate", "experiences", "athlete":
		return true
	default:
		return false
	}
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
	action := resumeActionVerb(firstString(claim.Actions))
	capability := firstNonEmpty(firstString(claim.Artifacts), firstString(claim.Objects), firstString(claim.Capabilities), claim.ClaimText)
	tools := strings.Join(limitStrings(claim.Technologies, 3), "/")
	scope := strings.Join(limitStrings(firstNonEmptyStringList(claim.Scope, claim.Objects, claim.Artifacts), 3), ", ")
	outcome := strings.Join(limitStrings(claim.Outcomes, 2), ", ")
	metrics := strings.Join(limitStrings(claim.Metrics, 2), ", ")
	parts := []string{action}
	if capability != "" {
		parts = append(parts, capability)
	}
	if tools != "" {
		parts = append(parts, "with "+tools)
	}
	if scope != "" && !strings.Contains(strings.ToLower(strings.Join(parts, " ")), strings.ToLower(scope)) {
		parts = append(parts, "for "+scope)
	}
	if outcome != "" {
		parts = append(parts, "to support "+outcome)
	}
	if metrics != "" && !strings.Contains(strings.ToLower(strings.Join(parts, " ")), strings.ToLower(metrics)) {
		parts = append(parts, "measured by "+metrics)
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

func resumeActionVerb(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "add", "added":
		return "Added"
	case "automate", "automated":
		return "Automated"
	case "create", "created":
		return "Created"
	case "deliver", "delivered":
		return "Delivered"
	case "deploy", "deployed":
		return "Deployed"
	case "design", "designed":
		return "Designed"
	case "develop", "developed":
		return "Developed"
	case "implement", "implemented":
		return "Implemented"
	case "improve", "improved":
		return "Improved"
	case "integrate", "integrated":
		return "Integrated"
	case "migrate", "migrated":
		return "Migrated"
	case "optimize", "optimized":
		return "Optimized"
	case "reduce", "reduced":
		return "Reduced"
	case "ship", "shipped":
		return "Shipped"
	case "support", "supported":
		return "Supported"
	case "test", "tested":
		return "Tested"
	default:
		return "Built"
	}
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
			if strings.Contains(text, term) && !weakJobMatchTerm(term) {
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
			if strings.Contains(text, term) && !weakJobMatchTerm(term) {
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
			claim_ids_json, origin_heading, origin_type, selection_score, selected_for_resume,
			resume_value_score, jd_relevance_score, origin_weight, risk_penalty, unsupported_context_penalty, selection_reason,
			value_theme, display_order
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
		job.RawText = normalizePastedText(job.RawText)
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
			&draft.ResumeValueScore,
			&draft.JDRelevanceScore,
			&draft.OriginWeight,
			&draft.RiskPenalty,
			&draft.UnsupportedPenalty,
			&draft.SelectionReason,
			&draft.ValueTheme,
			&draft.DisplayOrder,
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

func scanBulletGenerationEvents(rows *sql.Rows) ([]BulletGenerationEvent, error) {
	events := []BulletGenerationEvent{}
	for rows.Next() {
		var event BulletGenerationEvent
		if err := rows.Scan(
			&event.ID,
			&event.JobID,
			&event.OriginHeading,
			&event.Stage,
			&event.Status,
			&event.Reason,
			&event.DraftText,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
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

func minFloat(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func filterLLMRiskFlags(flags []string) []string {
	allowed := map[string]bool{
		"weak_evidence":         true,
		"ambiguous_metric":      true,
		"ambiguous_scope":       true,
		"ambiguous_technology":  true,
		"inferred_tailoring":    true,
		"approved_restricted":   true,
		"unsupported_metric":    true,
		"unsupported_tool":      true,
		"unsupported_seniority": true,
		"unsupported_ownership": true,
	}

	filtered := []string{}
	for _, flag := range flags {
		normalized := strings.TrimSpace(strings.ToLower(flag))
		if allowed[normalized] {
			filtered = append(filtered, normalized)
		}
	}

	return normalizeStringList(filtered)
}

func invalidBulletText(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return true
	}

	badPhrases := []string{
		"design backend development",
		"built api development",
		"api development across",
		"using postgresql/sql across",
		"using fastapi/sql across",
		"across activities, and",
		"and ai-assisted workflows, and",
		"worked on",
		"helped with",
		"used various",
		"contributed to team",
		"collaborated with",
		"supported development",
		"participated in",
		"business outcomes",
	}

	for _, phrase := range badPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	if strings.Contains(lower, "2:3") || strings.Contains(lower, "3:2") {
		return true
	}

	if strings.Contains(lower, " a and ") {
		return true
	}

	words := strings.Fields(lower)
	if len(words) < 10 {
		return true
	}

	first := strings.Trim(words[0], " ,.;:")
	validStarts := map[string]bool{
		"built":         true,
		"designed":      true,
		"implemented":   true,
		"refactored":    true,
		"developed":     true,
		"modeled":       true,
		"shipped":       true,
		"created":       true,
		"integrated":    true,
		"containerized": true,
		"automated":     true,
		"optimized":     true,
		"improved":      true,
	}

	if !validStarts[first] {
		return true
	}

	return false
}
