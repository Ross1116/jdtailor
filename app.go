package main

import (
	"context"
)

const appVersion = "0.1.0"

// App struct
type App struct {
	ctx   context.Context
	store *Store
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	store, err := NewStore(".")
	if err != nil {
		println("startup error:", err.Error())
		return
	}

	a.store = store
	if err := a.store.LogEvent("info", "app started"); err != nil {
		a.store.Logger().Error("failed to record app start", "error", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			println("shutdown error:", err.Error())
		}
	}
}

func (a *App) GetHealth() (Health, error) {
	if err := a.ensureStore(); err != nil {
		return Health{}, err
	}

	return a.store.Health(appVersion), nil
}

func (a *App) GetSettings() (Settings, error) {
	if err := a.ensureStore(); err != nil {
		return Settings{}, err
	}

	return a.store.GetSettings()
}

func (a *App) GetToolStatus() (ToolStatus, error) {
	if err := a.ensureStore(); err != nil {
		return ToolStatus{}, err
	}

	return a.store.ToolStatus(), nil
}

func (a *App) SaveSettings(input SaveSettingsInput) (Settings, error) {
	if err := a.ensureStore(); err != nil {
		return Settings{}, err
	}

	return a.store.SaveSettings(input)
}

func (a *App) SaveAPIKey(input SaveAPIKeyInput) (ToolStatus, error) {
	if err := a.ensureStore(); err != nil {
		return ToolStatus{}, err
	}

	return a.store.SaveAPIKey(input)
}

func (a *App) TestLLM() (LLMTestResult, error) {
	if err := a.ensureStore(); err != nil {
		return LLMTestResult{}, err
	}

	return a.store.TestLLM(a.ctx, nil)
}

func (a *App) InstallTectonic() (InstallTectonicResult, error) {
	if err := a.ensureStore(); err != nil {
		return InstallTectonicResult{}, err
	}

	return a.store.InstallTectonic(a.ctx)
}

func (a *App) RenderSamplePDF() (RenderPDFResult, error) {
	if err := a.ensureStore(); err != nil {
		return RenderPDFResult{}, err
	}

	return a.store.RenderSamplePDF(a.ctx)
}

func (a *App) GetRecentEvents() ([]AppEvent, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.GetRecentEvents(20)
}

func (a *App) GetCandidateProfile() (CandidateProfile, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateProfile{}, err
	}

	return a.store.GetCandidateProfile()
}

func (a *App) SaveCandidateProfile(input CandidateProfile) (CandidateProfile, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateProfile{}, err
	}

	return a.store.SaveCandidateProfile(input)
}

func (a *App) DraftCandidateProfileFromSource(sourceID int64) (CandidateProfile, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateProfile{}, err
	}

	return a.store.DraftCandidateProfileFromSource(sourceID)
}

func (a *App) ListCandidateSources() ([]CandidateSource, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListCandidateSources()
}

func (a *App) CreateCandidateSource(input CreateCandidateSourceInput) (CandidateSource, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateSource{}, err
	}

	return a.store.CreateCandidateSource(input)
}

func (a *App) ImportCandidateSourceFile(input ImportCandidateSourceFileInput) (CandidateSource, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateSource{}, err
	}

	return a.store.ImportCandidateSourceFile(input)
}

func (a *App) DeleteCandidateSource(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteCandidateSource(input)
}

func (a *App) ListSourceSections(sourceID int64) ([]SourceSection, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListSourceSections(sourceID)
}

func (a *App) DetectSourceSections(sourceID int64) ([]SourceSection, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.DetectSourceSections(sourceID)
}

func (a *App) UpdateSourceSection(input UpdateSourceSectionInput) (SourceSection, error) {
	if err := a.ensureStore(); err != nil {
		return SourceSection{}, err
	}

	return a.store.UpdateSourceSection(input)
}

func (a *App) DeleteSourceSection(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteSourceSection(input)
}

func (a *App) ExtractEvidenceFacts(input ExtractEvidenceFactsInput) ([]EvidenceFact, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ExtractEvidenceFacts(a.ctx, input, nil)
}

func (a *App) ListEvidenceFacts(status string) ([]EvidenceFact, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}

	return a.store.ListEvidenceFacts(status)
}

