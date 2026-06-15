import {FormEvent, useEffect, useMemo, useRef, useState} from 'react';
import {
  Activity,
  AlertCircle,
  Bot,
  BriefcaseBusiness,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clipboard,
  Cpu,
  Database,
  FilePlus2,
  FileText,
  Folder,
  Gauge,
  KeyRound,
  Layers3,
  ListChecks,
  LoaderCircle,
  Play,
  RefreshCcw,
  Save,
  Search,
  Settings as SettingsIcon,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  Trash2,
  Upload,
  UserRound,
  Wrench,
  Square,
  X,
  ExternalLink,
  Plus,
  Eye,
  Pencil,
} from 'lucide-react';
import {
  AnalyzeJobDescription,
  ApplicationStrategy,
  AutoSelectResumeBullets,
  BuildResumeContext,
  BulletGenerationEvent,
  CandidateProfile,
  CandidateProfileRecord,
  CandidateSource,
  BuildJobMatchMap,
  BlockedClaim,
  CreateCandidateSource,
  CreateBlockedClaim,
  CreateJobDescription,
  DeleteCandidateSource,
  DeleteAllCandidateClaims,
  DeleteAllEvidenceFacts,
  DeleteBlockedClaim,
  DeleteCandidateClaim,
  DeleteEvidenceFact,
  DeleteJobDescription,
  DeleteSourceSection,
  DeleteTailoredBulletDraft,
  DetectSourceSections,
  DraftCandidateProfileFromSource,
  EvidenceFact,
  ExtractEvidenceFacts,
  GenerateCandidateClaims,
  GenerateTailoredBulletDrafts,
  GenerateApplicationStrategy,
  GenerateFitAnalysis,
  GetCandidateProfile,
  GetContextAgentRun,
  GetApplicationStrategy,
  GetFitAnalysis,
  GetHealth,
  GetJobAnalysis,
  GetRecentEvents,
  GetSettings,
  GetToolStatus,
  JobDescription,
  JobAnalysis,
  JobFactMatch,
  JobFitAnalysis,
  JobRequirement,
  ListCandidateSources,
  ListBlockedClaims,
  ListCandidateClaims,
  ListContextAgentRuns,
  ListContextAgentSteps,
  ListEvidenceFacts,
  ListJobDescriptions,
  ListJobFactMatches,
  ListJobRequirements,
  ListBulletGenerationEvents,
  ListPromptResearchSources,
  ListPromptRules,
  ListSourceSections,
  ListTailoredBulletDrafts,
  RenderSamplePDF,
  SaveAPIKey,
  SaveCandidateProfile,
  SaveSettings,
  SelectTailoredBulletDraft,
  SourceSection,
  ContextAgentRun,
  ContextAgentStep,
  TailoredBulletDraft,
  TestLLM,
  UpdateEvidenceFactReview,
  UpdateJobDescription,
  UpdateSourceSection,
  UpdateTailoredBulletDraft,
  InstallTectonic,
  ParseJobDescription,
  PromptResearchSource,
  PromptRule,
  CandidateClaim,
  UpdateBlockedClaim,
  UpdateCandidateClaimReview,
  UpdatePromptRule,
  StartContextAgent,
  StopContextAgent,
  GenerateResumeJSON,
  ValidateResumeJSON,
  RenderResumePDF,
  SaveResumeVersion,
  ListResumeVersions,
  SaveApplication,
  GetApplication,
  ListApplications,
  UpdateApplicationStatus,
  LogCorrection,
  ListCorrections,
  ResumeJSON,
  ResumeVersion,
  Application,
  CorrectionLog,
  ValidationResult,
  GenerateResumeJSONInput,
} from './backend';

type Health = {
  version: string;
  storage_status: string;
  db_path: string;
  log_path: string;
  generated_path: string;
  pdf_renderer: string;
};

type Settings = {
  provider: string;
  model: string;
  embedding_model: string;
  api_key_configured: boolean;
};

type ToolStatus = {
  api_key_configured: boolean;
  api_key_source: string;
  env_local_path: string;
  tectonic_status: string;
  tectonic_path: string;
  generated_path: string;
};

type LLMTestResult = {
  success: boolean;
  provider: string;
  model: string;
  text: string;
  latency_ms: number;
  status_code: number;
  error: string;
};

type RenderPDFResult = {
  success: boolean;
  tex_path: string;
  pdf_path: string;
  error: string;
};

type AppEvent = {
  id: number;
  level: string;
  message: string;
  created_at: string;
};

type WorkItem = {
  key: string;
  title: string;
  detail: string;
  progress: number;
  kind?: 'context-agent' | 'action';
  runID?: number;
};

type LoadState = 'loading' | 'ready' | 'error';
type View = 'pipeline' | 'sources' | 'settings';
type PipelineStep = 'job' | 'bullets' | 'resume' | 'tracker';
type JobDraft = {company: string; title: string; url: string; raw_text: string};

const emptyProfile: CandidateProfile = {
  contact: {
    full_name: '',
    email: '',
    phone: '',
    location: '',
    linkedin: '',
    github: '',
    portfolio: '',
    links: [],
    verified: false,
    updated_at: '',
  },
  records: [],
};

const APP_STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  ready_to_apply: 'Ready',
  applied: 'Applied',
  interviewing: 'Interviewing',
  rejected: 'Rejected',
  offer: 'Offer',
};

