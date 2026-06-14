package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	claimStatusApproved           = "approved"
	claimStatusApprovedRestricted = "approved_restricted"
	claimStatusNeedsReview        = "needs_review"
	claimStatusRejected           = "rejected"
	claimStatusBlocked            = "blocked"
)

type CandidateClaim struct {
	ID               int64    `json:"id"`
	ClaimText        string   `json:"claim_text"`
	ClaimType        string   `json:"claim_type"`
	SourceFactIDs    []int64  `json:"source_fact_ids"`
	EvidenceQuotes   []string `json:"evidence_quotes"`
	Technologies     []string `json:"technologies"`
	Actions          []string `json:"actions"`
	Capabilities     []string `json:"capabilities"`
	Objects          []string `json:"objects"`
	Domains          []string `json:"domains"`
	Artifacts        []string `json:"artifacts"`
	Scope            []string `json:"scope"`
	Metrics          []string `json:"metrics"`
	Outcomes         []string `json:"outcomes"`
	ProfileContext   []string `json:"profile_context"`
	EvidenceStrength string   `json:"evidence_strength"`
	Strength         string   `json:"strength"`
	AllowedUse       []string `json:"allowed_use"`
	AllowedContexts  []string `json:"allowed_contexts"`
	BlockedContexts  []string `json:"blocked_contexts"`
	SafePhrasings    []string `json:"safe_phrasings"`
	UnsafePhrasings  []string `json:"unsafe_phrasings"`
	OriginHeading    string   `json:"origin_heading"`
	OriginType       string   `json:"origin_type"`
	Status           string   `json:"status"`
	RiskFlags        []string `json:"risk_flags"`
	SimilarityKey    string   `json:"similarity_key"`
	SimilarityScore  float64  `json:"similarity_score"`
	DuplicateOfID    int64    `json:"duplicate_of_id"`
	ReviewNote       string   `json:"review_note"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type UpdateCandidateClaimReviewInput struct {
	ID               int64    `json:"id"`
	ClaimText        string   `json:"claim_text"`
	ClaimType        string   `json:"claim_type"`
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
	EvidenceStrength string   `json:"evidence_strength"`
	Strength         string   `json:"strength"`
	AllowedUse       []string `json:"allowed_use"`
	AllowedContexts  []string `json:"allowed_contexts"`
	BlockedContexts  []string `json:"blocked_contexts"`
	SafePhrasings    []string `json:"safe_phrasings"`
	UnsafePhrasings  []string `json:"unsafe_phrasings"`
	Status           string   `json:"status"`
	RiskFlags        []string `json:"risk_flags"`
	ReviewNote       string   `json:"review_note"`
}

type BlockedClaim struct {
	ID        int64  `json:"id"`
	Pattern   string `json:"pattern"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateBlockedClaimInput struct {
	Pattern  string `json:"pattern"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Enabled  bool   `json:"enabled"`
}

type UpdateBlockedClaimInput struct {
	ID       int64  `json:"id"`
	Pattern  string `json:"pattern"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Enabled  bool   `json:"enabled"`
}

type parsedClaimsResponse struct {
	Claims []parsedCandidateClaim `json:"claims"`
}

type parsedCandidateClaim struct {
	ClaimText        string   `json:"claim_text"`
	Label            string   `json:"label"`
	ClaimType        string   `json:"claim_type"`
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
	EvidenceStrength string   `json:"evidence_strength"`
	Strength         string   `json:"strength"`
	AllowedUse       []string `json:"allowed_use"`
	AllowedContexts  []string `json:"allowed_contexts"`
	BlockedContexts  []string `json:"blocked_contexts"`
	SafePhrasings    []string `json:"safe_phrasings"`
	UnsafePhrasings  []string `json:"unsafe_phrasings"`
	RiskFlags        []string `json:"risk_flags"`
}

func (s *Store) GenerateCandidateClaims(ctx context.Context, client *http.Client) ([]CandidateClaim, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	facts, err := s.listFactsForClaims()
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return nil, errors.New("approve or review evidence facts before generating claims")
	}
	promptFacts := compactFactsForClaims(facts)
	factJSON, _ := json.Marshal(promptFacts)
	rules := s.promptRuleDigest("resume", "validation")
	system := `You are JD Tailor's claim-ledger builder. Return strict JSON only.
Convert evidence facts into compact profile-bank atoms without inventing unsupported scope.`
	user := fmt.Sprintf(`# Task
Generate atom-first candidate profile records from evidence facts.

# Output JSON schema
{"claims":[{"label":"","claim_type":"experience|project|skills|education","source_fact_ids":[0],"actions":[],"capabilities":[],"objects":[],"technologies":[],"domains":[],"artifacts":[],"scope":[],"metrics":[],"outcomes":[],"profile_context":[],"evidence_strength":"direct|inferred|weak","strength":"strong|moderate|weak","allowed_use":[],"allowed_contexts":[],"blocked_contexts":[],"safe_phrasings":[],"unsafe_phrasings":[],"risk_flags":[]}]}

# Rules
- Use only IDs from <evidence_facts_json>.
- label must be a short searchable label, not a resume bullet and not a full sentence.
- Store reusable keywords/phrases in atom arrays. Keep each atom short and atomic: one action, artifact, tool, metric, scope, or outcome per array item.
- Combine multiple source_fact_ids only when the facts share the same origin and support one coherent future resume insight.
- Do not output polished resume bullets, marketing phrasing, or complete achievement sentences.
- Do not add tools, metrics, leadership, seniority, cloud ownership, ML research, Kubernetes ownership, production scope, or domain claims unless facts prove them.
- Prefer records that can later support resume bullets, summary lines, or skills entries.
- Set blocked_contexts and unsafe_phrasings for likely overclaims.
- Mark project evidence as project unless the fact origin is employment.
- Keep labels and atoms plain, technical, and compact.

# Style and validation rules
%s

<evidence_facts_json>
%s
</evidence_facts_json>`, firstNonEmpty(rules, "Use only sourced facts. Avoid inflated phrasing."), string(factJSON))

	parsed := []parsedCandidateClaim{}
	text, err := s.GenerateLLMText(ctx, client, system, user, 1800)
	if err == nil {
		parsed, err = parseCandidateClaims(text)
	}
	if err != nil {
		parsed = fallbackCandidateClaims(facts)
		_ = s.LogEvent("warning", "candidate claims used local fallback: "+err.Error())
	}
	return s.replaceCandidateClaims(parsed, facts)
}

