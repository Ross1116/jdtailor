package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type JobAnalysis struct {
	JobID            int64    `json:"job_id"`
	Company          string   `json:"company"`
	RoleTitle        string   `json:"role_title"`
	Location         string   `json:"location"`
	WorkArrangement  string   `json:"work_arrangement"`
	Salary           string   `json:"salary"`
	TopPainPoints    []string `json:"top_pain_points"`
	RequiredSkills   []string `json:"required_skills"`
	PreferredSkills  []string `json:"preferred_skills"`
	Responsibilities []string `json:"responsibilities"`
	SeniorityLevel   string   `json:"seniority_level"`
	RoleArchetype    string   `json:"role_archetype"`
	Keywords         []string `json:"keywords"`
	RiskFlags        []string `json:"risk_flags"`
	JobPoster        string   `json:"job_poster"`
	CompanyURL       string   `json:"company_url"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type FitNeedAnalysis struct {
	RequirementID    int64   `json:"requirement_id"`
	JDNeed           string  `json:"jd_need"`
	MatchingFactIDs  []int64 `json:"matching_fact_ids"`
	EvidenceStrength string  `json:"evidence_strength"`
	GapLevel         string  `json:"gap_level"`
	Confidence       string  `json:"confidence"`
	Risk             string  `json:"risk"`
}

type JobFitAnalysis struct {
	JobID          int64             `json:"job_id"`
	OverallScore   int               `json:"overall_score"`
	Recommendation string            `json:"recommendation"`
	Strengths      []string          `json:"strengths"`
	CriticalGaps   []string          `json:"critical_gaps"`
	RealityCheck   string            `json:"reality_check"`
	Analysis       []FitNeedAnalysis `json:"analysis"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

type ApplicationStrategy struct {
	JobID               int64             `json:"job_id"`
	ApprovedFactIDs     []int64           `json:"approved_fact_ids"`
	RejectedFactIDs     []int64           `json:"rejected_fact_ids"`
	WeakOrMissing       []string          `json:"weak_or_missing_requirements"`
	ResumeHeadline      string            `json:"resume_headline"`
	ExperienceTitles    map[string]string `json:"experience_titles"`
	PositioningStrategy string            `json:"positioning_strategy"`
	Keywords            []string          `json:"keywords"`
	DoNotOverclaim      []string          `json:"do_not_overclaim"`
	FitSummary          string            `json:"fit_summary"`
	CreatedAt           string            `json:"created_at"`
	UpdatedAt           string            `json:"updated_at"`
}

func (s *Store) AnalyzeJobDescription(jobID int64) (JobAnalysis, error) {
	job, err := s.getJobDescription(jobID)
	if err != nil {
		return JobAnalysis{}, err
	}
	requirements, err := s.ListJobRequirements(jobID)
	if err != nil {
		return JobAnalysis{}, err
	}
	if len(requirements) == 0 {
		requirements, err = s.ParseJobDescription(context.Background(), jobID, nil)
		if err != nil {
			return JobAnalysis{}, err
		}
	}
	analysis := buildJobAnalysis(job, requirements)
	if err := s.replaceJobAnalysis(analysis); err != nil {
		return JobAnalysis{}, err
	}
	return s.GetJobAnalysis(jobID)
}

func (s *Store) GetJobAnalysis(jobID int64) (JobAnalysis, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT job_id, company, role_title, location, work_arrangement, salary, top_pain_points_json,
			required_skills_json, preferred_skills_json, responsibilities_json, seniority_level, role_archetype,
			keywords_json, risk_flags_json, job_poster, company_url, created_at, updated_at
		FROM job_analyses WHERE job_id = ?`,
		jobID,
	)
	if err != nil {
		return JobAnalysis{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return JobAnalysis{}, sql.ErrNoRows
	}
	return scanJobAnalysisRow(rows)
}

func (s *Store) GenerateFitAnalysis(jobID int64) (JobFitAnalysis, error) {
	requirements, err := s.ListJobRequirements(jobID)
	if err != nil {
		return JobFitAnalysis{}, err
	}
	if len(requirements) == 0 {
		return JobFitAnalysis{}, errors.New("parse job requirements before fit analysis")
	}
	matches, err := s.ListJobFactMatches(jobID)
	if err != nil {
		return JobFitAnalysis{}, err
	}
	fit := buildFitAnalysis(jobID, requirements, matches)
	if err := s.replaceFitAnalysis(fit); err != nil {
		return JobFitAnalysis{}, err
	}
	return s.GetFitAnalysis(jobID)
}

func (s *Store) GetFitAnalysis(jobID int64) (JobFitAnalysis, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT job_id, overall_score, recommendation, strengths_json, critical_gaps_json, reality_check,
			analysis_json, created_at, updated_at FROM job_fit_analyses WHERE job_id = ?`,
		jobID,
	)
	if err != nil {
		return JobFitAnalysis{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return JobFitAnalysis{}, sql.ErrNoRows
	}
	return scanFitAnalysisRow(rows)
}

func (s *Store) GenerateApplicationStrategy(jobID int64) (ApplicationStrategy, error) {
	job, err := s.getJobDescription(jobID)
	if err != nil {
		return ApplicationStrategy{}, err
	}
	analysis, err := s.GetJobAnalysis(jobID)
	if errors.Is(err, sql.ErrNoRows) {
		analysis, err = s.AnalyzeJobDescription(jobID)
	}
	if err != nil {
		return ApplicationStrategy{}, err
	}
	fit, err := s.GetFitAnalysis(jobID)
	if errors.Is(err, sql.ErrNoRows) {
		fit, err = s.GenerateFitAnalysis(jobID)
	}
	if err != nil {
		return ApplicationStrategy{}, err
	}
	matches, err := s.ListJobFactMatches(jobID)
	if err != nil {
		return ApplicationStrategy{}, err
	}
	strategy := buildApplicationStrategy(job, analysis, fit, matches)
	if err := s.replaceApplicationStrategy(strategy); err != nil {
		return ApplicationStrategy{}, err
	}
	return s.GetApplicationStrategy(jobID)
}

func (s *Store) GetApplicationStrategy(jobID int64) (ApplicationStrategy, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT job_id, approved_fact_ids_json, rejected_fact_ids_json, weak_or_missing_json, resume_headline,
			experience_titles_json, positioning_strategy, keywords_json, do_not_overclaim_json, fit_summary,
			created_at, updated_at FROM application_strategies WHERE job_id = ?`,
		jobID,
	)
	if err != nil {
		return ApplicationStrategy{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return ApplicationStrategy{}, sql.ErrNoRows
	}
	return scanApplicationStrategyRow(rows)
}

func buildJobAnalysis(job JobDescription, requirements []JobRequirement) JobAnalysis {
	required := []string{}
	preferred := []string{}
	responsibilities := []string{}
	keywords := []string{}
	riskFlags := []string{}
	for _, req := range requirements {
		keywords = append(keywords, req.Keywords...)
		switch req.Category {
		case "nice_to_have":
			preferred = append(preferred, req.RequirementText)
		case "responsibility":
			responsibilities = append(responsibilities, req.RequirementText)
		default:
			if req.Priority == "high" || req.Category == "must_have" {
				required = append(required, req.RequirementText)
			}
		}
		if req.Category == "domain" {
			riskFlags = append(riskFlags, "domain_requirement")
		}
	}
	painPoints := []string{}
	for _, req := range requirements {
		if requirementCanBeTopPainPoint(req) {
			painPoints = append(painPoints, req.RequirementText)
		}
	}
	if len(painPoints) == 0 {
		painPoints = append(responsibilities, required...)
	}
	painPoints = preserveStringListOrder(painPoints)
	if len(painPoints) > 3 {
		painPoints = painPoints[:3]
	}
	return JobAnalysis{
		JobID:            job.ID,
		Company:          job.Company,
		RoleTitle:        job.Title,
		Location:         inferJobLocation(job.RawText),
		WorkArrangement:  inferWorkArrangement(job.RawText),
		Salary:           firstSalaryLike(job.RawText),
		TopPainPoints:    painPoints,
		RequiredSkills:   normalizeStringList(flattenRequirementKeywords(required, requirements, "must_have")),
		PreferredSkills:  normalizeStringList(flattenRequirementKeywords(preferred, requirements, "nice_to_have")),
		Responsibilities: normalizeStringList(responsibilities),
		SeniorityLevel:   inferSeniority(requirements),
		RoleArchetype:    inferRoleArchetype(job.Title, keywords),
		Keywords:         normalizeStringList(keywords),
		RiskFlags:        normalizeStringList(riskFlags),
		JobPoster:        "",
		CompanyURL:       firstURL(job.RawText),
	}
}

func requirementCanBeTopPainPoint(req JobRequirement) bool {
	if !requirementCanDriveBullet(req) {
		return false
	}

	text := strings.ToLower(req.RequirementText)
	bad := []string{
		"degree",
		"computer science",
		"equivalent practical experience",
		"solid understanding",
		"communication",
		"team-first",
		"team first",
		"collaborative",
		"collaborate",
		"team culture",
		"ownership mentality",
		"seek clarity",
		"shared outcomes",
		"values",
		"benefits",
		"application instruction",
	}

	for _, word := range bad {
		if strings.Contains(text, word) {
			return false
		}
	}

	return true
}

func preserveStringListOrder(values []string) []string {
	seen := map[string]bool{}
	ordered := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		ordered = append(ordered, value)
	}
	return ordered
}

func buildFitAnalysis(jobID int64, requirements []JobRequirement, matches []JobFactMatch) JobFitAnalysis {
	matchesByRequirement := map[int64][]JobFactMatch{}
	for _, match := range matches {
		matchesByRequirement[match.RequirementID] = append(matchesByRequirement[match.RequirementID], match)
	}
	analysis := []FitNeedAnalysis{}
	strengths := []string{}
	gaps := []string{}
	total := 0.0
	weighted := 0.0
	highPriorityTotal := 0
	highPriorityCovered := 0
	partialNeeds := 0
	riskyEvidence := 0
	for _, req := range requirements {
		weight := 1.0
		if req.Priority == "high" {
			weight = 1.7
			highPriorityTotal++
		}
		switch req.Category {
		case "must_have", "seniority":
			weight += 0.4
		case "responsibility":
			weight += 0.2
		case "nice_to_have":
			weight *= 0.65
		}
		total += weight
		reqMatches := matchesByRequirement[req.ID]
		bestScore := 0.0
		bestCoverage := "gap"
		factIDs := []int64{}
		risk := ""
		for _, match := range reqMatches {
			factIDs = append(factIDs, match.FactID)
			if match.Score > bestScore {
				bestScore = match.Score
				bestCoverage = match.CoverageStatus
			}
			if match.FactStatus != factStatusApproved {
				risk = "uses " + match.FactStatus + " evidence"
				riskyEvidence++
			}
		}
		weighted += bestScore * weight
		gapLevel := "critical"
		if bestScore >= 0.75 {
			gapLevel = "covered"
			if req.Priority == "high" {
				highPriorityCovered++
			}
			strengths = append(strengths, "Strong evidence for: "+req.RequirementText)
		} else if bestScore >= 0.45 {
			gapLevel = "partial"
			partialNeeds++
			if req.Priority == "high" || req.Category == "must_have" {
				gaps = append(gaps, "Partial evidence only: "+req.RequirementText)
			}
		} else {
			gapPrefix := "Missing evidence for"
			if req.Priority == "high" || req.Category == "must_have" || req.Category == "seniority" {
				gapPrefix = "Critical missing evidence for"
			}
			gaps = append(gaps, gapPrefix+": "+req.RequirementText)
		}
		confidence := "medium"
		if bestCoverage == "strong" {
			confidence = "high"
		}
		if len(reqMatches) == 0 {
			confidence = "low"
			risk = "no matching evidence"
		}
		analysis = append(analysis, FitNeedAnalysis{
			RequirementID:    req.ID,
			JDNeed:           req.RequirementText,
			MatchingFactIDs:  normalizeInt64List(factIDs),
			EvidenceStrength: bestCoverage,
			GapLevel:         gapLevel,
			Confidence:       confidence,
			Risk:             risk,
		})
	}
	score := 0
	if total > 0 {
		score = int((weighted / total) * 100)
	}
	recommendation := fitRecommendation(score, highPriorityTotal, highPriorityCovered)
	return JobFitAnalysis{
		JobID:          jobID,
		OverallScore:   score,
		Recommendation: recommendation,
		Strengths:      normalizeStringList(strengths),
		CriticalGaps:   normalizeStringList(gaps),
		RealityCheck:   fitRealityCheck(score, len(requirements), highPriorityTotal, highPriorityCovered, partialNeeds, riskyEvidence),
		Analysis:       analysis,
	}
}

func fitRecommendation(score int, highPriorityTotal int, highPriorityCovered int) string {
	highCoverage := 1.0
	if highPriorityTotal > 0 {
		highCoverage = float64(highPriorityCovered) / float64(highPriorityTotal)
	}
	recommendation := "Look Elsewhere"
	switch {
	case score >= 72 && highCoverage >= 0.6:
		recommendation = "Apply"
	case score >= 58 && highCoverage >= 0.4:
		recommendation = "Apply With Caution"
	case score >= 38:
		recommendation = "Upskill First"
	default:
		recommendation = "Look Elsewhere"
	}
	return recommendation
}

func fitRealityCheck(score int, requirementCount int, highPriorityTotal int, highPriorityCovered int, partialNeeds int, riskyEvidence int) string {
	base := fmt.Sprintf("%d%% evidence-backed fit across %d parsed requirements; %d/%d high-priority needs are strongly covered.", score, requirementCount, highPriorityCovered, highPriorityTotal)
	if riskyEvidence > 0 {
		base += fmt.Sprintf(" %d match(es) rely on unapproved or rejected evidence, so treat those claims cautiously.", riskyEvidence)
	}
	if partialNeeds > 0 {
		base += fmt.Sprintf(" %d requirement(s) are only partially supported.", partialNeeds)
	}
	switch {
	case score >= 75:
		return base + " Competitive application if the resume leads with the strongest matching evidence and avoids unsupported extras."
	case score >= 58:
		return base + " Plausible but not dominant; apply only with tight tailoring around the covered requirements."
	case score >= 38:
		return base + " Interview odds are weak unless the role is flexible or the resume can credibly close the critical gaps."
	default:
		return base + " Low-probability fit; spend effort on stronger matches or filling the missing requirements first."
	}
}

func buildApplicationStrategy(job JobDescription, analysis JobAnalysis, fit JobFitAnalysis, matches []JobFactMatch) ApplicationStrategy {
	approvedFactIDs := []int64{}
	rejectedFactIDs := []int64{}
	for _, match := range matches {
		if match.FactStatus == factStatusApproved {
			approvedFactIDs = append(approvedFactIDs, match.FactID)
		} else {
			rejectedFactIDs = append(rejectedFactIDs, match.FactID)
		}
	}
	keywords := analysis.Keywords
	if len(keywords) > 12 {
		keywords = keywords[:12]
	}
	headline := strings.TrimSpace(strings.Join([]string{analysis.RoleArchetype, "for", job.Title}, " "))
	if strings.TrimSpace(analysis.RoleArchetype) == "" {
		headline = job.Title
	}
	return ApplicationStrategy{
		JobID:               job.ID,
		ApprovedFactIDs:     normalizeInt64List(approvedFactIDs),
		RejectedFactIDs:     normalizeInt64List(rejectedFactIDs),
		WeakOrMissing:       fit.CriticalGaps,
		ResumeHeadline:      headline,
		ExperienceTitles:    map[string]string{"default": job.Title},
		PositioningStrategy: "Lead with the strongest approved evidence that maps to the top JD pain points; avoid unsupported gaps.",
		Keywords:            normalizeStringList(keywords),
		DoNotOverclaim:      fit.CriticalGaps,
		FitSummary:          fit.RealityCheck,
	}
}

func (s *Store) replaceJobAnalysis(analysis JobAnalysis) error {
	now := time.Now().UTC().Format(time.RFC3339)
	topPainPointsJSON, _ := encodeStringList(analysis.TopPainPoints)
	requiredJSON, _ := encodeStringList(analysis.RequiredSkills)
	preferredJSON, _ := encodeStringList(analysis.PreferredSkills)
	responsibilitiesJSON, _ := encodeStringList(analysis.Responsibilities)
	keywordsJSON, _ := encodeStringList(analysis.Keywords)
	riskFlagsJSON, _ := encodeStringList(analysis.RiskFlags)
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO job_analyses
			(job_id, company, role_title, location, work_arrangement, salary, top_pain_points_json,
			required_skills_json, preferred_skills_json, responsibilities_json, seniority_level, role_archetype,
			keywords_json, risk_flags_json, job_poster, company_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			company = excluded.company, role_title = excluded.role_title, location = excluded.location,
			work_arrangement = excluded.work_arrangement, salary = excluded.salary,
			top_pain_points_json = excluded.top_pain_points_json, required_skills_json = excluded.required_skills_json,
			preferred_skills_json = excluded.preferred_skills_json, responsibilities_json = excluded.responsibilities_json,
			seniority_level = excluded.seniority_level, role_archetype = excluded.role_archetype,
			keywords_json = excluded.keywords_json, risk_flags_json = excluded.risk_flags_json,
			job_poster = excluded.job_poster, company_url = excluded.company_url, updated_at = excluded.updated_at`,
		analysis.JobID, analysis.Company, analysis.RoleTitle, analysis.Location, analysis.WorkArrangement, analysis.Salary,
		topPainPointsJSON, requiredJSON, preferredJSON, responsibilitiesJSON, analysis.SeniorityLevel, analysis.RoleArchetype,
		keywordsJSON, riskFlagsJSON, analysis.JobPoster, analysis.CompanyURL, now, now,
	)
	return err
}

