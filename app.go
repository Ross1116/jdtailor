package main

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"sync"
)

const appVersion = "0.1.0"

// App struct
type App struct {
	ctx                 context.Context
	store               *Store
	contextAgentMu      sync.Mutex
	contextAgentWorkers map[int64]context.CancelFunc
}

func NewApp() *App {
	return &App{contextAgentWorkers: make(map[int64]context.CancelFunc)}
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
	a.contextAgentMu.Lock()
	for runID, cancel := range a.contextAgentWorkers {
		cancel()
		delete(a.contextAgentWorkers, runID)
	}
	a.contextAgentMu.Unlock()

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

func (a *App) StartContextAgent(sourceID int64) (ContextAgentRun, error) {
	if err := a.ensureStore(); err != nil {
		return ContextAgentRun{}, err
	}
	run, err := a.store.StartContextAgent(sourceID)
	if err != nil {
		return ContextAgentRun{}, err
	}
	a.startContextAgentWorker(run)
	return run, nil
}

func (a *App) StopContextAgent(runID int64) (ContextAgentRun, error) {
	if err := a.ensureStore(); err != nil {
		return ContextAgentRun{}, err
	}
	if runID <= 0 {
		return ContextAgentRun{}, errors.New("run id is required")
	}

	a.contextAgentMu.Lock()
	cancel := a.contextAgentWorkers[runID]
	if cancel != nil {
		delete(a.contextAgentWorkers, runID)
	}
	a.contextAgentMu.Unlock()

	if cancel != nil {
		cancel()
	}

	return a.store.CancelContextAgentRun(runID, "cancelled by user")
}

func (a *App) GetContextAgentRun(runID int64) (ContextAgentRun, error) {
	if err := a.ensureStore(); err != nil {
		return ContextAgentRun{}, err
	}
	run, err := a.store.GetContextAgentRun(runID)
	if err != nil {
		return ContextAgentRun{}, err
	}
	// Start/revive the background worker for this run. This is idempotent:
	// startContextAgentWorker uses a mutexed contextAgentWorkers map to prevent
	// duplicate goroutines. If the run is already complete/cancelled this is a
	// no-op. See TestAppRevivesQueuedTest and recoverCompletedContextAgentRun.
	a.startContextAgentWorker(run)
	return run, nil
}

func (a *App) ListContextAgentRuns(sourceID int64) ([]ContextAgentRun, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	runs, err := a.store.ListContextAgentRuns(sourceID)
	if err != nil {
		return nil, err
	}
	// Start/revive background workers for each returned run. This is idempotent:
	// startContextAgentWorker uses a mutexed contextAgentWorkers map to prevent
	// duplicate goroutines. Already-complete/cancelled runs are no-ops.
	for _, run := range runs {
		a.startContextAgentWorker(run)
	}
	return runs, nil
}

func (a *App) ListContextAgentSteps(runID int64) ([]ContextAgentStep, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListContextAgentSteps(runID)
}

func (a *App) BuildResumeContext(sourceID int64) (ResumeContext, error) {
	if err := a.ensureStore(); err != nil {
		return ResumeContext{}, err
	}
	return a.store.BuildResumeContext(sourceID)
}

func (a *App) startContextAgentWorker(run ContextAgentRun) {
	if run.Status != contextAgentStatusRunning {
		return
	}

	a.contextAgentMu.Lock()
	if a.contextAgentWorkers == nil {
		a.contextAgentWorkers = make(map[int64]context.CancelFunc)
	}
	if _, exists := a.contextAgentWorkers[run.ID]; exists {
		a.contextAgentMu.Unlock()
		return
	}

	baseCtx := a.ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	ctx, cancel := context.WithCancel(baseCtx)
	a.contextAgentWorkers[run.ID] = cancel
	a.contextAgentMu.Unlock()

	go func(runID int64) {
		defer func() {
			a.contextAgentMu.Lock()
			delete(a.contextAgentWorkers, runID)
			a.contextAgentMu.Unlock()
		}()

		if _, err := a.store.RunContextAgent(ctx, runID, nil); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			a.store.Logger().Error("context agent failed", "run_id", runID, "error", err)
		}
	}(run.ID)
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

func (a *App) ListBulletGenerationEvents(jobID int64) ([]BulletGenerationEvent, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListBulletGenerationEvents(jobID)
}

func (a *App) UpdateTailoredBulletDraft(input UpdateTailoredBulletDraftInput) (TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return TailoredBulletDraft{}, err
	}
	return a.store.UpdateTailoredBulletDraft(input)
}