func (s *Store) listFactsForClaims() ([]factPromptContext, error) {
	facts, err := s.listFactPromptContext()
	if err != nil {
		return nil, err
	}
	filtered := []factPromptContext{}
	for _, fact := range facts {
		if fact.Status == factStatusRejected {
			continue
		}
		filtered = append(filtered, fact)
	}
	return filtered, nil
}

func (s *Store) ListCandidateClaims(status string) ([]CandidateClaim, error) {
	query := `SELECT id, claim_text, claim_type, source_fact_ids_json, evidence_quotes_json, technologies_json,
		strength, allowed_use_json, allowed_contexts_json, blocked_contexts_json, safe_phrasings_json,
		unsafe_phrasings_json, origin_heading, origin_type, status, risk_flags_json, review_note, created_at, updated_at,
		actions_json, capabilities_json, objects_json, domains_json, artifacts_json, scope_json, metrics_json,
		outcomes_json, profile_context_json, evidence_strength, similarity_key, similarity_score, duplicate_of_id
		FROM candidate_claims`
	args := []any{}
	if strings.TrimSpace(status) != "" && status != "all" {
		query += ` WHERE status = ?`
		args = append(args, normalizeClaimStatus(status))
	}
	query += ` ORDER BY CASE status WHEN 'needs_review' THEN 0 WHEN 'approved_restricted' THEN 1 WHEN 'approved' THEN 2 WHEN 'blocked' THEN 3 ELSE 4 END, id DESC`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidateClaims(rows)
}

func (s *Store) UpdateCandidateClaimReview(input UpdateCandidateClaimReviewInput) (CandidateClaim, error) {
	if input.ID <= 0 {
		return CandidateClaim{}, errors.New("claim id is required")
	}
	claimText := strings.TrimSpace(input.ClaimText)
	if claimText == "" {
		return CandidateClaim{}, errors.New("claim text is required")
	}
	techJSON, _ := encodeStringList(input.Technologies)
	actionsJSON, _ := encodeStringList(input.Actions)
	capabilitiesJSON, _ := encodeStringList(input.Capabilities)
	objectsJSON, _ := encodeStringList(input.Objects)
	domainsJSON, _ := encodeStringList(input.Domains)
	artifactsJSON, _ := encodeStringList(input.Artifacts)
	scopeJSON, _ := encodeStringList(input.Scope)
	metricsJSON, _ := encodeStringList(input.Metrics)
	outcomesJSON, _ := encodeStringList(input.Outcomes)
	profileContextJSON, _ := encodeStringList(input.ProfileContext)
	similarityKey := normalizedSimilarityKey(strings.Join([]string{
		claimText,
		strings.Join(input.Actions, " "),
		strings.Join(input.Capabilities, " "),
		strings.Join(input.Objects, " "),
		strings.Join(input.Domains, " "),
		strings.Join(input.Artifacts, " "),
		strings.Join(input.Scope, " "),
		strings.Join(input.Metrics, " "),
		strings.Join(input.Outcomes, " "),
	}, " "), input.Technologies, input.ProfileContext)
	allowedUseJSON, _ := encodeStringList(input.AllowedUse)
	allowedContextsJSON, _ := encodeStringList(input.AllowedContexts)
	blockedContextsJSON, _ := encodeStringList(input.BlockedContexts)
	safeJSON, _ := encodeStringList(input.SafePhrasings)
	unsafeJSON, _ := encodeStringList(input.UnsafePhrasings)
	riskFlags := normalizeStringList(append(input.RiskFlags, styleRiskFlags(strings.Join([]string{claimText, strings.Join(input.Actions, " "), strings.Join(input.Capabilities, " ")}, " "))...))
	riskJSON, _ := encodeStringList(riskFlags)
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE candidate_claims
		SET claim_text = ?, claim_type = ?, technologies_json = ?, actions_json = ?, capabilities_json = ?,
			objects_json = ?, domains_json = ?, artifacts_json = ?, scope_json = ?, metrics_json = ?,
			outcomes_json = ?, profile_context_json = ?, evidence_strength = ?, strength = ?,
			allowed_use_json = ?, allowed_contexts_json = ?, blocked_contexts_json = ?, safe_phrasings_json = ?,
			unsafe_phrasings_json = ?, status = ?, risk_flags_json = ?, review_note = ?, updated_at = ?
			, similarity_key = ?, similarity_score = 1, duplicate_of_id = 0
		WHERE id = ?`,
		claimText,
		normalizeClaimType(input.ClaimType),
		techJSON,
		actionsJSON,
		capabilitiesJSON,
		objectsJSON,
		domainsJSON,
		artifactsJSON,
		scopeJSON,
		metricsJSON,
		outcomesJSON,
		profileContextJSON,
		normalizeEvidenceStrength(input.EvidenceStrength),
		normalizeClaimStrength(input.Strength),
		allowedUseJSON,
		allowedContextsJSON,
		blockedContextsJSON,
		safeJSON,
		unsafeJSON,
		normalizeClaimStatus(input.Status),
		riskJSON,
		strings.TrimSpace(input.ReviewNote),
		time.Now().UTC().Format(time.RFC3339),
		similarityKey,
		input.ID,
	)
	if err != nil {
		return CandidateClaim{}, err
	}
	updated, err := s.getCandidateClaim(input.ID)
	if err != nil {
		return CandidateClaim{}, err
	}
	if err := s.DeleteAllEvidenceFacts(); err != nil {
		_ = s.LogEvent("error", "failed to invalidate evidence facts after claim update: "+err.Error())
	}
	return updated, nil
}

func (s *Store) DeleteCandidateClaim(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("claim id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM candidate_claims WHERE id = ?`, input.ID)
	if err != nil {
		return err
	}
	if err := s.DeleteAllEvidenceFacts(); err != nil {
		_ = s.LogEvent("error", "failed to invalidate evidence facts after claim delete: "+err.Error())
	}
	return nil
}