func (s *Store) replaceFitAnalysis(fit JobFitAnalysis) error {
	now := time.Now().UTC().Format(time.RFC3339)
	strengthsJSON, _ := encodeStringList(fit.Strengths)
	gapsJSON, _ := encodeStringList(fit.CriticalGaps)
	analysisJSON, _ := json.Marshal(fit.Analysis)
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO job_fit_analyses (job_id, overall_score, recommendation, strengths_json, critical_gaps_json, reality_check, analysis_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			overall_score = excluded.overall_score, recommendation = excluded.recommendation,
			strengths_json = excluded.strengths_json, critical_gaps_json = excluded.critical_gaps_json,
			reality_check = excluded.reality_check, analysis_json = excluded.analysis_json, updated_at = excluded.updated_at`,
		fit.JobID, fit.OverallScore, fit.Recommendation, strengthsJSON, gapsJSON, fit.RealityCheck, string(analysisJSON), now, now,
	)
	return err
}

func (s *Store) replaceApplicationStrategy(strategy ApplicationStrategy) error {
	now := time.Now().UTC().Format(time.RFC3339)
	approvedJSON, _ := encodeInt64List(strategy.ApprovedFactIDs)
	rejectedJSON, _ := encodeInt64List(strategy.RejectedFactIDs)
	weakJSON, _ := encodeStringList(strategy.WeakOrMissing)
	titlesJSON, _ := json.Marshal(strategy.ExperienceTitles)
	keywordsJSON, _ := encodeStringList(strategy.Keywords)
	overclaimJSON, _ := encodeStringList(strategy.DoNotOverclaim)
	_, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO application_strategies
			(job_id, approved_fact_ids_json, rejected_fact_ids_json, weak_or_missing_json, resume_headline,
			experience_titles_json, positioning_strategy, keywords_json, do_not_overclaim_json, fit_summary, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			approved_fact_ids_json = excluded.approved_fact_ids_json,
			rejected_fact_ids_json = excluded.rejected_fact_ids_json,
			weak_or_missing_json = excluded.weak_or_missing_json,
			resume_headline = excluded.resume_headline,
			experience_titles_json = excluded.experience_titles_json,
			positioning_strategy = excluded.positioning_strategy,
			keywords_json = excluded.keywords_json,
			do_not_overclaim_json = excluded.do_not_overclaim_json,
			fit_summary = excluded.fit_summary,
			updated_at = excluded.updated_at`,
		strategy.JobID, approvedJSON, rejectedJSON, weakJSON, strategy.ResumeHeadline, string(titlesJSON),
		strategy.PositioningStrategy, keywordsJSON, overclaimJSON, strategy.FitSummary, now, now,
	)
	return err
}

