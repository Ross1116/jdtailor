package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

const maxAgenticJDLength = 50000

type JobAgentWorkflowInput struct {
	Job                CreateJobDescriptionInput `json:"job"`
	JobID              int64                     `json:"job_id"`
	AutoSelectBullets  bool                      `json:"auto_select_bullets"`
	BuildResume        bool                      `json:"build_resume"`
	MinSelectedBullets int                       `json:"min_selected_bullets"`
	MaxSelectedBullets int                       `json:"max_selected_bullets"`
}

type JobAgentWorkflowStage struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type JobAgentWorkflowResult struct {
	Job             JobDescription          `json:"job"`
	Stages          []JobAgentWorkflowStage `json:"stages"`
	Requirements    []JobRequirement        `json:"requirements"`
	Matches         []JobFactMatch          `json:"matches"`
	Drafts          []TailoredBulletDraft   `json:"drafts"`
	Analysis        JobAnalysis             `json:"analysis"`
	Fit             JobFitAnalysis          `json:"fit"`
	Strategy        ApplicationStrategy     `json:"strategy"`
	Resume          ResumeJSON              `json:"resume"`
	Validation      ValidationResult        `json:"validation"`
	ResumeGenerated bool                    `json:"resume_generated"`
	CreatedAt       string                  `json:"created_at"`
}