function App() {
  const [activeView, setActiveView] = useState<View>('pipeline');
  const [pipelineStep, setPipelineStep] = useState<PipelineStep>('job');
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [error, setError] = useState('');
  const [runtimeError, setRuntimeError] = useState('');
  const [health, setHealth] = useState<Health | null>(null);
  const [settings, setSettings] = useState<Settings>({
    provider: 'openrouter',
    model: '',
    embedding_model: '',
    api_key_configured: false,
  });
  const [toolStatus, setToolStatus] = useState<ToolStatus | null>(null);
  const [events, setEvents] = useState<AppEvent[]>([]);
  const [profile, setProfile] = useState<CandidateProfile>(emptyProfile);
  const [sources, setSources] = useState<CandidateSource[]>([]);
  const [sections, setSections] = useState<SourceSection[]>([]);
  const [facts, setFacts] = useState<EvidenceFact[]>([]);
  const [claims, setClaims] = useState<CandidateClaim[]>([]);
  const [blockedClaims, setBlockedClaims] = useState<BlockedClaim[]>([]);
  const [contextRuns, setContextRuns] = useState<ContextAgentRun[]>([]);
  const [contextSteps, setContextSteps] = useState<ContextAgentStep[]>([]);
  const [jobs, setJobs] = useState<JobDescription[]>([]);
  const [jobRequirements, setJobRequirements] = useState<JobRequirement[]>([]);
  const [jobMatches, setJobMatches] = useState<JobFactMatch[]>([]);
  const [bulletDrafts, setBulletDrafts] = useState<TailoredBulletDraft[]>([]);
  const [bulletEvents, setBulletEvents] = useState<BulletGenerationEvent[]>([]);
  const [jobAnalysis, setJobAnalysis] = useState<JobAnalysis | null>(null);
  const [fitAnalysis, setFitAnalysis] = useState<JobFitAnalysis | null>(null);
  const [applicationStrategy, setApplicationStrategy] = useState<ApplicationStrategy | null>(null);
  const [selectedSourceID, setSelectedSourceID] = useState<number>(0);
  const [selectedJobID, setSelectedJobID] = useState<number>(0);
  const [sourceDraft, setSourceDraft] = useState({
    source_type: 'current_resume',
    trust_tier: 'verified',
    title: '',
    raw_text: '',
  });
  const [jobDraft, setJobDraft] = useState<JobDraft>({
    company: '',
    title: '',
    url: '',
    raw_text: '',
  });
  const [apiKey, setAPIKey] = useState('');
  const [llmResult, setLLMResult] = useState<LLMTestResult | null>(null);
  const [pdfResult, setPDFResult] = useState<RenderPDFResult | null>(null);
  const [busyAction, setBusyAction] = useState('');
  const [applications, setApplications] = useState<Application[]>([]);
  const [resumeVersions, setResumeVersions] = useState<ResumeVersion[]>([]);
  const [activeResume, setActiveResume] = useState<ResumeJSON | null>(null);
  const [editingResume, setEditingResume] = useState<ResumeJSON | null>(null);
  const [activeValidation, setActiveValidation] = useState<ValidationResult | null>(null);
  const [selectedApplicationID, setSelectedApplicationID] = useState<number>(0);
  const [applicationNotes, setApplicationNotes] = useState('');
  const pollingContextRunIDs = useRef<Set<number>>(new Set());
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});
  const [showAddJob, setShowAddJob] = useState(false);
  const [showImportSource, setShowImportSource] = useState(false);
  const [editingProfile, setEditingProfile] = useState(false);

  const selectedSource = sources.find((source) => source.id === selectedSourceID);
  const selectedJob = jobs.find((job) => job.id === selectedJobID);
  const selectedApplicationInfo = applications.find((a) => a.id === selectedApplicationID);
  const queuedFacts = facts.filter((fact) => fact.status === 'needs_review');
  const queuedClaims = claims.filter((claim) => claim.status === 'needs_review');
  const apiConfigured = toolStatus?.api_key_configured ?? settings.api_key_configured;
  const tectonicStatus = toolStatus?.tectonic_status ?? health?.pdf_renderer ?? 'checking';
  const runningContextRuns = contextRuns.filter((run) => run.status === 'running');

  const jobAppsMap = useMemo(() => {
    const map = new Map<number, Application>();
    for (const app of applications) {
      map.set(app.job_id, app);
    }
    return map;
  }, [applications]);

  const activeWorkItems: WorkItem[] = [
    ...runningContextRuns.map((run) => {
      const latestStep = latestContextStep(contextSteps, run.id);
      return {
        key: `context-${run.id}`,
        kind: 'context-agent' as const,
        runID: run.id,
        title: `Context: ${sourceTitle(sources, run.source_id)}`,
        detail: latestStep?.message || contextStageLabel(latestStep?.stage || 'queued'),
        progress: contextRunProgress(run, latestStep),
      };
    }),
    ...(busyAction ? [{...workItemForAction(busyAction), kind: 'action' as const}] : []),
  ];

  function toggleSection(key: string) {
    setExpandedSections((prev) => ({...prev, [key]: !prev[key]}));
  }

  async function load() {
    setLoadState('loading');
    setError('');
    try {
      const results = await Promise.all([
        GetHealth().catch((e: any) => { throw new Error('GetHealth: ' + e) }),
        GetSettings().catch((e: any) => { throw new Error('GetSettings: ' + e) }),
        GetToolStatus().catch((e: any) => { throw new Error('GetToolStatus: ' + e) }),
        GetRecentEvents().catch((e: any) => { throw new Error('GetRecentEvents: ' + e) }),
        GetCandidateProfile().catch((e: any) => { throw new Error('GetCandidateProfile: ' + e) }),
        ListCandidateSources().catch((e: any) => { throw new Error('ListCandidateSources: ' + e) }),
        ListSourceSections(0).catch((e: any) => { throw new Error('ListSourceSections: ' + e) }),
        ListEvidenceFacts('all').catch((e: any) => { throw new Error('ListEvidenceFacts: ' + e) }),
        ListCandidateClaims('all').catch((e: any) => { throw new Error('ListCandidateClaims: ' + e) }),
        ListBlockedClaims().catch((e: any) => { throw new Error('ListBlockedClaims: ' + e) }),
        ListContextAgentRuns(0).catch((e: any) => { throw new Error('ListContextAgentRuns: ' + e) }),
        ListJobDescriptions().catch((e: any) => { throw new Error('ListJobDescriptions: ' + e) }),
        ListApplications().catch((e: any) => { throw new Error('ListApplications: ' + e) }),
      ]);
      const [nh, ns, nst, ne, nprof, nsrc, nsec, nf, ncl, nblk, ncr, nj, na] = results;
      setHealth(nh as Health);
      setSettings(ns as Settings);
      setToolStatus(nst as ToolStatus);
      setEvents((ne ?? []) as AppEvent[]);
      setProfile(normalizeProfile(nprof as CandidateProfile));
      setSources(normalizeSources(nsrc as CandidateSource[] | null | undefined));
      setSections((nsec ?? []) as SourceSection[]);
      setFacts(normalizeFacts(nf as EvidenceFact[] | null | undefined));
      setClaims(normalizeClaims(ncl as CandidateClaim[] | null | undefined));
      setBlockedClaims(normalizeBlockedClaims(nblk as BlockedClaim[] | null | undefined));
      setContextRuns(normalizeContextRuns(ncr as ContextAgentRun[] | null | undefined));
      setJobs((nj ?? []) as JobDescription[]);
      setApplications((na ?? []) as Application[]);
      const firstSrc = (nsrc as CandidateSource[] | undefined)?.[0];
      if (!selectedSourceID && firstSrc) setSelectedSourceID(firstSrc.id);
      const firstJob = (nj as JobDescription[] | undefined)?.[0];
      if (!selectedJobID && firstJob) {
        setSelectedJobID(firstJob.id);
        setJobDraft({company: firstJob.company, title: firstJob.title, url: firstJob.url, raw_text: firstJob.raw_text});
        await refreshJobContext(firstJob.id);
      }
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      const msg = err instanceof Error ? err.message : String(err);
      setError('Load failed: ' + msg);
      setRuntimeError(msg);
      console.error('App load error:', err);
    }
  }

  async function refreshEvents() {
    const ne = await GetRecentEvents();
    setEvents((ne ?? []) as AppEvent[]);
  }

  async function refreshContextRuns(sourceID = 0) {
    const runs = (await ListContextAgentRuns(sourceID)) as ContextAgentRun[];
    setContextRuns((prev) => {
      const scoped = normalizeContextRuns(runs);
      return sourceID <= 0 ? scoped : normalizeContextRuns([...scoped, ...prev.filter((r) => r.source_id !== sourceID)]);
    });
    const latest = normalizeContextRuns(runs)[0];
    if (latest) {
      const steps = (await ListContextAgentSteps(latest.id)) as ContextAgentStep[];
      setContextSteps((prev) => normalizeContextSteps([...prev.filter((s) => s.run_id !== latest.id), ...steps]));
    }
  }

  function ensureContextAgentPolling(runID: number, sourceID: number) {
    if (!runID || pollingContextRunIDs.current.has(runID)) return;
    pollingContextRunIDs.current.add(runID);
    void pollContextAgent(runID, sourceID).finally(() => pollingContextRunIDs.current.delete(runID));
  }

  async function pollContextAgent(runID: number, sourceID: number) {
    let waitMS = 1000;
    for (;;) {
      await delay(waitMS);
      let normalizedRun: ContextAgentRun;
      let steps: ContextAgentStep[];
      try {
        const [run, ns] = await Promise.all([GetContextAgentRun(runID), ListContextAgentSteps(runID)]);
        normalizedRun = run as ContextAgentRun;
        steps = normalizeContextSteps(ns as ContextAgentStep[]);
      } catch (err) { setLoadState('error'); setError(err instanceof Error ? err.message : String(err)); return; }
      setContextRuns((prev) => normalizeContextRuns([normalizedRun, ...prev.filter((r) => r.id !== normalizedRun.id)]));
      setContextSteps((prev) => normalizeContextSteps([...prev.filter((s) => s.run_id !== runID), ...steps]));
      if (normalizedRun.status !== 'running') {
        await refreshWorkflow();
        if (normalizedRun.status === 'complete') {
          const sid = sourceID || normalizedRun.source_id;
          const secs = (await ListSourceSections(sid)) as SourceSection[];
          if (secs[0]) setSelectedJobID(selectedJobID);
          const np = (await GetCandidateProfile()) as CandidateProfile;
          setProfile(normalizeProfile(np));
          await BuildResumeContext(sid).catch(() => null);
        } else if (normalizedRun.status === 'failed' && normalizedRun.error) {
          setLoadState('error'); setError(normalizedRun.error);
        }
        return;
      }
      waitMS = Math.min(3000, waitMS + 500);
    }
  }

  async function beginContextAgent(sourceID: number) {
    const run = (await StartContextAgent(sourceID)) as ContextAgentRun;
    setContextRuns((prev) => normalizeContextRuns([run, ...prev.filter((r) => r.id !== run.id)]));
    const steps = (await ListContextAgentSteps(run.id)) as ContextAgentStep[];
    setContextSteps((prev) => normalizeContextSteps([...prev.filter((s) => s.run_id !== run.id), ...steps]));
    if (run.status === 'running') ensureContextAgentPolling(run.id, sourceID);
    return run;
  }

  async function refreshWorkflow() {
    const [nSrc, nSec, nF, nCl, nRuns, nJobs] = await Promise.all([
      ListCandidateSources(), ListSourceSections(0), ListEvidenceFacts('all'),
      ListCandidateClaims('all'), ListContextAgentRuns(0), ListJobDescriptions(),
    ]);
    setSources(normalizeSources(nSrc as CandidateSource[] | null | undefined));
    setSections((nSec ?? []) as SourceSection[]);
    setFacts(normalizeFacts(nF as EvidenceFact[] | null | undefined));
    setClaims(normalizeClaims(nCl as CandidateClaim[] | null | undefined));
    setContextRuns(normalizeContextRuns(nRuns as ContextAgentRun[] | null | undefined));
    setJobs((nJobs ?? []) as JobDescription[]);
    const na = (await ListApplications()) as Application[];
    setApplications(na);
    await refreshEvents();
  }

  async function generateResume() {
    if (!selectedJobID) return;
    await runAction('generate-resume', async () => {
      const input: GenerateResumeJSONInput = {
        job_id: selectedJobID,
        selected_bullet_ids: bulletDrafts.filter((d) => d.selected_for_resume).map((d) => d.id),
      };
      const resume = normalizeResumeJSON((await GenerateResumeJSON(input)) as ResumeJSON | null | undefined);
      setActiveResume(resume);
      const validation = normalizeValidationResult((await ValidateResumeJSON(resume, selectedJobID)) as ValidationResult | null | undefined);
      setActiveValidation(validation);
      const version = (await SaveResumeVersion({
        job_id: selectedJobID, resume_json: resume, tex_source: '', pdf_path: '', validation_result: validation,
      })) as ResumeVersion;
      setResumeVersions((prev) => [normalizeResumeVersion(version), ...prev]);
      setPipelineStep('resume');
    });
  }

  function startEditingResume() {
    if (!activeResume) return;
    setEditingResume(JSON.parse(JSON.stringify(activeResume)));
  }

  function cancelEditingResume() {
    setEditingResume(null);
  }

  async function saveEditedResume() {
    if (!editingResume) return;
    const cleaned = normalizeResumeJSON(editingResume);
    setActiveResume(cleaned);
    setEditingResume(null);
    const validation = normalizeValidationResult((await ValidateResumeJSON(cleaned, selectedJobID)) as ValidationResult | null | undefined);
    setActiveValidation(validation);
    const version = (await SaveResumeVersion({
      job_id: selectedJobID, resume_json: cleaned, tex_source: '', pdf_path: '', validation_result: validation,
    })) as ResumeVersion;
    setResumeVersions((prev) => [normalizeResumeVersion(version), ...prev]);
  }

  function updateEditingField(field: string, value: string) {
    if (!editingResume) return;
    setEditingResume({ ...editingResume, [field]: value });
  }

  function updateEditingSkill(catIdx: number, itemsStr: string) {
    if (!editingResume) return;
    const skills = [...editingResume.skills];
    skills[catIdx] = { ...skills[catIdx], items: itemsStr.split(',').map(s => s.trim()).filter(Boolean) };
    setEditingResume({ ...editingResume, skills });
  }

  function updateEditingEntry(section: 'experience' | 'projects' | 'education', idx: number, field: string, value: string) {
    if (!editingResume) return;
    const entries = [...editingResume[section]];
    entries[idx] = { ...entries[idx], [field]: value };
    setEditingResume({ ...editingResume, [section]: entries });
  }

  function updateEditingBullet(section: 'experience' | 'projects', entryIdx: number, bulletIdx: number, value: string) {
    if (!editingResume) return;
    const entries = [...editingResume[section]];
    const bullets = [...entries[entryIdx].bullets];
    bullets[bulletIdx] = value;
    entries[entryIdx] = { ...entries[entryIdx], bullets };
    setEditingResume({ ...editingResume, [section]: entries });
  }

  function addEditingBullet(section: 'experience' | 'projects', entryIdx: number) {
    if (!editingResume) return;
    const entries = [...editingResume[section]];
    entries[entryIdx] = { ...entries[entryIdx], bullets: [...entries[entryIdx].bullets, ''] };
    setEditingResume({ ...editingResume, [section]: entries });
  }

  function removeEditingBullet(section: 'experience' | 'projects', entryIdx: number, bulletIdx: number) {
    if (!editingResume) return;
    const entries = [...editingResume[section]];
    const bullets = entries[entryIdx].bullets.filter((_, i) => i !== bulletIdx);
    entries[entryIdx] = { ...entries[entryIdx], bullets };
    setEditingResume({ ...editingResume, [section]: entries });
  }

  function addEditingSkillCategory() {
    if (!editingResume) return;
    setEditingResume({ ...editingResume, skills: [...editingResume.skills, { category: 'New Category', items: [] }] });
  }

  function removeEditingSkillCategory(catIdx: number) {
    if (!editingResume) return;
    setEditingResume({ ...editingResume, skills: editingResume.skills.filter((_, i) => i !== catIdx) });
  }

  function addEditingEntry(section: 'experience' | 'projects' | 'education') {
    if (!editingResume) return;
    const newEntry = section === 'education'
      ? { organization: '', degree: '', location: '', end_date: '', start_date: '', bullets: [], claim_ids: [], bullet_ids: [] }
      : { company: '', title: '', location: '', start_date: '', end_date: '', bullets: [''], claim_ids: [], bullet_ids: [], url: '' };
    setEditingResume({ ...editingResume, [section]: [...editingResume[section], newEntry] });
  }

  function removeEditingEntry(section: 'experience' | 'projects' | 'education', idx: number) {
    if (!editingResume) return;
    setEditingResume({ ...editingResume, [section]: editingResume[section].filter((_, i) => i !== idx) });
  }

  async function exportResumeJSON() {
    if (!activeResume) return;
    const blob = new Blob([JSON.stringify(activeResume, null, 2)], {type: 'application/json'});
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `resume-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }

  async function renderPDF() {
    if (!activeResume) return;
    await runAction('render-pdf', async () => {
      const result = (await RenderResumePDF(activeResume)) as RenderPDFResult;
      if (!result.success && result.error) setError(result.error);
    });
  }

  async function saveApplication() {
    if (!selectedJobID) return;
    await runAction('save-application', async () => {
      const app = {
        id: selectedApplicationID || 0, job_id: selectedJobID, status: 'draft',
        fit_score: fitAnalysis?.overall_score || 0, resume_version_id: resumeVersions[0]?.id || 0,
        cover_letter_version_id: 0, notes: applicationNotes,
      } as Application;
      const saved = (await SaveApplication(app)) as Application;
      setSelectedApplicationID(saved.id);
      const na = (await ListApplications()) as Application[];
      setApplications(na);
    });
  }

  async function updateApplicationStatus(id: number, status: string) {
    await runAction(`upd-app-${id}`, async () => {
      const u = (await UpdateApplicationStatus(id, status)) as Application;
      setApplications((prev) => prev.map((a) => a.id === id ? u : a));
    });
  }

  async function deleteJob(jobID: number) {
    await runAction(`del-job-${jobID}`, async () => {
      await DeleteJobDescription({id: jobID});
      if (selectedJobID === jobID) {
        setSelectedJobID(0); setActiveResume(null); setActiveValidation(null);
      }
      setJobs((await ListJobDescriptions()) as JobDescription[]);
    });
  }

  async function saveJob(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-job', async () => {
      const saved = selectedJobID
        ? (await UpdateJobDescription({id: selectedJobID, ...jobDraft})) as JobDescription
        : (await CreateJobDescription(jobDraft)) as JobDescription;
      setSelectedJobID(saved.id);
      setJobDraft({company: saved.company, title: saved.title, url: saved.url, raw_text: saved.raw_text});
      setJobs((await ListJobDescriptions()) as JobDescription[]);
      await refreshJobContext(saved.id);
      setShowAddJob(false);
    });
  }

  function selectJob(job: JobDescription) {
    setSelectedJobID(job.id);
    setJobDraft({company: job.company, title: job.title, url: job.url, raw_text: job.raw_text});
    setActiveResume(null); setActiveValidation(null);
    setPipelineStep('job');
    refreshJobContext(job.id);
  }

  function updateJobDraft(nextDraft: JobDraft) {
    setJobDraft((prev) => {
      if (nextDraft.raw_text === prev.raw_text) return nextDraft;
      const inf = inferJobDetailsFromText(nextDraft.raw_text);
      return {...nextDraft, company: nextDraft.company.trim() ? nextDraft.company : inf.company, title: nextDraft.title.trim() ? nextDraft.title : inf.title};
    });
  }

  async function refreshJobContext(jobID = selectedJobID) {
    if (!jobID) { setJobRequirements([]); setJobMatches([]); setBulletDrafts([]); setJobAnalysis(null); setFitAnalysis(null); setApplicationStrategy(null); return; }
    const [req, mat, dra, gen, ana, fit, strat] = await Promise.all([
      ListJobRequirements(jobID), ListJobFactMatches(jobID), ListTailoredBulletDrafts(jobID),
      ListBulletGenerationEvents(jobID).catch(() => []),
      GetJobAnalysis(jobID).catch(() => null), GetFitAnalysis(jobID).catch(() => null),
      GetApplicationStrategy(jobID).catch(() => null),
    ]);
    setJobRequirements(normalizeRequirements(req as JobRequirement[] | null | undefined));
    setJobMatches(normalizeMatches(mat as JobFactMatch[] | null | undefined));
    setBulletDrafts(normalizeDrafts(dra as TailoredBulletDraft[] | null | undefined));
    setJobAnalysis(normalizeJobAnalysis(ana as JobAnalysis | null | undefined));
    setFitAnalysis(normalizeFitAnalysis(fit as JobFitAnalysis | null | undefined));
    setApplicationStrategy(normalizeApplicationStrategy(strat as ApplicationStrategy | null | undefined));
  }

  async function runAction(name: string, action: () => Promise<void>) {
    setBusyAction(name); setError('');
    try { await action(); setLoadState('ready'); }
    catch (err) { setLoadState('error'); setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusyAction(''); }
  }

  function handleSaveJob(e: FormEvent) {
    e.preventDefault();
    void runAction('save-job', async () => {
      const saved = selectedJobID
        ? (await UpdateJobDescription({id: selectedJobID, ...jobDraft})) as JobDescription
        : (await CreateJobDescription(jobDraft)) as JobDescription;
      setSelectedJobID(saved.id);
      setJobDraft({company: saved.company, title: saved.title, url: saved.url, raw_text: saved.raw_text});
      setJobs((await ListJobDescriptions()) as JobDescription[]);
      await refreshJobContext(saved.id);
      setShowAddJob(false);
    });
  }

  function handleParseJD() {
    if (!selectedJobID) return;
    void runAction('parse', async () => {
      const r = (await ParseJobDescription(selectedJobID)) as JobRequirement[];
      setJobRequirements(normalizeRequirements(r));
      const a = (await AnalyzeJobDescription(selectedJobID)) as JobAnalysis;
      setJobAnalysis(normalizeJobAnalysis(a));
      setJobMatches([]); setBulletDrafts([]); setFitAnalysis(null); setApplicationStrategy(null);
    });
  }

  function handleMatch() {
    if (!selectedJobID) return;
    void runAction('match', async () => {
      const m = (await BuildJobMatchMap(selectedJobID)) as JobFactMatch[];
      setJobMatches(normalizeMatches(m));
      setFitAnalysis(null); setApplicationStrategy(null);
    });
  }

  function handleFit() {
    if (!selectedJobID) return;
    void runAction('fit', async () => {
      const f = (await GenerateFitAnalysis(selectedJobID)) as JobFitAnalysis;
      setFitAnalysis(normalizeFitAnalysis(f));
    });
  }

  function handleBullets() {
    if (!selectedJobID) return;
    void runAction('bullets', async () => {
      const d = (await GenerateTailoredBulletDrafts(selectedJobID)) as TailoredBulletDraft[];
      setBulletDrafts(normalizeDrafts(d));
      const ev = (await ListBulletGenerationEvents(selectedJobID)) as BulletGenerationEvent[];
      setBulletEvents(normalizeBulletEvents(ev));
    });
  }

  function handleStrategy() {
    if (!selectedJobID) return;
    void runAction('strategy', async () => {
      const s = (await GenerateApplicationStrategy(selectedJobID)) as ApplicationStrategy;
      setApplicationStrategy(normalizeApplicationStrategy(s));
    });
  }

  function handleEditBullet(draft: TailoredBulletDraft, newText: string) {
    void runAction(`edit-${draft.id}`, async () => {
      const saved = (await UpdateTailoredBulletDraft({
        id: draft.id, draft_text: newText, rationale: draft.rationale, status: draft.status, risk_flags: draft.risk_flags,
      })) as TailoredBulletDraft;
      setBulletDrafts((prev) => normalizeDrafts(prev.map((x) => x.id === saved.id ? saved : x)));
    });
  }

  function handleDeleteBullet(draftID: number) {
    void runAction(`del-${draftID}`, async () => {
      await DeleteTailoredBulletDraft({id: draftID});
      setBulletDrafts((prev) => prev.filter((d) => d.id !== draftID));
    });
  }

  function handleAutoSelect() {
    if (!selectedJobID) return;
    void runAction('auto-sel', async () => {
      const d = (await AutoSelectResumeBullets(selectedJobID)) as TailoredBulletDraft[];
      setBulletDrafts(normalizeDrafts(d));
    });
  }

  function handleToggleBullet(draft: TailoredBulletDraft) {
    void runAction(`sel-${draft.id}`, async () => {
      const s = (await SelectTailoredBulletDraft({id: draft.id, selected: !draft.selected_for_resume})) as TailoredBulletDraft;
      setBulletDrafts((prev) => normalizeDrafts(prev.map((x) => x.id === s.id ? s : x)));
    });
  }

  function handleAddSource(e: FormEvent) {
    e.preventDefault();
    void runAction('add-src', async () => {
      const result = (await CreateCandidateSource(sourceDraft)) as CandidateSource;
      setSelectedSourceID(result.id);
      setSourceDraft({...sourceDraft, title: '', raw_text: ''});
      await beginContextAgent(result.id);
      setShowImportSource(false);
    });
  }

  function handleStopAgent(run: ContextAgentRun) {
    void runAction(`stop-agent-${run.id}`, async () => {
      const s = (await StopContextAgent(run.id)) as ContextAgentRun;
      setContextRuns((prev) => normalizeContextRuns([s, ...prev.filter((r) => r.id !== s.id)]));
    });
  }

  function handleDeleteSource(sourceID: number) {
    void runAction(`del-src-${sourceID}`, async () => {
      await DeleteCandidateSource({id: sourceID});
      await refreshWorkflow();
    });
  }

  function handleTestSettings() {
    void (async () => {
      const r = (await SaveSettings({provider: settings.provider, model: settings.model, embedding_model: settings.embedding_model})) as Settings;
      setSettings(r);
      await load();
    })();
  }

  function handleSaveProfile() {
    void (async () => {
      const s = await SaveCandidateProfile(profile);
      setProfile(normalizeProfile(s as CandidateProfile));
      setEditingProfile(false);
    })();
  }

  useEffect(() => { load(); }, []);

  useEffect(() => { for (const run of runningContextRuns) ensureContextAgentPolling(run.id, run.source_id); }, [contextRuns]);

  const statusText = loadState === 'loading' ? 'Loading' : loadState === 'error' ? 'Needs attention' : 'Ready';

  const pipelineSteps: {key: PipelineStep; label: string; icon: React.ReactNode}[] = [
    {key: 'job', label: 'Job', icon: <BriefcaseBusiness size={14} />},
    {key: 'bullets', label: 'Bullets', icon: <ListChecks size={14} />},
    {key: 'resume', label: 'Resume', icon: <FileText size={14} />},
    {key: 'tracker', label: 'Tracker', icon: <Gauge size={14} />},
  ];

  function getJobProgress(jobID: number): {step: PipelineStep; pct: number} {
    const app = jobAppsMap.get(jobID);
    if (app && app.status !== 'draft') return {step: 'tracker', pct: 100};
    if (resumeVersions.some((v) => v.job_id === jobID)) return {step: 'resume', pct: 75};
    if (bulletDrafts.filter((d) => d.job_id === jobID).length > 0) return {step: 'bullets', pct: 50};
    return {step: 'job', pct: 10};
  }

  return (
    <div className="flex h-screen bg-[#f6f8fb]">
      <aside className="flex w-80 shrink-0 flex-col border-r border-slate-200 bg-white">
        <div className="border-b border-slate-200 px-4 py-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-900 text-white">
                <FileText size={16} />
              </div>
              <span className="text-sm font-semibold text-slate-950">JD Tailor</span>
            </div>
            <div className="flex items-center gap-1">
              <span className={`inline-block h-2 w-2 rounded-full ${loadState === 'ready' ? 'bg-green-500' : loadState === 'error' ? 'bg-red-500' : 'bg-amber-400'}`} />
              <span className="text-xs text-slate-400">{statusText}</span>
              <button
                type="button"
                onClick={load}
                disabled={busyAction !== '' || loadState === 'loading'}
                className="ml-1 inline-flex h-7 w-7 items-center justify-center rounded-md border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 hover:text-slate-800 active:translate-y-px disabled:pointer-events-none disabled:opacity-40"
                title="Refresh"
                aria-label="Refresh app data"
              >
                <RefreshCcw size={13} className={loadState === 'loading' ? 'animate-spin' : ''} />
              </button>
            </div>
          </div>
        </div>

        <div className="flex border-b border-slate-200">
          <button
            className={`flex-1 py-2.5 text-xs font-medium ${activeView === 'pipeline' ? 'border-b-2 border-slate-900 text-slate-950' : 'text-slate-500 hover:text-slate-700'}`}
            onClick={() => setActiveView('pipeline')}
          >
            Workspace
          </button>
          <button
            className={`flex-1 py-2.5 text-xs font-medium ${activeView === 'sources' ? 'border-b-2 border-slate-900 text-slate-950' : 'text-slate-500 hover:text-slate-700'}`}
            onClick={() => setActiveView('sources')}
          >
            Sources
          </button>
          <button
            className={`flex-1 py-2.5 text-xs font-medium ${activeView === 'settings' ? 'border-b-2 border-slate-900 text-slate-950' : 'text-slate-500 hover:text-slate-700'}`}
            onClick={() => setActiveView('settings')}
          >
            Settings
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-3">
          <div className="mb-3 flex items-center justify-between px-1">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jobs</span>
            <button className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600" onClick={() => setShowAddJob(!showAddJob)}>
              <Plus size={14} />
            </button>
          </div>

          {showAddJob && (
            <div className="mb-3 rounded-lg border border-slate-200 bg-slate-50 p-3">
              <form onSubmit={saveJob} className="space-y-2">
                <TextInput label="Company" value={jobDraft.company} onChange={(v) => updateJobDraft({...jobDraft, company: v})} />
                <TextInput label="Role" value={jobDraft.title} onChange={(v) => updateJobDraft({...jobDraft, title: v})} />
                <TextArea label="Job description" rows={6} value={jobDraft.raw_text} onChange={(v) => updateJobDraft({...jobDraft, raw_text: v})} />
                <div className="flex gap-2">
                  <button type="submit" className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800">Save</button>
                  <button type="button" className="rounded-md border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100" onClick={() => setShowAddJob(false)}>Cancel</button>
                </div>
              </form>
            </div>
          )}

          {jobs.length === 0 ? (
            <p className="px-1 py-4 text-center text-xs text-slate-400">No jobs yet.</p>
          ) : (
            <div className="space-y-1">
              {jobs.map((job) => {
                const app = jobAppsMap.get(job.id);
                const prog = getJobProgress(job.id);
                const isSelected = job.id === selectedJobID;
                return (
                  <button
                    key={job.id}
                    className={`w-full rounded-lg p-2.5 text-left transition-colors ${isSelected ? 'bg-slate-100 ring-1 ring-slate-300' : 'hover:bg-slate-50'}`}
                    onClick={() => selectJob(job)}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-xs font-semibold text-slate-950">{job.title || 'Untitled'}</p>
                        <p className="truncate text-xs text-slate-500">{job.company || 'No company'}</p>
                      </div>
                      <span className={`shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${app ? 'bg-blue-100 text-blue-700' : 'bg-slate-100 text-slate-500'}`}>
                        {app ? APP_STATUS_LABELS[app.status] || app.status : `${prog.pct}%`}
                      </span>
                    </div>
                    <div className="mt-2 h-1.5 w-full rounded-full bg-slate-200">
                      <div className="h-1.5 rounded-full bg-slate-600 transition-all" style={{width: `${prog.pct}%`}} />
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        <div className="border-t border-slate-200 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className={`inline-block h-2 w-2 rounded-full ${apiConfigured ? 'bg-green-500' : 'bg-slate-300'}`} />
            <span className="text-xs text-slate-500">{apiConfigured ? 'API ready' : 'API key needed'}</span>
          </div>
          {tectonicStatus === 'installed' && (
            <div className="mt-1 flex items-center gap-1">
              <span className="inline-block h-2 w-2 rounded-full bg-green-500" />
              <span className="text-xs text-slate-500">PDF ready</span>
            </div>
          )}
        </div>
      </aside>

      <main className="flex flex-1 flex-col overflow-hidden">
        {error && (
          <div className="mx-6 mt-4 flex items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">
            <AlertCircle className="mt-0.5 shrink-0" size={16} />
            <div className="flex-1">
              <span>{error}</span>
              {runtimeError && <pre className="mt-2 text-xs whitespace-pre-wrap text-red-700">{runtimeError}</pre>}
            </div>
            <button onClick={() => { setError(''); setRuntimeError(''); }}><X size={14} /></button>
          </div>
        )}

        {activeWorkItems.length > 0 && (
          <WorkBanner busyAction={busyAction} items={activeWorkItems} onStopAgent={() => {}} />
        )}

        {activeView === 'pipeline' && selectedJob && (
          <div className="flex flex-1 flex-col overflow-hidden">
            <div className="border-b border-slate-200 bg-white px-6 py-3">
              <div className="flex items-center gap-3">
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold text-slate-950">{selectedJob.title || 'Untitled'}</p>
                  <p className="truncate text-xs text-slate-500">{selectedJob.company || 'No company'}</p>
                </div>
                <div className="flex gap-1 rounded-lg border border-slate-200 bg-slate-50 p-1">
                  {pipelineSteps.map((s, i) => (
                    <button
                      key={s.key}
                      className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${pipelineStep === s.key ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-700'}`}
                      onClick={() => setPipelineStep(s.key)}
                    >
                      <span className={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold ${pipelineStep === s.key ? 'bg-slate-900 text-white' : 'bg-slate-200 text-slate-500'}`}>{i + 1}</span>
                      {s.label}
                    </button>
                  ))}
                </div>
                <IconButton label="Refresh" onClick={load} disabled={busyAction !== ''}>
                  <RefreshCcw size={14} />
                </IconButton>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-6">
              {pipelineStep === 'job' && (
                <div className="mx-auto max-w-3xl space-y-4">
                  <Panel icon={<BriefcaseBusiness size={16} />} title="Job Description" compact>
                    <form onSubmit={handleSaveJob} className="space-y-3">
                      <div className="grid gap-3 md:grid-cols-2">
                        <TextInput label="Company" value={jobDraft.company} onChange={(v) => updateJobDraft({...jobDraft, company: v})} />
                        <TextInput label="Role" value={jobDraft.title} onChange={(v) => updateJobDraft({...jobDraft, title: v})} />
                      </div>
                      <TextInput label="URL" value={jobDraft.url} onChange={(v) => updateJobDraft({...jobDraft, url: v})} />
                      <TextArea label="Description" rows={12} value={jobDraft.raw_text} onChange={(v) => updateJobDraft({...jobDraft, raw_text: v})} />
                      <div className="flex gap-2">
                        <IconButton label="Save" submit><Save size={14} /></IconButton>
                        {selectedJobID > 0 && (
                          <IconButton label="Delete" onClick={() => deleteJob(selectedJobID)}><Trash2 size={14} /></IconButton>
                        )}
                      </div>
                    </form>
                  </Panel>

                  <div className="flex flex-wrap gap-2">
                    <PipelineButton label="Extract" onClick={handleParseJD} tone="primary" />
                    <PipelineButton label="Match" onClick={handleMatch} />
                    <PipelineButton label="Fit" onClick={handleFit} />
                    <PipelineButton label="Strategy" onClick={handleStrategy} />
                    <PipelineButton label="Bullets" onClick={handleBullets} />
                  </div>

                  {jobAnalysis && (
                    <CollapsibleSection label={`Analysis — ${jobAnalysis.role_archetype}`}>
                      <div className="mt-2 space-y-2 text-xs text-slate-600">
                        {jobAnalysis.company && <p><span className="font-medium text-slate-700">Company:</span> {jobAnalysis.company}</p>}
                        {jobAnalysis.location && <p><span className="font-medium text-slate-700">Location:</span> {jobAnalysis.location}</p>}
                        {jobAnalysis.seniority_level && <p><span className="font-medium text-slate-700">Seniority:</span> {jobAnalysis.seniority_level}</p>}
                        {jobAnalysis.work_arrangement && <p><span className="font-medium text-slate-700">Work:</span> {jobAnalysis.work_arrangement}</p>}
                        {jobAnalysis.salary && <p><span className="font-medium text-slate-700">Salary:</span> {jobAnalysis.salary}</p>}
                        {jobAnalysis.responsibilities.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Responsibilities:</p>
                            {jobAnalysis.responsibilities.map((r, i) => <p key={i} className="mt-0.5">• {r}</p>)}
                          </div>
                        )}
                        {jobAnalysis.required_skills.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Required skills:</p>
                            <p className="mt-0.5">{jobAnalysis.required_skills.join(', ')}</p>
                          </div>
                        )}
                        {jobAnalysis.preferred_skills.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Preferred skills:</p>
                            <p className="mt-0.5">{jobAnalysis.preferred_skills.join(', ')}</p>
                          </div>
                        )}
                        {jobAnalysis.top_pain_points.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Pain points:</p>
                            {jobAnalysis.top_pain_points.map((p, i) => <p key={i} className="mt-0.5">• {p}</p>)}
                          </div>
                        )}
                      </div>
                    </CollapsibleSection>
                  )}

                  {jobRequirements.length > 0 && (
                    <CollapsibleSection label={`Requirements (${jobRequirements.length})`}>
                      <div className="mt-2 space-y-1">
                        {jobRequirements.map((req) => (
                          <div key={req.id} className="flex items-start gap-2 text-xs">
                            <span className={`mt-1 shrink-0 rounded-sm px-1.5 py-0.5 text-[10px] font-bold ${req.priority === 'required' ? 'bg-red-100 text-red-700' : 'bg-slate-100 text-slate-500'}`}>{req.priority === 'required' ? 'REQ' : 'pref'}</span>
                            <span className="text-slate-600">{req.requirement_text}</span>
                          </div>
                        ))}
                      </div>
                    </CollapsibleSection>
                  )}

                  {jobMatches.length > 0 && (
                    <CollapsibleSection label={`Match map (${jobMatches.length})`}>
                      <div className="mt-2 space-y-2">
                        {jobMatches.map((m) => (
                          <div key={m.id} className="flex items-start gap-2 text-xs">
                            <span className={`mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold ${m.coverage_status === 'covered' ? 'bg-green-100 text-green-800' : m.coverage_status === 'partial' ? 'bg-amber-100 text-amber-800' : 'bg-red-100 text-red-800'}`}>
                              {m.coverage_status}
                            </span>
                            <div className="min-w-0">
                              <p className="text-slate-700">{m.fact_text}</p>
                              {m.rationale && <p className="text-slate-400">{m.rationale}</p>}
                            </div>
                          </div>
                        ))}
                      </div>
                    </CollapsibleSection>
                  )}

                  {applicationStrategy && (
                    <CollapsibleSection label="Strategy">
                      <div className="mt-2 space-y-2 text-xs text-slate-600">
                        {applicationStrategy.resume_headline && <p><span className="font-medium text-slate-700">Headline:</span> {applicationStrategy.resume_headline}</p>}
                        {applicationStrategy.weak_or_missing_requirements.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Weak/missing:</p>
                            {applicationStrategy.weak_or_missing_requirements.map((w, i) => <p key={i} className="mt-0.5">• {w}</p>)}
                          </div>
                        )}
                      </div>
                    </CollapsibleSection>
                  )}

                  {fitAnalysis && (
                    <Panel icon={<Gauge size={16} />} title={`Fit: ${fitAnalysis.overall_score}% — ${fitAnalysis.recommendation}`} compact summary={fitAnalysis.reality_check}>
                      <div className="space-y-2 mt-3 text-xs text-slate-600">
                        {fitAnalysis.strengths.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Strengths:</p>
                            {fitAnalysis.strengths.map((s, i) => <p key={i} className="mt-0.5">• {s}</p>)}
                          </div>
                        )}
                        {fitAnalysis.critical_gaps.length > 0 && (
                          <div>
                            <p className="font-medium text-slate-700">Gaps:</p>
                            {fitAnalysis.critical_gaps.map((g, i) => <p key={i} className="mt-0.5">• {g}</p>)}
                          </div>
                        )}
                      </div>
                    </Panel>
                  )}

                  <div className="pt-2">
                    <button className="w-full rounded-lg border-2 border-dashed border-slate-300 py-3 text-xs font-medium text-slate-500 hover:border-slate-400 hover:text-slate-600" onClick={() => setPipelineStep('bullets')}>
                      → Continue to bullet selection
                    </button>
                  </div>
                </div>
              )}

              {pipelineStep === 'bullets' && (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <p className="text-xs text-slate-500">{bulletDrafts.filter((d) => d.job_id === selectedJobID).length} bullets · {bulletDrafts.filter((d) => d.job_id === selectedJobID && d.selected_for_resume).length} selected</p>
                    <SecondaryButton label="Auto-select" onClick={handleAutoSelect} />
                  </div>
                  {bulletEvents.length > 0 && (
                    <CollapsibleSection label={`Generation log (${bulletEvents.length})`}>
                      <div className="mt-2 space-y-1">
                        {bulletEvents.slice(-5).map((ev) => (
                          <div key={ev.id} className="flex items-start gap-2 text-xs">
                            <span className={`mt-0.5 shrink-0 rounded px-1 py-0.5 text-[10px] font-medium ${ev.status === 'selected' ? 'bg-green-100 text-green-700' : ev.status === 'rejected' ? 'bg-red-100 text-red-700' : 'bg-slate-100 text-slate-500'}`}>
                              {ev.status}
                            </span>
                            <div className="min-w-0">
                              <p className="text-slate-700">{ev.draft_text || ev.origin_heading}</p>
                              {ev['reason'] && <p className="text-slate-400">{ev['reason']}</p>}
                            </div>
                          </div>
                        ))}
                      </div>
                    </CollapsibleSection>
                  )}
                  {bulletDrafts.filter((d) => d.job_id === selectedJobID).length === 0 ? (
                    <EmptyState text="No bullets yet. Generate bullets from the Job step." />
                  ) : (
                    <div className="space-y-2">
                      {bulletDrafts.filter((d) => d.job_id === selectedJobID).map((draft) => (
                        <BulletCard key={draft.id} draft={draft} onToggle={() => handleToggleBullet(draft)} onEdit={handleEditBullet} onDelete={handleDeleteBullet} />
                      ))}
                    </div>
                  )}
                  <div className="pt-2">
                    <button className="w-full rounded-lg border-2 border-dashed border-slate-300 py-3 text-xs font-medium text-slate-500 hover:border-slate-400 hover:text-slate-600" onClick={() => setPipelineStep('resume')}>
                      → Continue to resume assembly
                    </button>
                  </div>
                </div>
              )}

              {pipelineStep === 'resume' && (
                <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
                  <div className="space-y-4">
                    {!activeResume ? (
                      <div className="rounded-lg border border-slate-200 bg-white p-6 text-center">
                        <p className="text-sm text-slate-600">{bulletDrafts.filter((d) => d.selected_for_resume).length} bullets ready.</p>
                        <div className="mt-4 flex justify-center gap-2">
                          <IconButton label="Generate Resume" onClick={generateResume}><Sparkles size={14} /></IconButton>
                          <SecondaryButton label="Edit bullets" onClick={() => setPipelineStep('bullets')} />
                        </div>
                      </div>
                    ) : (
                      <>
                        <div className="rounded-lg border border-slate-200 bg-white p-5 space-y-4">
                          {editingResume ? (
                            <>
                              <div className="flex items-center justify-between">
                                <span className="text-xs font-semibold text-amber-700 bg-amber-50 px-2 py-0.5 rounded">Editing Resume</span>
                                <div className="flex gap-1">
                                  <SecondaryButton label="Save" onClick={saveEditedResume} variant="green" />
                                  <SecondaryButton label="Cancel" onClick={cancelEditingResume} variant="red" />
                                </div>
                              </div>
                              <div>
                                <label className="text-[10px] font-medium text-slate-500 uppercase">Headline</label>
                                <input className="mt-0.5 w-full rounded border border-slate-200 px-2 py-1 text-xs" value={editingResume.headline} onChange={e => updateEditingField('headline', e.target.value)} />
                              </div>
                              <div>
                                <label className="text-[10px] font-medium text-slate-500 uppercase">Summary</label>
                                <textarea className="mt-0.5 w-full rounded border border-slate-200 px-2 py-1 text-xs" rows={3} value={editingResume.summary} onChange={e => updateEditingField('summary', e.target.value)} />
                              </div>
                              <div>
                                <div className="flex items-center justify-between">
                                  <label className="text-[10px] font-medium text-slate-500 uppercase">Skills</label>
                                  <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={addEditingSkillCategory}>+ Add category</button>
                                </div>
                                {editingResume.skills.map((s, i) => (
                                  <div key={i} className="mt-1 flex items-center gap-1">
                                    <input className="w-24 rounded border border-slate-200 px-1.5 py-0.5 text-[10px] font-semibold" value={s.category} onChange={e => updateEditingSkill(i, e.target.value.split(',').map(v => v.trim()).join(', '))} placeholder="Category" />
                                    <input className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={s.items.join(', ')} onChange={e => updateEditingSkill(i, e.target.value)} placeholder="item1, item2, ..." />
                                    <button className="text-[10px] text-red-400 hover:text-red-600" onClick={() => removeEditingSkillCategory(i)}>✕</button>
                                  </div>
                                ))}
                              </div>
                              <div>
                                <div className="flex items-center justify-between">
                                  <label className="text-[10px] font-medium text-slate-500 uppercase">Experience</label>
                                  <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={() => addEditingEntry('experience')}>+ Add</button>
                                </div>
                                {editingResume.experience.map((e, i) => (
                                  <div key={i} className="mt-2 rounded border border-slate-100 p-2 space-y-1">
<div className="flex items-center gap-1">
                                       <input className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px] font-semibold" value={e.title} onChange={ev => updateEditingEntry('experience', i, 'title', ev.target.value)} placeholder="Title" />
                                       <input className="w-24 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.company} onChange={ev => updateEditingEntry('experience', i, 'company', ev.target.value)} placeholder="Company" />
                                       <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.url} onChange={ev => updateEditingEntry('experience', i, 'url', ev.target.value)} placeholder="Company URL" />
                                       <button className="text-[10px] text-red-400 hover:text-red-600" onClick={() => removeEditingEntry('experience', i)}>✕</button>
                                     </div>
                                     <input className="w-full rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.location} onChange={ev => updateEditingEntry('experience', i, 'location', ev.target.value)} placeholder="Location" />
                                    <div className="flex gap-1">
                                      <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.start_date} onChange={ev => updateEditingEntry('experience', i, 'start_date', ev.target.value)} placeholder="Start" />
                                      <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.end_date} onChange={ev => updateEditingEntry('experience', i, 'end_date', ev.target.value)} placeholder="End" />
                                    </div>
                                    {e.bullets.map((b, bi) => (
                                      <div key={bi} className="flex items-center gap-1">
                                        <textarea className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" rows={2} value={b} onChange={ev => updateEditingBullet('experience', i, bi, ev.target.value)} placeholder="Bullet point" />
                                        <button className="text-[10px] text-red-400 hover:text-red-600 self-start mt-1" onClick={() => removeEditingBullet('experience', i, bi)}>✕</button>
                                      </div>
                                    ))}
                                    <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={() => addEditingBullet('experience', i)}>+ Add bullet</button>
                                  </div>
                                ))}
                              </div>
                              <div>
                                <div className="flex items-center justify-between">
                                  <label className="text-[10px] font-medium text-slate-500 uppercase">Projects</label>
                                  <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={() => addEditingEntry('projects')}>+ Add</button>
                                </div>
                                {editingResume.projects.map((e, i) => (
                                  <div key={i} className="mt-2 rounded border border-slate-100 p-2 space-y-1">
                                    <div className="flex items-center gap-1">
                                      <input className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px] font-semibold" value={e.title} onChange={ev => updateEditingEntry('projects', i, 'title', ev.target.value)} placeholder="Project name" />
                                      <input className="w-24 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.url} onChange={ev => updateEditingEntry('projects', i, 'url', ev.target.value)} placeholder="URL" />
                                      <button className="text-[10px] text-red-400 hover:text-red-600" onClick={() => removeEditingEntry('projects', i)}>✕</button>
                                    </div>
                                    <input className="w-full rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.company} onChange={ev => updateEditingEntry('projects', i, 'company', ev.target.value)} placeholder="Stack (e.g. Go, React)" />
                                    <div className="flex gap-1">
                                      <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.start_date} onChange={ev => updateEditingEntry('projects', i, 'start_date', ev.target.value)} placeholder="Start" />
                                      <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.end_date} onChange={ev => updateEditingEntry('projects', i, 'end_date', ev.target.value)} placeholder="End" />
                                    </div>
                                    {e.bullets.map((b, bi) => (
                                      <div key={bi} className="flex items-center gap-1">
                                        <textarea className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" rows={2} value={b} onChange={ev => updateEditingBullet('projects', i, bi, ev.target.value)} placeholder="Bullet point" />
                                        <button className="text-[10px] text-red-400 hover:text-red-600 self-start mt-1" onClick={() => removeEditingBullet('projects', i, bi)}>✕</button>
                                      </div>
                                    ))}
                                    <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={() => addEditingBullet('projects', i)}>+ Add bullet</button>
                                  </div>
                                ))}
                              </div>
                              <div>
                                <div className="flex items-center justify-between">
                                  <label className="text-[10px] font-medium text-slate-500 uppercase">Education</label>
                                  <button className="text-[10px] text-blue-600 hover:text-blue-800" onClick={() => addEditingEntry('education')}>+ Add</button>
                                </div>
                                {editingResume.education.map((e, i) => (
                                  <div key={i} className="mt-2 rounded border border-slate-100 p-2 space-y-1">
                                    <div className="flex items-center gap-1">
                                      <input className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px] font-semibold" value={e.organization} onChange={ev => updateEditingEntry('education', i, 'organization', ev.target.value)} placeholder="Institution" />
                                      <button className="text-[10px] text-red-400 hover:text-red-600" onClick={() => removeEditingEntry('education', i)}>✕</button>
                                    </div>
                                    <input className="w-full rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.degree} onChange={ev => updateEditingEntry('education', i, 'degree', ev.target.value)} placeholder="Degree" />
                                    <div className="flex gap-1">
                                      <input className="flex-1 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.location} onChange={ev => updateEditingEntry('education', i, 'location', ev.target.value)} placeholder="Location" />
                                      <input className="w-20 rounded border border-slate-200 px-1.5 py-0.5 text-[10px]" value={e.end_date} onChange={ev => updateEditingEntry('education', i, 'end_date', ev.target.value)} placeholder="Year" />
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </>
                          ) : (
                            <>
                              <div className="flex items-start justify-between">
                                <div className="flex-1">
                                  <h3 className="text-base font-semibold text-slate-950">{activeResume.headline}</h3>
                                  <p className="mt-1 text-xs text-slate-500">{activeResume.summary}</p>
                                </div>
                                <button className="ml-2 rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600" onClick={startEditingResume} title="Edit resume">
                                  <Pencil size={14} />
                                </button>
                              </div>
                              {activeResume.skills.length > 0 && (
                                <div className="mt-4">
                                  <CollapsibleSection label={`Skills (${activeResume.skills.length})`}>
                                    <div className="space-y-1">
                                      {activeResume.skills.map((s, i) => (
                                        <p key={i} className="text-xs text-slate-600"><span className="font-semibold text-slate-800">{s.category}:</span> {s.items.join(', ')}</p>
                                      ))}
                                    </div>
                                  </CollapsibleSection>
                                </div>
                              )}
                              {activeResume.experience.length > 0 && (
                                <div className="mt-4">
                                  <CollapsibleSection label={`Experience (${activeResume.experience.length})`} defaultOpen>
                                    <div className="space-y-3">
                                      {activeResume.experience.map((e, i) => (
                                        <div key={i}>
                                          <p className="text-xs font-semibold text-slate-900">{e.title}</p>
                                          {e.company && <p className="text-xs text-slate-500">{e.company}{e.location ? ` · ${e.location}` : ''}</p>}
                                          {e.bullets.map((b, bi) => <p key={bi} className="mt-0.5 text-xs text-slate-600">• {b}</p>)}
                                        </div>
                                      ))}
                                    </div>
                                  </CollapsibleSection>
                                </div>
                              )}
                              {activeResume.projects.length > 0 && (
                                <div className="mt-3">
                                  <CollapsibleSection label={`Projects (${activeResume.projects.length})`}>
                                    <div className="space-y-3">
                                      {activeResume.projects.map((e, i) => (
                                        <div key={i}>
                                          <p className="text-xs font-semibold text-slate-900">{e.title}</p>
                                          {e.company && <p className="text-xs text-slate-500">{e.company}</p>}
                                          {e.bullets.map((b, bi) => <p key={bi} className="mt-0.5 text-xs text-slate-600">• {b}</p>)}
                                        </div>
                                      ))}
                                    </div>
                                  </CollapsibleSection>
                                </div>
                              )}
                              {activeResume.education.length > 0 && (
                                <div className="mt-3">
                                  <CollapsibleSection label={`Education (${activeResume.education.length})`}>
                                    <div className="space-y-2">
                                      {activeResume.education.map((e, i) => (
                                        <div key={i}>
                                          <p className="text-xs font-semibold text-slate-900">{e.organization}</p>
                                          <p className="text-xs text-slate-600">{e.degree}{e.location ? ` · ${e.location}` : ''}{e.end_date ? ` · ${e.end_date}` : ''}</p>
                                        </div>
                                      ))}
                                    </div>
                                  </CollapsibleSection>
                                </div>
                              )}
                            </>
                          )}
                        </div>
                        <div className="flex items-center gap-2">
                          {resumeVersions.length > 1 && (
                            <select className="rounded border border-slate-200 px-2 py-1 text-xs" value={resumeVersions[0]?.id ?? 0} onChange={async (e) => {
                              const v = resumeVersions.find((rv) => rv.id === Number(e.target.value));
                              if (v) { setActiveResume(normalizeResumeJSON(v.resume_json)); setActiveValidation(normalizeValidationResult(v.validation_result)); }
                            }}>
                              {resumeVersions.map((v) => <option key={v.id} value={v.id}>v{v.id} — {v.created_at?.slice(0, 16)}</option>)}
                            </select>
                          )}
                          {activeResume && !editingResume && (
                            <SecondaryButton label="Edit" onClick={startEditingResume} variant="blue" />
                          )}
                          <IconButton label="Export JSON" onClick={exportResumeJSON}><FileText size={14} /></IconButton>
                          <IconButton label="PDF" onClick={renderPDF}><FilePlus2 size={14} /></IconButton>
                        </div>
                      </>
                    )}
                  </div>
                  <div className="space-y-4">
                    {activeValidation && (
                      <div className={`rounded-lg border p-4 ${activeValidation.passed ? 'border-green-200 bg-green-50' : 'border-red-200 bg-red-50'}`}>
                        <p className={`text-xs font-semibold ${activeValidation.passed ? 'text-green-900' : 'text-red-900'}`}>
                          {activeValidation.passed ? 'Passed' : 'Issues found'}
                        </p>
                        {activeValidation.errors.length > 0 && (
                          <CollapsibleSection label={`${activeValidation.errors.length} errors`}>
                            {activeValidation.errors.map((e, i) => <p key={i} className="mt-1 text-xs text-red-700">• {e}</p>)}
                          </CollapsibleSection>
                        )}
                        {activeValidation.warnings.length > 0 && (
                          <CollapsibleSection label={`${activeValidation.warnings.length} warnings`}>
                            {activeValidation.warnings.slice(0, 3).map((w, i) => <p key={i} className="mt-1 text-xs text-amber-700">• {w}</p>)}
                            {activeValidation.warnings.length > 3 && <p className="mt-1 text-xs text-amber-600">+{activeValidation.warnings.length - 3} more</p>}
                          </CollapsibleSection>
                        )}
                      </div>
                    )}
                    <Panel icon={<BriefcaseBusiness size={14} />} title="Application" compact>
                      <div className="space-y-2 mt-2">
                        <TextInput label="Notes" value={applicationNotes} onChange={setApplicationNotes} />
                        <IconButton label="Save" onClick={saveApplication} disabled={!selectedJobID}><Save size={14} /></IconButton>
                      </div>
                    </Panel>
                  </div>
                </div>
              )}

              {pipelineStep === 'tracker' && (
                <div className="space-y-4">
                  {applications.length === 0 ? (
                    <EmptyState text="No applications yet. Save one from the Resume step." />
                  ) : (
                    <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
                      <table className="w-full text-xs">
                        <thead>
                          <tr className="border-b border-slate-200 text-left font-semibold text-slate-500">
                            <th className="px-4 py-3">Company</th>
                            <th className="px-4 py-3">Role</th>
                            <th className="px-4 py-3">Status</th>
                            <th className="px-4 py-3">Fit</th>
                          </tr>
                        </thead>
                        <tbody>
                          {applications.map((app) => {
                            const j = jobs.find((jj) => jj.id === app.job_id);
                            return (
                              <tr key={app.id} className="border-b border-slate-100 hover:bg-slate-50">
                                <td className="px-4 py-3 font-medium">{j?.company || '—'}</td>
                                <td className="px-4 py-3">{j?.title || '—'}</td>
                                 <td className="px-4 py-3">
                                  <select className="rounded-lg border border-slate-200 px-2 py-1 text-xs font-medium focus:border-slate-400 focus:outline-none" value={app.status} onChange={(e) => updateApplicationStatus(app.id, e.target.value)}>
                                     {Object.entries(APP_STATUS_LABELS).map(([k, l]) => <option key={k} value={k}>{l}</option>)}
                                  </select>
                                 </td>
                                <td className="px-4 py-3">{app.fit_score}%</td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {activeView === 'pipeline' && !selectedJob && (
          <div className="flex flex-1 items-center justify-center">
            <div className="text-center">
              <BriefcaseBusiness size={40} className="mx-auto text-slate-300" />
              <p className="mt-3 text-sm text-slate-500">Select a job from the sidebar</p>
              <p className="text-xs text-slate-400">or add a new one to get started</p>
            </div>
          </div>
        )}

        {activeView === 'sources' && (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="mx-auto max-w-4xl space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-xs text-slate-500">{sources.length} sources · {facts.length} facts · {claims.length} claims</p>
                <button className="flex items-center gap-1 rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800" onClick={() => setShowImportSource(!showImportSource)}>
                  <Plus size={14} /> Add source
                </button>
              </div>
              {showImportSource && (
                <div className="rounded-lg border border-slate-200 bg-white p-4">
                  <form onSubmit={handleAddSource} className="space-y-3">
                    <div className="grid gap-3 md:grid-cols-3">
                      <SelectInput label="Type" value={sourceDraft.source_type} onChange={(v) => setSourceDraft({...sourceDraft, source_type: v, trust_tier: defaultSourceTrust(v)})} options={[['current_resume', 'Current resume'], ['extended_resume', 'Extended'], ['project_notes', 'Project notes'], ['manual_notes', 'Manual']]} />
                      <SelectInput label="Trust" value={sourceDraft.trust_tier} onChange={(v) => setSourceDraft({...sourceDraft, trust_tier: v})} options={[['verified', 'Verified'], ['trusted_ai_summary', 'AI summary'], ['raw_source', 'Raw'], ['unverified_ai', 'Unverified']]} />
                      <TextInput label="Title" value={sourceDraft.title} onChange={(v) => setSourceDraft({...sourceDraft, title: v})} />
                    </div>
                    <TextArea label="Raw text" rows={8} value={sourceDraft.raw_text} onChange={(v) => setSourceDraft({...sourceDraft, raw_text: v})} />
                    <div className="flex gap-2">
                      <button type="submit" className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800">Save</button>
                      <button type="button" className="rounded-md border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100" onClick={() => setShowImportSource(false)}>Cancel</button>
                    </div>
                  </form>
                </div>
              )}
              {sources.map((source) => {
                const run = contextRuns.filter((r) => r.source_id === source.id).sort((a, b) => b.id - a.id)[0];
                const step = run ? latestContextStep(contextSteps, run.id) : undefined;
                return (
                  <div key={source.id} className="rounded-lg border border-slate-200 bg-white p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 flex-1">
                        <p className="text-xs font-semibold text-slate-950">{source.title}</p>
                        <p className="mt-1 line-clamp-2 text-xs text-slate-500">{source.raw_text}</p>
                        {run && (
                          <div className="mt-2 flex items-center gap-2">
                            <span className={`inline-block h-2 w-2 rounded-full ${run.status === 'complete' ? 'bg-green-500' : run.status === 'running' ? 'bg-blue-500 animate-pulse' : 'bg-red-400'}`} />
                            <span className="text-xs text-slate-500">{run.facts_created} facts · {run.claims_created} claims</span>
                          </div>
                        )}
                      </div>
                      <div className="flex shrink-0 items-center gap-1">
                        {run?.status === 'running' ? (
                          <button className="rounded p-1.5 text-slate-400 hover:bg-slate-100" onClick={() => handleStopAgent(run)}>
                            <Square size={14} />
                          </button>
                        ) : (
                          <button className="rounded p-1.5 text-slate-400 hover:bg-slate-100" onClick={() => beginContextAgent(source.id)}>
                            <Play size={14} />
                          </button>
                        )}
                        <button className="rounded p-1.5 text-slate-400 hover:bg-slate-100" onClick={() => handleDeleteSource(source.id)}>
                          <Trash2 size={14} />
                        </button>
                      </div>
                    </div>
                    {run?.status === 'running' && (
                      <div className="mt-3">
                        <div className="h-1.5 w-full rounded-full bg-slate-200">
                          <div className="h-1.5 rounded-full bg-blue-500 transition-all" style={{width: `${contextRunProgress(run, step)}%`}} />
                        </div>
                        <p className="mt-1 text-xs text-slate-400">{step?.message || '...'}</p>
                      </div>
                    )}
                  </div>
                );
              })}
              <TestLLMBanner />
            </div>
          </div>
        )}

        {activeView === 'settings' && (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="mx-auto max-w-2xl space-y-4">
              <p className="text-xs text-slate-500">Configure LLM provider, API keys, and profile</p>
              <Panel icon={<KeyRound size={14} />} title="API" compact>
                <div className="space-y-2 mt-2">
                  <SelectInput label="Provider" value={settings.provider} onChange={(v) => setSettings({...settings, provider: v})} options={[['openrouter', 'OpenRouter'], ['openai', 'OpenAI']]} />
                  <TextInput label="Model" value={settings.model} onChange={(v) => setSettings({...settings, model: v})} />
                  <TextInput label="API Key" value={apiKey} onChange={setAPIKey} />
                  <button className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800" onClick={handleTestSettings}>Save</button>
                </div>
              </Panel>
              <Panel icon={<UserRound size={14} />} title="Profile" compact>
                <div className="mt-2">
                  {editingProfile ? (
                    <div className="space-y-2">
                      <TextInput label="Name" value={profile.contact.full_name} onChange={(v) => setProfile({...profile, contact: {...profile.contact, full_name: v}})} />
                      <TextInput label="Email" value={profile.contact.email} onChange={(v) => setProfile({...profile, contact: {...profile.contact, email: v}})} />
                      <button className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-800" onClick={handleSaveProfile}>Save</button>
                    </div>
                  ) : (
                    <button className="text-xs text-slate-600 hover:text-slate-900" onClick={() => setEditingProfile(true)}>{profile.contact.full_name || 'Edit profile'}</button>
                  )}
                </div>
              </Panel>
              <Panel icon={<Wrench size={14} />} title="Tools" compact>
                <div className="mt-2 flex flex-wrap gap-2">
                  <SecondaryButton label="Install Tectonic" onClick={async () => { await InstallTectonic(); await load(); }} />
                  <SecondaryButton label="Render test PDF" onClick={async () => { const r = (await RenderSamplePDF()) as RenderPDFResult; setPDFResult(r); }} />
                </div>
                {pdfResult && <p className={`mt-1 text-xs ${pdfResult.success ? 'text-green-600' : 'text-red-600'}`}>{pdfResult.success ? 'PDF rendered' : pdfResult.error}</p>}
              </Panel>
              {events.length > 0 && (
                <CollapsibleSection label={`Events (${events.length})`}>
                  <div className="mt-2 max-h-48 space-y-1 overflow-y-auto">
                    {events.slice(-20).reverse().map((ev) => (
                      <div key={ev.id} className="flex items-start gap-2 text-xs">
                        <span className={`mt-0.5 shrink-0 rounded px-1 py-0.5 text-[10px] font-medium ${ev.level === 'error' ? 'bg-red-100 text-red-700' : ev.level === 'warn' ? 'bg-amber-100 text-amber-700' : 'bg-slate-100 text-slate-500'}`}>
                          {ev.level}
                        </span>
                        <p className="min-w-0 text-slate-600">{ev.message}</p>
                        <span className="shrink-0 text-slate-400">{ev.created_at?.slice(11, 19)}</span>
                      </div>
                    ))}
                  </div>
                </CollapsibleSection>
              )}
              <TestLLMBanner />
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

function CollapsibleSection({label, children, defaultOpen = false}: {label: string; children: React.ReactNode; defaultOpen?: boolean}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="border-l-2 border-slate-200 pl-3">
      <button className="flex items-center gap-1 text-xs font-semibold text-slate-700 hover:text-slate-900" onClick={() => setOpen(!open)}>
        {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        {label}
      </button>
      {open && <div className="mt-2">{children}</div>}
    </div>
  );
}

function BulletCard({draft, onToggle, onEdit, onDelete}: {draft: TailoredBulletDraft; onToggle: () => void; onEdit: (draft: TailoredBulletDraft, text: string) => void; onDelete: (id: number) => void}) {
  const [expanded, setExpanded] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState(draft.draft_text);
  const riskFlags = draft.risk_flags.filter((f) => f !== 'style_too_long' && f !== 'style_too_thin');
  return (
    <div className={`rounded-lg border bg-white ${draft.selected_for_resume ? 'border-blue-300 ring-1 ring-blue-100' : 'border-slate-200'}`}>
      <div className="flex items-start gap-2 p-3">
        <button className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 transition-colors ${draft.selected_for_resume ? 'border-blue-500 bg-blue-500 text-white' : 'border-slate-300 hover:border-slate-400'}`} onClick={onToggle}>
          {draft.selected_for_resume && <CheckCircle2 size={13} />}
        </button>
        <div className="min-w-0 flex-1">
          {editing ? (
            <div className="flex items-center gap-2">
              <input className="w-full rounded-lg border border-slate-200 px-2 py-1 text-xs text-slate-900 focus:border-slate-400 focus:outline-none" value={editText} onChange={(e) => setEditText(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') { onEdit(draft, editText); setEditing(false); } if (e.key === 'Escape') setEditing(false); }} autoFocus />
              <button className="shrink-0 rounded-lg bg-slate-900 px-2.5 py-1 text-[10px] font-semibold text-white hover:bg-slate-800" onClick={() => { onEdit(draft, editText); setEditing(false); }}>Save</button>
              <button className="shrink-0 rounded-lg border border-slate-200 px-2.5 py-1 text-[10px] font-medium text-slate-500 hover:bg-slate-50" onClick={() => setEditing(false)}>Cancel</button>
            </div>
          ) : (
            <p className="text-xs text-slate-900">{draft.draft_text}</p>
          )}
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-500">{draft.origin_heading || draft.origin_type}</span>
            <span className="text-[10px] text-slate-400">{Math.round(draft.selection_score * 100)}%</span>
            {riskFlags.map((f) => <span key={f} className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700">{f}</span>)}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button className="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600" onClick={() => setExpanded(!expanded)} title="Details">
            <Eye size={13} />
          </button>
          <button className="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-blue-50 hover:text-blue-600" onClick={() => { setEditText(draft.draft_text); setEditing(true); }} title="Edit">
            <Pencil size={13} />
          </button>
          <button className="rounded-md p-1.5 text-slate-400 transition-colors hover:bg-red-50 hover:text-red-500" onClick={() => onDelete(draft.id)} title="Delete">
            <Trash2 size={13} />
          </button>
        </div>
        {expanded && draft.rationale && (
          <div className="border-t border-slate-100 bg-slate-50 px-3 py-2">
            <p className="text-xs text-slate-500">{draft.rationale}</p>
          </div>
        )}
      </div>
    </div>
  );
}

function TestLLMBanner() {
  const [result, setResult] = useState<LLMTestResult | null>(null);
  function handleTestLLM() {
    void (async () => { setResult((await TestLLM()) as LLMTestResult); })();
  }
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-slate-700">LLM Connection</p>
        <button className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50" onClick={handleTestLLM}>Test</button>
      </div>
      {result && <p className={`mt-1 text-xs ${result.success ? 'text-green-600' : 'text-red-600'}`}>{result.success ? `${result.provider} OK` : result.error}</p>}
    </div>
  );
}

function Panel({icon, title, subtitle, children, compact, summary}: {icon?: React.ReactNode; title: string; subtitle?: string; children?: React.ReactNode; compact?: boolean; summary?: string}) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white">
      <div className={`flex items-center gap-2 ${compact ? 'px-4 py-2.5' : 'px-4 py-3'}`}>
        {icon && <span className="text-slate-500">{icon}</span>}
        <span className={`font-semibold text-slate-950 ${compact ? 'text-xs' : 'text-sm'}`}>{title}</span>
        {subtitle && <span className="text-xs text-slate-400">{subtitle}</span>}
      </div>
      {summary && <p className="border-t border-slate-100 px-4 py-2 text-xs text-slate-500">{summary}</p>}
      {children && <div className={`border-t border-slate-100 ${compact ? 'px-4 py-2.5' : 'px-4 py-3'}`}>{children}</div>}
    </div>
  );
}