func (s *Store) DeleteAllCandidateClaims() error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM candidate_claims`)
	if err != nil {
		return err
	}
	_ = s.LogEvent("info", "all candidate claims deleted")
	return nil
}

func (s *Store) ListBlockedClaims() ([]BlockedClaim, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, pattern, reason, severity, source, enabled, created_at, updated_at FROM blocked_claims ORDER BY enabled DESC, severity, pattern`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlockedClaims(rows)
}

func (s *Store) CreateBlockedClaim(input CreateBlockedClaimInput) (BlockedClaim, error) {
	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" {
		return BlockedClaim{}, errors.New("blocked claim pattern is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO blocked_claims (pattern, reason, severity, source, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		pattern,
		strings.TrimSpace(input.Reason),
		normalizeSeverity(input.Severity),
		strings.TrimSpace(input.Source),
		boolToInt(input.Enabled),
		now,
		now,
	)
	if err != nil {
		return BlockedClaim{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return BlockedClaim{}, err
	}
	return s.getBlockedClaim(id)
}

func (s *Store) UpdateBlockedClaim(input UpdateBlockedClaimInput) (BlockedClaim, error) {
	if input.ID <= 0 {
		return BlockedClaim{}, errors.New("blocked claim id is required")
	}
	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" {
		return BlockedClaim{}, errors.New("blocked claim pattern is required")
	}
	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE blocked_claims SET pattern = ?, reason = ?, severity = ?, source = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		pattern,
		strings.TrimSpace(input.Reason),
		normalizeSeverity(input.Severity),
		strings.TrimSpace(input.Source),
		boolToInt(input.Enabled),
		time.Now().UTC().Format(time.RFC3339),
		input.ID,
	)
	if err != nil {
		return BlockedClaim{}, err
	}
	_, _ = s.db.ExecContext(context.Background(), `DELETE FROM candidate_claims`)
	_ = s.LogEvent("info", "blocked claim updated")
	return s.getBlockedClaim(input.ID)
}