func scanJobAnalysisRow(rows *sql.Rows) (JobAnalysis, error) {
	var analysis JobAnalysis
	var topPainPointsJSON, requiredJSON, preferredJSON, responsibilitiesJSON, keywordsJSON, riskFlagsJSON string
	if err := rows.Scan(&analysis.JobID, &analysis.Company, &analysis.RoleTitle, &analysis.Location, &analysis.WorkArrangement,
		&analysis.Salary, &topPainPointsJSON, &requiredJSON, &preferredJSON, &responsibilitiesJSON, &analysis.SeniorityLevel,
		&analysis.RoleArchetype, &keywordsJSON, &riskFlagsJSON, &analysis.JobPoster, &analysis.CompanyURL, &analysis.CreatedAt, &analysis.UpdatedAt); err != nil {
		return JobAnalysis{}, err
	}
	analysis.TopPainPoints = decodeStringList(topPainPointsJSON)
	analysis.RequiredSkills = decodeStringList(requiredJSON)
	analysis.PreferredSkills = decodeStringList(preferredJSON)
	analysis.Responsibilities = decodeStringList(responsibilitiesJSON)
	analysis.Keywords = decodeStringList(keywordsJSON)
	analysis.RiskFlags = decodeStringList(riskFlagsJSON)
	return analysis, nil
}