function SecondaryButton({label, onClick, icon, disabled, variant}: {label: string; onClick: () => void; icon?: React.ReactNode; disabled?: boolean; variant?: 'default' | 'blue' | 'green' | 'amber' | 'red'}) {
  const variants = {
    default: 'border-slate-300 text-slate-700 before:bg-slate-400 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-900',
    blue: 'border-slate-300 text-slate-700 before:bg-blue-500 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-900',
    green: 'border-slate-300 text-slate-700 before:bg-emerald-500 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-900',
    amber: 'border-slate-300 text-slate-700 before:bg-amber-500 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-900',
    red: 'border-red-300 text-red-700 before:bg-red-500 hover:border-red-400 hover:bg-red-50 hover:text-red-800',
  };
  return (
    <button onClick={onClick} disabled={disabled} className={`relative flex items-center gap-1.5 overflow-hidden rounded-md border bg-white px-3 py-1.5 pl-3.5 text-xs font-semibold shadow-sm shadow-slate-200/70 ring-1 ring-white/70 transition-all before:absolute before:left-0 before:top-0 before:h-full before:w-0.5 active:translate-y-px active:shadow-none disabled:pointer-events-none disabled:opacity-40 ${variants[variant ?? 'default']}`}>
      {icon}{label}
    </button>
  );
}

