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
  ExtensionJobDraft,
  FetchJobDescription,
  FetchJobDescriptionResult,
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
  GetPendingExtensionJobDraft,
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
  RunJobAgentWorkflow,
  ValidateResumeJSON,
  RenderResumePDF,
  OpenFolder,
  SaveResumeVersion,
  ListResumeVersions,
  SaveApplication,
  GetApplication,
  ListApplications,
  UpdateApplicationStatus,
  UpdateApplicationResumeVersion,
  LogCorrection,
  ListCorrections,
  ResumeJSON,
  ResumeVersion,
  Application,
  CorrectionLog,
  ValidationResult,
  GenerateResumeJSONInput,
  JobAgentWorkflowResult,
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
type View = 'dashboard' | 'pipeline' | 'sources' | 'settings';
type PipelineStep = 'job' | 'bullets' | 'resume' | 'tracker';
type JobDraft = {company: string; title: string; url: string; raw_text: string};
type GeneratedResumeDraft = {
  id: string;
  job_id: number;
  resume: ResumeJSON;
  validation: ValidationResult;
  created_at: string;
};
type JobWorkflowState = {
  pct: number;
  stage: PipelineStep;
  label: string;
  tone: 'slate' | 'blue' | 'green' | 'amber' | 'red';
  selectedBullets: number;
  totalBullets: number;
};