func scanFitAnalysisRow(rows *sql.Rows) (JobFitAnalysis, error) {
	var fit JobFitAnalysis
	var strengthsJSON, gapsJSON, analysisJSON string
	if err := rows.Scan(&fit.JobID, &fit.OverallScore, &fit.Recommendation, &strengthsJSON, &gapsJSON, &fit.RealityCheck, &analysisJSON, &fit.CreatedAt, &fit.UpdatedAt); err != nil {
		return JobFitAnalysis{}, err
	}
	fit.Strengths = decodeStringList(strengthsJSON)
	fit.CriticalGaps = decodeStringList(gapsJSON)
	_ = json.Unmarshal([]byte(analysisJSON), &fit.Analysis)
	return fit, nil
}

func scanApplicationStrategyRow(rows *sql.Rows) (ApplicationStrategy, error) {
	var strategy ApplicationStrategy
	var approvedJSON, rejectedJSON, weakJSON, titlesJSON, keywordsJSON, overclaimJSON string
	if err := rows.Scan(&strategy.JobID, &approvedJSON, &rejectedJSON, &weakJSON, &strategy.ResumeHeadline, &titlesJSON,
		&strategy.PositioningStrategy, &keywordsJSON, &overclaimJSON, &strategy.FitSummary, &strategy.CreatedAt, &strategy.UpdatedAt); err != nil {
		return ApplicationStrategy{}, err
	}
	strategy.ApprovedFactIDs = decodeInt64List(approvedJSON)
	strategy.RejectedFactIDs = decodeInt64List(rejectedJSON)
	strategy.WeakOrMissing = decodeStringList(weakJSON)
	strategy.Keywords = decodeStringList(keywordsJSON)
	strategy.DoNotOverclaim = decodeStringList(overclaimJSON)
	strategy.ExperienceTitles = map[string]string{}
	_ = json.Unmarshal([]byte(titlesJSON), &strategy.ExperienceTitles)
	return strategy, nil
}