func (s *Store) DeleteBlockedClaim(input DeleteInput) error {
	if input.ID <= 0 {
		return errors.New("blocked claim id is required")
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM blocked_claims WHERE id = ?`, input.ID)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(context.Background(), `DELETE FROM candidate_claims`)
	_ = s.LogEvent("info", "blocked claim deleted")
	return nil
}

func (s *Store) seedBlockedClaimDefaults(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, blocked := range defaultBlockedClaims() {
		if _, err := s.db.ExecContext(
			ctx,
			`INSERT INTO blocked_claims (pattern, reason, severity, source, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, ?, ?)
			ON CONFLICT(pattern) DO NOTHING`,
			blocked.Pattern,
			blocked.Reason,
			blocked.Severity,
			blocked.Source,
			now,
			now,
		); err != nil {
			return err
		}
	}
	return nil
}

func defaultBlockedClaims() []BlockedClaim {
	source := "plans/JOB_COPILOT_FULL_PLAN.md"
	return []BlockedClaim{
		{Pattern: "senior/staff/lead title", Reason: "Unsupported seniority framing unless title evidence proves it.", Severity: "high", Source: source},
		{Pattern: "ML Engineer positioning", Reason: "Do not position as ML Engineer unless evidence supports ML engineering work.", Severity: "high", Source: source},
		{Pattern: "Data Engineer positioning", Reason: "Do not position as Data Engineer unless evidence supports data engineering ownership.", Severity: "high", Source: source},
		{Pattern: "DevOps/SRE ownership", Reason: "Do not imply DevOps or SRE ownership unless evidenced.", Severity: "high", Source: source},
		{Pattern: "Kubernetes ownership", Reason: "Do not claim Kubernetes ownership unless evidenced.", Severity: "high", Source: source},
		{Pattern: "cloud infrastructure ownership", Reason: "Do not claim cloud infrastructure ownership unless evidenced.", Severity: "high", Source: source},
		{Pattern: "invented metrics", Reason: "Metrics must be copied from source evidence.", Severity: "high", Source: source},
		{Pattern: "team leadership", Reason: "Do not imply team leadership unless evidenced.", Severity: "high", Source: source},
		{Pattern: "project work as employment", Reason: "Do not represent project work as employment.", Severity: "high", Source: source},
		{Pattern: "model training or ML research", Reason: "Do not claim model training or ML research unless evidenced.", Severity: "high", Source: source},
	}
}

func (s *Store) replaceCandidateClaims(parsed []parsedCandidateClaim, facts []factPromptContext) ([]CandidateClaim, error) {
	factsByID := map[int64]factPromptContext{}
	for _, fact := range facts {
		factsByID[fact.ID] = fact
	}
	parsed = mergeSimilarParsedClaims(parsed, factsByID)
	blocked, err := s.ListBlockedClaims()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM candidate_claims`); err != nil {
		return nil, err
	}
	ids := []int64{}
	seen := map[string]bool{}
	for _, claim := range parsed {
		claim.ClaimText = normalizeClaimLabel(claim)
		if claim.ClaimText == "" {
			continue
		}
		validFactIDs, sourceFacts := validClaimFacts(claim.SourceFactIDs, factsByID)
		if len(validFactIDs) == 0 {
			continue
		}
		claim = enrichClaimAtoms(claim, sourceFacts)
		key := strings.ToLower(strings.Join(append([]string{claim.ClaimText}, claim.SourceFactIDsKey()...), "|"))
		if seen[key] {
			continue
		}
		seen[key] = true
		evidenceQuotes, tech, originHeading, originType := claimEvidenceContext(sourceFacts)
		riskFlags := verifyClaimRiskFlags(claim, sourceFacts, blocked)
		status := claimStatusForFacts(sourceFacts, riskFlags)
		if normalizeClaimStatus(claimStatusBlocked) == status {
			claim.BlockedContexts = normalizeStringList(append(claim.BlockedContexts, "blocked_by_negative_profile"))
		}
		sourceFactJSON, _ := encodeInt64List(validFactIDs)
		similarityKey := claimSimilarityKey(claim, sourceFacts)
		evidenceJSON, _ := encodeStringList(evidenceQuotes)
		techJSON, _ := encodeStringList(firstNonEmptyList(claimTechnologies(claim), tech))
		actionsJSON, _ := encodeStringList(claim.Actions)
		capabilitiesJSON, _ := encodeStringList(claim.Capabilities)
		objectsJSON, _ := encodeStringList(claim.Objects)
		domainsJSON, _ := encodeStringList(claim.Domains)
		artifactsJSON, _ := encodeStringList(claim.Artifacts)
		scopeJSON, _ := encodeStringList(claim.Scope)
		metricsJSON, _ := encodeStringList(claim.Metrics)
		outcomesJSON, _ := encodeStringList(claim.Outcomes)
		profileContextJSON, _ := encodeStringList(claim.ProfileContext)
		allowedUseJSON, _ := encodeStringList(defaultAllowedUse(claim.AllowedUse, originType))
		allowedContextsJSON, _ := encodeStringList(defaultAllowedContexts(claim.AllowedContexts, sourceFacts))
		blockedContextsJSON, _ := encodeStringList(defaultBlockedContexts(claim.BlockedContexts, claim.ClaimText))
		safeJSON, _ := encodeStringList(defaultSafePhrasings(claim.SafePhrasings, claim.ClaimText))
		unsafeJSON, _ := encodeStringList(defaultUnsafePhrasings(claim.UnsafePhrasings, claim.ClaimText, riskFlags))
		riskJSON, _ := encodeStringList(riskFlags)
		result, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO candidate_claims
				(claim_text, claim_type, source_fact_ids_json, evidence_quotes_json, technologies_json, strength,
				allowed_use_json, allowed_contexts_json, blocked_contexts_json, safe_phrasings_json, unsafe_phrasings_json,
				origin_heading, origin_type, status, risk_flags_json, review_note, created_at, updated_at,
				actions_json, capabilities_json, objects_json, domains_json, artifacts_json, scope_json, metrics_json,
				outcomes_json, profile_context_json, evidence_strength, similarity_key, similarity_score, duplicate_of_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
			claim.ClaimText,
			normalizeClaimType(firstNonEmpty(claim.ClaimType, originType)),
			sourceFactJSON,
			evidenceJSON,
			techJSON,
			normalizeClaimStrength(firstNonEmpty(claim.Strength, inferClaimStrength(sourceFacts))),
			allowedUseJSON,
			allowedContextsJSON,
			blockedContextsJSON,
			safeJSON,
			unsafeJSON,
			originHeading,
			originType,
			status,
			riskJSON,
			now,
			now,
			actionsJSON,
			capabilitiesJSON,
			objectsJSON,
			domainsJSON,
			artifactsJSON,
			scopeJSON,
			metricsJSON,
			outcomesJSON,
			profileContextJSON,
			normalizeEvidenceStrength(claim.EvidenceStrength),
			similarityKey,
			1.0,
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
		return nil, errors.New("no usable candidate claims generated")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = s.LogEvent("info", "candidate claims generated")
	return s.ListCandidateClaims("all")
}

func parseCandidateClaims(text string) ([]parsedCandidateClaim, error) {
	var parsed parsedClaimsResponse
	if err := parseJSONObject(text, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Claims) == 0 {
		return nil, errors.New("LLM returned no claims")
	}
	return parsed.Claims, nil
}

func mergeSimilarParsedClaims(parsed []parsedCandidateClaim, factsByID map[int64]factPromptContext) []parsedCandidateClaim {
	merged := []parsedCandidateClaim{}
	indexByKey := map[string]int{}
	for _, claim := range parsed {
		validIDs, sourceFacts := validClaimFacts(claim.SourceFactIDs, factsByID)
		if len(validIDs) == 0 {
			continue
		}
		claim.SourceFactIDs = validIDs
		claim.ClaimText = normalizeClaimLabel(claim)
		claim = enrichClaimAtoms(claim, sourceFacts)
		key := claimSimilarityKey(claim, sourceFacts)
		if key == "" {
			key = strings.ToLower(claim.ClaimText)
		}
		if existingIndex, ok := indexByKey[key]; ok {
			existing := merged[existingIndex]
			existing.SourceFactIDs = uniqueInt64s(append(existing.SourceFactIDs, claim.SourceFactIDs...))
			existing.Actions = normalizeStringList(append(existing.Actions, claim.Actions...))
			existing.Capabilities = normalizeStringList(append(existing.Capabilities, claim.Capabilities...))
			existing.Objects = normalizeStringList(append(existing.Objects, claim.Objects...))
			existing.Technologies = normalizeStringList(append(existing.Technologies, claim.Technologies...))
			existing.Domains = normalizeStringList(append(existing.Domains, claim.Domains...))
			existing.Artifacts = normalizeStringList(append(existing.Artifacts, claim.Artifacts...))
			existing.Scope = normalizeStringList(append(existing.Scope, claim.Scope...))
			existing.Metrics = normalizeStringList(append(existing.Metrics, claim.Metrics...))
			existing.Outcomes = normalizeStringList(append(existing.Outcomes, claim.Outcomes...))
			existing.ProfileContext = normalizeStringList(append(existing.ProfileContext, claim.ProfileContext...))
			existing.AllowedUse = normalizeStringList(append(existing.AllowedUse, claim.AllowedUse...))
			existing.AllowedContexts = normalizeStringList(append(existing.AllowedContexts, claim.AllowedContexts...))
			existing.BlockedContexts = normalizeStringList(append(existing.BlockedContexts, claim.BlockedContexts...))
			existing.SafePhrasings = normalizeStringList(append(existing.SafePhrasings, claim.SafePhrasings...))
			existing.UnsafePhrasings = normalizeStringList(append(existing.UnsafePhrasings, claim.UnsafePhrasings...))
			existing.RiskFlags = normalizeStringList(append(existing.RiskFlags, claim.RiskFlags...))
			merged[existingIndex] = existing
			continue
		}
		indexByKey[key] = len(merged)
		merged = append(merged, claim)
	}
	return merged
}

func claimSimilarityKey(claim parsedCandidateClaim, facts []factPromptContext) string {
	context := append([]string{}, claim.ProfileContext...)
	for _, fact := range facts {
		context = append(context, fact.Context...)
		context = append(context, fact.SectionHeading, fact.SectionType)
	}
	text := strings.Join([]string{
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
	}, " ")
	return normalizedSimilarityKey(text, claim.Technologies, context)
}

func fallbackCandidateClaims(facts []factPromptContext) []parsedCandidateClaim {
	claims := []parsedCandidateClaim{}
	for _, fact := range facts {
		if fact.Status == factStatusRejected {
			continue
		}
		atoms := atomsFromFact(fact)
		label := claimLabelFromAtoms(atoms)
		if label == "" {
			continue
		}
		claims = append(claims, parsedCandidateClaim{
			ClaimText:        label,
			Label:            label,
			ClaimType:        normalizeClaimType(fact.SectionType),
			SourceFactIDs:    []int64{fact.ID},
			Actions:          atoms.Actions,
			Capabilities:     atoms.Capabilities,
			Objects:          atoms.Objects,
			Technologies:     atoms.Technologies,
			Domains:          atoms.Domains,
			Artifacts:        atoms.Artifacts,
			Scope:            atoms.Scope,
			Metrics:          atoms.Metrics,
			Outcomes:         atoms.Outcomes,
			ProfileContext:   atoms.ProfileContext,
			EvidenceStrength: inferEvidenceStrengthFromFact(fact),
			Strength:         inferClaimStrength([]factPromptContext{fact}),
			AllowedUse:       defaultAllowedUse(nil, fact.SectionType),
			AllowedContexts:  claimContextsFromFact(fact),
			BlockedContexts:  defaultBlockedContexts(nil, label),
			SafePhrasings:    []string{label},
			UnsafePhrasings:  []string{},
			RiskFlags:        fact.RiskFlags,
		})
	}
	return claims
}

func claimTextFromFact(fact factPromptContext) string {
	text := strings.TrimSpace(fact.FactText)
	if text == "" {
		return ""
	}
	parts := parseKeyValueFact(text)
	action := parts["actions"]
	if action == "" {
		action = parts["action"]
	}
	artifact := firstNonEmpty(parts["artifact"], parts["scope"], strings.TrimPrefix(parts["evidence"], "- "))
	tools := firstNonEmpty(parts["tools"], strings.Join(fact.Technologies, ", "))
	outcome := parts["outcome"]
	segments := []string{}
	if action != "" && artifact != "" {
		segments = append(segments, strings.Title(strings.Split(action, ",")[0])+" "+artifact)
	} else if artifact != "" {
		segments = append(segments, artifact)
	} else {
		segments = append(segments, text)
	}
	if tools != "" {
		segments = append(segments, "using "+tools)
	}
	if outcome != "" {
		segments = append(segments, "with "+outcome)
	}
	claim := strings.TrimSpace(strings.Join(segments, " "))
	claim = regexp.MustCompile(`\s+`).ReplaceAllString(claim, " ")
	return strings.TrimSuffix(claim, ".") + "."
}

type claimAtoms struct {
	Actions        []string
	Capabilities   []string
	Objects        []string
	Technologies   []string
	Domains        []string
	Artifacts      []string
	Scope          []string
	Metrics        []string
	Outcomes       []string
	ProfileContext []string
}

func atomsFromFact(fact factPromptContext) claimAtoms {
	parts := parseKeyValueFact(fact.FactText)
	text := strings.TrimSpace(strings.Join([]string{fact.FactText, fact.EvidenceQuote}, " "))
	atoms := claimAtoms{
		Actions:        splitAtomList(firstNonEmpty(parts["actions"], parts["action"], inferActionAtom(text))),
		Capabilities:   splitAtomList(firstNonEmpty(parts["capabilities"], parts["capability"], inferCapabilityAtom(text))),
		Objects:        splitAtomList(firstNonEmpty(parts["objects"], parts["object"])),
		Technologies:   normalizeStringList(append(splitAtomList(parts["tools"]), fact.Technologies...)),
		Domains:        splitAtomList(firstNonEmpty(parts["domain"], parts["domains"], inferDomainAtom(text))),
		Artifacts:      splitAtomList(firstNonEmpty(parts["artifact"], parts["artifacts"])),
		Scope:          splitAtomList(firstNonEmpty(parts["scope"], parts["evidence"])),
		Metrics:        extractMetricAtoms(text),
		Outcomes:       splitAtomList(firstNonEmpty(parts["outcome"], parts["outcomes"])),
		ProfileContext: normalizeStringList(append(fact.Context, fact.SectionHeading, fact.SectionType)),
	}
	if len(atoms.Objects) == 0 && len(atoms.Artifacts) > 0 {
		atoms.Objects = atoms.Artifacts
	}
	return atoms
}

func enrichClaimAtoms(claim parsedCandidateClaim, facts []factPromptContext) parsedCandidateClaim {
	for _, fact := range facts {
		atoms := atomsFromFact(fact)
		claim.Actions = normalizeStringList(append(claim.Actions, atoms.Actions...))
		claim.Capabilities = normalizeStringList(append(claim.Capabilities, atoms.Capabilities...))
		claim.Objects = normalizeStringList(append(claim.Objects, atoms.Objects...))
		claim.Technologies = normalizeStringList(append(claim.Technologies, atoms.Technologies...))
		claim.Domains = normalizeStringList(append(claim.Domains, atoms.Domains...))
		claim.Artifacts = normalizeStringList(append(claim.Artifacts, atoms.Artifacts...))
		claim.Scope = normalizeStringList(append(claim.Scope, atoms.Scope...))
		claim.Metrics = normalizeStringList(append(claim.Metrics, atoms.Metrics...))
		claim.Outcomes = normalizeStringList(append(claim.Outcomes, atoms.Outcomes...))
		claim.ProfileContext = normalizeStringList(append(claim.ProfileContext, atoms.ProfileContext...))
		if claim.EvidenceStrength == "" {
			claim.EvidenceStrength = inferEvidenceStrengthFromFact(fact)
		}
	}
	claim.Actions = normalizeStringList(claim.Actions)
	claim.Capabilities = normalizeStringList(claim.Capabilities)
	claim.Objects = normalizeStringList(claim.Objects)
	claim.Technologies = normalizeStringList(claim.Technologies)
	claim.Domains = normalizeStringList(claim.Domains)
	claim.Artifacts = normalizeStringList(claim.Artifacts)
	claim.Scope = normalizeStringList(claim.Scope)
	claim.Metrics = normalizeStringList(claim.Metrics)
	claim.Outcomes = normalizeStringList(claim.Outcomes)
	if claim.ClaimText == "" {
		claim.ClaimText = claimLabelFromAtoms(claimAtoms{
			Actions:      claim.Actions,
			Capabilities: claim.Capabilities,
			Objects:      claim.Objects,
			Technologies: claim.Technologies,
			Domains:      claim.Domains,
			Artifacts:    claim.Artifacts,
			Scope:        claim.Scope,
		})
	}
	return claim
}

func normalizeClaimLabel(claim parsedCandidateClaim) string {
	label := firstNonEmpty(claim.Label, claim.ClaimText)
	label = strings.TrimSpace(strings.TrimSuffix(label, "."))
	label = regexp.MustCompile(`\s+`).ReplaceAllString(label, " ")
	if label == "" {
		return ""
	}
	metrics := extractMetricAtoms(label)
	words := strings.Fields(label)
	if len(words) > 9 {
		label = strings.Join(words[:9], " ")
	}
	if len(metrics) > 0 && !strings.Contains(label, metrics[0]) {
		label = strings.TrimSpace(label + " " + metrics[0])
	}
	return label
}

func claimLabelFromAtoms(atoms claimAtoms) string {
	parts := []string{}
	if len(atoms.Technologies) > 0 {
		parts = append(parts, strings.Join(limitStrings(atoms.Technologies, 3), "/"))
	}
	if len(atoms.Capabilities) > 0 {
		parts = append(parts, atoms.Capabilities[0])
	} else if len(atoms.Objects) > 0 {
		parts = append(parts, atoms.Objects[0])
	} else if len(atoms.Artifacts) > 0 {
		parts = append(parts, atoms.Artifacts[0])
	} else if len(atoms.Scope) > 0 {
		parts = append(parts, atoms.Scope[0])
	}
	if len(atoms.Domains) > 0 {
		parts = append(parts, atoms.Domains[0])
	}
	return normalizeClaimLabel(parsedCandidateClaim{Label: strings.Join(parts, " ")})
}

func splitAtomList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '|' || r == '\n'
	})
	result := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, ".-"))
		if part != "" {
			result = append(result, part)
		}
	}
	return normalizeStringList(result)
}