const emptyJobDraft: JobDraft = {company: '', title: '', url: '', raw_text: ''};

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
  const [activeView, setActiveView] = useState<View>('dashboard');
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
  const bulletDraftsRef = useRef<Map<number, TailoredBulletDraft[]>>(new Map());
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
  const [newJobDraft, setNewJobDraft] = useState<JobDraft>(emptyJobDraft);
  const [newJobFetchWarnings, setNewJobFetchWarnings] = useState<string[]>([]);
  const [jobFetchWarnings, setJobFetchWarnings] = useState<string[]>([]);
  const [extensionImportNotice, setExtensionImportNotice] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [llmResult, setLLMResult] = useState<LLMTestResult | null>(null);
  const [pdfResult, setPDFResult] = useState<RenderPDFResult | null>(null);
  const [pdfSuccess, setPDFSuccess] = useState<{ path: string; jobTitle: string; jobID: number } | null>(null);
  const [busyAction, setBusyAction] = useState('');
  const [applications, setApplications] = useState<Application[]>([]);
  const [resumeVersions, setResumeVersions] = useState<ResumeVersion[]>([]);
  const [generatedResumeDrafts, setGeneratedResumeDrafts] = useState<GeneratedResumeDraft[]>([]);
  const [activeResume, setActiveResume] = useState<ResumeJSON | null>(null);
  const [editingResume, setEditingResume] = useState<ResumeJSON | null>(null);
  const [activeValidation, setActiveValidation] = useState<ValidationResult | null>(null);
  const [selectedApplicationID, setSelectedApplicationID] = useState<number>(0);
  const [applicationNotes, setApplicationNotes] = useState('');
  const [applicationStatusDraft, setApplicationStatusDraft] = useState('draft');
  const [jobAgentStages, setJobAgentStages] = useState<JobAgentWorkflowResult['stages']>([]);
  const [resumeSaveNotice, setResumeSaveNotice] = useState('');
  const [jobSearch, setJobSearch] = useState('');
  const [jobStatusFilter, setJobStatusFilter] = useState('all');
  const pollingContextRunIDs = useRef<Set<number>>(new Set());
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});
  const [showAddJob, setShowAddJob] = useState(false);
  const [showImportSource, setShowImportSource] = useState(false);
  const [editingProfile, setEditingProfile] = useState(false);
  const jobContextRequestRef = useRef(0);

  const selectedSource = sources.find((source) => source.id === selectedSourceID);
  const selectedJob = jobs.find((job) => job.id === selectedJobID);
  const selectedJobDrafts = useMemo(() => bulletDrafts.filter((draft) => draft.job_id === selectedJobID), [bulletDrafts, selectedJobID]);
  const selectedGeneratedResumes = useMemo(() => generatedResumeDrafts.filter((draft) => draft.job_id === selectedJobID), [generatedResumeDrafts, selectedJobID]);
  const newJobInferred = useMemo(() => inferJobDetailsFromText(newJobDraft.raw_text), [newJobDraft.raw_text]);
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

  const resumeVersionsByJobID = useMemo(() => {
    const map = new Map<number, ResumeVersion[]>();
    for (const version of resumeVersions) {
      const jobVersions = map.get(version.job_id) || [];
      map.set(version.job_id, [version, ...jobVersions]);
    }
    return map;
  }, [resumeVersions]);

  const selectedSavedResumes = useMemo(() => resumeVersionsByJobID.get(selectedJobID) || [], [resumeVersionsByJobID, selectedJobID]);
  const activeResumeFingerprint = useMemo(() => activeResume ? JSON.stringify(activeResume) : '', [activeResume]);
  const activeResumeSaved = useMemo(() => activeResumeFingerprint !== '' && selectedSavedResumes.some((version) => JSON.stringify(normalizeResumeJSON(version.resume_json)) === activeResumeFingerprint), [activeResumeFingerprint, selectedSavedResumes]);
  const activeResumeDraftOnly = Boolean(activeResume && !activeResumeSaved);

  const jobWorkflowByID = useMemo(() => {
    const map = new Map<number, JobWorkflowState>();
    for (const job of jobs) {
      const app = jobAppsMap.get(job.id);
      const versions = resumeVersionsByJobID.get(job.id) || [];
      const drafts = bulletDrafts.filter((draft) => draft.job_id === job.id);
      map.set(job.id, buildJobWorkflowState(job, app, versions, drafts, selectedJobID === job.id ? jobRequirements : []));
    }
    return map;
  }, [jobs, jobAppsMap, resumeVersionsByJobID, bulletDrafts, selectedJobID, jobRequirements]);

  const selectedJobWorkflow = selectedJob ? jobWorkflowByID.get(selectedJob.id) : undefined;
  const selectedJobApplication = selectedJobID ? jobAppsMap.get(selectedJobID) : undefined;

  const filteredJobs = useMemo(() => {
    const query = jobSearch.trim().toLowerCase();
    return jobs.filter((job) => {
      const app = jobAppsMap.get(job.id);
      if (jobStatusFilter === 'active' && app?.status && ['rejected', 'offer'].includes(app.status)) return false;
      if (jobStatusFilter === 'no_resume' && (resumeVersionsByJobID.get(job.id)?.length || generatedResumeDrafts.some((draft) => draft.job_id === job.id))) return false;
      if (jobStatusFilter !== 'all' && jobStatusFilter !== 'active' && jobStatusFilter !== 'no_resume' && app?.status !== jobStatusFilter) return false;
      if (!query) return true;
      return [job.company, job.title, job.url, app?.status || ''].join(' ').toLowerCase().includes(query);
    });
  }, [jobs, jobAppsMap, resumeVersionsByJobID, generatedResumeDrafts, jobSearch, jobStatusFilter]);

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
      setJobs(normalizeJobs(nj as JobDescription[] | null | undefined));
      setApplications((na ?? []) as Application[]);
      const loadedJobs = (nj as JobDescription[]) ?? [];
      bulletDraftsRef.current.clear();
      for (const job of loadedJobs) {
        const drafts = normalizeDrafts(await ListTailoredBulletDrafts(job.id).catch(() => []) as TailoredBulletDraft[] | null | undefined);
        bulletDraftsRef.current.set(job.id, drafts);
      }
      setBulletDrafts(Array.from(bulletDraftsRef.current.values()).flat());
      const allVersions: ResumeVersion[] = [];
      for (const job of loadedJobs) {
        const vers = normalizeResumeVersions(await ListResumeVersions(job.id).catch(() => []) as ResumeVersion[] | null | undefined);
        allVersions.push(...vers);
      }
      setResumeVersions(normalizeResumeVersions(allVersions));
      const firstSrc = (nsrc as CandidateSource[] | undefined)?.[0];
      if (!selectedSourceID && firstSrc) setSelectedSourceID(firstSrc.id);
      const firstJob = loadedJobs[0];
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

  async function loadAllResumeVersions() {
    const fetchedJobs = jobs.length > 0 ? jobs : ((await ListJobDescriptions()) as JobDescription[]);
    const allVersions: ResumeVersion[] = [];
    for (const job of fetchedJobs) {
      const vers = normalizeResumeVersions(await ListResumeVersions(job.id).catch(() => []) as ResumeVersion[] | null | undefined);
      allVersions.push(...vers);
    }
    setResumeVersions(normalizeResumeVersions(allVersions));
  }

  async function refreshJobsAndVersions(): Promise<JobDescription[]> {
    const nextJobs = normalizeJobs((await ListJobDescriptions()) as JobDescription[] | null | undefined);
    setJobs(nextJobs);
    const allVersions: ResumeVersion[] = [];
    for (const job of nextJobs) {
      const vers = normalizeResumeVersions(await ListResumeVersions(job.id).catch(() => []) as ResumeVersion[] | null | undefined);
      allVersions.push(...vers);
    }
    setResumeVersions(normalizeResumeVersions(allVersions));
    return nextJobs;
  }

  function replaceDraftsForJob(jobID: number, drafts: TailoredBulletDraft[] | null | undefined) {
    const normalized = normalizeDrafts(drafts);
    bulletDraftsRef.current.set(jobID, normalized);
    setBulletDrafts(Array.from(bulletDraftsRef.current.values()).flat());
  }

  function updateDraftInCache(saved: TailoredBulletDraft) {
    const existing = bulletDraftsRef.current.get(saved.job_id) || [];
    bulletDraftsRef.current.set(saved.job_id, normalizeDrafts(existing.map((draft) => draft.id === saved.id ? saved : draft)));
    setBulletDrafts(Array.from(bulletDraftsRef.current.values()).flat());
  }

  function removeDraftFromCache(draftID: number) {
    for (const [jobID, drafts] of bulletDraftsRef.current.entries()) {
      if (drafts.some((draft) => draft.id === draftID)) {
        bulletDraftsRef.current.set(jobID, drafts.filter((draft) => draft.id !== draftID));
        break;
      }
    }
    setBulletDrafts(Array.from(bulletDraftsRef.current.values()).flat());
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
    const [nSrc, nSec, nF, nCl, nRuns, nJobs, nApps] = await Promise.all([
      ListCandidateSources(), ListSourceSections(0), ListEvidenceFacts('all'),
      ListCandidateClaims('all'), ListContextAgentRuns(0), ListJobDescriptions(), ListApplications(),
    ]);
    setSources(normalizeSources(nSrc as CandidateSource[] | null | undefined));
    setSections((nSec ?? []) as SourceSection[]);
    setFacts(normalizeFacts(nF as EvidenceFact[] | null | undefined));
    setClaims(normalizeClaims(nCl as CandidateClaim[] | null | undefined));
    setContextRuns(normalizeContextRuns(nRuns as ContextAgentRun[] | null | undefined));
    setJobs(normalizeJobs(nJobs as JobDescription[] | null | undefined));
    setApplications((nApps ?? []) as Application[]);
    await refreshEvents();
  }

  async function generateResume() {
    if (!selectedJobID) return;
    await runAction('generate-resume', async () => {
      await buildResumeFromBullets(selectedJobID, selectedJobDrafts);
    });
  }

  async function buildResumeFromBullets(jobID: number, drafts: TailoredBulletDraft[]) {
    const input: GenerateResumeJSONInput = {
      job_id: jobID,
      selected_bullet_ids: drafts.filter((d) => d.selected_for_resume).map((d) => d.id),
    };
    const resume = normalizeResumeJSON((await GenerateResumeJSON(input)) as ResumeJSON | null | undefined);
    setActiveResume(resume);
    const validation = normalizeValidationResult((await ValidateResumeJSON(resume, jobID)) as ValidationResult | null | undefined);
    setActiveValidation(validation);
    const generated: GeneratedResumeDraft = {id: `draft-${Date.now()}`, job_id: jobID, resume, validation, created_at: new Date().toISOString()};
    setGeneratedResumeDrafts((prev) => [generated, ...prev]);
    setPipelineStep('resume');
  }

  function applyJobAgentResult(result: JobAgentWorkflowResult) {
    const saved = result.job;
    setSelectedJobID(saved.id);
    setJobDraft({company: saved.company, title: saved.title, url: saved.url, raw_text: saved.raw_text});
    setJobs((prev) => normalizeJobs([saved, ...prev.filter((job) => job.id !== saved.id)]));
    setJobRequirements(normalizeRequirements(result.requirements));
    setJobMatches(normalizeMatches(result.matches));
    replaceDraftsForJob(saved.id, result.drafts);
    setJobAnalysis(normalizeJobAnalysis(result.analysis));
    setFitAnalysis(normalizeFitAnalysis(result.fit));
    setApplicationStrategy(normalizeApplicationStrategy(result.strategy));
    setJobAgentStages(result.stages || []);
    if (result.resume_generated) {
      const resume = normalizeResumeJSON(result.resume);
      const validation = normalizeValidationResult(result.validation);
      setActiveResume(resume);
      setActiveValidation(validation);
      setGeneratedResumeDrafts((prev) => [{id: `agent-${Date.now()}`, job_id: saved.id, resume, validation, created_at: result.created_at || new Date().toISOString()}, ...prev]);
      setPipelineStep('resume');
    } else {
      setActiveResume(null);
      setActiveValidation(null);
      setPipelineStep('bullets');
    }
  }

  async function runJobAgent(jobID: number, draft: JobDraft) {
    const result = (await RunJobAgentWorkflow({
      job: draft,
      job_id: jobID,
      auto_select_bullets: true,
      build_resume: true,
      min_selected_bullets: 4,
      max_selected_bullets: 10,
      require_resume_review: true,
    })) as JobAgentWorkflowResult;
    applyJobAgentResult(result);
    await refreshJobsAndVersions();
    await refreshEvents();
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
    const generated: GeneratedResumeDraft = {id: `draft-${Date.now()}`, job_id: selectedJobID, resume: cleaned, validation, created_at: new Date().toISOString()};
    setGeneratedResumeDrafts((prev) => [generated, ...prev]);
    setResumeSaveNotice('Edits saved as a new draft. Save a version when it is ready to use.');
  }

  async function saveActiveResumeVersion() {
    if (!activeResume || !selectedJobID) return;
    await runAction('save-resume-version', async () => {
      const validation = activeValidation ?? normalizeValidationResult((await ValidateResumeJSON(activeResume, selectedJobID)) as ValidationResult | null | undefined);
      const version = (await SaveResumeVersion({
        job_id: selectedJobID, resume_json: activeResume, tex_source: '', pdf_path: '', validation_result: validation,
      })) as ResumeVersion;
      setResumeVersions((prev) => [normalizeResumeVersion(version), ...prev.filter((v) => v.id !== version.id)]);
      setApplicationStatusDraft((prev) => prev === 'draft' ? 'ready_to_apply' : prev);
      setResumeSaveNotice('Resume version saved. You can render a PDF or mark the application ready/applied.');
    });
  }

  function updateEditingField(field: string, value: string) {
    setEditingResume((prev) => prev ? { ...prev, [field]: value } : prev);
  }

  function updateEditingSkillCategory(catIdx: number, category: string) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const skills = [...prev.skills];
      skills[catIdx] = { ...skills[catIdx], category };
      return { ...prev, skills };
    });
  }

  function updateEditingSkill(catIdx: number, itemsStr: string) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const skills = [...prev.skills];
      skills[catIdx] = { ...skills[catIdx], items: itemsStr.split(',').map(s => s.trim()).filter(Boolean) };
      return { ...prev, skills };
    });
  }

  function updateEditingEntry(section: 'experience' | 'projects' | 'education', idx: number, field: string, value: string) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const entries = [...prev[section]];
      entries[idx] = { ...entries[idx], [field]: value };
      return { ...prev, [section]: entries };
    });
  }

  function updateEditingBullet(section: 'experience' | 'projects', entryIdx: number, bulletIdx: number, value: string) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const entries = [...prev[section]];
      const bullets = [...entries[entryIdx].bullets];
      bullets[bulletIdx] = value;
      entries[entryIdx] = { ...entries[entryIdx], bullets };
      return { ...prev, [section]: entries };
    });
  }

  function addEditingBullet(section: 'experience' | 'projects', entryIdx: number) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const entries = [...prev[section]];
      entries[entryIdx] = { ...entries[entryIdx], bullets: [...entries[entryIdx].bullets, ''] };
      return { ...prev, [section]: entries };
    });
  }

  function removeEditingBullet(section: 'experience' | 'projects', entryIdx: number, bulletIdx: number) {
    setEditingResume((prev) => {
      if (!prev) return prev;
      const entries = [...prev[section]];
      const bullets = entries[entryIdx].bullets.filter((_, i) => i !== bulletIdx);
      entries[entryIdx] = { ...entries[entryIdx], bullets };
      return { ...prev, [section]: entries };
    });
  }

  function addEditingSkillCategory() {
    setEditingResume((prev) => prev ? { ...prev, skills: [...prev.skills, { category: 'New Category', items: [] }] } : prev);
  }

  function removeEditingSkillCategory(catIdx: number) {
    setEditingResume((prev) => prev ? { ...prev, skills: prev.skills.filter((_, i) => i !== catIdx) } : prev);
  }

  function addEditingEntry(section: 'experience' | 'projects' | 'education') {
    const newEntry = section === 'education'
      ? { organization: '', degree: '', location: '', end_date: '', start_date: '', bullets: [], claim_ids: [], bullet_ids: [] }
      : { company: '', title: '', location: '', start_date: '', end_date: '', bullets: [''], claim_ids: [], bullet_ids: [], url: '' };
    setEditingResume((prev) => prev ? { ...prev, [section]: [...prev[section], newEntry] } : prev);
  }

  function removeEditingEntry(section: 'experience' | 'projects' | 'education', idx: number) {
    setEditingResume((prev) => prev ? { ...prev, [section]: prev[section].filter((_, i) => i !== idx) } : prev);
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
      if (!result.success && result.error) {
        setError(result.error);
      } else if (result.success && result.pdf_path) {
        setPDFSuccess({ path: result.pdf_path, jobTitle: selectedJob?.title || 'Resume', jobID: selectedJobID });
        await OpenFolder(result.pdf_path);
      }
    });
  }

  async function saveApplication(statusOverride?: string, jobIDOverride?: number) {
    const targetJobID = jobIDOverride || selectedJobID;
    if (!targetJobID) return;
    await runAction('save-application', async () => {
      const jobID = targetJobID;
      const existingApp = applications.find((a) => a.job_id === jobID);
      const app = {
        id: existingApp?.id || 0, job_id: jobID, status: statusOverride || applicationStatusDraft || existingApp?.status || 'draft',
        fit_score: fitAnalysis?.overall_score || existingApp?.fit_score || 0, resume_version_id: resumeVersionsByJobID.get(jobID)?.[0]?.id || existingApp?.resume_version_id || 0,
        cover_letter_version_id: 0, notes: applicationNotes,
      } as Application;
      const saved = (await SaveApplication(app)) as Application;
      setSelectedApplicationID(saved.id);
      setApplicationStatusDraft(saved.status);
      const na = (await ListApplications()) as Application[];
      setApplications(na);
    });
  }

  async function markApplicationStatus(status: string, jobID = selectedJobID) {
    const existingApp = applications.find((app) => app.job_id === jobID);
    if (existingApp) {
      await updateApplicationStatus(existingApp.id, status);
      if (jobID === selectedJobID) setApplicationStatusDraft(status);
      return;
    }
    await saveApplication(status, jobID);
  }

  async function updateApplicationStatus(id: number, status: string) {
    await runAction(`upd-app-${id}`, async () => {
      const u = (await UpdateApplicationStatus(id, status)) as Application;
      setApplications((prev) => prev.map((a) => a.id === id ? u : a));
    });
  }

  async function updateApplicationResumeVersion(id: number, resumeVersionID: number) {
    await runAction(`upd-resume-${id}`, async () => {
      const u = (await UpdateApplicationResumeVersion(id, resumeVersionID)) as Application;
      setApplications((prev) => prev.map((a) => a.id === id ? u : a));
    });
  }