func (a *App) UpdateEvidenceFactReview(input UpdateEvidenceFactReviewInput) (EvidenceFact, error) {
	if err := a.ensureStore(); err != nil {
		return EvidenceFact{}, err
	}

	return a.store.UpdateEvidenceFactReview(input)
}

func (a *App) DeleteEvidenceFact(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteEvidenceFact(input)
}

func (a *App) DeleteAllEvidenceFacts() error {
	if err := a.ensureStore(); err != nil {
		return err
	}

	return a.store.DeleteAllEvidenceFacts()
}

func (a *App) ListJobDescriptions() ([]JobDescription, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListJobDescriptions()
}

func (a *App) CreateJobDescription(input CreateJobDescriptionInput) (JobDescription, error) {
	if err := a.ensureStore(); err != nil {
		return JobDescription{}, err
	}
	return a.store.CreateJobDescription(input)
}

func (a *App) UpdateJobDescription(input UpdateJobDescriptionInput) (JobDescription, error) {
	if err := a.ensureStore(); err != nil {
		return JobDescription{}, err
	}
	return a.store.UpdateJobDescription(input)
}

func (a *App) DeleteJobDescription(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteJobDescription(input)
}

func (a *App) ParseJobDescription(jobID int64) ([]JobRequirement, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ParseJobDescription(a.ctx, jobID, nil)
}

func (a *App) BuildJobMatchMap(jobID int64) ([]JobFactMatch, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.BuildJobMatchMap(a.ctx, jobID, nil)
}

func (a *App) GenerateTailoredBulletDrafts(jobID int64) ([]TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.GenerateTailoredBulletDrafts(a.ctx, jobID, nil)
}

func (a *App) ListJobRequirements(jobID int64) ([]JobRequirement, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListJobRequirements(jobID)
}

func (a *App) ListJobFactMatches(jobID int64) ([]JobFactMatch, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListJobFactMatches(jobID)
}

func (a *App) ListTailoredBulletDrafts(jobID int64) ([]TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListTailoredBulletDrafts(jobID)
}

func (a *App) UpdateTailoredBulletDraft(input UpdateTailoredBulletDraftInput) (TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return TailoredBulletDraft{}, err
	}
	return a.store.UpdateTailoredBulletDraft(input)
}

func (a *App) DeleteTailoredBulletDraft(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteTailoredBulletDraft(input)
}

func (a *App) ListPromptRules() ([]PromptRule, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListPromptRules()
}

func (a *App) UpdatePromptRule(input UpdatePromptRuleInput) (PromptRule, error) {
	if err := a.ensureStore(); err != nil {
		return PromptRule{}, err
	}
	return a.store.UpdatePromptRule(input)
}

func (a *App) ListPromptResearchSources() ([]PromptResearchSource, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListPromptResearchSources()
}

func (a *App) AnalyzeJobDescription(jobID int64) (JobAnalysis, error) {
	if err := a.ensureStore(); err != nil {
		return JobAnalysis{}, err
	}
	return a.store.AnalyzeJobDescription(jobID)
}

func (a *App) GetJobAnalysis(jobID int64) (JobAnalysis, error) {
	if err := a.ensureStore(); err != nil {
		return JobAnalysis{}, err
	}
	return a.store.GetJobAnalysis(jobID)
}

func (a *App) GenerateFitAnalysis(jobID int64) (JobFitAnalysis, error) {
	if err := a.ensureStore(); err != nil {
		return JobFitAnalysis{}, err
	}
	return a.store.GenerateFitAnalysis(jobID)
}

func (a *App) GetFitAnalysis(jobID int64) (JobFitAnalysis, error) {
	if err := a.ensureStore(); err != nil {
		return JobFitAnalysis{}, err
	}
	return a.store.GetFitAnalysis(jobID)
}

func (a *App) GenerateApplicationStrategy(jobID int64) (ApplicationStrategy, error) {
	if err := a.ensureStore(); err != nil {
		return ApplicationStrategy{}, err
	}
	return a.store.GenerateApplicationStrategy(jobID)
}

func (a *App) GetApplicationStrategy(jobID int64) (ApplicationStrategy, error) {
	if err := a.ensureStore(); err != nil {
		return ApplicationStrategy{}, err
	}
	return a.store.GetApplicationStrategy(jobID)
}

func (a *App) ensureStore() error {
	if a.store != nil {
		return nil
	}

	store, err := NewStore(".")
	if err != nil {
		return err
	}
	a.store = store
	return nil
}