func extractMetricAtoms(text string) []string {
	matches := regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*(?:%|x|\b(?:users|requests|seconds|minutes|hours|days|ms|milliseconds|thousand|million|manual workload|workload)\b)`).FindAllString(text, -1)
	return normalizeStringList(matches)
}

func inferActionAtom(text string) string {
	lower := strings.ToLower(text)
	for _, action := range []string{"built", "shipped", "implemented", "designed", "developed", "created", "automated", "tested", "supported", "integrated", "optimized", "documented"} {
		if strings.Contains(lower, action) {
			return action
		}
	}
	return ""
}

func inferCapabilityAtom(text string) string {
	lower := strings.ToLower(text)
	for needle, label := range map[string]string{
		"backend":    "backend development",
		"api":        "API development",
		"postgres":   "database-backed systems",
		"test":       "testing",
		"automation": "automation",
		"rbac":       "access control",
		"audit":      "audit logging",
		"workflow":   "workflow systems",
		"frontend":   "frontend development",
	} {
		if strings.Contains(lower, needle) {
			return label
		}
	}
	return ""
}

func inferDomainAtom(text string) string {
	lower := strings.ToLower(text)
	for needle, label := range map[string]string{
		"construction": "construction planning",
		"planning":     "planning platform",
		"legal":        "legal technology",
		"finance":      "finance",
	} {
		if strings.Contains(lower, needle) {
			return label
		}
	}
	return ""
}

func inferEvidenceStrengthFromFact(fact factPromptContext) string {
	if fact.Confidence == "high" && fact.Status == factStatusApproved {
		return "direct"
	}
	if fact.Confidence == "low" || len(fact.RiskFlags) > 0 {
		return "weak"
	}
	return "inferred"
}

func limitStrings(values []string, limit int) []string {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func parseKeyValueFact(text string) map[string]string {
	result := map[string]string{}
	for _, part := range strings.Split(text, ";") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		result[strings.TrimSpace(strings.ToLower(key))] = strings.TrimSpace(value)
	}
	return result
}

func (c parsedCandidateClaim) SourceFactIDsKey() []string {
	keys := []string{}
	for _, id := range c.SourceFactIDs {
		keys = append(keys, fmt.Sprint(id))
	}
	return keys
}

func validClaimFacts(ids []int64, factsByID map[int64]factPromptContext) ([]int64, []factPromptContext) {
	validIDs := []int64{}
	facts := []factPromptContext{}
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		fact, ok := factsByID[id]
		if !ok || fact.Status == factStatusRejected {
			continue
		}
		seen[id] = true
		validIDs = append(validIDs, id)
		facts = append(facts, fact)
	}
	return validIDs, facts
}

func claimEvidenceContext(facts []factPromptContext) ([]string, []string, string, string) {
	quotes := []string{}
	tech := []string{}
	heading := ""
	sectionType := ""
	for _, fact := range facts {
		quotes = append(quotes, fact.EvidenceQuote)
		tech = append(tech, fact.Technologies...)
		if heading == "" {
			heading = fact.SectionHeading
		}
		if sectionType == "" {
			sectionType = fact.SectionType
		}
	}
	return normalizeStringList(quotes), normalizeStringList(tech), heading, sectionType
}

func verifyClaimRiskFlags(claim parsedCandidateClaim, facts []factPromptContext, blocked []BlockedClaim) []string {
	text := strings.Join([]string{
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
	}, " ")
	lower := strings.ToLower(text)
	evidence := strings.ToLower(joinFactEvidence(facts))
	flags := []string{}
	flags = append(flags, claim.RiskFlags...)
	flags = append(flags, styleRiskFlags(text)...)
	if containsMetric(text) && !containsMetric(evidence) {
		flags = append(flags, "unsupported_metric")
	}
	for _, tech := range extractTechnologies(text) {
		if !strings.Contains(evidence, strings.ToLower(tech)) {
			flags = append(flags, "unsupported_tool")
		}
	}
	for _, pattern := range []string{"senior", "staff", "lead", "managed", "mentored", "led a team", "team leadership", "kubernetes", "cloud infrastructure", "model training", "ml research"} {
		if strings.Contains(lower, pattern) && !strings.Contains(evidence, pattern) {
			flags = append(flags, "blocked_context")
			break
		}
	}
	for _, blockedClaim := range blocked {
		if !blockedClaim.Enabled {
			continue
		}
		if blockedClaimHits(lower, blockedClaim.Pattern) {
			flags = append(flags, "blocked_claim")
			break
		}
	}
	for _, fact := range facts {
		if fact.Status != factStatusApproved {
			flags = append(flags, "uses_"+fact.Status+"_fact")
		}
		flags = append(flags, fact.RiskFlags...)
	}
	return normalizeStringList(flags)
}

func joinFactEvidence(facts []factPromptContext) string {
	parts := []string{}
	for _, fact := range facts {
		parts = append(parts, fact.FactText, fact.EvidenceQuote, strings.Join(fact.Technologies, " "), strings.Join(fact.Context, " "))
	}
	return strings.Join(parts, " ")
}

func blockedClaimHits(lowerText string, pattern string) bool {
	pattern = strings.ToLower(pattern)
	switch pattern {
	case "senior/staff/lead title":
		return strings.Contains(lowerText, "senior") || strings.Contains(lowerText, "staff") || strings.Contains(lowerText, "lead ")
	case "invented metrics":
		return false
	default:
		for _, part := range strings.FieldsFunc(pattern, func(r rune) bool { return r == '/' || r == ',' }) {
			part = strings.TrimSpace(part)
			if len(part) > 3 && strings.Contains(lowerText, part) {
				return true
			}
		}
	}
	return false
}

func containsMetric(text string) bool {
	return regexp.MustCompile(`\d+[%x]?|\b\d+\s*(users|requests|seconds|minutes|hours|days|ms|millions|thousand)\b`).MatchString(strings.ToLower(text))
}

func claimStatusForFacts(facts []factPromptContext, riskFlags []string) string {
	for _, flag := range riskFlags {
		if flag == "blocked_claim" || flag == "blocked_context" || flag == "unsupported_metric" || flag == "unsupported_tool" {
			return claimStatusBlocked
		}
	}
	for _, fact := range facts {
		if fact.Status != factStatusApproved {
			return claimStatusNeedsReview
		}
	}
	if len(riskFlags) > 0 {
		return claimStatusApprovedRestricted
	}
	return claimStatusApproved
}

func inferClaimStrength(facts []factPromptContext) string {
	for _, fact := range facts {
		switch fact.SectionType {
		case "experience":
			if fact.Status == factStatusApproved && fact.Confidence == "high" {
				return "strong"
			}
			return "moderate"
		case "project":
			return "moderate"
		}
	}
	return "weak"
}

func defaultAllowedUse(values []string, originType string) []string {
	if len(values) > 0 {
		return normalizeStringList(values)
	}
	switch normalizeClaimType(originType) {
	case "skills":
		return []string{"skills", "summary"}
	case "education":
		return []string{"education", "summary"}
	default:
		return []string{"experience_bullet", "summary", "skills"}
	}
}

func defaultAllowedContexts(values []string, facts []factPromptContext) []string {
	if len(values) > 0 {
		return normalizeStringList(values)
	}
	contexts := []string{}
	for _, fact := range facts {
		contexts = append(contexts, claimContextsFromFact(fact)...)
	}
	return normalizeStringList(contexts)
}

func claimContextsFromFact(fact factPromptContext) []string {
	contexts := []string{}
	text := strings.ToLower(strings.Join([]string{fact.FactText, fact.EvidenceQuote, strings.Join(fact.Technologies, " ")}, " "))
	for _, context := range []string{"backend engineering", "api design", "product engineering", "testing", "data systems", "ai workflow", "frontend engineering", "database design"} {
		needle := strings.Split(context, " ")[0]
		if strings.Contains(text, needle) || strings.Contains(strings.ToLower(fact.SectionHeading), needle) {
			contexts = append(contexts, context)
		}
	}
	if len(contexts) == 0 && fact.SectionType != "" {
		contexts = append(contexts, fact.SectionType)
	}
	return contexts
}

func defaultBlockedContexts(values []string, claimText string) []string {
	contexts := append([]string{}, values...)
	lower := strings.ToLower(claimText)
	for _, pair := range map[string]string{
		"machine learning":     "ML engineering unless evidenced",
		"model training":       "model training unless evidenced",
		"kubernetes":           "Kubernetes ownership unless evidenced",
		"cloud infrastructure": "cloud infrastructure ownership unless evidenced",
		"senior":               "senior/staff/lead title unless evidenced",
		"staff":                "senior/staff/lead title unless evidenced",
		"team leadership":      "team leadership unless evidenced",
		"managed":              "team leadership unless evidenced",
		"led a team":           "team leadership unless evidenced",
		"production at scale":  "large-scale production scope unless evidenced",
		"millions":             "large-scale metric unless evidenced",
	} {
		if strings.Contains(lower, pair) {
			contexts = append(contexts, pair)
		}
	}
	return normalizeStringList(contexts)
}

func defaultSafePhrasings(values []string, claimText string) []string {
	if len(values) > 0 {
		return normalizeStringList(values)
	}
	return normalizeStringList([]string{claimText})
}

func defaultUnsafePhrasings(values []string, claimText string, riskFlags []string) []string {
	unsafe := append([]string{}, values...)
	if listContains(riskFlags, "blocked_context") || listContains(riskFlags, "blocked_claim") {
		unsafe = append(unsafe, "inflated seniority, ownership, or unsupported specialization")
	}
	if listContains(riskFlags, "unsupported_metric") {
		unsafe = append(unsafe, "metric phrasing not present in source evidence")
	}
	if listContains(riskFlags, "unsupported_tool") {
		unsafe = append(unsafe, "tool phrasing not present in source evidence")
	}
	return normalizeStringList(unsafe)
}

func listContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func compactFactsForClaims(facts []factPromptContext) []factPromptContext {
	selected := []factPromptContext{}
	for _, fact := range facts {
		if fact.Status == factStatusRejected {
			continue
		}
		selected = append(selected, compactFactPromptContext(fact))
		if len(selected) >= 120 {
			break
		}
	}
	return selected
}

func claimTechnologies(c parsedCandidateClaim) []string {
	return normalizeStringList(append(c.Technologies, extractTechnologies(c.ClaimText)...))
}

func firstNonEmptyList(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func normalizeClaimStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case claimStatusApproved, claimStatusApprovedRestricted, claimStatusNeedsReview, claimStatusRejected, claimStatusBlocked:
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return claimStatusNeedsReview
	}
}

func normalizeClaimType(claimType string) string {
	switch strings.TrimSpace(strings.ToLower(claimType)) {
	case "experience", "project", "skills", "education", "summary":
		return strings.TrimSpace(strings.ToLower(claimType))
	default:
		return "experience"
	}
}

func normalizeClaimStrength(strength string) string {
	switch strings.TrimSpace(strings.ToLower(strength)) {
	case "strong", "moderate", "weak":
		return strings.TrimSpace(strings.ToLower(strength))
	default:
		return "moderate"
	}
}

func normalizeEvidenceStrength(strength string) string {
	switch strings.TrimSpace(strings.ToLower(strength)) {
	case "direct", "inferred", "weak":
		return strings.TrimSpace(strings.ToLower(strength))
	default:
		return "direct"
	}
}

func normalizeSeverity(severity string) string {
	switch strings.TrimSpace(strings.ToLower(severity)) {
	case "low", "medium", "high":
		return strings.TrimSpace(strings.ToLower(severity))
	default:
		return "medium"
	}
}

func (s *Store) getCandidateClaim(id int64) (CandidateClaim, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, claim_text, claim_type, source_fact_ids_json, evidence_quotes_json, technologies_json,
		strength, allowed_use_json, allowed_contexts_json, blocked_contexts_json, safe_phrasings_json,
		unsafe_phrasings_json, origin_heading, origin_type, status, risk_flags_json, review_note, created_at, updated_at,
		actions_json, capabilities_json, objects_json, domains_json, artifacts_json, scope_json, metrics_json,
		outcomes_json, profile_context_json, evidence_strength, similarity_key, similarity_score, duplicate_of_id
		FROM candidate_claims WHERE id = ?`,
		id,
	)
	if err != nil {
		return CandidateClaim{}, err
	}
	defer rows.Close()
	claims, err := scanCandidateClaims(rows)
	if err != nil {
		return CandidateClaim{}, err
	}
	if len(claims) == 0 {
		return CandidateClaim{}, sql.ErrNoRows
	}
	return claims[0], nil
}