func flattenRequirementKeywords(requirementTexts []string, requirements []JobRequirement, category string) []string {
	values := []string{}
	textSet := map[string]bool{}
	for _, text := range requirementTexts {
		textSet[text] = true
	}
	for _, req := range requirements {
		if req.Category == category || textSet[req.RequirementText] {
			values = append(values, req.Keywords...)
		}
	}
	return values
}

func inferSeniority(requirements []JobRequirement) string {
	joined := strings.ToLower(strings.Join(requirementTexts(requirements), " "))
	if strings.Contains(joined, "senior") || strings.Contains(joined, "lead") || strings.Contains(joined, "mentor") {
		return "senior"
	}
	if strings.Contains(joined, "graduate") || strings.Contains(joined, "junior") {
		return "junior"
	}
	return "mid"
}

func inferRoleArchetype(title string, keywords []string) string {
	joined := strings.ToLower(title + " " + strings.Join(keywords, " "))
	if strings.Contains(joined, "cloud") || strings.Contains(joined, "azure") || strings.Contains(joined, "aws") {
		return "cloud software engineer"
	}
	if strings.Contains(joined, "backend") || strings.Contains(joined, "api") || strings.Contains(joined, "microservice") {
		return "backend engineer"
	}
	if strings.Contains(joined, "full stack") || strings.Contains(joined, "react") {
		return "full stack engineer"
	}
	return "software engineer"
}

