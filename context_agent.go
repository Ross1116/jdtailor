package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	contextAgentStatusRunning   = "running"
	contextAgentStatusComplete  = "complete"
	contextAgentStatusFailed    = "failed"
	contextAgentStatusCancelled = "cancelled"
)

type ContextAgentRun struct {
	ID            int64  `json:"id"`
	SourceID      int64  `json:"source_id"`
	Status        string `json:"status"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	Error         string `json:"error"`
	FactsCreated  int    `json:"facts_created"`
	ClaimsCreated int    `json:"claims_created"`
}

type ContextAgentStep struct {
	ID        int64  `json:"id"`
	RunID     int64  `json:"run_id"`
	Stage     string `json:"stage"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type ResumeContext struct {
	SourceID int64                 `json:"source_id"`
	Origins  []ResumeContextOrigin `json:"origins"`
}

type ResumeContextOrigin struct {
	OriginHeading string               `json:"origin_heading"`
	OriginType    string               `json:"origin_type"`
	Facts         []ResumeContextFact  `json:"facts"`
	Claims        []ResumeContextClaim `json:"claims"`
	Keywords      []string             `json:"keywords"`
	RiskFlags     []string             `json:"risk_flags"`
}

type ResumeContextFact struct {
	ID            int64    `json:"id"`
	Atoms         string   `json:"atoms"`
	Technologies  []string `json:"technologies"`
	Status        string   `json:"status"`
	RiskFlags     []string `json:"risk_flags"`
	EvidenceQuote string   `json:"evidence_quote"`
}

type ResumeContextClaim struct {
	ID               int64    `json:"id"`
	Label            string   `json:"label"`
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
	EvidenceStrength string   `json:"evidence_strength"`
	Status           string   `json:"status"`
	RiskFlags        []string `json:"risk_flags"`
}

func (s *Store) StartContextAgent(sourceID int64) (ContextAgentRun, error) {
	if sourceID <= 0 {
		return ContextAgentRun{}, errors.New("source id is required")
	}
	if _, err := s.getCandidateSource(sourceID); err != nil {
		return ContextAgentRun{}, err
	}
	active, err := s.activeContextAgentRun(sourceID)
	if err != nil {
		return ContextAgentRun{}, err
	}
	if active.ID > 0 {
		return active, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO context_agent_runs (source_id, status, started_at) VALUES (?, ?, ?)`,
		sourceID,
		contextAgentStatusRunning,
		now,
	)
	if err != nil {
		return ContextAgentRun{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return ContextAgentRun{}, err
	}
	_ = s.recordContextAgentStep(id, "queued", "ok", "context agent started")
	_ = s.LogEvent("info", "context agent started")
	return s.GetContextAgentRun(id)
}

func (s *Store) RunContextAgent(ctx context.Context, runID int64, client *http.Client) (ContextAgentRun, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	run, err := s.GetContextAgentRun(runID)
	if err != nil {
		return ContextAgentRun{}, err
	}
	if run.Status != contextAgentStatusRunning {
		return run, nil
	}

	factsCreated := run.FactsCreated
	claimsCreated := run.ClaimsCreated

	fail := func(stage string, err error) (ContextAgentRun, error) {
		if errors.Is(err, context.Canceled) {
			return s.cancelContextAgentFromWorker(runID, stage, factsCreated, claimsCreated)
		}

		_ = s.recordContextAgentStep(runID, stage, "failed", err.Error())
		_ = s.finishContextAgentRun(runID, contextAgentStatusFailed, err.Error(), factsCreated, claimsCreated)
		_ = s.LogEvent("error", "context agent failed: "+err.Error())

		finished, getErr := s.GetContextAgentRun(runID)
		if getErr != nil {
			return ContextAgentRun{}, getErr
		}
		return finished, err
	}

	checkCancelled := func(stage string) (ContextAgentRun, bool, error) {
		if ctx.Err() != nil {
			cancelled, cancelErr := s.cancelContextAgentFromWorker(runID, stage, factsCreated, claimsCreated)
			return cancelled, true, cancelErr
		}

		current, err := s.GetContextAgentRun(runID)
		if err != nil {
			return ContextAgentRun{}, false, err
		}
		if current.Status == contextAgentStatusCancelled {
			return current, true, context.Canceled
		}

		return ContextAgentRun{}, false, nil
	}

	if cancelled, stopped, err := checkCancelled("source_preprocess"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "source_preprocess", "ok", "source normalized")

	sections, err := s.DetectSourceSections(run.SourceID)
	if err != nil {
		return fail("section_detect", err)
	}

	if cancelled, stopped, err := checkCancelled("section_detect"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "section_detect", "ok", pluralCount(len(sections), "section"))

	for _, section := range sections {
		if cancelled, stopped, err := checkCancelled("fact_extract"); stopped || err != nil {
			return cancelled, err
		}

		facts, err := s.ExtractEvidenceFacts(ctx, ExtractEvidenceFactsInput{
			SourceID:  run.SourceID,
			SectionID: section.ID,
		}, client)
		if err != nil {
			return fail("fact_extract", err)
		}

		factsCreated += len(facts)

		if cancelled, stopped, err := checkCancelled("fact_extract"); stopped || err != nil {
			return cancelled, err
		}

		_ = s.recordContextAgentStep(runID, "fact_extract", "ok", section.Heading+": "+pluralCount(len(facts), "fact"))
	}

	if cancelled, stopped, err := checkCancelled("fact_compact"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "fact_compact", "ok", pluralCount(factsCreated, "fact")+" compacted")

	if cancelled, stopped, err := checkCancelled("profile_draft"); stopped || err != nil {
		return cancelled, err
	}

	profile, err := s.DraftCandidateProfileFromSource(run.SourceID)
	if err != nil {
		return fail("profile_draft", err)
	}
	if err := s.mergeAndSaveCandidateProfile(profile); err != nil {
		return fail("profile_draft", err)
	}

	if cancelled, stopped, err := checkCancelled("profile_draft"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "profile_draft", "ok", "candidate profile draft merged")

	if cancelled, stopped, err := checkCancelled("claim_generate"); stopped || err != nil {
		return cancelled, err
	}

	claims, err := s.GenerateCandidateClaims(ctx, client)
	if err != nil {
		return fail("claim_generate", err)
	}

	claimsCreated = len(claims)

	if cancelled, stopped, err := checkCancelled("claim_generate"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "claim_generate", "ok", pluralCount(claimsCreated, "claim"))

	if cancelled, stopped, err := checkCancelled("claim_compact"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "claim_compact", "ok", "claim atoms compacted")

	if cancelled, stopped, err := checkCancelled("dedupe"); stopped || err != nil {
		return cancelled, err
	}

	_ = s.recordContextAgentStep(runID, "dedupe", "ok", "similarity keys applied")

	if cancelled, stopped, err := checkCancelled("done"); stopped || err != nil {
		return cancelled, err
	}

	if err := s.finishContextAgentRun(runID, contextAgentStatusComplete, "", factsCreated, claimsCreated); err != nil {
		return ContextAgentRun{}, err
	}

	_ = s.recordContextAgentStep(runID, "done", "ok", "context agent complete")
	_ = s.LogEvent("info", "context agent complete")

	return s.GetContextAgentRun(runID)
}

func (s *Store) ListContextAgentRuns(sourceID int64) ([]ContextAgentRun, error) {
	query := `SELECT id, source_id, status, started_at, finished_at, error, facts_created, claims_created FROM context_agent_runs`
	args := []any{}
	if sourceID > 0 {
		query += ` WHERE source_id = ?`
		args = append(args, sourceID)
	}
	query += ` ORDER BY id DESC LIMIT 50`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContextAgentRuns(rows)
}

func (s *Store) GetContextAgentRun(runID int64) (ContextAgentRun, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_id, status, started_at, finished_at, error, facts_created, claims_created
		FROM context_agent_runs WHERE id = ?`,
		runID,
	)
	if err != nil {
		return ContextAgentRun{}, err
	}
	defer rows.Close()
	runs, err := scanContextAgentRuns(rows)
	if err != nil {
		return ContextAgentRun{}, err
	}
	if len(runs) == 0 {
		return ContextAgentRun{}, sql.ErrNoRows
	}
	return runs[0], nil
}

func (s *Store) CancelContextAgentRun(runID int64, message string) (ContextAgentRun, error) {
	if runID <= 0 {
		return ContextAgentRun{}, errors.New("run id is required")
	}

	run, err := s.GetContextAgentRun(runID)
	if err != nil {
		return ContextAgentRun{}, err
	}

	if run.Status != contextAgentStatusRunning {
		return run, nil
	}

	message = strings.TrimSpace(message)
	if message == "" {
		message = "cancelled"
	}

	_ = s.recordContextAgentStep(runID, "cancelled", "cancelled", message)

	if err := s.finishContextAgentRun(
		runID,
		contextAgentStatusCancelled,
		message,
		run.FactsCreated,
		run.ClaimsCreated,
	); err != nil {
		return ContextAgentRun{}, err
	}

	_ = s.LogEvent("info", "context agent cancelled")

	return s.GetContextAgentRun(runID)
}

func (s *Store) cancelContextAgentFromWorker(runID int64, stage string, factsCreated int, claimsCreated int) (ContextAgentRun, error) {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = "cancelled"
	}

	_ = s.recordContextAgentStep(runID, stage, "cancelled", "context agent cancelled")

	if err := s.finishContextAgentRun(
		runID,
		contextAgentStatusCancelled,
		"context agent cancelled",
		factsCreated,
		claimsCreated,
	); err != nil {
		return ContextAgentRun{}, err
	}

	_ = s.LogEvent("info", "context agent cancelled")

	finished, err := s.GetContextAgentRun(runID)
	if err != nil {
		return ContextAgentRun{}, err
	}

	return finished, context.Canceled
}