func (s *Store) getBlockedClaim(id int64) (BlockedClaim, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, pattern, reason, severity, source, enabled, created_at, updated_at FROM blocked_claims WHERE id = ?`, id)
	if err != nil {
		return BlockedClaim{}, err
	}
	defer rows.Close()
	blocked, err := scanBlockedClaims(rows)
	if err != nil {
		return BlockedClaim{}, err
	}
	if len(blocked) == 0 {
		return BlockedClaim{}, sql.ErrNoRows
	}
	return blocked[0], nil
}

func scanCandidateClaims(rows *sql.Rows) ([]CandidateClaim, error) {
	claims := []CandidateClaim{}
	for rows.Next() {
		var claim CandidateClaim
		var sourceFactJSON, evidenceJSON, techJSON, allowedUseJSON, allowedContextsJSON, blockedContextsJSON, safeJSON, unsafeJSON, riskJSON string
		var actionsJSON, capabilitiesJSON, objectsJSON, domainsJSON, artifactsJSON, scopeJSON, metricsJSON, outcomesJSON, profileContextJSON string
		if err := rows.Scan(
			&claim.ID,
			&claim.ClaimText,
			&claim.ClaimType,
			&sourceFactJSON,
			&evidenceJSON,
			&techJSON,
			&claim.Strength,
			&allowedUseJSON,
			&allowedContextsJSON,
			&blockedContextsJSON,
			&safeJSON,
			&unsafeJSON,
			&claim.OriginHeading,
			&claim.OriginType,
			&claim.Status,
			&riskJSON,
			&claim.ReviewNote,
			&claim.CreatedAt,
			&claim.UpdatedAt,
			&actionsJSON,
			&capabilitiesJSON,
			&objectsJSON,
			&domainsJSON,
			&artifactsJSON,
			&scopeJSON,
			&metricsJSON,
			&outcomesJSON,
			&profileContextJSON,
			&claim.EvidenceStrength,
			&claim.SimilarityKey,
			&claim.SimilarityScore,
			&claim.DuplicateOfID,
		); err != nil {
			return nil, err
		}
		claim.SourceFactIDs = decodeInt64List(sourceFactJSON)
		claim.EvidenceQuotes = decodeStringList(evidenceJSON)
		claim.Technologies = decodeStringList(techJSON)
		claim.Actions = decodeStringList(actionsJSON)
		claim.Capabilities = decodeStringList(capabilitiesJSON)
		claim.Objects = decodeStringList(objectsJSON)
		claim.Domains = decodeStringList(domainsJSON)
		claim.Artifacts = decodeStringList(artifactsJSON)
		claim.Scope = decodeStringList(scopeJSON)
		claim.Metrics = decodeStringList(metricsJSON)
		claim.Outcomes = decodeStringList(outcomesJSON)
		claim.ProfileContext = decodeStringList(profileContextJSON)
		claim.EvidenceStrength = normalizeEvidenceStrength(claim.EvidenceStrength)
		claim.AllowedUse = decodeStringList(allowedUseJSON)
		claim.AllowedContexts = decodeStringList(allowedContextsJSON)
		claim.BlockedContexts = decodeStringList(blockedContextsJSON)
		claim.SafePhrasings = decodeStringList(safeJSON)
		claim.UnsafePhrasings = decodeStringList(unsafeJSON)
		claim.RiskFlags = decodeStringList(riskJSON)
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func scanBlockedClaims(rows *sql.Rows) ([]BlockedClaim, error) {
	blocked := []BlockedClaim{}
	for rows.Next() {
		var item BlockedClaim
		var enabled int
		if err := rows.Scan(&item.ID, &item.Pattern, &item.Reason, &item.Severity, &item.Source, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Enabled = intToBool(enabled)
		blocked = append(blocked, item)
	}
	return blocked, rows.Err()
}