func (s *Store) RunJobAgentWorkflow(ctx context.Context, input JobAgentWorkflowInput, client *http.Client) (JobAgentWorkflowResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	input = normalizeJobAgentWorkflowInput(input)
	if strings.TrimSpace(input.Job.RawText) == "" && input.JobID <= 0 {
		return JobAgentWorkflowResult{}, errors.New("job description text is required")
	}
	if len(input.Job.RawText) > maxAgenticJDLength {
		return JobAgentWorkflowResult{}, errors.New("job description is too long for the agentic workflow")
	}

	result := JobAgentWorkflowResult{CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	stage := func(key, label, status, message string) {
		result.Stages = append(result.Stages, JobAgentWorkflowStage{Key: key, Label: label, Status: status, Message: message})
	}

	stage("intake", "JD intake", "running", "normalizing job description")
	var job JobDescription
	var err error
	if input.JobID > 0 {
		job, err = s.getJobDescription(input.JobID)
		if err == nil {
			updated := UpdateJobDescriptionInput{ID: job.ID, Company: job.Company, Title: job.Title, URL: job.URL, RawText: job.RawText}
			changed := false
			if strings.TrimSpace(input.Job.Company) != "" {
				updated.Company = input.Job.Company
				changed = true
			}
			if strings.TrimSpace(input.Job.Title) != "" {
				updated.Title = input.Job.Title
				changed = true
			}
			if strings.TrimSpace(input.Job.URL) != "" {
				updated.URL = input.Job.URL
				changed = true
			}
			if strings.TrimSpace(input.Job.RawText) != "" {
				updated.RawText = input.Job.RawText
				changed = true
			}
			if changed {
				job, err = s.UpdateJobDescription(updated)
			}
		}
	} else {
		job, err = s.CreateJobDescription(input.Job)
	}
	if err != nil {
		stage("intake", "JD intake", "failed", err.Error())
		return result, err
	}
	result.Job = job
	stage("intake", "JD intake", "ok", "job saved")

	stage("parse", "Requirement parser", "running", "extracting explicit requirements")
	result.Requirements, err = s.ParseJobDescription(ctx, job.ID, client)
	if err != nil {
		stage("parse", "Requirement parser", "failed", err.Error())
		return result, err
	}
	stage("parse", "Requirement parser", "ok", pluralCount(len(result.Requirements), "requirement"))

	stage("analysis", "JD analyst", "running", "identifying role shape and employer pain")
	result.Analysis, err = s.AnalyzeJobDescription(job.ID)
	if err != nil {
		stage("analysis", "JD analyst", "failed", err.Error())
		return result, err
	}
	stage("analysis", "JD analyst", "ok", result.Analysis.RoleArchetype)

	stage("match", "Evidence matcher", "running", "matching JD needs to approved candidate evidence")
	result.Matches, err = s.BuildJobMatchMap(ctx, job.ID, client)
	if err != nil {
		stage("match", "Evidence matcher", "failed", err.Error())
		return result, err
	}
	stage("match", "Evidence matcher", "ok", pluralCount(len(result.Matches), "evidence match"))

	stage("fit", "Fit critic", "running", "scoring realistic application fit")
	result.Fit, err = s.GenerateFitAnalysis(job.ID)
	if err != nil {
		stage("fit", "Fit critic", "failed", err.Error())
		return result, err
	}
	stage("fit", "Fit critic", "ok", result.Fit.Recommendation)

	stage("strategy", "Application strategist", "running", "deciding positioning and weak spots")
	result.Strategy, err = s.GenerateApplicationStrategy(job.ID)
	if err != nil {
		stage("strategy", "Application strategist", "failed", err.Error())
		return result, err
	}
	stage("strategy", "Application strategist", "ok", "strategy ready")

	stage("draft", "Resume drafter", "running", "drafting evidence-grounded bullets")
	result.Drafts, err = s.GenerateTailoredBulletDrafts(ctx, job.ID, client)
	if err != nil {
		stage("draft", "Resume drafter", "failed", err.Error())
		return result, err
	}
	stage("draft", "Resume drafter", "ok", pluralCount(len(result.Drafts), "bullet draft"))

	if input.AutoSelectBullets {
		stage("select", "Bullet selector", "running", "selecting highest-value supported bullets")
		result.Drafts, err = s.AutoSelectResumeBullets(job.ID)
		if err != nil {
			stage("select", "Bullet selector", "failed", err.Error())
			return result, err
		}
		selected := countSelectedDrafts(result.Drafts)
		if selected < input.MinSelectedBullets {
			stage("select", "Bullet selector", "warning", "fewer than target selected; preserving truth constraints")
		} else if input.MaxSelectedBullets > 0 && selected > input.MaxSelectedBullets {
			stage("select", "Bullet selector", "warning", "more than target selected; resume generator will still enforce compactness")
		} else {
			stage("select", "Bullet selector", "ok", pluralCount(selected, "selected bullet"))
		}
	}

	if input.BuildResume {
		selectedIDs := selectedDraftIDs(result.Drafts)
		if len(selectedIDs) == 0 {
			stage("resume", "Resume assembler", "failed", "no selected bullets available")
			return result, errors.New("no selected bullets available for resume generation")
		}
		stage("resume", "Resume assembler", "running", "assembling and polishing an in-memory resume draft")
		result.Resume, err = s.GenerateResumeJSON(ctx, GenerateResumeJSONInput{JobID: job.ID, SelectedBulletIDs: selectedIDs})
		if err != nil {
			stage("resume", "Resume assembler", "failed", err.Error())
			return result, err
		}
		stage("human", "Human editor", "ok", "language reviewed in the resume assembler")
		result.Validation, err = s.ValidateResumeJSON(result.Resume, job.ID)
		if err != nil {
			stage("validate", "Resume validator", "failed", err.Error())
			return result, err
		}
		result.ResumeGenerated = true
		result.Fit = buildTailoredResumeFitAnalysis(job.ID, result.Requirements, result.Resume, result.Validation)
		stage("final_fit", "Final fit", "ok", result.Fit.Recommendation)
		if result.Validation.Passed {
			stage("validate", "Resume validator", "ok", "resume passed validation")
		} else {
			stage("validate", "Resume validator", "warning", pluralCount(len(result.Validation.Errors)+len(result.Validation.Warnings), "issue"))
		}
	}

	_ = s.LogEvent("info", "job agent workflow completed")
	return result, nil
}

func normalizeJobAgentWorkflowInput(input JobAgentWorkflowInput) JobAgentWorkflowInput {
	if input.MinSelectedBullets <= 0 {
		input.MinSelectedBullets = 4
	}
	if input.MaxSelectedBullets <= 0 {
		input.MaxSelectedBullets = 10
	}
	return input
}

func countSelectedDrafts(drafts []TailoredBulletDraft) int {
	return len(selectedDraftIDs(drafts))
}

func selectedDraftIDs(drafts []TailoredBulletDraft) []int64 {
	ids := make([]int64, 0, len(drafts))
	for _, draft := range drafts {
		if draft.SelectedForResume {
			ids = append(ids, draft.ID)
		}
	}
	return ids
}

func buildTailoredResumeFitAnalysis(jobID int64, requirements []JobRequirement, resume ResumeJSON, validation ValidationResult) JobFitAnalysis {
	resumeText := strings.ToLower(resumeToPlainText(resume))
	analyses := make([]FitNeedAnalysis, 0, len(requirements))
	coveredRequired := 0
	totalRequired := 0
	partial := 0
	gaps := []string{}
	strengths := []string{}
	for _, req := range requirements {
		if req.Priority == "required" {
			totalRequired++
		}
		terms := requirementResumeTerms(req)
		matched := 0
		for _, term := range terms {
			if containsNormalizedTerm(resumeText, term) {
				matched++
			}
		}
		strength := "missing"
		gap := "high"
		if len(terms) > 0 && matched >= maxResumeFitInt(1, len(terms)/2) {
			strength = "strong"
			gap = "none"
			if req.Priority == "required" {
				coveredRequired++
			}
			if len(strengths) < 4 {
				strengths = append(strengths, req.RequirementText)
			}
		} else if matched > 0 {
			strength = "partial"
			gap = "medium"
			partial++
		} else if req.Priority == "required" && len(gaps) < 5 {
			gaps = append(gaps, req.RequirementText)
		}
		analyses = append(analyses, FitNeedAnalysis{RequirementID: req.ID, JDNeed: req.RequirementText, EvidenceStrength: strength, GapLevel: gap, Confidence: "resume_text", Risk: "based on tailored resume text"})
	}
	score := 45
	if totalRequired > 0 {
		score = 35 + int(float64(coveredRequired)/float64(totalRequired)*55)
	} else if len(requirements) > 0 {
		score = 55
	}
	score += minResumeFitInt(partial*2, 8)
	score -= len(validation.Errors) * 8
	score -= len(validation.Warnings) * 2
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	recommendation := fitRecommendation(score, totalRequired, coveredRequired, coveredRequired)
	reality := fitRealityCheck(score, len(requirements), totalRequired, coveredRequired, coveredRequired, partial, 0, len(validation.Errors)+len(validation.Warnings)) + " Final score is based on the tailored resume draft, not just pre-resume evidence."
	return JobFitAnalysis{JobID: jobID, OverallScore: score, Recommendation: recommendation, Strengths: strengths, CriticalGaps: gaps, RealityCheck: reality, Analysis: analyses, CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
}

func resumeToPlainText(resume ResumeJSON) string {
	parts := []string{resume.Headline, resume.Summary, resume.SkillsLine}
	for _, skill := range resume.Skills {
		parts = append(parts, skill.Category, strings.Join(skill.Items, " "))
	}
	for _, entry := range append(resume.Experience, resume.Projects...) {
		parts = append(parts, entry.Company, entry.Title, strings.Join(entry.Bullets, " "))
	}
	return strings.Join(parts, " ")
}

func requirementResumeTerms(req JobRequirement) []string {
	terms := []string{}
	for _, kw := range req.Keywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if len(kw) > 1 {
			terms = append(terms, kw)
		}
	}
	if len(terms) == 0 {
		for _, word := range strings.Fields(strings.ToLower(req.RequirementText)) {
			word = strings.Trim(word, ".,;:()[]{}")
			if len(word) > 1 && !commonResumeFitWord(word) {
				terms = append(terms, word)
			}
		}
	}
	return normalizeStringList(terms)
}

func commonResumeFitWord(word string) bool {
	common := map[string]bool{"experience": true, "strong": true, "ability": true, "working": true, "knowledge": true, "skills": true, "build": true, "develop": true, "support": true, "using": true, "with": true, "team": true}
	return common[word]
}

func minResumeFitInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxResumeFitInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