func (s *Store) ListContextAgentSteps(runID int64) ([]ContextAgentStep, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, run_id, stage, status, message, created_at
		FROM context_agent_steps WHERE run_id = ? ORDER BY id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	steps := []ContextAgentStep{}
	for rows.Next() {
		var step ContextAgentStep
		if err := rows.Scan(&step.ID, &step.RunID, &step.Stage, &step.Status, &step.Message, &step.CreatedAt); err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (s *Store) BuildResumeContext(sourceID int64) (ResumeContext, error) {
	facts, err := s.listResumeContextFacts(sourceID)
	if err != nil {
		return ResumeContext{}, err
	}
	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return ResumeContext{}, err
	}

	factByID := map[int64]ResumeContextFact{}
	factSourceIDs := map[int64]bool{}
	origins := map[string]*ResumeContextOrigin{}

	for _, fact := range facts {
		if fact.Status == factStatusRejected || fact.DuplicateOfID > 0 {
			continue
		}
		key := originKey(fact.OriginHeading, fact.OriginType)
		origin := origins[key]
		if origin == nil {
			origin = &ResumeContextOrigin{OriginHeading: fact.OriginHeading, OriginType: fact.OriginType}
			origins[key] = origin
		}
		item := ResumeContextFact{
			ID:            fact.ID,
			Atoms:         compactFactAtoms(fact.FactText),
			Technologies:  normalizeStringList(fact.Technologies),
			Status:        fact.Status,
			RiskFlags:     normalizeStringList(fact.RiskFlags),
			EvidenceQuote: strings.TrimSpace(fact.EvidenceQuote),
		}
		origin.Facts = append(origin.Facts, item)
		origin.Keywords = normalizeStringList(append(origin.Keywords, fact.Technologies...))
		origin.RiskFlags = normalizeStringList(append(origin.RiskFlags, fact.RiskFlags...))
		factByID[fact.ID] = item
		factSourceIDs[fact.ID] = true
	}

	for _, claim := range claims {
		if claim.Status == claimStatusRejected || claim.Status == claimStatusBlocked {
			continue
		}
		if sourceID > 0 && !claimIntersectsFacts(claim, factSourceIDs) {
			continue
		}
		key := originKey(claim.OriginHeading, claim.OriginType)
		origin := origins[key]
		if origin == nil {
			origin = &ResumeContextOrigin{OriginHeading: claim.OriginHeading, OriginType: claim.OriginType}
			origins[key] = origin
		}
		origin.Claims = append(origin.Claims, ResumeContextClaim{
			ID:               claim.ID,
			Label:            strings.TrimSpace(claim.ClaimText),
			SourceFactIDs:    uniqueInt64s(claim.SourceFactIDs),
			Actions:          normalizeStringList(claim.Actions),
			Capabilities:     normalizeStringList(claim.Capabilities),
			Objects:          normalizeStringList(claim.Objects),
			Technologies:     normalizeStringList(claim.Technologies),
			Domains:          normalizeStringList(claim.Domains),
			Artifacts:        normalizeStringList(claim.Artifacts),
			Scope:            normalizeStringList(claim.Scope),
			Metrics:          normalizeStringList(claim.Metrics),
			Outcomes:         normalizeStringList(claim.Outcomes),
			EvidenceStrength: claim.EvidenceStrength,
			Status:           claim.Status,
			RiskFlags:        normalizeStringList(claim.RiskFlags),
		})
		origin.Keywords = normalizeStringList(append(origin.Keywords, claim.Technologies...))
		origin.RiskFlags = normalizeStringList(append(origin.RiskFlags, claim.RiskFlags...))
		for _, factID := range claim.SourceFactIDs {
			if fact, ok := factByID[factID]; ok {
				origin.Keywords = normalizeStringList(append(origin.Keywords, fact.Technologies...))
			}
		}
	}

	result := ResumeContext{SourceID: sourceID}
	for _, origin := range origins {
		sort.Slice(origin.Facts, func(i, j int) bool { return origin.Facts[i].ID < origin.Facts[j].ID })
		sort.Slice(origin.Claims, func(i, j int) bool { return origin.Claims[i].ID < origin.Claims[j].ID })
		result.Origins = append(result.Origins, *origin)
	}
	sort.Slice(result.Origins, func(i, j int) bool {
		if result.Origins[i].OriginType != result.Origins[j].OriginType {
			return result.Origins[i].OriginType < result.Origins[j].OriginType
		}
		return result.Origins[i].OriginHeading < result.Origins[j].OriginHeading
	})
	return result, nil
}

func (s *Store) activeContextAgentRun(sourceID int64) (ContextAgentRun, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, source_id, status, started_at, finished_at, error, facts_created, claims_created
		FROM context_agent_runs WHERE source_id = ? AND status = ? ORDER BY id DESC LIMIT 1`,
		sourceID,
		contextAgentStatusRunning,
	)
	if err != nil {
		return ContextAgentRun{}, err
	}
	defer rows.Close()
	runs, err := scanContextAgentRuns(rows)
	if err != nil {
		return ContextAgentRun{}, err
	}
	if len(runs) == 0 {
		return ContextAgentRun{}, nil
	}
	return runs[0], nil
}

func (s *Store) recordContextAgentStep(runID int64, stage, status, message string) error {
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO context_agent_steps (run_id, stage, status, message, created_at) VALUES (?, ?, ?, ?, ?)`,
		runID,
		strings.TrimSpace(stage),
		strings.TrimSpace(status),
		strings.TrimSpace(message),
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) finishContextAgentRun(runID int64, status, message string, factsCreated int, claimsCreated int) error {
	if status == contextAgentStatusCancelled {
		_, err := s.db.ExecContext(
			context.Background(),
			`UPDATE context_agent_runs
			SET status = ?, finished_at = ?, error = ?, facts_created = ?, claims_created = ?
			WHERE id = ?`,
			status,
			time.Now().UTC().Format(time.RFC3339),
			strings.TrimSpace(message),
			factsCreated,
			claimsCreated,
			runID,
		)
		return err
	}

	_, err := s.db.ExecContext(
		context.Background(),
		`UPDATE context_agent_runs
		SET status = ?, finished_at = ?, error = ?, facts_created = ?, claims_created = ?
		WHERE id = ? AND status != ?`,
		status,
		time.Now().UTC().Format(time.RFC3339),
		strings.TrimSpace(message),
		factsCreated,
		claimsCreated,
		runID,
		contextAgentStatusCancelled,
	)
	return err
}

func (s *Store) mergeAndSaveCandidateProfile(draft CandidateProfile) error {
	current, err := s.GetCandidateProfile()
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.Contact.FullName) == "" {
		current.Contact.FullName = draft.Contact.FullName
	}
	if strings.TrimSpace(current.Contact.Email) == "" {
		current.Contact.Email = draft.Contact.Email
	}
	if strings.TrimSpace(current.Contact.Phone) == "" {
		current.Contact.Phone = draft.Contact.Phone
	}
	if strings.TrimSpace(current.Contact.Location) == "" {
		current.Contact.Location = draft.Contact.Location
	}
	if strings.TrimSpace(current.Contact.LinkedIn) == "" {
		current.Contact.LinkedIn = draft.Contact.LinkedIn
	}
	if strings.TrimSpace(current.Contact.GitHub) == "" {
		current.Contact.GitHub = draft.Contact.GitHub
	}
	if strings.TrimSpace(current.Contact.Portfolio) == "" {
		current.Contact.Portfolio = draft.Contact.Portfolio
	}
	current.Contact.Links = normalizeStringList(append(current.Contact.Links, draft.Contact.Links...))
	if !current.Contact.Verified {
		current.Contact.Verified = draft.Contact.Verified
	}

	seen := map[string]bool{}
	records := []CandidateProfileRecord{}
	for _, record := range append(current.Records, draft.Records...) {
		key := strings.ToLower(strings.Join([]string{
			record.RecordType,
			record.Label,
			record.Organization,
			record.Role,
			record.StartDate,
			record.EndDate,
			record.Value,
		}, "|"))
		if seen[key] {
			continue
		}
		seen[key] = true
		record.Verified = false
		records = append(records, record)
	}
	current.Records = records
	_, err = s.SaveCandidateProfile(current)
	return err
}