function PipelineButton({label, onClick, tone}: {label: string; onClick: () => void; tone?: 'primary'}) {
  const classes = tone === 'primary'
    ? 'border-slate-950 bg-slate-900 text-white shadow-slate-300/80 hover:bg-slate-800 hover:shadow-slate-400/80'
    : 'border-slate-300 bg-white text-slate-800 shadow-slate-200/90 hover:border-slate-400 hover:bg-slate-50 hover:text-slate-950 hover:shadow-slate-300/90';
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex min-w-20 items-center justify-center rounded-md border px-4 py-2 text-xs font-bold shadow-sm transition-all active:translate-y-px active:shadow-none ${classes}`}
    >
      {label}
    </button>
  );
}

function IconButton({label, onClick, children, submit, full, disabled}: {label: string; onClick?: () => void; children?: React.ReactNode; submit?: boolean; full?: boolean; disabled?: boolean}) {
  return (
    <button onClick={onClick} disabled={disabled || submit} type={submit ? 'submit' : 'button'}
      className={`flex items-center gap-1.5 rounded-md border border-slate-950 bg-slate-900 px-3 py-1.5 text-xs font-semibold text-white shadow-sm shadow-slate-300/80 transition-all hover:bg-slate-800 active:translate-y-px active:shadow-none disabled:pointer-events-none disabled:opacity-40 ${full ? 'flex-1 justify-center' : ''}`}>
      {children}{label}
    </button>
  );
}

function StatusPill({state, text}: {state: string; text: string}) {
  const color = state === 'ready' ? 'bg-green-100 text-green-700' : state === 'error' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700';
  return <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${color}`}>{text}</span>;
}