async function deleteJob(jobID: number) {
    await runAction(`del-job-${jobID}`, async () => {
      await DeleteJobDescription({id: jobID});
      bulletDraftsRef.current.delete(jobID);
      setBulletDrafts(Array.from(bulletDraftsRef.current.values()).flat());
      if (selectedJobID === jobID) {
        setSelectedJobID(0); setActiveResume(null); setActiveValidation(null);
      }
      const nextJobs = await refreshJobsAndVersions();
      if (selectedJobID === jobID && nextJobs[0]) selectJob(nextJobs[0]);
    });
  }

async function saveNewJob(event: FormEvent<HTMLFormElement>) {
     event.preventDefault();
     await runAction('save-new-job', async () => {
       const saved = (await CreateJobDescription(newJobDraft)) as JobDescription;
        setSelectedJobID(saved.id);
        setJobDraft({company: saved.company, title: saved.title, url: saved.url, raw_text: saved.raw_text});
        setNewJobDraft(emptyJobDraft);
        setExtensionImportNotice('');
        await refreshJobsAndVersions();
       await refreshJobContext(saved.id);
       setShowAddJob(false);
     });
   }

  async function saveNewJobAndRun() {
    await runAction('save-new-job-agent', async () => {
      setActiveView('pipeline');
      setJobAgentStages([]);
      const result = (await RunJobAgentWorkflow({
        job: newJobDraft,
        job_id: 0,
        auto_select_bullets: true,
        build_resume: true,
        min_selected_bullets: 4,
        max_selected_bullets: 10,
        require_resume_review: true,
      })) as JobAgentWorkflowResult;
      applyJobAgentResult(result);
      setNewJobDraft(emptyJobDraft);
      setExtensionImportNotice('');
      setShowAddJob(false);
      await refreshJobsAndVersions();
      await refreshEvents();
    });
  }

  function draftFromFetch(result: FetchJobDescriptionResult): JobDraft {
    return {
      company: result.company || '',
      title: result.title || '',
      url: result.url || '',
      raw_text: result.raw_text || '',
    };
  }

  async function handleFetchNewJobFromURL() {
    const url = newJobDraft.url.trim();
    if (!url) { setError('Posting URL is required.'); setLoadState('error'); return; }
    await runAction('fetch-new-job', async () => {
      const result = (await FetchJobDescription({url})) as FetchJobDescriptionResult;
      setNewJobDraft(draftFromFetch(result));
      setNewJobFetchWarnings(result.warnings || []);
    });
  }

  async function handleFetchNewJobAndParse() {
    const url = newJobDraft.url.trim();
    if (!url) { setError('Posting URL is required.'); setLoadState('error'); return; }
    await runAction('fetch-new-job-parse', async () => {
      const result = (await FetchJobDescription({url})) as FetchJobDescriptionResult;
      const draft = draftFromFetch(result);
      setNewJobDraft(draft);
      setNewJobFetchWarnings(result.warnings || []);
      const saved = (await CreateJobDescription(draft)) as JobDescription;
      setSelectedJobID(saved.id);
      setJobDraft({company: saved.company, title: saved.title, url: saved.url, raw_text: saved.raw_text});
      const requirements = (await ParseJobDescription(saved.id)) as JobRequirement[];
      setJobRequirements(normalizeRequirements(requirements));
      const analysis = (await AnalyzeJobDescription(saved.id)) as JobAnalysis;
      setJobAnalysis(normalizeJobAnalysis(analysis));
      setJobMatches([]); setFitAnalysis(null); setApplicationStrategy(null); replaceDraftsForJob(saved.id, []);
      setShowAddJob(false);
      setPipelineStep('job');
      await refreshJobsAndVersions();
      await refreshEvents();
    });
  }

  function selectJob(job: JobDescription) {
    setShowAddJob(false);
    setExtensionImportNotice('');
    setResumeSaveNotice('');
    setSelectedJobID(job.id);
    setJobDraft({company: job.company, title: job.title, url: job.url, raw_text: job.raw_text});
    const existingApp = jobAppsMap.get(job.id);
    setApplicationNotes(existingApp?.notes || '');
    setApplicationStatusDraft(existingApp?.status || 'draft');
    const generated = generatedResumeDrafts.find((draft) => draft.job_id === job.id);
    const jobVersions = resumeVersionsByJobID.get(job.id) || [];
    if (generated) {
      setActiveResume(generated.resume);
      setActiveValidation(generated.validation);
    } else if (jobVersions.length > 0) {
      setActiveResume(normalizeResumeJSON(jobVersions[0].resume_json));
      setActiveValidation(normalizeValidationResult(jobVersions[0].validation_result));
    } else {
      setActiveResume(null);
      setActiveValidation(null);
    }
    setPipelineStep('job');
    void refreshJobContext(job.id);
  }

  function updateJobDraft(nextDraft: JobDraft) {
    setJobDraft((prev) => applyJobDraftInference(prev, nextDraft));
  }

  function updateNewJobDraft(nextDraft: JobDraft) {
    setNewJobDraft((prev) => applyJobDraftInference(prev, nextDraft));
  }

  function cancelNewJob() {
    setNewJobDraft(emptyJobDraft);
    setExtensionImportNotice('');
    setShowAddJob(false);
  }

  function toggleNewJobForm() {
    setNewJobDraft(emptyJobDraft);
    setExtensionImportNotice('');
    setShowAddJob((prev) => !prev);
  }

  async function refreshJobContext(jobID = selectedJobID) {
    const requestID = ++jobContextRequestRef.current;
    if (!jobID) { setJobRequirements([]); setJobMatches([]); setBulletDrafts([]); setJobAnalysis(null); setFitAnalysis(null); setApplicationStrategy(null); return; }
    const [req, mat, dra, gen, ana, fit, strat, vers] = await Promise.all([
      ListJobRequirements(jobID), ListJobFactMatches(jobID), ListTailoredBulletDrafts(jobID),
      ListBulletGenerationEvents(jobID).catch(() => []),
      GetJobAnalysis(jobID).catch(() => null), GetFitAnalysis(jobID).catch(() => null),
      GetApplicationStrategy(jobID).catch(() => null), ListResumeVersions(jobID).catch(() => []),
    ]);
    if (requestID !== jobContextRequestRef.current) return;
    setJobRequirements(normalizeRequirements(req as JobRequirement[] | null | undefined));
    setJobMatches(normalizeMatches(mat as JobFactMatch[] | null | undefined));
    replaceDraftsForJob(jobID, dra as TailoredBulletDraft[] | null | undefined);
    setJobAnalysis(normalizeJobAnalysis(ana as JobAnalysis | null | undefined));
    setFitAnalysis(normalizeFitAnalysis(fit as JobFitAnalysis | null | undefined));
    setApplicationStrategy(normalizeApplicationStrategy(strat as ApplicationStrategy | null | undefined));
    const jobVersions = normalizeResumeVersions(vers as ResumeVersion[] | null | undefined);
    setResumeVersions((prev) => normalizeResumeVersions([...jobVersions, ...prev.filter((v) => v.job_id !== jobID)]));
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
       await refreshJobsAndVersions();
       await refreshJobContext(saved.id);
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

  function handleFetchSelectedJobFromURL() {
    const url = jobDraft.url.trim();
    if (!url) { setError('Posting URL is required.'); setLoadState('error'); return; }
    void runAction('fetch-job', async () => {
      const result = (await FetchJobDescription({url})) as FetchJobDescriptionResult;
      setJobDraft(draftFromFetch(result));
      setJobFetchWarnings(result.warnings || []);
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
        replaceDraftsForJob(selectedJobID, d);
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

function runAgenticPipeline() {
     void runAction('agentic-pipeline', async () => {
       setActiveResume(null);
       setActiveValidation(null);
       setJobAgentStages([]);
       await runJobAgent(selectedJobID, jobDraft);
    });
  }

  function handleEditBullet(draft: TailoredBulletDraft, newText: string) {
    void runAction(`edit-${draft.id}`, async () => {
      const saved = (await UpdateTailoredBulletDraft({
        id: draft.id, draft_text: newText, rationale: draft.rationale, status: draft.status, risk_flags: draft.risk_flags,
      })) as TailoredBulletDraft;
      updateDraftInCache(saved);
    });
  }

  function handleDeleteBullet(draftID: number) {
    void runAction(`del-${draftID}`, async () => {
      await DeleteTailoredBulletDraft({id: draftID});
      removeDraftFromCache(draftID);
    });
  }

  function handleAutoSelect() {
    if (!selectedJobID) return;
    void runAction('auto-sel', async () => {
      const d = (await AutoSelectResumeBullets(selectedJobID)) as TailoredBulletDraft[];
      replaceDraftsForJob(selectedJobID, d);
    });
  }

  function handleToggleBullet(draft: TailoredBulletDraft) {
    void runAction(`sel-${draft.id}`, async () => {
      const s = (await SelectTailoredBulletDraft({id: draft.id, selected: !draft.selected_for_resume})) as TailoredBulletDraft;
      updateDraftInCache(s);
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

  async function pollExtensionJobDraft() {
    const imported = (await GetPendingExtensionJobDraft()) as ExtensionJobDraft | null;
    if (!imported?.raw_text) return;
    setNewJobDraft({company: imported.company || '', title: imported.title || '', url: imported.url || '', raw_text: imported.raw_text || ''});
    setNewJobFetchWarnings(imported.warnings || []);
    setExtensionImportNotice(`Imported from ${imported.source || 'Firefox extension'} at ${imported.received_at ? imported.received_at.slice(11, 19) : 'now'}.`);
    setShowAddJob(true);
    setActiveView('pipeline');
    setPipelineStep('job');
  }

  useEffect(() => { load(); }, []);

  useEffect(() => {
    const id = window.setInterval(() => { void pollExtensionJobDraft().catch(() => null); }, 2000);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => { for (const run of runningContextRuns) ensureContextAgentPolling(run.id, run.source_id); }, [contextRuns]);

  useEffect(() => {
    if (jobs.length > 0) {
      loadAllResumeVersions();
    }
  }, [jobs]);

  const statusText = loadState === 'loading' ? 'Loading' : loadState === 'error' ? 'Needs attention' : 'Ready';

  const pipelineSteps: {key: PipelineStep; label: string; icon: React.ReactNode}[] = [
    {key: 'job', label: 'JD', icon: <BriefcaseBusiness size={14} />},
    {key: 'bullets', label: 'Evidence', icon: <ListChecks size={14} />},
    {key: 'resume', label: 'Resume', icon: <FileText size={14} />},
    {key: 'tracker', label: 'Track', icon: <Gauge size={14} />},
  ];

  function getJobProgress(jobID: number): {step: PipelineStep; pct: number} {
    const wf = jobWorkflowByID.get(jobID);
    return {step: wf?.stage || 'job', pct: wf?.pct || 10};
  }

  function openJob(job: JobDescription, step: PipelineStep = 'job') {
    selectJob(job);
    setActiveView('pipeline');
    setPipelineStep(step);
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

        <div className="grid grid-cols-4 gap-1 border-b border-slate-200 p-2">
          <button
            className={`rounded-lg py-2 text-xs font-semibold transition ${activeView === 'dashboard' ? 'bg-slate-950 text-white shadow-sm' : 'text-slate-500 hover:bg-slate-100 hover:text-slate-800'}`}
            onClick={() => setActiveView('dashboard')}
          >
            Home
          </button>
          <button
            className={`rounded-lg py-2 text-xs font-semibold transition ${activeView === 'pipeline' ? 'bg-slate-950 text-white shadow-sm' : 'text-slate-500 hover:bg-slate-100 hover:text-slate-800'}`}
            onClick={() => setActiveView('pipeline')}
          >
            Work
          </button>
          <button
            className={`rounded-lg py-2 text-xs font-semibold transition ${activeView === 'sources' ? 'bg-slate-950 text-white shadow-sm' : 'text-slate-500 hover:bg-slate-100 hover:text-slate-800'}`}
            onClick={() => setActiveView('sources')}
          >
            Sources
          </button>
          <button
            className={`rounded-lg py-2 text-xs font-semibold transition ${activeView === 'settings' ? 'bg-slate-950 text-white shadow-sm' : 'text-slate-500 hover:bg-slate-100 hover:text-slate-800'}`}
            onClick={() => setActiveView('settings')}
          >
            Setup
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-3">
          <div className="mb-3 flex items-center justify-between px-1">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Jobs</span>
            <span className="text-[10px] font-semibold text-slate-400">{filteredJobs.length}/{jobs.length}</span>
            <button className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700" onClick={() => { setShowAddJob(true); setActiveView('pipeline'); }} title="Import job">
              <Plus size={14} />
            </button>
          </div>

          <div className="mb-3 space-y-2">
            <input type="search" value={jobSearch} onChange={(event) => setJobSearch(event.target.value)} placeholder="Search jobs..." className="w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-medium text-slate-800 outline-none transition placeholder:text-slate-400 focus:border-slate-300 focus:bg-white" />
            <select value={jobStatusFilter} onChange={(event) => setJobStatusFilter(event.target.value)} className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-600 outline-none focus:border-slate-300">
              <option value="all">All jobs</option>
              <option value="active">Active jobs</option>
              <option value="no_resume">Needs resume</option>
              {Object.entries(APP_STATUS_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
            </select>
          </div>

          {jobs.length === 0 ? (
            <button type="button" onClick={() => { setShowAddJob(true); setActiveView('pipeline'); }} className="w-full rounded-2xl border border-dashed border-slate-200 px-3 py-8 text-center text-xs font-semibold text-slate-400 transition hover:border-slate-300 hover:bg-slate-50 hover:text-slate-600">Import your first job</button>
          ) : (
            <div className="space-y-1">
              {filteredJobs.map((job) => {
                const app = jobAppsMap.get(job.id);
                const prog = getJobProgress(job.id);
                const wf = jobWorkflowByID.get(job.id);
                const isSelected = job.id === selectedJobID;
                const jobVersions = resumeVersionsByJobID.get(job.id) || [];
                const latestVer = jobVersions[0];
                return (
                  <div
                    key={job.id}
                    className={`w-full rounded-lg p-2.5 text-left transition-colors ${isSelected ? 'bg-slate-100 ring-1 ring-slate-300' : 'hover:bg-slate-50'}`}
                  >
                    <div className="flex items-start gap-2">
                      <button type="button" className="min-w-0 flex-1 text-left" onClick={() => selectJob(job)}>
                        <div className="flex items-start justify-between gap-2">
                          <div className="min-w-0">
                            <p className="truncate text-xs font-semibold text-slate-950">{job.title || 'Untitled'}</p>
                            <p className="truncate text-xs text-slate-500">{job.company || 'No company'}</p>
                          </div>
                          <span className={`shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium ${workflowToneClass(wf?.tone)}`}>
                            {app ? APP_STATUS_LABELS[app.status] || app.status : latestVer ? 'Saved' : wf?.label || `${prog.pct}%`}
                          </span>
                        </div>
                      </button>
                      <button
                        type="button"
                        className="shrink-0 rounded-md p-1 text-slate-400 hover:bg-red-50 hover:text-red-600"
                        title="Delete job"
                        onClick={(event) => { event.stopPropagation(); deleteJob(job.id); }}
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                    <div className="mt-2 h-1.5 w-full rounded-full bg-slate-200">
                      <div className="h-1.5 rounded-full bg-slate-600" style={{width: `${prog.pct}%`}} />
                    </div>
                  </div>
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

        {pdfSuccess && (
          <div className="mx-6 mt-4 flex flex-wrap items-center gap-3 rounded-2xl border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-900 shadow-sm shadow-green-100/70">
            <FileText className="mt-0.5 shrink-0" size={16} />
            <div className="flex-1 min-w-0">
              <p className="font-medium">PDF generated: {pdfSuccess.jobTitle}</p>
              <p className="mt-1 text-xs text-green-700 truncate">{pdfSuccess.path}</p>
            </div>
            {pdfSuccess.jobID > 0 && (
              <>
                <button
                  onClick={() => markApplicationStatus('ready_to_apply', pdfSuccess.jobID)}
                  className="shrink-0 rounded-xl border border-green-300 bg-white px-3 py-2 text-xs font-bold text-green-800 shadow-sm transition hover:-translate-y-0.5 hover:bg-green-50 hover:shadow-md"
                >
                  Mark Ready
                </button>
                <button
                  onClick={() => markApplicationStatus('applied', pdfSuccess.jobID)}
                  className="shrink-0 rounded-xl bg-green-700 px-3 py-2 text-xs font-bold text-white shadow-sm transition hover:-translate-y-0.5 hover:bg-green-800 hover:shadow-md"
                >
                  Mark Applied
                </button>
              </>
            )}
            <button
              onClick={async () => { await OpenFolder(pdfSuccess.path); setPDFSuccess(null); }}
              className="shrink-0 rounded-xl border border-green-300 bg-white px-3 py-2 text-xs font-bold text-green-700 transition hover:bg-green-50"
            >
              <Folder className="inline mr-1" size={12} /> Open Folder
            </button>
            <button onClick={() => setPDFSuccess(null)} className="shrink-0 text-green-600 hover:text-green-800"><X size={14} /></button>
          </div>
        )}

{activeWorkItems.length > 0 && (
           <WorkBanner busyAction={busyAction} items={activeWorkItems} onStopAgent={() => {}} />
         )}

         {activeView === 'dashboard' && (
           <div className="flex-1 overflow-y-auto p-6">
             <div className="mx-auto max-w-6xl space-y-6">
               <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                 <div className="flex flex-wrap items-center justify-between gap-4">
                   <div>
                      <h2 className="text-2xl font-bold tracking-tight text-slate-950">Resume application tracker</h2>
                      <p className="mt-1 text-sm text-slate-500">Import jobs, generate tailored resumes, save versions, and track each application.</p>
                   </div>
                   <button className="inline-flex items-center gap-2 rounded-xl bg-slate-950 px-4 py-2.5 text-xs font-bold text-white shadow-sm shadow-slate-300/80 transition hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-md" onClick={() => { setShowAddJob(true); setActiveView('pipeline'); }}>
                      <Plus size={14} /> Import job
                   </button>
                 </div>
                 <div className="mt-5 grid gap-3 sm:grid-cols-3">
                    <TrackerStat label="Jobs" value={jobs.length} detail="tracked postings" />
                    <TrackerStat label="Resumes" value={resumeVersions.length + generatedResumeDrafts.length} detail="drafts + saved versions" />
                   <TrackerStat label="Sources" value={sources.length} detail="evidence inputs" />
                 </div>
               </div>

                {jobs.length === 0 ? (
                  <EmptyState text="No jobs yet. Add a JD to start the agentic workflow." />
                ) : filteredJobs.length === 0 ? (
                  <EmptyState text="No jobs match the current search/filter." />
                ) : (
                  <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
                    {filteredJobs.map((job) => {
                     const app = jobAppsMap.get(job.id);
                     const wf = jobWorkflowByID.get(job.id) || buildJobWorkflowState(job, app, resumeVersionsByJobID.get(job.id) || [], [], []);
                     const jobVersions = resumeVersionsByJobID.get(job.id) || [];
                     return (
                       <div key={job.id} className="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm shadow-slate-200/60 transition hover:-translate-y-0.5 hover:shadow-md">
                         <div className="flex items-start justify-between gap-3">
                           <button type="button" className="min-w-0 flex-1 text-left" onClick={() => openJob(job, wf.stage)}>
                             <p className="truncate text-sm font-bold text-slate-950">{job.title || 'Untitled role'}</p>
                             <p className="truncate text-xs text-slate-500">{job.company || 'No company'}</p>
                           </button>
                           <StatusBadge text={wf.label} color={wf.tone} />
                         </div>
                         <div className="mt-4">
                           <div className="flex items-center justify-between text-[10px] font-semibold uppercase tracking-wider text-slate-400">
                             <span>Workflow progress</span><span>{wf.pct}%</span>
                           </div>
                           <div className="mt-1.5 h-2 w-full rounded-full bg-slate-100">
                             <div className="h-2 rounded-full bg-gradient-to-r from-blue-500 via-slate-800 to-emerald-500 transition-all" style={{width: `${wf.pct}%`}} />
                           </div>
                         </div>
                         <div className="mt-3 grid grid-cols-2 gap-2 text-center text-xs">
                           <MiniMetric label="Resume" value={jobVersions.length ? 'Saved' : generatedResumeDrafts.some((draft) => draft.job_id === job.id) ? 'Draft' : '-'} />
                           <MiniMetric label="Fit" value={app?.fit_score ? `${app.fit_score}%` : '-'} />
                         </div>
                         <div className="mt-3 flex flex-wrap items-center gap-2">
                           <PipelineButton label="Open" onClick={() => openJob(job, wf.stage)} />
                           {app ? (
                             <select
                               className="rounded-md border border-slate-200 bg-white px-2 py-1.5 text-xs font-semibold text-slate-700 focus:border-slate-400 focus:outline-none"
                               value={app.status}
                               onClick={(e) => e.stopPropagation()}
                               onChange={(e) => { e.stopPropagation(); updateApplicationStatus(app.id, e.target.value); }}
                             >
                               {Object.entries(APP_STATUS_LABELS).map(([k, l]) => <option key={k} value={k}>{l}</option>)}
                             </select>
                           ) : (
                             <span className="rounded-md border border-dashed border-slate-200 px-2 py-1.5 text-xs font-medium text-slate-400">No application</span>
                           )}
                         </div>
                       </div>
                     );
                   })}
                 </div>
               )}
             </div>
           </div>
         )}

         {activeView === 'pipeline' && showAddJob && (
           <div className="flex-1 overflow-y-auto p-6">
             <div className="mx-auto max-w-5xl">
               <div className="mb-5 flex items-center justify-between gap-4">
                 <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Import job</p>
                    <h2 className="mt-1 text-2xl font-bold tracking-tight text-slate-950">Turn a posting into a tailored resume draft</h2>
                    <p className="mt-1 text-sm text-slate-500">Use the Firefox LinkedIn button, fetch a public URL, or paste text. Then run the workflow once.</p>
                 </div>
                 <SecondaryButton label="Cancel" onClick={cancelNewJob} />
               </div>

               <form onSubmit={saveNewJob} className="grid gap-5 lg:grid-cols-[1fr_320px]">
                  <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                    {extensionImportNotice && (
                      <div className="mb-4 rounded-2xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-xs text-emerald-800">
                        <span className="font-bold">LinkedIn import ready.</span> {extensionImportNotice}
                      </div>
                    )}
                    <TextArea label="Job description" rows={22} value={newJobDraft.raw_text} onChange={(v) => updateNewJobDraft({...newJobDraft, raw_text: v})} />
                   <div className="mt-3 flex flex-wrap gap-1.5">
                     <StatusBadge text={`${newJobDraft.raw_text.length}/50000 chars`} color={newJobDraft.raw_text.length > 50000 ? 'red' : 'slate'} />
                     {(newJobDraft.company || newJobInferred.company) && <StatusBadge text={newJobDraft.company || newJobInferred.company} color="blue" />}
                     {(newJobDraft.title || newJobInferred.title) && <StatusBadge text={newJobDraft.title || newJobInferred.title} color="green" />}
                   </div>
                 </div>

                  <div className="space-y-4">
                    <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                      <div className="space-y-3">
                        <TextInput label="Company" value={newJobDraft.company} onChange={(v) => updateNewJobDraft({...newJobDraft, company: v})} />
                        <TextInput label="Role" value={newJobDraft.title} onChange={(v) => updateNewJobDraft({...newJobDraft, title: v})} />
                        <TextInput label="Posting URL" value={newJobDraft.url} onChange={(v) => updateNewJobDraft({...newJobDraft, url: v})} />
                      </div>
                      <div className="mt-4 rounded-2xl border border-slate-100 bg-slate-50 p-3">
                        <p className="text-xs font-bold uppercase tracking-[0.16em] text-slate-400">Import source</p>
                        <p className="mt-1 text-xs leading-5 text-slate-500">LinkedIn works best through the Firefox extension. Public company career pages can be fetched here.</p>
                        <div className="mt-3 grid gap-2">
                          <button type="button" disabled={!newJobDraft.url.trim() || busyAction !== ''} onClick={handleFetchNewJobFromURL} className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-xs font-bold text-slate-700 shadow-sm shadow-slate-200/70 transition hover:bg-slate-50 disabled:pointer-events-none disabled:opacity-40">
                            Fetch from URL
                          </button>
                          <button type="button" disabled={!newJobDraft.url.trim() || busyAction !== ''} onClick={handleFetchNewJobAndParse} className="rounded-xl bg-blue-600 px-4 py-2.5 text-xs font-bold text-white shadow-sm shadow-blue-200 transition hover:bg-blue-700 disabled:pointer-events-none disabled:opacity-40">
                            Fetch and analyze
                          </button>
                        </div>
                        {(newJobFetchWarnings.length > 0 || newJobDraft.raw_text) && (
                          <div className="mt-3 space-y-1 text-[11px] text-slate-500">
                            {newJobDraft.raw_text && <p>{newJobDraft.raw_text.length} extracted characters.</p>}
                            {newJobFetchWarnings.map((warning, index) => <p key={index} className="text-amber-700">{warning}</p>)}
                          </div>
                        )}
                      </div>
                    </div>

                   <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                      <p className="text-sm font-bold text-slate-950">Next step</p>
                      <p className="mt-1 text-xs leading-5 text-slate-500">Create a resume draft in one run. Save a version from the Resume step when you are happy with it.</p>
                      <div className="mt-4 grid gap-2">
                        <button type="button" disabled={!newJobDraft.raw_text.trim() || busyAction !== '' || newJobDraft.raw_text.length > 50000} onClick={saveNewJobAndRun} className="rounded-xl bg-slate-950 px-4 py-3 text-xs font-bold text-white shadow-sm shadow-slate-300/80 transition hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-md disabled:pointer-events-none disabled:opacity-40">
                          Create resume draft
                        </button>
                        <button type="submit" disabled={!newJobDraft.raw_text.trim() || busyAction !== '' || newJobDraft.raw_text.length > 50000} className="rounded-xl border border-slate-200 bg-white px-4 py-3 text-xs font-bold text-slate-700 shadow-sm shadow-slate-200/70 transition hover:-translate-y-0.5 hover:border-slate-300 hover:bg-slate-50 hover:shadow-md disabled:pointer-events-none disabled:opacity-40">
                          Save JD for later
                        </button>
                     </div>
                   </div>
                 </div>
               </form>
             </div>
           </div>
         )}

{activeView === 'pipeline' && selectedJob && !showAddJob && (
           <div className="flex flex-1 flex-col overflow-hidden">
             <div className="border-b border-slate-200 bg-white px-6 py-3">
               <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="truncate text-sm font-semibold text-slate-950">{selectedJob.title || 'Untitled'}</p>
                      {selectedJobWorkflow && <StatusBadge text={selectedJobWorkflow.label} color={selectedJobWorkflow.tone} />}
                      {selectedJobApplication?.status && <StatusBadge text={APP_STATUS_LABELS[selectedJobApplication.status] || selectedJobApplication.status} color="green" />}
                    </div>
                    <p className="truncate text-xs text-slate-500">{selectedJob.company || 'No company'} · {selectedJobWorkflow?.pct || 18}% workflow</p>
                  </div>
                  <div className="flex items-center gap-2">
                    {pipelineSteps.map((step) => (
                      <button
                        key={step.key}
                        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${pipelineStep === step.key ? 'bg-slate-900 text-white shadow-sm' : 'border border-slate-200 bg-white text-slate-600 hover:bg-slate-50'}`}
                        onClick={() => setPipelineStep(step.key)}
                      >
                        {step.icon}{step.label}
                      </button>
                    ))}
                    <IconButton label="Refresh" onClick={load} disabled={busyAction !== ''}>
                      <RefreshCcw size={14} />
                    </IconButton>
                  </div>
               </div>
             </div>

             <div className="flex-1 overflow-y-auto p-6">
               {pipelineStep === 'job' && (
                <div className="mx-auto max-w-3xl space-y-4">
                   <Panel icon={<BriefcaseBusiness size={16} />} title="Job workspace" compact>
                     <form onSubmit={handleSaveJob} className="space-y-4">
                      <div className="grid gap-3 md:grid-cols-2">
                        <TextInput label="Company" value={jobDraft.company} onChange={(v) => updateJobDraft({...jobDraft, company: v})} />
                        <TextInput label="Role" value={jobDraft.title} onChange={(v) => updateJobDraft({...jobDraft, title: v})} />
                      </div>
                       <div className="grid gap-2 md:grid-cols-[1fr_auto] md:items-end">
                         <TextInput label="URL" value={jobDraft.url} onChange={(v) => updateJobDraft({...jobDraft, url: v})} />
                         <button type="button" disabled={!jobDraft.url.trim() || busyAction !== ''} onClick={handleFetchSelectedJobFromURL} className="inline-flex h-10 items-center justify-center gap-1.5 rounded-md border border-slate-200 bg-white px-4 text-xs font-bold text-slate-700 shadow-sm shadow-slate-200/70 transition-all hover:bg-slate-50 disabled:pointer-events-none disabled:opacity-40">
                            <Search size={14} /> Refresh from URL
                         </button>
                       </div>
                       {(jobFetchWarnings.length > 0 || jobDraft.raw_text) && (
                         <div className="flex flex-wrap gap-1.5">
                           {jobDraft.raw_text && <StatusBadge text={`${jobDraft.raw_text.length}/50000 chars`} color={jobDraft.raw_text.length > 50000 ? 'red' : 'slate'} />}
                           {jobFetchWarnings.map((warning, index) => <StatusBadge key={index} text={warning} color="amber" />)}
                         </div>
                       )}
                       <TextArea label="Description" rows={12} value={jobDraft.raw_text} onChange={(v) => updateJobDraft({...jobDraft, raw_text: v})} />
                        <div className="rounded-2xl border border-blue-100 bg-blue-50/60 p-4">
                          <div className="flex flex-wrap items-center justify-between gap-3">
                            <div>
                              <p className="text-sm font-bold text-slate-950">Resume workflow</p>
                              <p className="mt-1 text-xs text-slate-600">Updates the JD, reruns evidence matching, selects bullets, and creates a reviewable resume draft.</p>
                            </div>
                            <PipelineButton label="Create resume draft" onClick={runAgenticPipeline} tone="primary" disabled={busyAction !== '' || !jobDraft.raw_text.trim()} />
                          </div>
                          {jobAgentStages.length > 0 && (
                            <div className="mt-3 grid gap-1.5">
                              {jobAgentStages.map((stage, index) => (
                                <div key={`${stage.key}-${index}`} className="flex items-center gap-2 rounded-lg bg-white/80 px-3 py-2 text-xs ring-1 ring-blue-100">
                                  <span className={`h-2 w-2 rounded-full ${stage.status === 'ok' ? 'bg-emerald-500' : stage.status === 'warning' ? 'bg-amber-500' : stage.status === 'failed' ? 'bg-red-500' : 'bg-blue-500'}`} />
                                  <span className="font-semibold text-slate-800">{stage.label}</span>
                                  <span className="min-w-0 truncate text-slate-500">{stage.message}</span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        <div className="flex flex-wrap gap-2">
                          <button type="submit" disabled={busyAction !== ''} className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-4 py-2 text-xs font-bold text-slate-700 shadow-sm shadow-slate-200/70 transition-all hover:bg-slate-50 disabled:pointer-events-none disabled:opacity-40">
                            <Save size={14} /> Update JD
                          </button>
                          {selectedJobID > 0 && <SecondaryButton label="Delete" onClick={() => deleteJob(selectedJobID)} variant="red" />}
                        </div>
                    </form>
                  </Panel>

                   {(fitAnalysis || jobAnalysis || applicationStrategy || jobRequirements.length > 0 || jobMatches.length > 0) && (
                     <CollapsibleSection label="Diagnostics">
                      <div className="mt-2 space-y-2 text-xs text-slate-600">
                        {fitAnalysis && <p><span className="font-semibold text-slate-800">Fit:</span> {fitAnalysis.overall_score}% — {fitAnalysis.recommendation}. {fitAnalysis.reality_check}</p>}
                        {jobAnalysis?.role_archetype && <p><span className="font-semibold text-slate-800">Role:</span> {jobAnalysis.role_archetype}</p>}
                        {applicationStrategy?.resume_headline && <p><span className="font-semibold text-slate-800">Headline:</span> {applicationStrategy.resume_headline}</p>}
                        <p><span className="font-semibold text-slate-800">Parsed:</span> {jobRequirements.length} requirements, {jobMatches.length} evidence matches</p>
                      </div>
                    </CollapsibleSection>
                  )}
                </div>
              )}

              {pipelineStep === 'bullets' && (
                <div className="mx-auto max-w-4xl space-y-4">
                  <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                    <div className="flex flex-wrap items-start justify-between gap-3">
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-blue-600">Evidence</p>
                        <h3 className="mt-1 text-lg font-bold text-slate-950">Resume bullet source</h3>
                        <p className="mt-1 text-sm text-slate-500">Use this only when you want to inspect or edit the evidence behind the resume.</p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <PipelineButton label="Generate evidence" onClick={handleBullets} tone="primary" disabled={busyAction !== ''} />
                        <SecondaryButton label="Select best" onClick={handleAutoSelect} icon={<Sparkles size={13} />} disabled={selectedJobDrafts.length === 0 || busyAction !== ''} />
                        <SecondaryButton label="Build resume" onClick={generateResume} icon={<FileText size={13} />} disabled={!selectedJobDrafts.some((draft) => draft.selected_for_resume) || busyAction !== ''} variant="green" />
                      </div>
                    </div>
                  </div>

                  {selectedJobDrafts.length === 0 ? (
                    <EmptyState text="No bullets generated yet. Run the agentic pipeline or generate bullets from this step." />
                  ) : (
                    <div className="space-y-2">
                      {selectedJobDrafts
                        .slice()
                        .sort((a, b) => a.display_order - b.display_order || b.selection_score - a.selection_score)
                        .map((draft) => (
                          <BulletCard key={draft.id} draft={draft} onToggle={() => handleToggleBullet(draft)} onEdit={handleEditBullet} onDelete={handleDeleteBullet} />
                        ))}
                    </div>
                  )}

                  {bulletEvents.length > 0 && (
                    <CollapsibleSection label={`Generation log (${bulletEvents.length})`}>
                      <div className="space-y-1">
                        {bulletEvents.slice(-8).reverse().map((event) => (
                          <div key={event.id} className="rounded-md bg-white px-3 py-2 text-xs text-slate-500 ring-1 ring-slate-100">
                            <span className="font-semibold text-slate-700">{event.stage}</span> {event.reason || event.status}
                          </div>
                        ))}
                      </div>
                    </CollapsibleSection>
                  )}
                </div>
              )}

              {pipelineStep === 'resume' && (
                <div className="mx-auto grid max-w-5xl gap-4">
                  <div className="space-y-4">
                    {!activeResume ? (
                      <div className="rounded-lg border border-slate-200 bg-white p-6 text-center">
                        <p className="text-sm text-slate-600">No active resume draft yet. Run the JD workflow, or build one from selected evidence bullets.</p>
                        <div className="mt-4 flex justify-center gap-2">
                          <PipelineButton label="Build from selected bullets" onClick={generateResume} tone="primary" disabled={!selectedJobDrafts.some((draft) => draft.selected_for_resume) || busyAction !== ''} />
                          <SecondaryButton label="Review evidence" onClick={() => setPipelineStep('bullets')} disabled={selectedJobDrafts.length === 0} />
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
                                    <input className="w-24 rounded border border-slate-200 px-1.5 py-0.5 text-[10px] font-semibold" value={s.category} onChange={e => updateEditingSkillCategory(i, e.target.value)} placeholder="Category" />
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
                        <div className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm shadow-slate-200/70">
                          <div className="flex flex-wrap items-center justify-between gap-3">
                            <div>
                              <p className="text-xs font-bold uppercase tracking-[0.16em] text-slate-400">Resume version</p>
                              <div className="mt-1 flex flex-wrap items-center gap-1.5">
                                {activeResumeSaved && <StatusBadge text="Saved version" color="green" />}
                                {activeResumeDraftOnly && <StatusBadge text="Draft only" color="amber" />}
                                {activeValidation && <StatusBadge text={activeValidation.passed ? 'Validated' : 'Needs fixes'} color={activeValidation.passed ? 'green' : 'red'} />}
                              </div>
                              <p className="mt-1 text-xs text-slate-500">Save a version before rendering a PDF or marking the job ready to apply.</p>
                            </div>
                            {(selectedGeneratedResumes.length > 0 || selectedSavedResumes.length > 0) && (
                              <select className="rounded-xl border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-700" value="" onChange={(e) => {
                              const value = e.target.value;
                              const generated = selectedGeneratedResumes.find((draft) => draft.id === value);
                              if (generated) { setActiveResume(generated.resume); setActiveValidation(generated.validation); return; }
                              const v = selectedSavedResumes.find((rv) => `version-${rv.id}` === value);
                              if (v) { setActiveResume(normalizeResumeJSON(v.resume_json)); setActiveValidation(normalizeValidationResult(v.validation_result)); }
                            }}>
                              <option value="">Switch version...</option>
                              {selectedGeneratedResumes.map((draft) => <option key={draft.id} value={draft.id}>Draft — {formatDateTime(draft.created_at)}</option>)}
                              {selectedSavedResumes.map((v) => <option key={v.id} value={`version-${v.id}`}>Saved — {formatDateTime(v.created_at)}</option>)}
                            </select>
                            )}
                          </div>
                          {resumeSaveNotice && <p className="mt-3 rounded-xl bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-800">{resumeSaveNotice}</p>}
                          <div className="mt-4 grid gap-3 lg:grid-cols-2">
                            <div className="rounded-xl border border-slate-100 bg-slate-50 p-3">
                              <p className="text-xs font-bold text-slate-800">Review draft</p>
                              <div className="mt-3 flex flex-wrap gap-2">
                                <PipelineButton label="Regenerate" onClick={generateResume} disabled={busyAction !== ''} />
                                {activeResume && !editingResume && <SecondaryButton label="Edit" onClick={startEditingResume} variant="blue" disabled={busyAction !== ''} />}
                                <SecondaryButton label={activeResumeSaved ? 'Saved' : 'Save version'} onClick={saveActiveResumeVersion} disabled={!activeResume || activeResumeSaved || busyAction !== ''} variant="green" />
                              </div>
                            </div>
                            <div className="rounded-xl border border-emerald-100 bg-emerald-50/60 p-3">
                              <p className="text-xs font-bold text-emerald-950">Finish application</p>
                              <div className="mt-3 flex flex-wrap gap-2">
                                <SecondaryButton label="Render PDF" onClick={renderPDF} icon={<FilePlus2 size={13} />} disabled={!activeResume || activeResumeDraftOnly || busyAction !== ''} variant="green" />
                                <SecondaryButton label="Mark ready" onClick={() => markApplicationStatus('ready_to_apply')} disabled={!selectedJobID || activeResumeDraftOnly || busyAction !== ''} variant="blue" />
                                <SecondaryButton label="Mark applied" onClick={() => markApplicationStatus('applied')} disabled={!selectedJobID || activeResumeDraftOnly || busyAction !== ''} variant="green" />
                              </div>
                            </div>
                          </div>
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
                    <Panel icon={<BriefcaseBusiness size={14} />} title="Application notes" compact>
                      <div className="mt-2 grid gap-3 md:grid-cols-[220px_1fr_auto] md:items-end">
                        <div>
                          <label className="mb-1 block text-xs font-medium text-slate-500">Status</label>
                          <select value={applicationStatusDraft} onChange={(event) => setApplicationStatusDraft(event.target.value)} className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-800 outline-none focus:border-slate-300">
                            {Object.entries(APP_STATUS_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                          </select>
                        </div>
                        <TextInput label="Notes" value={applicationNotes} onChange={setApplicationNotes} />
                        <IconButton label="Save application" onClick={() => saveApplication()} disabled={!selectedJobID}><Save size={14} /></IconButton>
                      </div>
                    </Panel>
                  </div>
</div>
              )}

              {pipelineStep === 'tracker' && (
                <div className="mx-auto max-w-4xl space-y-4">
                  <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-200/70">
                    <div className="flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-emerald-600">Application tracker</p>
                        <h3 className="mt-1 text-xl font-bold text-slate-950">{selectedJob.title || 'Untitled role'}</h3>
                        <p className="mt-1 text-sm text-slate-500">{selectedJob.company || 'No company'} · keep resume versions, PDF state, and application notes together.</p>
                      </div>
                      <StatusBadge text={APP_STATUS_LABELS[selectedJobApplication?.status || applicationStatusDraft] || applicationStatusDraft} color={selectedJobApplication?.status === 'applied' ? 'green' : 'blue'} />
                    </div>
                    <div className="mt-5 grid gap-3 sm:grid-cols-3">
                      <MiniMetric label="Saved resumes" value={selectedSavedResumes.length} />
                      <MiniMetric label="Draft resumes" value={selectedGeneratedResumes.length} />
                      <MiniMetric label="Selected bullets" value={selectedJobDrafts.filter((draft) => draft.selected_for_resume).length} />
                    </div>
                  </div>

                  <Panel icon={<FileText size={14} />} title="Resume delivery" compact>
                    <div className="mt-2 flex flex-wrap items-center gap-2">
                      <SecondaryButton label="Open resume" onClick={() => setPipelineStep('resume')} variant="blue" />
                      <SecondaryButton label="Save current version" onClick={saveActiveResumeVersion} disabled={!activeResume || activeResumeSaved || busyAction !== ''} variant="green" />
                      <SecondaryButton label="Render PDF" onClick={renderPDF} icon={<FilePlus2 size={13} />} disabled={!activeResume || activeResumeDraftOnly || busyAction !== ''} variant="green" />
                    </div>
                    <p className="mt-3 text-xs text-slate-500">PDF rendering is enabled after a resume version is saved. Unsaved drafts stay editable in the Resume step.</p>
                  </Panel>

                  <Panel icon={<BriefcaseBusiness size={14} />} title="Application status" compact>
                    <div className="mt-2 grid gap-3 md:grid-cols-[220px_1fr_auto] md:items-end">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-slate-500">Status</label>
                        <select value={applicationStatusDraft} onChange={(event) => setApplicationStatusDraft(event.target.value)} className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-800 outline-none focus:border-slate-300">
                          {Object.entries(APP_STATUS_LABELS).map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                        </select>
                      </div>
                      <TextInput label="Notes" value={applicationNotes} onChange={setApplicationNotes} />
                      <IconButton label="Save status" onClick={() => saveApplication()} disabled={!selectedJobID}><Save size={14} /></IconButton>
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      <SecondaryButton label="Mark ready" onClick={() => markApplicationStatus('ready_to_apply')} disabled={!selectedJobID || busyAction !== ''} variant="blue" />
                      <SecondaryButton label="Mark applied" onClick={() => markApplicationStatus('applied')} disabled={!selectedJobID || busyAction !== ''} variant="green" />
                    </div>
                  </Panel>
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
                   <TextInput label="Embedding model" value={settings.embedding_model} onChange={(v) => setSettings({...settings, embedding_model: v})} />
                   <p className="text-xs text-slate-400">OpenAI default: text-embedding-3-small. OpenRouter default: openai/text-embedding-3-small.</p>
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
                  <SecondaryButton label="Render test PDF" onClick={async () => { const r = (await RenderSamplePDF()) as RenderPDFResult; setPDFResult(r); if (r.success && r.pdf_path) setPDFSuccess({ path: r.pdf_path, jobTitle: 'Sample PDF', jobID: 0 }); }} />
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

function TrackerStat({label, value, detail}: {label: string; value: number; detail: string}) {
  return (
    <div className="rounded-xl border border-slate-100 bg-slate-50 px-4 py-3">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-400">{label}</p>
      <p className="mt-1 text-2xl font-bold text-slate-950">{value}</p>
      <p className="text-xs text-slate-500">{detail}</p>
    </div>
  );
}

function MiniMetric({label, value}: {label: string; value: React.ReactNode}) {
  return (
    <div className="rounded-lg border border-slate-100 bg-slate-50 px-3 py-2">
      <p className="text-[10px] font-semibold uppercase tracking-wider text-slate-400">{label}</p>
      <p className="mt-0.5 text-sm font-bold text-slate-900">{value}</p>
    </div>
  );
}

function buildJobWorkflowState(job: JobDescription, app: Application | undefined, versions: ResumeVersion[], drafts: TailoredBulletDraft[], requirements: JobRequirement[]): JobWorkflowState {
  const selectedBullets = drafts.filter((draft) => draft.selected_for_resume).length;
  if (app && app.status !== 'draft') return {pct: 100, stage: 'tracker', label: APP_STATUS_LABELS[app.status] || app.status, tone: 'green', selectedBullets, totalBullets: drafts.length};
  if (app) return {pct: 90, stage: 'tracker', label: 'Ready', tone: 'blue', selectedBullets, totalBullets: drafts.length};
  if (versions.length > 0) return {pct: 82, stage: 'resume', label: 'Saved resume', tone: 'green', selectedBullets, totalBullets: drafts.length};
  if (selectedBullets > 0) return {pct: 68, stage: 'resume', label: 'Bullets selected', tone: 'blue', selectedBullets, totalBullets: drafts.length};
  if (drafts.length > 0) return {pct: 56, stage: 'resume', label: 'Resume ready', tone: 'amber', selectedBullets, totalBullets: drafts.length};
  if (requirements.length > 0) return {pct: 36, stage: 'job', label: 'Matched', tone: 'blue', selectedBullets, totalBullets: drafts.length};
  if (job.raw_text.trim()) return {pct: 18, stage: 'job', label: 'JD saved', tone: 'slate', selectedBullets, totalBullets: drafts.length};
  return {pct: 8, stage: 'job', label: 'Draft', tone: 'slate', selectedBullets, totalBullets: drafts.length};
}

function workflowToneClass(tone: JobWorkflowState['tone'] | undefined) {
  const classes = {
    slate: 'bg-slate-100 text-slate-600',
    blue: 'bg-blue-100 text-blue-700',
    green: 'bg-emerald-100 text-emerald-700',
    amber: 'bg-amber-100 text-amber-700',
    red: 'bg-red-100 text-red-700',
  };
  return classes[tone ?? 'slate'];
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
    default: 'border-slate-200 text-slate-700 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950',
    blue: 'border-blue-200 text-blue-700 hover:border-blue-300 hover:bg-blue-50',
    green: 'border-emerald-200 text-emerald-700 hover:border-emerald-300 hover:bg-emerald-50',
    amber: 'border-amber-200 text-amber-700 hover:border-amber-300 hover:bg-amber-50',
    red: 'border-red-200 text-red-700 hover:border-red-300 hover:bg-red-50',
  };
  return (
    <button onClick={onClick} disabled={disabled} className={`inline-flex items-center justify-center gap-1.5 rounded-xl border bg-white px-3.5 py-2 text-xs font-bold shadow-sm shadow-slate-200/60 transition hover:-translate-y-0.5 hover:shadow-md active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${variants[variant ?? 'default']}`}>
      {icon}{label}
    </button>
  );
}

function PipelineButton({label, onClick, tone, disabled}: {label: string; onClick: () => void; tone?: 'primary'; disabled?: boolean}) {
  const classes = tone === 'primary'
    ? 'border-slate-950 bg-slate-950 text-white shadow-slate-300/80 hover:bg-slate-800 hover:shadow-md'
    : 'border-slate-200 bg-white text-slate-800 shadow-slate-200/70 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950 hover:shadow-md';
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex min-w-20 items-center justify-center rounded-xl border px-4 py-2 text-xs font-bold shadow-sm transition-all hover:-translate-y-0.5 active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${classes}`}
    >
      {label}
    </button>
  );
}

function IconButton({label, onClick, children, submit, full, disabled}: {label: string; onClick?: () => void; children?: React.ReactNode; submit?: boolean; full?: boolean; disabled?: boolean}) {
  return (
    <button onClick={onClick} disabled={disabled || submit} type={submit ? 'submit' : 'button'}
      className={`inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-950 bg-slate-950 px-3.5 py-2 text-xs font-bold text-white shadow-sm shadow-slate-300/80 transition-all hover:-translate-y-0.5 hover:bg-slate-800 hover:shadow-md active:translate-y-0 active:shadow-sm disabled:pointer-events-none disabled:opacity-40 ${full ? 'flex-1 justify-center' : ''}`}>
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

function applyJobDraftInference(prev: JobDraft, nextDraft: JobDraft) {
  if (nextDraft.raw_text === prev.raw_text) return nextDraft;
  const inf = inferJobDetailsFromText(nextDraft.raw_text);
  return {...nextDraft, company: nextDraft.company.trim() ? nextDraft.company : inf.company, title: nextDraft.title.trim() ? nextDraft.title : inf.title};
}

function inferJobDetailsFromText(text: string) {
  const lines = text.split('\n').map(cleanJobHeaderLine).filter((line) => line && !isJobHeaderNoise(line));
  const explicitTitle = firstRegexGroup(text, /(?:job\s*title|role|position)\s*[:\-]\s*([^\n]+)/i);
  const explicitCompany = firstRegexGroup(text, /(?:company|employer|organisation|organization)\s*[:\-]\s*([^\n]+)/i);
  const headerPair = inferHeaderPair(lines);
  const title = cleanInferredValue(explicitTitle || headerPair.title || bestJobTitleLine(lines), 80);
  const company = cleanInferredValue(explicitCompany || headerPair.company || bestCompanyLine(lines, title), 60);
  return {company, title};
}

function cleanJobHeaderLine(line: string) {
  return line.replace(/\s+/g, ' ').replace(/\blogo\b/gi, '').replace(/^[•\-*\s]+/, '').trim();
}

function isJobHeaderNoise(line: string) {
  const lower = line.toLowerCase();
  return !line || lower === 'logo' || lower.includes('premium') || lower.includes('meet the hiring team') ||
    lower.includes('job poster') || lower.includes('promoted by') || lower.includes('actively reviewing') ||
    lower.includes('how your profile') || lower === 'about the job' || lower === 'job description' ||
    lower === 'about us' || lower === 'overview' || /^\d+(st|nd|rd|th)$/i.test(line);
}

function isLocationOrMetaLine(line: string) {
  const lower = line.toLowerCase();
  return lower.includes('applicant') || lower.includes('ago') || lower.includes('reposted') || lower.includes('promoted') ||
    lower.includes('full-time') || lower.includes('part-time') || lower.includes('contract') || lower.includes('internship') ||
    lower.includes('hybrid') || lower.includes('remote') || lower.includes('onsite') || lower.includes('on-site') ||
    lower.includes('australia') || lower.includes('victoria') || lower.includes('melbourne') || lower.includes('sydney') ||
    lower.includes('brisbane') || lower.includes('perth') || lower.includes('canberra') || lower.includes('new south wales') ||
    lower.includes('linkedin') || lower.includes('easy apply') || /^\d+\s+applicants?/i.test(line);
}

function isLikelyJobTitle(line: string) {
  const lower = line.toLowerCase();
  if (isJobHeaderNoise(line) || isLocationOrMetaLine(line)) return false;
  return ['engineer', 'developer', 'manager', 'designer', 'analyst', 'consultant', 'architect', 'lead', 'specialist', 'coordinator', 'administrator', 'officer', 'associate', 'director', 'intern', 'graduate', 'product', 'data', 'software', 'frontend', 'backend', 'full stack', 'devops', 'security', 'qa'].some((word) => lower.includes(word));
}

function firstRegexGroup(text: string, pattern: RegExp) {
  return cleanJobHeaderLine(text.match(pattern)?.[1] ?? '');
}

function inferHeaderPair(lines: string[]) {
  for (const line of lines.slice(0, 12)) {
    const parts = line.split(/[·|]/).map(cleanJobHeaderLine).filter(Boolean);
    if (parts.length < 2) continue;
    const titlePart = parts.find(isLikelyJobTitle) || '';
    const companyPart = parts.find((part) => part !== titlePart && isLikelyCompanyLine(part)) || '';
    if (titlePart || companyPart) return {title: titlePart, company: companyPart};
  }
  return {title: '', company: ''};
}

function bestJobTitleLine(lines: string[]) {
  return lines
    .map((line, index) => ({line, score: jobTitleScore(line, index)}))
    .filter((item) => item.score > 0)
    .sort((a, b) => b.score - a.score)[0]?.line || '';
}

function jobTitleScore(line: string, index: number) {
  if (!isLikelyJobTitle(line)) return 0;
  const words = line.split(/\s+/).length;
  let score = Math.max(0, 30 - index * 2);
  if (words >= 2 && words <= 8) score += 20;
  if (/\b(senior|junior|lead|principal|staff|graduate|intern)\b/i.test(line)) score += 8;
  if (line.includes(':') || line.length > 90) score -= 15;
  return score;
}

function bestCompanyLine(lines: string[], title: string) {
  return lines.slice(0, 12).find((line) => line !== title && isLikelyCompanyLine(line)) || '';
}

function isLikelyCompanyLine(line: string) {
  if (!line || isJobHeaderNoise(line) || isLocationOrMetaLine(line) || isLikelyJobTitle(line)) return false;
  const words = line.split(/\s+/).length;
  return words <= 6 && !/[.!?]$/.test(line) && !/^\d/.test(line);
}

function cleanInferredValue(value: string, maxLength: number) {
  return cleanJobHeaderLine(value).replace(/\s+[·|].*$/, '').slice(0, maxLength);
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
function normalizeJobs(v: JobDescription[] | null | undefined): JobDescription[] { return v ?? []; }

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

function normalizeResumeVersions(v: ResumeVersion[] | null | undefined): ResumeVersion[] {
  return (v ?? []).map(normalizeResumeVersion);
}

function formatDate(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, {month: 'short', day: 'numeric'}).replace(/,/g, '');
}

function formatDateTime(iso: string): string {
  if (!iso) return 'saved resume';
  const d = new Date(iso);
  return d.toLocaleString(undefined, {month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'}).replace(/,/g, '');
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