func (s *Store) listResumeContextFacts(sourceID int64) ([]EvidenceFact, error) {
	query := `SELECT id, source_id, section_id, fact_text, evidence_quote, technologies_json, confidence, risk_flags_json, origin_heading, origin_type, context_json, status, auto_approved, similarity_key, similarity_score, duplicate_of_id, review_note, created_at, updated_at FROM evidence_facts`
	args := []any{}
	if sourceID > 0 {
		query += ` WHERE source_id = ?`
		args = append(args, sourceID)
	}
	query += ` ORDER BY source_id DESC, origin_type, origin_heading, id`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvidenceFacts(rows)
}

func scanContextAgentRuns(rows *sql.Rows) ([]ContextAgentRun, error) {
	runs := []ContextAgentRun{}
	for rows.Next() {
		var run ContextAgentRun
		if err := rows.Scan(&run.ID, &run.SourceID, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Error, &run.FactsCreated, &run.ClaimsCreated); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func claimIntersectsFacts(claim CandidateClaim, factIDs map[int64]bool) bool {
	for _, factID := range claim.SourceFactIDs {
		if factIDs[factID] {
			return true
		}
	}
	return false
}

func compactFactAtoms(value string) string {
	parts := parseKeyValueFact(value)
	if len(parts) == 0 {
		return compactCoreEvidenceFact(value)
	}
	order := []string{"actions", "action", "artifact", "artifacts", "objects", "object", "tools", "scope", "metric", "outcome", "domain"}
	atoms := []string{}
	seen := map[string]bool{}
	for _, key := range order {
		raw := strings.TrimSpace(parts[key])
		if raw == "" {
			continue
		}
		canonical := key
		if canonical == "action" {
			canonical = "actions"
		}
		if canonical == "artifacts" {
			canonical = "artifact"
		}
		if canonical == "objects" || canonical == "object" {
			canonical = "artifact"
		}
		if seen[canonical+"="+raw] {
			continue
		}
		seen[canonical+"="+raw] = true
		atoms = append(atoms, canonical+"="+raw)
	}
	if len(atoms) == 0 {
		return compactCoreEvidenceFact(value)
	}
	return strings.Join(atoms, "; ")
}

func pluralCount(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}