func (a *App) SelectTailoredBulletDraft(input SelectTailoredBulletDraftInput) (TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return TailoredBulletDraft{}, err
	}
	return a.store.SelectTailoredBulletDraft(input)
}

func (a *App) AutoSelectResumeBullets(jobID int64) ([]TailoredBulletDraft, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.AutoSelectResumeBullets(jobID)
}

func (a *App) DeleteTailoredBulletDraft(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteTailoredBulletDraft(input)
}

func (a *App) GenerateCandidateClaims() ([]CandidateClaim, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.GenerateCandidateClaims(a.ctx, nil)
}

func (a *App) ListCandidateClaims(status string) ([]CandidateClaim, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListCandidateClaims(status)
}

func (a *App) UpdateCandidateClaimReview(input UpdateCandidateClaimReviewInput) (CandidateClaim, error) {
	if err := a.ensureStore(); err != nil {
		return CandidateClaim{}, err
	}
	return a.store.UpdateCandidateClaimReview(input)
}

func (a *App) DeleteCandidateClaim(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteCandidateClaim(input)
}

func (a *App) DeleteAllCandidateClaims() error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteAllCandidateClaims()
}

func (a *App) ListBlockedClaims() ([]BlockedClaim, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListBlockedClaims()
}

func (a *App) CreateBlockedClaim(input CreateBlockedClaimInput) (BlockedClaim, error) {
	if err := a.ensureStore(); err != nil {
		return BlockedClaim{}, err
	}
	return a.store.CreateBlockedClaim(input)
}

func (a *App) UpdateBlockedClaim(input UpdateBlockedClaimInput) (BlockedClaim, error) {
	if err := a.ensureStore(); err != nil {
		return BlockedClaim{}, err
	}
	return a.store.UpdateBlockedClaim(input)
}

func (a *App) DeleteBlockedClaim(input DeleteInput) error {
	if err := a.ensureStore(); err != nil {
		return err
	}
	return a.store.DeleteBlockedClaim(input)
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

func (a *App) GenerateResumeJSON(input GenerateResumeJSONInput) (ResumeJSON, error) {
	if err := a.ensureStore(); err != nil {
		return ResumeJSON{}, err
	}
	return a.store.GenerateResumeJSON(a.ctx, input)
}

func (a *App) ValidateResumeJSON(resume ResumeJSON, jobID int64) (ValidationResult, error) {
	if err := a.ensureStore(); err != nil {
		return ValidationResult{}, err
	}
	return a.store.ValidateResumeJSON(resume, jobID)
}

func (a *App) RenderResumePDF(resume ResumeJSON) (RenderPDFResult, error) {
	if err := a.ensureStore(); err != nil {
		return RenderPDFResult{}, err
	}
	return a.store.RenderResumePDF(a.ctx, resume)
}

func (a *App) OpenFolder(path string) error {
	if path == "" {
		return errors.New("path is required")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func (a *App) SaveResumeVersion(version ResumeVersion) (ResumeVersion, error) {
	if err := a.ensureStore(); err != nil {
		return ResumeVersion{}, err
	}
	return a.store.SaveResumeVersion(version)
}

func (a *App) GetResumeVersion(id int64) (ResumeVersion, error) {
	if err := a.ensureStore(); err != nil {
		return ResumeVersion{}, err
	}
	return a.store.GetResumeVersion(id)
}

func (a *App) ListResumeVersions(jobID int64) ([]ResumeVersion, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListResumeVersions(jobID)
}

func (a *App) SaveApplication(app Application) (Application, error) {
	if err := a.ensureStore(); err != nil {
		return Application{}, err
	}
	return a.store.SaveApplication(app)
}

func (a *App) GetApplication(id int64) (Application, error) {
	if err := a.ensureStore(); err != nil {
		return Application{}, err
	}
	return a.store.GetApplication(id)
}

func (a *App) ListApplications() ([]Application, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListApplications()
}

func (a *App) UpdateApplicationStatus(id int64, status string) (Application, error) {
	if err := a.ensureStore(); err != nil {
		return Application{}, err
	}
	return a.store.UpdateApplicationStatus(id, status)
}

func (a *App) LogCorrection(correction CorrectionLog) (CorrectionLog, error) {
	if err := a.ensureStore(); err != nil {
		return CorrectionLog{}, err
	}
	return a.store.LogCorrection(correction)
}

func (a *App) ListCorrections(applicationID int64) ([]CorrectionLog, error) {
	if err := a.ensureStore(); err != nil {
		return nil, err
	}
	return a.store.ListCorrections(applicationID)
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
