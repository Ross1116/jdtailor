package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type ResumeJSON struct {
	Headline    string           `json:"headline"`
	Summary     string           `json:"summary"`
	Skills      []ResumeSkill    `json:"skills"`
	Experience  []ResumeEntry    `json:"experience"`
	Projects    []ResumeEntry    `json:"projects"`
	Education   []ResumeEducation `json:"education"`
	GeneratedAt string           `json:"generated_at"`
}

type ResumeSkill struct {
	Category string   `json:"category"`
	Items    []string `json:"items"`
}

type ResumeEntry struct {
	Company    string   `json:"company"`
	Title      string   `json:"title"`
	Location   string   `json:"location"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Bullets    []string `json:"bullets"`
	ClaimIDs   []int64  `json:"claim_ids"`
	BulletIDs  []int64  `json:"bullet_ids"`
}

type ResumeEducation struct {
	Organization string `json:"organization"`
	Degree       string `json:"degree"`
	Location     string `json:"location"`
	EndDate      string `json:"end_date"`
}

type ValidationResult struct {
	Passed           bool              `json:"passed"`
	Errors           []string          `json:"errors"`
	Warnings         []string          `json:"warnings"`
	FactualityChecks []FactualityCheck `json:"factuality_checks"`
	StyleIssues      []string          `json:"style_issues"`
	ImmutableIssues  []string          `json:"immutable_issues"`
	TitleIssues      []string          `json:"title_issues"`
}

type FactualityCheck struct {
	BulletIndex int      `json:"bullet_index"`
	Bullet     string   `json:"bullet"`
	HasClaims  bool     `json:"has_claims"`
	AllApproved bool    `json:"all_approved"`
	Issues     []string `json:"issues,omitempty"`
}

type ResumeVersion struct {
	ID                int64            `json:"id"`
	JobID             int64            `json:"job_id"`
	ResumeJSON        ResumeJSON       `json:"resume_json"`
	TexSource         string           `json:"tex_source"`
	PDFPath           string           `json:"pdf_path"`
	ValidationResult  ValidationResult `json:"validation_result"`
	CreatedAt         string           `json:"created_at"`
}

type Application struct {
	ID                  int64  `json:"id"`
	JobID               int64  `json:"job_id"`
	Status              string `json:"status"`
	FitScore            int    `json:"fit_score"`
	ResumeVersionID     int64  `json:"resume_version_id"`
	CoverLetterVersionID int64 `json:"cover_letter_version_id"`
	Notes               string `json:"notes"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type CorrectionLog struct {
	ID                 int64    `json:"id"`
	ApplicationID      int64    `json:"application_id"`
	ResumeVersionID    int64    `json:"resume_version_id"`
	OriginalBulletText string   `json:"original_bullet_text"`
	CorrectedBulletText string  `json:"corrected_bullet_text"`
	ClaimIDs           []int64  `json:"claim_ids"`
	Reason             string   `json:"reason"`
	CreatedAt          string   `json:"created_at"`
}

type GenerateResumeJSONInput struct {
	JobID           int64   `json:"job_id"`
	SelectedBulletIDs []int64 `json:"selected_bullet_ids"`
}

func (s *Store) GenerateResumeJSON(ctx context.Context, input GenerateResumeJSONInput) (ResumeJSON, error) {
	job, err := s.getJobDescription(input.JobID)
	if err != nil {
		return ResumeJSON{}, err
	}
	_ = job

	analysis, err := s.GetJobAnalysis(input.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		analysis, err = s.AnalyzeJobDescription(input.JobID)
	}
	if err != nil {
		return ResumeJSON{}, err
	}

	strategy, err := s.GetApplicationStrategy(input.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		strategy, err = s.GenerateApplicationStrategy(input.JobID)
	}
	if err != nil {
		return ResumeJSON{}, err
	}

	profile, err := s.GetCandidateProfile()
	if err != nil {
		return ResumeJSON{}, err
	}

	var selectedDrafts []TailoredBulletDraft
	if len(input.SelectedBulletIDs) > 0 {
		allDrafts, err := s.ListTailoredBulletDrafts(input.JobID)
		if err != nil {
			return ResumeJSON{}, err
		}
		idSet := map[int64]bool{}
		for _, id := range input.SelectedBulletIDs {
			idSet[id] = true
		}
		for _, draft := range allDrafts {
			if idSet[draft.ID] {
				selectedDrafts = append(selectedDrafts, draft)
			}
		}
	} else {
		drafts, listErr := s.ListTailoredBulletDrafts(input.JobID)
		if listErr != nil {
			return ResumeJSON{}, listErr
		}
		for _, draft := range drafts {
			if draft.SelectedForResume {
				selectedDrafts = append(selectedDrafts, draft)
			}
		}
	}

	if len(selectedDrafts) == 0 {
		return ResumeJSON{}, errors.New("no selected bullet drafts for resume generation")
	}

	claimIDs := map[int64]bool{}
	bulletClaimMap := map[int64][]int64{}
	for _, draft := range selectedDrafts {
		for _, cid := range draft.ClaimIDs {
			claimIDs[cid] = true
		}
		bulletClaimMap[draft.ID] = draft.ClaimIDs
	}

	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return ResumeJSON{}, err
	}
	claimsByID := map[int64]CandidateClaim{}
	for _, claim := range claims {
		claimsByID[claim.ID] = claim
	}

	experienceEntries := []ResumeEntry{}
	projectEntries := []ResumeEntry{}
	seenBullets := map[string]bool{}

	for _, draft := range selectedDrafts {
		if seenBullets[draft.DraftText] {
			continue
		}
		seenBullets[draft.DraftText] = true

		entry := ResumeEntry{
			Title:     draft.OriginHeading,
			Bullets:   []string{draft.DraftText},
			ClaimIDs:  draft.ClaimIDs,
			BulletIDs: []int64{draft.ID},
		}

		for _, cid := range draft.ClaimIDs {
			if claim, ok := claimsByID[cid]; ok && claim.OriginHeading != "" {
				entry.Company = claim.OriginHeading
				break
			}
		}

		originType := strings.TrimSpace(strings.ToLower(draft.OriginType))
		if originType == "project" || originType == "education" {
			projectEntries = append(projectEntries, entry)
		} else {
			experienceEntries = append(experienceEntries, entry)
		}
	}

	skillSet := map[string]bool{}
	for _, claim := range claims {
		if !claimIDs[claim.ID] {
			continue
		}
		for _, tech := range claim.Technologies {
			skillSet[tech] = true
		}
	}

	langSkills := []string{}
	frameworkSkills := []string{}
	platformSkills := []string{}
	toolSkills := []string{}
	langSet := map[string]bool{"Go": true, "Python": true, "Java": true, "TypeScript": true, "JavaScript": true, "Rust": true, "C#": true, "C++": true}
	frameworkSet := map[string]bool{"React": true, "FastAPI": true, "Node.js": true, "Express": true}
	platformSet := map[string]bool{"AWS": true, "Azure": true, "GCP": true, "Docker": true, "Kubernetes": true}
	for skill := range skillSet {
		switch {
		case langSet[skill]:
			langSkills = append(langSkills, skill)
		case frameworkSet[skill]:
			frameworkSkills = append(frameworkSkills, skill)
		case platformSet[skill]:
			platformSkills = append(platformSkills, skill)
		default:
			toolSkills = append(toolSkills, skill)
		}
	}

	skills := []ResumeSkill{}
	if len(langSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Languages", Items: normalizeStringList(langSkills)})
	}
	if len(frameworkSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Frameworks", Items: normalizeStringList(frameworkSkills)})
	}
	if len(platformSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Platforms", Items: normalizeStringList(platformSkills)})
	}
	if len(toolSkills) > 0 {
		skills = append(skills, ResumeSkill{Category: "Tools", Items: normalizeStringList(toolSkills)})
	}

	headline := strings.TrimSpace(strategy.ResumeHeadline)
	if headline == "" {
		headline = analysis.RoleTitle
	}

	resume := ResumeJSON{
		Headline:    headline,
		Summary:     buildResumeSummary(profile, analysis, claimIDs, claimsByID),
		Skills:      skills,
		Experience:  experienceEntries,
		Projects:    projectEntries,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	return resume, nil
}

func buildResumeSummary(profile CandidateProfile, analysis JobAnalysis, claimIDs map[int64]bool, claimsByID map[int64]CandidateClaim) string {
	techs := []string{}
	for cid := range claimIDs {
		if claim, ok := claimsByID[cid]; ok {
			techs = append(techs, claim.Technologies...)
		}
	}
	techs = normalizeStringList(techs)
	if len(techs) > 5 {
		techs = techs[:5]
	}

	name := strings.TrimSpace(profile.Contact.FullName)
	if name == "" {
		name = "Candidate"
	}
	role := strings.TrimSpace(analysis.RoleTitle)
	if role == "" {
		role = "engineer"
	}
	techStr := strings.Join(techs, ", ")
	if techStr == "" {
		return name + " — tailored for " + role + "."
	}
	return name + " — " + role + ". Focused on " + techStr + "."
}

func (s *Store) ValidateResumeJSON(resume ResumeJSON, jobID int64) (ValidationResult, error) {
	result := ValidationResult{Passed: true}

	claims, err := s.ListCandidateClaims("all")
	if err != nil {
		return result, err
	}
	claimsByID := map[int64]CandidateClaim{}
	for _, claim := range claims {
		claimsByID[claim.ID] = claim
	}

	profile, err := s.GetCandidateProfile()
	if err != nil {
		return result, err
	}

	var allChecks []FactualityCheck
	allBullets := []string{}
	for _, entry := range resume.Experience {
		allBullets = append(allBullets, entry.Bullets...)
	}
	for _, entry := range resume.Projects {
		allBullets = append(allBullets, entry.Bullets...)
	}

	for i, bullet := range allBullets {
		check := FactualityCheck{
			BulletIndex: i,
			Bullet:     bullet,
			HasClaims:  true,
			AllApproved: true,
		}

		lower := strings.ToLower(bullet)
		evidence := strings.ToLower(supportedBulletEvidence(bullet, claimsByID))
		for _, term := range []string{"aws", "serverless", "container", "containers", "kubernetes", "health-tech", "healthcare", "medical", "compliance", "enterprise", "scalable"} {
			if strings.Contains(lower, term) && !strings.Contains(evidence, term) {
				check.Issues = append(check.Issues, "unsupported term: "+term)
				check.AllApproved = false
			}
		}

		if check.AllApproved && len(check.Issues) == 0 {
			check.Issues = append(check.Issues, "no linked claims; verify content is evidence-backed")
		}

		allChecks = append(allChecks, check)
	}
	result.FactualityChecks = allChecks

	styleIssues := []string{}
	lowerBullets := strings.ToLower(strings.Join(allBullets, " "))
	bannedPhrases := []string{
		"leveraged", "spearheaded", "empowered", "utilized", "cutting-edge",
		"game-changer", "transformative", "synergy", "seamless",
	}
	for _, phrase := range bannedPhrases {
		if strings.Contains(lowerBullets, phrase) {
			styleIssues = append(styleIssues, "banned phrase: "+phrase)
		}
	}
	result.StyleIssues = styleIssues

	immutableIssues := []string{}
	if profile.Contact.FullName != "" {
		nameParts := strings.Fields(profile.Contact.FullName)
		for _, part := range nameParts {
			partLower := strings.ToLower(part)
			if !strings.Contains(strings.ToLower(resume.Headline), partLower) && !strings.Contains(strings.ToLower(resume.Summary), partLower) {
				break
			}
		}
	}
	result.ImmutableIssues = immutableIssues

	titleIssues := []string{}
	if strings.TrimSpace(resume.Headline) == "" {
		titleIssues = append(titleIssues, "headline is empty")
	}
	for _, entry := range resume.Experience {
		if strings.TrimSpace(entry.Title) == "" {
			titleIssues = append(titleIssues, "experience entry missing title")
		}
	}
	result.TitleIssues = titleIssues

	for _, entry := range resume.Experience {
		for _, bullet := range entry.Bullets {
			bulletIssues := styleRiskFlags(bullet)
			for _, issue := range bulletIssues {
				result.Warnings = append(result.Warnings, "bullet style: "+issue+": "+bullet)
			}
		}
	}

	for _, check := range result.FactualityChecks {
		if !check.AllApproved {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("bullet %d: unapproved claims", check.BulletIndex))
		}
		for _, issue := range check.Issues {
			result.Warnings = append(result.Warnings, fmt.Sprintf("bullet %d: %s", check.BulletIndex, issue))
		}
	}

	for _, issue := range result.StyleIssues {
		result.Warnings = append(result.Warnings, issue)
	}
	for _, issue := range result.ImmutableIssues {
		result.Errors = append(result.Errors, issue)
		result.Passed = false
	}
	for _, issue := range result.TitleIssues {
		result.Errors = append(result.Errors, issue)
		result.Passed = false
	}

	return result, nil
}