func inferJobLocation(raw string) string {
	for _, line := range meaningfulJobLines(raw, 20) {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "melbourne") || strings.Contains(lower, "sydney") || strings.Contains(lower, "australia") {
			return line
		}
	}
	return ""
}

func inferWorkArrangement(raw string) string {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "remote") {
		return "remote"
	}
	if strings.Contains(lower, "hybrid") {
		return "hybrid"
	}
	if strings.Contains(lower, "office") || strings.Contains(lower, "on-site") || strings.Contains(lower, "onsite") {
		return "onsite"
	}
	return ""
}

func firstSalaryLike(raw string) string {
	for _, line := range meaningfulJobLines(raw, 40) {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "salary") || strings.Contains(line, "$") || strings.Contains(line, "A$") {
			return line
		}
	}
	return ""
}

func firstURL(raw string) string {
	for _, part := range strings.Fields(raw) {
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			return strings.Trim(part, ".,;")
		}
	}
	return ""
}

func requirementTexts(requirements []JobRequirement) []string {
	values := []string{}
	for _, req := range requirements {
		values = append(values, req.RequirementText)
	}
	return values
}

func normalizeInt64List(values []int64) []int64 {
	seen := map[int64]bool{}
	next := []int64{}
	for _, value := range values {
		if value <= 0 || seen[value] {
			continue
		}
		seen[value] = true
		next = append(next, value)
	}
	return next
}