function StatusBadge({text, color}: {text: string; color?: 'slate' | 'green' | 'blue' | 'amber' | 'red'}) {
  const colors = {
    slate: 'bg-slate-100 text-slate-600',
    green: 'bg-green-100 text-green-700',
    blue: 'bg-blue-100 text-blue-700',
    amber: 'bg-amber-100 text-amber-700',
    red: 'bg-red-100 text-red-700',
  };
  return <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${colors[color ?? 'slate']}`}>{text}</span>;
}

function TextInput({label, value, onChange}: {label: string; value: string; onChange: (v: string) => void}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm text-slate-950 focus:border-slate-400 focus:outline-none" />
    </div>
  );
}

function TextArea({label, value, onChange, rows}: {label: string; value: string; onChange: (v: string) => void; rows?: number}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <textarea value={value} onChange={(e) => onChange(e.target.value)} rows={rows || 4} className="w-full rounded-md border border-slate-200 px-3 py-2 text-sm text-slate-950 focus:border-slate-400 focus:outline-none" />
    </div>
  );
}

function SelectInput({label, value, onChange, options}: {label: string; value: string; onChange: (v: string) => void; options: [string, string][]}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-slate-500">{label}</label>
      <select value={value} onChange={(e) => onChange(e.target.value)} className="w-full rounded-md border border-slate-200 px-3 py-1.5 text-sm text-slate-950 focus:border-slate-400 focus:outline-none">
        {options.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
      </select>
    </div>
  );
}

function CheckInput({checked, label, onChange}: {checked: boolean; label: string; onChange: (v: boolean) => void}) {
  return (
    <label className="flex items-center gap-2 text-xs text-slate-700">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="rounded border-slate-300" />
      {label}
    </label>
  );
}

function EmptyState({text}: {text: string}) {
  return <div className="rounded-lg border border-dashed border-slate-200 py-8 text-center text-xs text-slate-400">{text}</div>;
}

function WorkBanner({busyAction, items, onStopAgent}: {busyAction: string; items: WorkItem[]; onStopAgent: (id: number) => void}) {
  return (
    <div className="border-b border-blue-200 bg-blue-50 px-6 py-2">
      {items.map((item) => (
        <div key={item.key} className="flex items-center gap-2 text-xs text-blue-700">
          <LoaderCircle size={12} className="animate-spin" />
          <span className="font-medium">{item.title}</span>
          <span className="text-blue-500">{item.detail}</span>
          {item.runID && <button className="ml-2 text-blue-400" onClick={() => onStopAgent(item.runID!)}>stop</button>}
        </div>
      ))}
    </div>
  );
}

function InlineProgress({title, detail, progress}: {title: string; detail: string; progress: number}) {
  return (
    <div>
      <div className="flex items-center justify-between text-xs text-slate-500">
        <span>{title}</span><span>{Math.round(progress)}%</span>
      </div>
      <div className="mt-1 h-1.5 w-full rounded-full bg-slate-200">
        <div className="h-1.5 rounded-full bg-blue-500 transition-all" style={{width: `${progress}%`}} />
      </div>
      {detail && <p className="mt-1 text-xs text-slate-400">{detail}</p>}
    </div>
  );
}

function contextRunProgress(run: ContextAgentRun | undefined, step: ContextAgentStep | undefined) {
  if (!run) return 0;
  if (run.status === 'complete') return 100;
  if (run.status === 'failed' || run.status === 'cancelled') return 0;
  const stages = ['queued', 'source_preprocess', 'section_detect', 'fact_extract', 'fact_compact', 'profile_draft', 'claim_generate', 'claim_compact', 'dedupe', 'done'];
  const idx = stages.indexOf(step?.stage || 'queued');
  return Math.min(90, Math.round(((idx + 1) / stages.length) * 100));
}

function sourceTitle(sources: CandidateSource[], sourceID: number) {
  return sources.find((s) => s.id === sourceID)?.title || `Source ${sourceID}`;
}

function sourceName(sources: CandidateSource[], sourceID: number) {
  const s = sources.find((src) => src.id === sourceID);
  return s ? s.title : 'No source selected';
}

function contextStageLabel(stage: string) {
  return stage.replaceAll('_', ' ');
}

function workItemForAction(name: string) {
  return {key: name, title: name.replace(/-/g, ' '), detail: 'Running', progress: 50};
}

function inferJobDetailsFromText(text: string) {
  const lines = text.split('\n').filter((l) => l.trim());
  return {company: lines[0]?.trim()?.slice(0, 60) || '', title: lines[1]?.trim()?.slice(0, 80) || ''};
}

function defaultSourceTrust(sourceType: string) {
  if (sourceType === 'current_resume' || sourceType === 'project_notes') return 'verified';
  if (sourceType === 'extended_resume') return 'trusted_ai_summary';
  return 'raw_source';
}

function delay(ms: number) { return new Promise((r) => setTimeout(r, ms)); }

function normalizeSources(v: CandidateSource[] | null | undefined) { return v ?? []; }
function normalizeFacts(v: EvidenceFact[] | null | undefined) { return v ?? []; }
function normalizeClaims(v: CandidateClaim[] | null | undefined) { return v ?? []; }
function normalizeBlockedClaims(v: BlockedClaim[] | null | undefined) { return v ?? []; }
function normalizeContextRuns(v: ContextAgentRun[] | null | undefined) { return v ?? []; }
function normalizeContextSteps(v: ContextAgentStep[]) { return v; }
function normalizeRequirements(v: JobRequirement[] | null | undefined) { return v ?? []; }
function normalizeMatches(v: JobFactMatch[] | null | undefined) { return v ?? []; }
function normalizeDrafts(v: TailoredBulletDraft[] | null | undefined) { return v ?? []; }
function normalizeBulletEvents(v: BulletGenerationEvent[] | null | undefined) { return v ?? []; }
function normalizeJobAnalysis(v: JobAnalysis | null | undefined) { return v ?? null; }
function normalizeFitAnalysis(v: JobFitAnalysis | null | undefined) { return v ?? null; }
function normalizeApplicationStrategy(v: ApplicationStrategy | null | undefined) { return v ?? null; }

function normalizeResumeJSON(v: ResumeJSON | null | undefined): ResumeJSON {
  return {
    contact: {
      full_name: v?.contact?.full_name ?? '',
      email: v?.contact?.email ?? '',
      phone: v?.contact?.phone ?? '',
      location: v?.contact?.location ?? '',
      linkedin: v?.contact?.linkedin ?? '',
      github: v?.contact?.github ?? '',
    },
    headline: v?.headline ?? '',
    summary: v?.summary ?? '',
    contact_line: v?.contact_line ?? '',
    skills_line: v?.skills_line ?? '',
    skills: (v?.skills ?? []).map((s) => ({...s, items: s.items ?? []})),
    experience: (v?.experience ?? []).map((e) => ({...e, url: e.url ?? '', bullets: e.bullets ?? [], claim_ids: e.claim_ids ?? [], bullet_ids: e.bullet_ids ?? []})),
    projects: (v?.projects ?? []).map((e) => ({...e, url: e.url ?? '', bullets: e.bullets ?? [], claim_ids: e.claim_ids ?? [], bullet_ids: e.bullet_ids ?? []})),
    education: v?.education ?? [],
    tex_source: v?.tex_source ?? '',
    generated_at: v?.generated_at ?? '',
  };
}

function normalizeValidationResult(v: ValidationResult | null | undefined): ValidationResult {
  return {
    passed: Boolean(v?.passed),
    errors: v?.errors ?? [],
    warnings: v?.warnings ?? [],
    factuality_checks: v?.factuality_checks ?? [],
    style_issues: v?.style_issues ?? [],
    immutable_issues: v?.immutable_issues ?? [],
    title_issues: v?.title_issues ?? [],
  };
}

function normalizeResumeVersion(v: ResumeVersion): ResumeVersion {
  return {...v, resume_json: normalizeResumeJSON(v.resume_json), validation_result: normalizeValidationResult(v.validation_result)};
}

function normalizeProfile(p: CandidateProfile) {
  return {...p, contact: {...p.contact, links: p.contact.links || []}, records: p.records || []};
}

function newRecord(recordType: string): CandidateProfileRecord {
  return {id: 0, record_type: recordType, label: '', organization: '', role: '', start_date: '', end_date: '', value: '', verified: false, created_at: '', updated_at: ''};
}

function recordTypeLabel(type: string) {
  const labels: Record<string, string> = {education: 'Education', employment: 'Employment', project: 'Project', allowed_alias: 'Alias', blocked_alias: 'Blocked'};
  return labels[type] ?? type;
}

function latestContextStep(steps: ContextAgentStep[], runID: number) {
  return steps.filter((s) => s.run_id === runID).sort((a, b) => b.id - a.id)[0];
}

export default App;