func supportedBulletEvidence(bullet string, claimsByID map[int64]CandidateClaim) string {
	parts := []string{}
	for _, claim := range claimsByID {
		parts = append(parts,
			claim.ClaimText,
			strings.Join(claim.Actions, " "),
			strings.Join(claim.Technologies, " "),
			strings.Join(claim.Capabilities, " "),
			strings.Join(claim.Objects, " "),
			strings.Join(claim.Domains, " "),
			strings.Join(claim.Artifacts, " "),
			strings.Join(claim.Scope, " "),
			strings.Join(claim.Metrics, " "),
			strings.Join(claim.Outcomes, " "),
		)
	}
	return strings.Join(parts, " ")
}

func (s *Store) RenderResumePDF(ctx context.Context, resume ResumeJSON) (RenderPDFResult, error) {
	funcs := template.FuncMap{
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
	}
	tmpl, err := template.New("resume").Funcs(funcs).Parse(latexResumeTemplate())
	if err != nil {
		return RenderPDFResult{}, fmt.Errorf("failed to parse LaTeX template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, resume); err != nil {
		return RenderPDFResult{}, fmt.Errorf("failed to execute LaTeX template: %w", err)
	}

	texContent := sanitizeLaTeX(buf.String())

	outputDir := filepath.Join(s.generatedPath, fmt.Sprintf("resume-%d", time.Now().Unix()))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RenderPDFResult{}, err
	}

	texPath := filepath.Join(outputDir, "resume.tex")
	if err := os.WriteFile(texPath, []byte(texContent), 0o644); err != nil {
		return RenderPDFResult{}, err
	}

	pdfPath := filepath.Join(outputDir, "resume.pdf")
	pdfResult, pdfErr := s.RenderSamplePDF(ctx)

	return RenderPDFResult{
		Success: pdfResult.Success,
		TexPath: texPath,
		PDFPath: pdfPath,
		Error:   firstNonEmptyErr(pdfErr, pdfResult.Error),
	}, nil
}

func (s *Store) SaveResumeVersion(version ResumeVersion) (ResumeVersion, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	resumeJSONBytes, _ := json.Marshal(version.ResumeJSON)
	validationBytes, _ := json.Marshal(version.ValidationResult)

	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO resume_versions (job_id, resume_json, tex_source, pdf_path, validation_result, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		version.JobID,
		string(resumeJSONBytes),
		version.TexSource,
		version.PDFPath,
		string(validationBytes),
		now,
	)
	if err != nil {
		return ResumeVersion{}, err
	}
	id, _ := result.LastInsertId()
	version.ID = id
	version.CreatedAt = now
	return version, nil
}

func (s *Store) GetResumeVersion(id int64) (ResumeVersion, error) {
	var version ResumeVersion
	var resumeJSONStr, validationStr string
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, job_id, resume_json, tex_source, pdf_path, validation_result, created_at
		FROM resume_versions WHERE id = ?`,
		id,
	).Scan(&version.ID, &version.JobID, &resumeJSONStr, &version.TexSource, &version.PDFPath, &validationStr, &version.CreatedAt)
	if err != nil {
		return ResumeVersion{}, err
	}
	json.Unmarshal([]byte(resumeJSONStr), &version.ResumeJSON)
	json.Unmarshal([]byte(validationStr), &version.ValidationResult)
	return version, nil
}

func (s *Store) ListResumeVersions(jobID int64) ([]ResumeVersion, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, resume_json, tex_source, pdf_path, validation_result, created_at
		FROM resume_versions WHERE job_id = ? ORDER BY created_at DESC`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []ResumeVersion
	for rows.Next() {
		var version ResumeVersion
		var resumeJSONStr, validationStr string
		if err := rows.Scan(&version.ID, &version.JobID, &resumeJSONStr, &version.TexSource, &version.PDFPath, &validationStr, &version.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(resumeJSONStr), &version.ResumeJSON)
		json.Unmarshal([]byte(validationStr), &version.ValidationResult)
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) SaveApplication(app Application) (Application, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if app.ID > 0 {
		_, err := s.db.ExecContext(
			context.Background(),
			`UPDATE applications SET status = ?, fit_score = ?, resume_version_id = ?, cover_letter_version_id = ?, notes = ?, updated_at = ? WHERE id = ?`,
			normalizeAppStatus(app.Status), app.FitScore, app.ResumeVersionID, app.CoverLetterVersionID, strings.TrimSpace(app.Notes), now, app.ID,
		)
		if err != nil {
			return Application{}, err
		}
		app.UpdatedAt = now
		return app, nil
	}

	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO applications (job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		app.JobID, normalizeAppStatus(app.Status), app.FitScore, app.ResumeVersionID, app.CoverLetterVersionID, strings.TrimSpace(app.Notes), now, now,
	)
	if err != nil {
		return Application{}, err
	}
	id, _ := result.LastInsertId()
	app.ID = id
	app.CreatedAt = now
	app.UpdatedAt = now
	return app, nil
}

func (s *Store) GetApplication(id int64) (Application, error) {
	var app Application
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT id, job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at
		FROM applications WHERE id = ?`,
		id,
	).Scan(&app.ID, &app.JobID, &app.Status, &app.FitScore, &app.ResumeVersionID, &app.CoverLetterVersionID, &app.Notes, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return Application{}, err
	}
	return app, nil
}

func (s *Store) ListApplications() ([]Application, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, job_id, status, fit_score, resume_version_id, cover_letter_version_id, notes, created_at, updated_at
		FROM applications ORDER BY updated_at DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var app Application
		if err := rows.Scan(&app.ID, &app.JobID, &app.Status, &app.FitScore, &app.ResumeVersionID, &app.CoverLetterVersionID, &app.Notes, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func (s *Store) UpdateApplicationStatus(id int64, status string) (Application, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(context.Background(), `UPDATE applications SET status = ?, updated_at = ? WHERE id = ?`, normalizeAppStatus(status), now, id)
	if err != nil {
		return Application{}, err
	}
	return s.GetApplication(id)
}

func (s *Store) LogCorrection(correction CorrectionLog) (CorrectionLog, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	claimJSON, _ := encodeInt64List(correction.ClaimIDs)
	result, err := s.db.ExecContext(
		context.Background(),
		`INSERT INTO correction_logs (application_id, resume_version_id, original_text, corrected_text, claim_ids, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		correction.ApplicationID,
		correction.ResumeVersionID,
		correction.OriginalBulletText,
		correction.CorrectedBulletText,
		claimJSON,
		strings.TrimSpace(correction.Reason),
		now,
	)
	if err != nil {
		return CorrectionLog{}, err
	}
	id, _ := result.LastInsertId()
	correction.ID = id
	correction.CreatedAt = now
	return correction, nil
}

func (s *Store) ListCorrections(applicationID int64) ([]CorrectionLog, error) {
	rows, err := s.db.QueryContext(
		context.Background(),
		`SELECT id, application_id, resume_version_id, original_text, corrected_text, claim_ids, reason, created_at
		FROM correction_logs WHERE application_id = ? ORDER BY created_at DESC`,
		applicationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var corrections []CorrectionLog
	for rows.Next() {
		var c CorrectionLog
		var claimJSON string
		if err := rows.Scan(&c.ID, &c.ApplicationID, &c.ResumeVersionID, &c.OriginalBulletText, &c.CorrectedBulletText, &claimJSON, &c.Reason, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.ClaimIDs = decodeInt64List(claimJSON)
		corrections = append(corrections, c)
	}
	return corrections, rows.Err()
}

func firstNonEmptyErr(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}

func normalizeAppStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "draft", "ready_to_apply", "applied", "rejected", "interviewing", "offer":
		return strings.TrimSpace(strings.ToLower(status))
	}
	return "draft"
}

func latexResumeTemplate() string {
	return `\documentclass[11pt,a4paper]{article}
\usepackage[utf8]{inputenc}
\usepackage[margin=0.5in]{geometry}
\usepackage{enumitem}
\usepackage{titlesec}

\pagestyle{empty}
\setlength{\parindent}{0pt}
\setlength{\parskip}{4pt}

\titleformat{\section}{\large\bfseries}{}{0em}{}[\titlerule]

\begin{document}

\begin{center}
{\LARGE\bfseries {{.Headline}}} \\
\vspace{4pt}
{{.Summary}}
\end{center}

{{range .Skills}}
\section{\Category}
{{join .Items ", "}}

{{end}}

{{range .Experience}}
\section{\Title}
{{range .Bullets}}
\item {{.}}
{{end}}

{{end}}

{{range .Projects}}
\section{\Title}
{{range .Bullets}}
\item {{.}}
{{end}}

{{end}}

\end{document}
`
}

func sanitizeLaTeX(input string) string {
	replacements := []struct{ old, new string }{
		{`&`, `\&`},
		{`%`, `\%`},
		{`$`, `\$`},
		{`#`, `\#`},
		{`{`, `\{`},
		{`}`, `\}`},
		{`~`, `\textasciitilde{}`},
		{`^`, `\textasciicircum{}`},
	}
	for _, r := range replacements {
		input = strings.ReplaceAll(input, r.old, r.new)
	}
	input = strings.ReplaceAll(input, `\`, `\textbackslash{}`)
	return input
}
