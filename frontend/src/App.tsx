import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
  Activity,
  AlertCircle,
  BriefcaseBusiness,
  CheckCircle2,
  Clipboard,
  Cpu,
  Database,
  FileText,
  Folder,
  KeyRound,
  Layers3,
  ListChecks,
  Play,
  RefreshCcw,
  Save,
  Settings as SettingsIcon,
  Sparkles,
  Trash2,
  Upload,
  UserRound,
  Wrench,
} from 'lucide-react';
import {
  AnalyzeJobDescription,
  ApplicationStrategy,
  CandidateProfile,
  CandidateProfileRecord,
  CandidateSource,
  BuildJobMatchMap,
  CreateCandidateSource,
  CreateJobDescription,
  DeleteCandidateSource,
  DeleteAllEvidenceFacts,
  DeleteEvidenceFact,
  DeleteJobDescription,
  DeleteSourceSection,
  DeleteTailoredBulletDraft,
  DetectSourceSections,
  DraftCandidateProfileFromSource,
  EvidenceFact,
  ExtractEvidenceFacts,
  GenerateTailoredBulletDrafts,
  GenerateApplicationStrategy,
  GenerateFitAnalysis,
  GetCandidateProfile,
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
  ListEvidenceFacts,
  ListJobDescriptions,
  ListJobFactMatches,
  ListJobRequirements,
  ListPromptResearchSources,
  ListPromptRules,
  ListSourceSections,
  ListTailoredBulletDrafts,
  RenderSamplePDF,
  SaveAPIKey,
  SaveCandidateProfile,
  SaveSettings,
  SourceSection,
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
  UpdatePromptRule,
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

type LoadState = 'loading' | 'ready' | 'error';
type Tab = 'sources' | 'jobs' | 'profile' | 'sections' | 'facts' | 'settings';
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

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('jobs');
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [error, setError] = useState('');
  const [health, setHealth] = useState<Health | null>(null);
  const [settings, setSettings] = useState<Settings>({
    provider: 'openrouter',
    model: '',
    api_key_configured: false,
  });
  const [toolStatus, setToolStatus] = useState<ToolStatus | null>(null);
  const [events, setEvents] = useState<AppEvent[]>([]);
  const [promptRules, setPromptRules] = useState<PromptRule[]>([]);
  const [promptSources, setPromptSources] = useState<PromptResearchSource[]>([]);
  const [profile, setProfile] = useState<CandidateProfile>(emptyProfile);
  const [sources, setSources] = useState<CandidateSource[]>([]);
  const [sections, setSections] = useState<SourceSection[]>([]);
  const [facts, setFacts] = useState<EvidenceFact[]>([]);
  const [jobs, setJobs] = useState<JobDescription[]>([]);
  const [jobRequirements, setJobRequirements] = useState<JobRequirement[]>([]);
  const [jobMatches, setJobMatches] = useState<JobFactMatch[]>([]);
  const [bulletDrafts, setBulletDrafts] = useState<TailoredBulletDraft[]>([]);
  const [jobAnalysis, setJobAnalysis] = useState<JobAnalysis | null>(null);
  const [fitAnalysis, setFitAnalysis] = useState<JobFitAnalysis | null>(null);
  const [applicationStrategy, setApplicationStrategy] = useState<ApplicationStrategy | null>(null);
  const [selectedSourceID, setSelectedSourceID] = useState<number>(0);
  const [selectedSectionID, setSelectedSectionID] = useState<number>(0);
  const [selectedJobID, setSelectedJobID] = useState<number>(0);
  const [sourceDraft, setSourceDraft] = useState({
    source_type: 'current_resume',
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

  const selectedSource = sources.find((source) => source.id === selectedSourceID);
  const selectedSection = sections.find((section) => section.id === selectedSectionID);
  const selectedJob = jobs.find((job) => job.id === selectedJobID);
  const queuedFacts = facts.filter((fact) => fact.status === 'needs_review');
  const apiConfigured = toolStatus?.api_key_configured ?? settings.api_key_configured;
  const tectonicStatus = toolStatus?.tectonic_status ?? health?.pdf_renderer ?? 'checking';

  const statusText = useMemo(() => {
    if (loadState === 'loading') {
      return 'Loading';
    }
    if (loadState === 'error') {
      return 'Needs attention';
    }
    return 'Ready';
  }, [loadState]);

  async function load() {
    setLoadState('loading');
    setError('');
    try {
      const [
        nextHealth,
        nextSettings,
        nextStatus,
        nextEvents,
        nextPromptRules,
        nextPromptSources,
        nextProfile,
        nextSources,
        nextSections,
        nextFacts,
        nextJobs,
      ] = await Promise.all([
        GetHealth(),
        GetSettings(),
        GetToolStatus(),
        GetRecentEvents(),
        ListPromptRules(),
        ListPromptResearchSources(),
        GetCandidateProfile(),
        ListCandidateSources(),
        ListSourceSections(0),
        ListEvidenceFacts('all'),
        ListJobDescriptions(),
      ]);
      setHealth(nextHealth as Health);
      setSettings(nextSettings as Settings);
      setToolStatus(nextStatus as ToolStatus);
      setEvents((nextEvents ?? []) as AppEvent[]);
      setPromptRules((nextPromptRules ?? []) as PromptRule[]);
      setPromptSources((nextPromptSources ?? []) as PromptResearchSource[]);
      setProfile(normalizeProfile(nextProfile as CandidateProfile));
      setSources((nextSources ?? []) as CandidateSource[]);
      setSections((nextSections ?? []) as SourceSection[]);
      setFacts(normalizeFacts(nextFacts as EvidenceFact[] | null | undefined));
      setJobs((nextJobs ?? []) as JobDescription[]);
      const firstSource = (nextSources as CandidateSource[] | undefined)?.[0];
      if (!selectedSourceID && firstSource) {
        setSelectedSourceID(firstSource.id);
      }
      const firstJob = (nextJobs as JobDescription[] | undefined)?.[0];
      if (!selectedJobID && firstJob) {
        setSelectedJobID(firstJob.id);
        setJobDraft({
          company: firstJob.company,
          title: firstJob.title,
          url: firstJob.url,
          raw_text: firstJob.raw_text,
        });
        await refreshJobContext(firstJob.id);
      }
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function refreshEvents() {
    const nextEvents = await GetRecentEvents();
    setEvents((nextEvents ?? []) as AppEvent[]);
  }

  async function refreshWorkflow() {
    const [nextSources, nextSections, nextFacts, nextJobs] = await Promise.all([
      ListCandidateSources(),
      ListSourceSections(0),
      ListEvidenceFacts('all'),
      ListJobDescriptions(),
    ]);
    setSources((nextSources ?? []) as CandidateSource[]);
    setSections((nextSections ?? []) as SourceSection[]);
    setFacts(normalizeFacts(nextFacts as EvidenceFact[] | null | undefined));
    setJobs((nextJobs ?? []) as JobDescription[]);
    await refreshEvents();
  }

  async function refreshJobContext(jobID = selectedJobID) {
    if (!jobID) {
      setJobRequirements([]);
      setJobMatches([]);
      setBulletDrafts([]);
      setJobAnalysis(null);
      setFitAnalysis(null);
      setApplicationStrategy(null);
      return;
    }
    const [requirements, matches, drafts, analysis, fit, strategy] = await Promise.all([
      ListJobRequirements(jobID),
      ListJobFactMatches(jobID),
      ListTailoredBulletDrafts(jobID),
      GetJobAnalysis(jobID).catch(() => null),
      GetFitAnalysis(jobID).catch(() => null),
      GetApplicationStrategy(jobID).catch(() => null),
    ]);
    setJobRequirements(normalizeRequirements(requirements as JobRequirement[] | null | undefined));
    setJobMatches(normalizeMatches(matches as JobFactMatch[] | null | undefined));
    setBulletDrafts(normalizeDrafts(drafts as TailoredBulletDraft[] | null | undefined));
    setJobAnalysis(normalizeJobAnalysis(analysis as JobAnalysis | null | undefined));
    setFitAnalysis(normalizeFitAnalysis(fit as JobFitAnalysis | null | undefined));
    setApplicationStrategy(normalizeApplicationStrategy(strategy as ApplicationStrategy | null | undefined));
  }

  async function runAction(name: string, action: () => Promise<void>) {
    setBusyAction(name);
    setError('');
    try {
      await action();
      setLoadState('ready');
    } catch (err) {
      setLoadState('error');
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyAction('');
    }
  }

  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-profile', async () => {
      const saved = await SaveCandidateProfile(profile);
      setProfile(normalizeProfile(saved as CandidateProfile));
      await refreshEvents();
    });
  }

  async function draftProfileFromSource() {
    if (!selectedSourceID) {
      return;
    }
    await runAction('draft-profile', async () => {
      const draft = normalizeProfile((await DraftCandidateProfileFromSource(selectedSourceID)) as CandidateProfile);
      setProfile((current) => ({
        contact: {
          ...current.contact,
          full_name: draft.contact.full_name || current.contact.full_name,
          email: draft.contact.email || current.contact.email,
          phone: draft.contact.phone || current.contact.phone,
          location: draft.contact.location || current.contact.location,
          linkedin: draft.contact.linkedin || current.contact.linkedin,
          github: draft.contact.github || current.contact.github,
          portfolio: draft.contact.portfolio || current.contact.portfolio,
          links: draft.contact.links.length ? draft.contact.links : current.contact.links,
          verified: false,
        },
        records: [...current.records, ...draft.records],
      }));
      await refreshEvents();
    });
  }

  async function applyDraftFromSource(sourceID: number) {
    const draft = normalizeProfile((await DraftCandidateProfileFromSource(sourceID)) as CandidateProfile);
    setProfile((current) => ({
      contact: {
        ...current.contact,
        full_name: draft.contact.full_name || current.contact.full_name,
        email: draft.contact.email || current.contact.email,
        phone: draft.contact.phone || current.contact.phone,
        location: draft.contact.location || current.contact.location,
        linkedin: draft.contact.linkedin || current.contact.linkedin,
        github: draft.contact.github || current.contact.github,
        portfolio: draft.contact.portfolio || current.contact.portfolio,
          links: draft.contact.links.length ? draft.contact.links : current.contact.links,
        verified: false,
      },
      records: mergeDraftRecords(current.records, draft.records),
    }));
  }

  async function createSource(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('create-source', async () => {
      const source = (await CreateCandidateSource(sourceDraft)) as CandidateSource;
      setSelectedSourceID(source.id);
      setSourceDraft({...sourceDraft, title: '', raw_text: ''});
      const detected = (await DetectSourceSections(source.id)) as SourceSection[];
      setSelectedSectionID(detected[0]?.id ?? 0);
      await applyDraftFromSource(source.id);
      await refreshWorkflow();
      setActiveTab('profile');
    });
  }

  async function deleteSource(sourceID: number) {
    await runAction(`delete-source-${sourceID}`, async () => {
      await DeleteCandidateSource({id: sourceID});
      if (selectedSourceID === sourceID) {
        setSelectedSourceID(0);
        setSelectedSectionID(0);
      }
      await refreshWorkflow();
    });
  }

  async function importFile(file?: File) {
    if (!file) {
      return;
    }
    await runAction('import-file', async () => {
      const rawText = await file.text();
      const source = (await CreateCandidateSource({
        source_type: sourceDraft.source_type,
        title: sourceDraft.title || file.name.replace(/\.(txt|md|markdown|tex|latex)$/i, ''),
        raw_text: rawText,
      })) as CandidateSource;
      setSelectedSourceID(source.id);
      const detected = (await DetectSourceSections(source.id)) as SourceSection[];
      setSelectedSectionID(detected[0]?.id ?? 0);
      await applyDraftFromSource(source.id);
      await refreshWorkflow();
      setActiveTab('profile');
    });
  }

  async function detectSections() {
    if (!selectedSourceID) {
      return;
    }
    await runAction('detect-sections', async () => {
      const detected = (await DetectSourceSections(selectedSourceID)) as SourceSection[];
      setSections((previous) => [
        ...detected,
        ...previous.filter((section) => section.source_id !== selectedSourceID),
      ]);
      if (detected[0]) {
        setSelectedSectionID(detected[0].id);
      }
      await refreshEvents();
    });
  }

  async function saveSection(section: SourceSection) {
    await runAction(`save-section-${section.id}`, async () => {
      const saved = (await UpdateSourceSection({
        id: section.id,
        heading: section.heading,
        section_type: section.section_type,
        content: section.content,
      })) as SourceSection;
      setSections((previous) => previous.map((item) => item.id === saved.id ? saved : item));
      await refreshEvents();
    });
  }

  async function deleteSection(sectionID: number) {
    await runAction(`delete-section-${sectionID}`, async () => {
      await DeleteSourceSection({id: sectionID});
      if (selectedSectionID === sectionID) {
        setSelectedSectionID(0);
      }
      await refreshWorkflow();
    });
  }

  async function extractFacts() {
    if (!selectedSection) {
      return;
    }
    await runAction('extract-facts', async () => {
      await ExtractEvidenceFacts({
        source_id: selectedSection.source_id,
        section_id: selectedSection.id,
      });
      const nextFacts = (await ListEvidenceFacts('all')) as EvidenceFact[];
      setFacts(normalizeFacts(nextFacts));
      await refreshEvents();
      setActiveTab('facts');
    });
  }

  async function reviewFact(fact: EvidenceFact, status: string) {
    await runAction(`review-fact-${fact.id}`, async () => {
      const saved = (await UpdateEvidenceFactReview({
        id: fact.id,
        fact_text: fact.fact_text,
        evidence_quote: fact.evidence_quote,
        technologies: fact.technologies,
        confidence: fact.confidence,
        risk_flags: fact.risk_flags,
        status,
        review_note: fact.review_note,
      })) as EvidenceFact;
      setFacts((previous) => normalizeFacts(previous.map((item) => item.id === saved.id ? saved : item)));
      await refreshEvents();
    });
  }

  async function deleteFact(factID: number) {
    await runAction(`delete-fact-${factID}`, async () => {
      await DeleteEvidenceFact({id: factID});
      const nextFacts = (await ListEvidenceFacts('all')) as EvidenceFact[];
      setFacts(normalizeFacts(nextFacts));
      await refreshEvents();
    });
  }

  async function deleteAllFacts() {
    if (!window.confirm('Delete all evidence facts, match maps, and bullet drafts?')) {
      return;
    }
    await runAction('delete-all-facts', async () => {
      await DeleteAllEvidenceFacts();
      setFacts([]);
      setJobMatches([]);
      setBulletDrafts([]);
      setFitAnalysis(null);
      setApplicationStrategy(null);
      await refreshEvents();
    });
  }

  async function saveJob(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-job', async () => {
      const saved = selectedJobID
        ? (await UpdateJobDescription({id: selectedJobID, ...jobDraft})) as JobDescription
        : (await CreateJobDescription(jobDraft)) as JobDescription;
      setSelectedJobID(saved.id);
      setJobDraft({
        company: saved.company,
        title: saved.title,
        url: saved.url,
        raw_text: saved.raw_text,
      });
      const nextJobs = (await ListJobDescriptions()) as JobDescription[];
      setJobs(nextJobs);
      await refreshJobContext(saved.id);
      await refreshEvents();
    });
  }

  async function selectJob(job: JobDescription) {
    setSelectedJobID(job.id);
    setJobDraft({
      company: job.company,
      title: job.title,
      url: job.url,
      raw_text: job.raw_text,
    });
    await refreshJobContext(job.id);
  }

  async function newJob() {
    setSelectedJobID(0);
    setJobDraft({company: '', title: '', url: '', raw_text: ''});
    setJobRequirements([]);
    setJobMatches([]);
    setBulletDrafts([]);
    setJobAnalysis(null);
    setFitAnalysis(null);
    setApplicationStrategy(null);
  }

  function updateJobDraft(nextDraft: JobDraft) {
    setJobDraft((previous) => {
      if (nextDraft.raw_text === previous.raw_text) {
        return nextDraft;
      }
      const inferred = inferJobDetailsFromText(nextDraft.raw_text);
      return {
        ...nextDraft,
        company: nextDraft.company.trim() ? nextDraft.company : inferred.company,
        title: nextDraft.title.trim() ? nextDraft.title : inferred.title,
      };
    });
  }

  async function deleteJob(jobID: number) {
    await runAction(`delete-job-${jobID}`, async () => {
      await DeleteJobDescription({id: jobID});
      if (selectedJobID === jobID) {
        await newJob();
      }
      const nextJobs = (await ListJobDescriptions()) as JobDescription[];
      setJobs(nextJobs);
      await refreshEvents();
    });
  }

  async function parseJob() {
    if (!selectedJobID) return;
    await runAction('parse-job', async () => {
      const requirements = (await ParseJobDescription(selectedJobID)) as JobRequirement[];
      setJobRequirements(normalizeRequirements(requirements));
      const analysis = (await AnalyzeJobDescription(selectedJobID)) as JobAnalysis;
      setJobAnalysis(normalizeJobAnalysis(analysis));
      setJobMatches([]);
      setBulletDrafts([]);
      setFitAnalysis(null);
      setApplicationStrategy(null);
      await refreshEvents();
    });
  }

  async function buildMatchMap() {
    if (!selectedJobID) return;
    await runAction('build-match-map', async () => {
      const matches = (await BuildJobMatchMap(selectedJobID)) as JobFactMatch[];
      setJobMatches(normalizeMatches(matches));
      setFitAnalysis(null);
      setApplicationStrategy(null);
      await refreshEvents();
    });
  }

  async function generateFit() {
    if (!selectedJobID) return;
    await runAction('generate-fit', async () => {
      const fit = (await GenerateFitAnalysis(selectedJobID)) as JobFitAnalysis;
      setFitAnalysis(normalizeFitAnalysis(fit));
      setApplicationStrategy(null);
      await refreshEvents();
    });
  }

  async function generateStrategy() {
    if (!selectedJobID) return;
    await runAction('generate-strategy', async () => {
      const strategy = (await GenerateApplicationStrategy(selectedJobID)) as ApplicationStrategy;
      setApplicationStrategy(normalizeApplicationStrategy(strategy));
      await refreshEvents();
    });
  }

  async function generateBulletDrafts() {
    if (!selectedJobID) return;
    await runAction('generate-bullets', async () => {
      const drafts = (await GenerateTailoredBulletDrafts(selectedJobID)) as TailoredBulletDraft[];
      setBulletDrafts(normalizeDrafts(drafts));
      await refreshEvents();
    });
  }

  async function updateBulletDraft(draft: TailoredBulletDraft, status = draft.status) {
    await runAction(`update-draft-${draft.id}`, async () => {
      const saved = (await UpdateTailoredBulletDraft({
        id: draft.id,
        draft_text: draft.draft_text,
        rationale: draft.rationale,
        status,
        risk_flags: draft.risk_flags,
      })) as TailoredBulletDraft;
      setBulletDrafts((previous) => normalizeDrafts(previous.map((item) => item.id === saved.id ? saved : item)));
      await refreshEvents();
    });
  }

  async function deleteBulletDraft(draftID: number) {
    await runAction(`delete-draft-${draftID}`, async () => {
      await DeleteTailoredBulletDraft({id: draftID});
      setBulletDrafts((previous) => previous.filter((draft) => draft.id !== draftID));
      await refreshEvents();
    });
  }

  async function saveSettings(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-settings', async () => {
      const nextSettings = await SaveSettings({
        provider: settings.provider,
        model: settings.model,
      });
      setSettings(nextSettings as Settings);
      await load();
    });
  }

  async function saveAPIKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await runAction('save-key', async () => {
      const nextStatus = await SaveAPIKey({api_key: apiKey, provider: settings.provider});
      setToolStatus(nextStatus as ToolStatus);
      setSettings({...settings, api_key_configured: (nextStatus as ToolStatus).api_key_configured});
      setAPIKey('');
      await refreshEvents();
    });
  }

  async function runLLMTest() {
    await runAction('test-llm', async () => {
      const result = (await TestLLM()) as LLMTestResult;
      setLLMResult(result);
      await refreshEvents();
    });
  }

  async function installTectonic() {
    await runAction('install-tectonic', async () => {
      await InstallTectonic();
      await load();
    });
  }

  async function renderSamplePDF() {
    await runAction('render-pdf', async () => {
      const result = (await RenderSamplePDF()) as RenderPDFResult;
      setPDFResult(result);
      await load();
    });
  }

  async function savePromptRule(rule: PromptRule) {
    await runAction(`save-rule-${rule.id}`, async () => {
      const saved = (await UpdatePromptRule({
        id: rule.id,
        content: rule.content,
        enabled: rule.enabled,
      })) as PromptRule;
      setPromptRules((previous) => previous.map((item) => item.id === saved.id ? saved : item));
      await refreshEvents();
    });
  }

  useEffect(() => {
    load();
  }, []);

  return (
    <main className="min-h-screen bg-[#f6f8fb]">
      <section className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-5 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-500">
              JD Tailor
            </p>
            <h1 className="mt-1 text-2xl font-semibold text-slate-950">
              Candidate context builder
            </h1>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <StatusPill state={loadState} text={statusText} />
            <IconButton label="Refresh" onClick={load} disabled={busyAction !== ''}>
              <RefreshCcw size={16} />
            </IconButton>
          </div>
        </div>
      </section>

      <section className="border-b border-slate-200 bg-white">
        <nav className="mx-auto flex max-w-7xl gap-2 overflow-x-auto px-5 py-3">
          <TabButton active={activeTab === 'jobs'} label="Jobs" icon={<BriefcaseBusiness size={16} />} onClick={() => setActiveTab('jobs')} />
          <TabButton active={activeTab === 'sources'} label="Sources" icon={<Upload size={16} />} onClick={() => setActiveTab('sources')} />
          <TabButton active={activeTab === 'profile'} label="Profile" icon={<UserRound size={16} />} onClick={() => setActiveTab('profile')} />
          <TabButton active={activeTab === 'sections'} label="Sections" icon={<Layers3 size={16} />} onClick={() => setActiveTab('sections')} />
          <TabButton active={activeTab === 'facts'} label={`Fact Review${queuedFacts.length ? ` ${queuedFacts.length}` : ''}`} icon={<ListChecks size={16} />} onClick={() => setActiveTab('facts')} />
          <TabButton active={activeTab === 'settings'} label="Settings" icon={<SettingsIcon size={16} />} onClick={() => setActiveTab('settings')} />
        </nav>
      </section>

      <section className="mx-auto max-w-7xl px-5 py-5">
        {error && (
          <div className="mb-4 flex items-start gap-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">
            <AlertCircle className="mt-0.5 shrink-0" size={18} />
            <span>{error}</span>
          </div>
        )}

        {activeTab === 'profile' && (
          <ProfileView
            busy={busyAction === 'save-profile'}
            onAddRecord={(type) => setProfile({...profile, records: [...profile.records, newRecord(type)]})}
            onChange={setProfile}
            onDraftFromSource={draftProfileFromSource}
            onSave={saveProfile}
            profile={profile}
            selectedSourceID={selectedSourceID}
            sources={sources}
          />
        )}

        {activeTab === 'sources' && (
          <SourcesView
            busy={busyAction === 'create-source' || busyAction === 'import-file'}
            draft={sourceDraft}
            onDraftChange={setSourceDraft}
            onDelete={deleteSource}
            onFile={importFile}
            onSave={createSource}
            onSelect={(id) => {
              setSelectedSourceID(id);
              setActiveTab('sections');
            }}
            sources={sources}
          />
        )}

        {activeTab === 'jobs' && (
          <JobsView
            busyAction={busyAction}
            applicationStrategy={applicationStrategy}
            draft={jobDraft}
            facts={facts}
            fitAnalysis={fitAnalysis}
            jobAnalysis={jobAnalysis}
            jobs={jobs}
            matches={jobMatches}
            onBuildMatchMap={buildMatchMap}
            onChangeDraft={(draft) => setBulletDrafts((previous) => normalizeDrafts(previous.map((item) => item.id === draft.id ? draft : item)))}
            onDeleteDraft={deleteBulletDraft}
            onDeleteJob={deleteJob}
            onDraftChange={updateJobDraft}
            onGenerateDrafts={generateBulletDrafts}
            onGenerateFit={generateFit}
            onGenerateStrategy={generateStrategy}
            onNewJob={newJob}
            onParseJob={parseJob}
            onSaveDraft={(draft) => updateBulletDraft(draft)}
            onSaveJob={saveJob}
            onSelectJob={selectJob}
            onSetDraftStatus={updateBulletDraft}
            requirements={jobRequirements}
            selectedJobID={selectedJobID}
            tailoredDrafts={bulletDrafts}
          />
        )}

        {activeTab === 'sections' && (
          <SectionsView
            busyAction={busyAction}
            onDetect={detectSections}
            onDelete={deleteSection}
            onExtract={extractFacts}
            onSave={saveSection}
            onSectionChange={(section) => setSections((previous) => previous.map((item) => item.id === section.id ? section : item))}
            onSelectSection={setSelectedSectionID}
            onSelectSource={setSelectedSourceID}
            sections={sections.filter((section) => !selectedSourceID || section.source_id === selectedSourceID)}
            selectedSectionID={selectedSectionID}
            selectedSource={selectedSource}
            selectedSourceID={selectedSourceID}
            sources={sources}
          />
        )}

        {activeTab === 'facts' && (
          <FactsView
            busyAction={busyAction}
            facts={facts}
            onChange={(fact) => setFacts((previous) => normalizeFacts(previous.map((item) => item.id === fact.id ? fact : item)))}
            onDelete={deleteFact}
            onDeleteAll={deleteAllFacts}
            onReview={reviewFact}
          />
        )}

        {activeTab === 'settings' && (
          <SettingsView
            apiConfigured={apiConfigured}
            apiKey={apiKey}
            busyAction={busyAction}
            events={events}
            health={health}
            llmResult={llmResult}
            onAPIKeyChange={setAPIKey}
            onInstallTectonic={installTectonic}
            onRenderSamplePDF={renderSamplePDF}
            onRunLLMTest={runLLMTest}
            onPromptRuleChange={(rule) => setPromptRules((previous) => previous.map((item) => item.id === rule.id ? rule : item))}
            onSaveAPIKey={saveAPIKey}
            onSavePromptRule={savePromptRule}
            onSaveSettings={saveSettings}
            pdfResult={pdfResult}
            promptRules={promptRules}
            promptSources={promptSources}
            settings={settings}
            setSettings={setSettings}
            tectonicStatus={tectonicStatus}
            toolStatus={toolStatus}
          />
        )}
      </section>
    </main>
  );
}

function ProfileView({
  busy,
  onAddRecord,
  onChange,
  onDraftFromSource,
  onSave,
  profile,
  selectedSourceID,
  sources,
}: {
  busy: boolean;
  onAddRecord: (type: string) => void;
  onChange: (profile: CandidateProfile) => void;
  onDraftFromSource: () => void;
  onSave: (event: FormEvent<HTMLFormElement>) => void;
  profile: CandidateProfile;
  selectedSourceID: number;
  sources: CandidateSource[];
}) {
  const updateContact = (field: keyof CandidateProfile['contact'], value: string) => {
    onChange({...profile, contact: {...profile.contact, [field]: value}});
  };
  const updateRecord = (index: number, record: CandidateProfileRecord) => {
    onChange({...profile, records: profile.records.map((item, itemIndex) => itemIndex === index ? record : item)});
  };
  const removeRecord = (index: number) => {
    onChange({...profile, records: profile.records.filter((_, itemIndex) => itemIndex !== index)});
  };

  return (
    <form className="grid gap-4 lg:grid-cols-[0.85fr_1.15fr]" onSubmit={onSave}>
      <Panel icon={<UserRound size={18} />} title="Locked identity" subtitle="Fields the model must preserve exactly.">
        <div className="space-y-4">
          <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="text-sm font-semibold text-slate-950">Draft from source</p>
                <p className="mt-1 text-sm text-slate-500">{sourceName(sources, selectedSourceID)}</p>
              </div>
              <IconButton label="Draft profile" onClick={onDraftFromSource} disabled={!selectedSourceID || busy}>
                <Sparkles size={16} />
              </IconButton>
            </div>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            <TextInput label="Full name" value={profile.contact.full_name} onChange={(value) => updateContact('full_name', value)} />
            <TextInput label="Email" value={profile.contact.email} onChange={(value) => updateContact('email', value)} />
            <TextInput label="Phone" value={profile.contact.phone} onChange={(value) => updateContact('phone', value)} />
            <TextInput label="Location" value={profile.contact.location} onChange={(value) => updateContact('location', value)} />
            <TextInput label="LinkedIn" value={profile.contact.linkedin} onChange={(value) => updateContact('linkedin', value)} />
            <TextInput label="GitHub" value={profile.contact.github} onChange={(value) => updateContact('github', value)} />
            <div className="md:col-span-2">
              <TextInput label="Portfolio" value={profile.contact.portfolio} onChange={(value) => updateContact('portfolio', value)} />
            </div>
          </div>
          <CheckInput
            checked={profile.contact.verified}
            label="Verified locked identity"
            onChange={(checked) => onChange({...profile, contact: {...profile.contact, verified: checked}})}
          />
        </div>
      </Panel>

      <Panel icon={<ListChecks size={18} />} title="Structured records" subtitle="Education, employment, projects, aliases, and blocked aliases.">
        <div className="space-y-4">
          <div className="flex flex-wrap gap-2">
            {['education', 'employment', 'project', 'allowed_alias', 'blocked_alias'].map((type) => (
              <SecondaryButton key={type} label={recordTypeLabel(type)} onClick={() => onAddRecord(type)} />
            ))}
          </div>
          <div className="space-y-3">
            {profile.records.length === 0 ? (
              <EmptyState text="No locked records yet." />
            ) : (
              profile.records.map((record, index) => (
                <RecordEditor
                  key={`${record.id}-${index}`}
                  onChange={(next) => updateRecord(index, next)}
                  onRemove={() => removeRecord(index)}
                  record={record}
                />
              ))
            )}
          </div>
          <IconButton label="Save profile" submit full disabled={busy}>
            <Save size={16} />
          </IconButton>
        </div>
      </Panel>
    </form>
  );
}

function SourcesView({
  busy,
  draft,
  onDraftChange,
  onDelete,
  onFile,
  onSave,
  onSelect,
  sources,
}: {
  busy: boolean;
  draft: {source_type: string; title: string; raw_text: string};
  onDraftChange: (draft: {source_type: string; title: string; raw_text: string}) => void;
  onDelete: (id: number) => void;
  onFile: (file?: File) => void;
  onSave: (event: FormEvent<HTMLFormElement>) => void;
  onSelect: (id: number) => void;
  sources: CandidateSource[];
}) {
  return (
    <div className="grid gap-4 lg:grid-cols-[0.85fr_1.15fr]">
      <Panel icon={<Upload size={18} />} title="Import source" subtitle="Paste raw context or import TXT/MD material.">
        <form className="space-y-4" onSubmit={onSave}>
          <div className="grid gap-3 md:grid-cols-2">
            <SelectInput
              label="Source type"
              value={draft.source_type}
              onChange={(value) => onDraftChange({...draft, source_type: value})}
              options={[
                ['current_resume', 'Current resume'],
                ['extended_resume', 'Extended resume'],
                ['old_resume', 'Old resume'],
                ['project_notes', 'Project notes'],
                ['readme', 'README'],
                ['architecture_notes', 'Architecture notes'],
                ['interview_notes', 'Interview notes'],
                ['manual_notes', 'Manual notes'],
              ]}
            />
            <TextInput label="Title" value={draft.title} onChange={(value) => onDraftChange({...draft, title: value})} />
          </div>
          <TextArea label="Raw source text" rows={14} value={draft.raw_text} onChange={(value) => onDraftChange({...draft, raw_text: value})} />
          <div className="grid gap-3 md:grid-cols-2">
            <IconButton label="Save source" submit disabled={busy || draft.raw_text.trim() === ''}>
              <Save size={16} />
            </IconButton>
            <label className="inline-flex h-10 cursor-pointer items-center justify-center gap-2 rounded-md border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800 shadow-sm hover:bg-slate-50">
              <Upload size={16} />
              Import TXT/MD/TeX
              <input
                className="hidden"
                type="file"
                accept=".txt,.md,.markdown,.tex,.latex,text/plain,text/markdown,application/x-tex"
                onChange={(event) => onFile(event.currentTarget.files?.[0])}
              />
            </label>
          </div>
        </form>
      </Panel>

      <Panel icon={<FileText size={18} />} title="Source library" subtitle="Stored raw text can be sectioned and reprocessed.">
        <div className="space-y-3">
          {sources.length === 0 ? (
            <EmptyState text="No sources imported yet." />
          ) : (
            sources.map((source) => (
              <div
                key={source.id}
                className="rounded-md border border-slate-200 bg-slate-50 p-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <button type="button" onClick={() => onSelect(source.id)} className="min-w-0 flex-1 text-left">
                    <span className="text-sm font-semibold text-slate-950">{source.title}</span>
                    <span className="mt-2 line-clamp-3 block text-sm text-slate-600">{source.raw_text}</span>
                    <time className="mt-2 block text-xs text-slate-500">{formatDate(source.imported_at)}</time>
                  </button>
                  <div className="flex shrink-0 items-center gap-2">
                    <span className="rounded-md bg-white px-2 py-1 text-xs font-medium text-slate-600">{sourceTypeLabel(source.source_type)}</span>
                    <IconOnlyButton label="Delete source" onClick={() => onDelete(source.id)}>
                      <Trash2 size={16} />
                    </IconOnlyButton>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </Panel>
    </div>
  );
}

function JobsView({
  busyAction,
  applicationStrategy,
  draft,
  facts,
  fitAnalysis,
  jobAnalysis,
  jobs,
  matches,
  onBuildMatchMap,
  onChangeDraft,
  onDeleteDraft,
  onDeleteJob,
  onDraftChange,
  onGenerateDrafts,
  onGenerateFit,
  onGenerateStrategy,
  onNewJob,
  onParseJob,
  onSaveDraft,
  onSaveJob,
  onSelectJob,
  onSetDraftStatus,
  requirements,
  selectedJobID,
  tailoredDrafts,
}: {
  busyAction: string;
  applicationStrategy: ApplicationStrategy | null;
  draft: JobDraft;
  facts: EvidenceFact[];
  fitAnalysis: JobFitAnalysis | null;
  jobAnalysis: JobAnalysis | null;
  jobs: JobDescription[];
  matches: JobFactMatch[];
  onBuildMatchMap: () => void;
  onChangeDraft: (draft: TailoredBulletDraft) => void;
  onDeleteDraft: (id: number) => void;
  onDeleteJob: (id: number) => void;
  onDraftChange: (draft: JobDraft) => void;
  onGenerateDrafts: () => void;
  onGenerateFit: () => void;
  onGenerateStrategy: () => void;
  onNewJob: () => void;
  onParseJob: () => void;
  onSaveDraft: (draft: TailoredBulletDraft) => void;
  onSaveJob: (event: FormEvent<HTMLFormElement>) => void;
  onSelectJob: (job: JobDescription) => void;
  onSetDraftStatus: (draft: TailoredBulletDraft, status: string) => void;
  requirements: JobRequirement[];
  selectedJobID: number;
  tailoredDrafts: TailoredBulletDraft[];
}) {
  const matchesByRequirement = new Map<number, JobFactMatch[]>();
  matches.forEach((match) => {
    matchesByRequirement.set(match.requirement_id, [...(matchesByRequirement.get(match.requirement_id) ?? []), match]);
  });
  const requirementLabel = (id: number) => requirements.find((req) => req.id === id)?.requirement_text ?? `Requirement ${id}`;

  return (
    <div className="grid gap-4 lg:grid-cols-[340px_1fr]">
      <Panel icon={<BriefcaseBusiness size={18} />} title="Job descriptions" subtitle="Paste JD text and keep one match map per application.">
        <div className="space-y-4">
          <SecondaryButton label="New job" onClick={onNewJob} />
          <div className="space-y-2">
            {jobs.length === 0 ? (
              <EmptyState text="No jobs saved yet." />
            ) : (
              jobs.map((job) => (
                <div
                  key={job.id}
                  className={`flex items-start gap-2 rounded-md border p-3 text-sm ${job.id === selectedJobID ? 'border-sky-300 bg-sky-50' : 'border-slate-200 bg-slate-50'}`}
                >
                  <button type="button" onClick={() => onSelectJob(job)} className="min-w-0 flex-1 text-left">
                    <span className="font-semibold text-slate-950">{job.title}</span>
                    <span className="mt-1 block text-slate-600">{job.company || 'Company not set'}</span>
                    <span className="mt-2 line-clamp-3 block text-slate-500">{job.raw_text}</span>
                  </button>
                  <IconOnlyButton label="Delete job" onClick={() => onDeleteJob(job.id)}>
                    <Trash2 size={16} />
                  </IconOnlyButton>
                </div>
              ))
            )}
          </div>
        </div>
      </Panel>

      <div className="space-y-4">
        <Panel icon={<FileText size={18} />} title="JD intake" subtitle="Paste the job description and parse it into requirements.">
          <form className="space-y-4" onSubmit={onSaveJob}>
            <div className="grid gap-3 md:grid-cols-3">
              <TextInput label="Company" value={draft.company} onChange={(value) => onDraftChange({...draft, company: value})} />
              <TextInput label="Title" value={draft.title} onChange={(value) => onDraftChange({...draft, title: value})} />
              <TextInput label="URL" value={draft.url} onChange={(value) => onDraftChange({...draft, url: value})} />
            </div>
            <TextArea label="Raw JD text" rows={10} value={draft.raw_text} onChange={(value) => onDraftChange({...draft, raw_text: value})} />
            <div className="grid gap-3 md:grid-cols-6">
              <IconButton label={selectedJobID ? 'Update JD' : 'Save JD'} submit disabled={busyAction === 'save-job' || draft.raw_text.trim() === ''}>
                <Save size={16} />
              </IconButton>
              <IconButton label="Parse JD" onClick={onParseJob} disabled={!selectedJobID || busyAction === 'parse-job'}>
                <Sparkles size={16} />
              </IconButton>
              <IconButton label="Build matches" onClick={onBuildMatchMap} disabled={!selectedJobID || requirements.length === 0 || facts.length === 0 || busyAction === 'build-match-map'}>
                <ListChecks size={16} />
              </IconButton>
              <IconButton label="Fit" onClick={onGenerateFit} disabled={!selectedJobID || requirements.length === 0 || busyAction === 'generate-fit'}>
                <Activity size={16} />
              </IconButton>
              <IconButton label="Strategy" onClick={onGenerateStrategy} disabled={!selectedJobID || !fitAnalysis || busyAction === 'generate-strategy'}>
                <Wrench size={16} />
              </IconButton>
              <IconButton label="Draft bullets" onClick={onGenerateDrafts} disabled={!selectedJobID || matches.length === 0 || busyAction === 'generate-bullets'}>
                <Sparkles size={16} />
              </IconButton>
            </div>
          </form>
        </Panel>

        <Panel icon={<Activity size={18} />} title="JD analysis and strategy" subtitle="Top pain points, evidence-backed fit, and resume positioning.">
          <div className="grid gap-3 lg:grid-cols-3">
            <SummaryBlock
              title="Top pain points"
              empty="Parse a saved JD."
              items={jobAnalysis?.top_pain_points ?? []}
            />
            <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
              <p className="text-sm font-semibold text-slate-950">Fit</p>
              {fitAnalysis ? (
                <div className="mt-2 space-y-2 text-sm text-slate-700">
                  <div className="flex flex-wrap gap-2">
                    <StatusBadge text={`${fitAnalysis.overall_score}%`} />
                    <StatusBadge text={fitAnalysis.recommendation} />
                  </div>
                  <p>{fitAnalysis.reality_check}</p>
                </div>
              ) : (
                <p className="mt-2 text-sm text-slate-500">Generate fit after matches.</p>
              )}
            </div>
            <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
              <p className="text-sm font-semibold text-slate-950">Strategy</p>
              {applicationStrategy ? (
                <div className="mt-2 space-y-2 text-sm text-slate-700">
                  <p className="font-medium text-slate-900">{applicationStrategy.resume_headline}</p>
                  <p>{applicationStrategy.positioning_strategy}</p>
                </div>
              ) : (
                <p className="mt-2 text-sm text-slate-500">Generate strategy after fit.</p>
              )}
            </div>
            <SummaryBlock
              title="Keywords"
              empty="No keywords yet."
              items={(applicationStrategy?.keywords ?? jobAnalysis?.keywords ?? []).slice(0, 10)}
            />
            <SummaryBlock
              title="Do not overclaim"
              empty="No blocked gaps yet."
              items={applicationStrategy?.do_not_overclaim ?? fitAnalysis?.critical_gaps ?? []}
            />
            <SummaryBlock
              title="Required skills"
              empty="No required skills yet."
              items={jobAnalysis?.required_skills ?? []}
            />
          </div>
        </Panel>

        <Panel icon={<ListChecks size={18} />} title="Requirements and matches" subtitle="All fact statuses are shown so risky evidence stays visible.">
          <div className="space-y-3">
            {requirements.length === 0 ? (
              <EmptyState text="Parse a saved JD to create requirements." />
            ) : (
              requirements.map((requirement) => (
                <div key={requirement.id} className="rounded-md border border-slate-200 bg-slate-50 p-3">
                  <div className="mb-2 flex flex-wrap items-center gap-2">
                    <StatusBadge text={requirement.priority} />
                    <StatusBadge text={requirement.category} />
                    {asStringArray(requirement.keywords).map((keyword) => <StatusBadge key={keyword} text={keyword} />)}
                  </div>
                  <p className="text-sm font-semibold text-slate-950">{requirement.requirement_text}</p>
                  <p className="mt-1 text-sm text-slate-600">{requirement.source_quote}</p>
                  <div className="mt-3 space-y-2">
                    {(matchesByRequirement.get(requirement.id) ?? []).length === 0 ? (
                      <p className="text-sm text-slate-500">No matches yet.</p>
                    ) : (
                      (matchesByRequirement.get(requirement.id) ?? []).map((match) => (
                        <div key={match.id} className="rounded-md border border-slate-200 bg-white p-3">
                          <div className="mb-2 flex flex-wrap items-center gap-2">
                            <StatusBadge text={match.coverage_status} />
                            <StatusBadge text={`${Math.round(match.score * 100)}%`} />
                            <StatusBadge text={match.fact_status} />
                            {asStringArray(match.risk_flags).map((flag) => <StatusBadge key={flag} text={flag} />)}
                          </div>
                          <p className="text-sm text-slate-900">{match.fact_text}</p>
                          <p className="mt-1 text-xs text-slate-500">{match.rationale}</p>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel icon={<Sparkles size={18} />} title="Saved bullet drafts" subtitle="Suggestions only; they do not overwrite source truth.">
          <div className="space-y-3">
            {tailoredDrafts.length === 0 ? (
              <EmptyState text="Generate drafts after building a match map." />
            ) : (
              tailoredDrafts.map((item) => (
                <div key={item.id} className="rounded-md border border-slate-200 bg-slate-50 p-3">
                  <div className="mb-3 flex flex-wrap items-center gap-2">
                    <StatusBadge text={item.status} />
                    <StatusBadge text={requirementLabel(item.requirement_id)} />
                    {asStringArray(item.risk_flags).map((flag) => <StatusBadge key={flag} text={flag} />)}
                  </div>
                  <TextArea
                    label="Draft bullet"
                    rows={3}
                    value={item.draft_text}
                    onChange={(value) => onChangeDraft({...item, draft_text: value})}
                  />
                  <div className="mt-3 grid gap-3 md:grid-cols-[1fr_220px]">
                    <TextInput label="Rationale" value={item.rationale} onChange={(value) => onChangeDraft({...item, rationale: value})} />
                    <TextInput label="Risk flags" value={item.risk_flags.join(', ')} onChange={(value) => onChangeDraft({...item, risk_flags: splitList(value)})} />
                  </div>
                  <div className="mt-3 grid gap-3 md:grid-cols-5">
                    <IconButton label="Save" onClick={() => onSaveDraft(item)} disabled={busyAction === `update-draft-${item.id}`}>
                      <Save size={16} />
                    </IconButton>
                    <SecondaryButton label="Accept" onClick={() => onSetDraftStatus(item, 'accepted')} />
                    <SecondaryButton label="Needs review" onClick={() => onSetDraftStatus(item, 'needs_review')} />
                    <DangerButton label="Reject" onClick={() => onSetDraftStatus(item, 'rejected')} />
                    <IconButton label="Copy" onClick={() => navigator.clipboard?.writeText(item.draft_text)} disabled={false}>
                      <Clipboard size={16} />
                    </IconButton>
                  </div>
                  <div className="mt-3">
                    <DangerButton label="Delete draft" onClick={() => onDeleteDraft(item.id)} />
                  </div>
                </div>
              ))
            )}
          </div>
        </Panel>
      </div>
    </div>
  );
}

function SectionsView({
  busyAction,
  onDetect,
  onDelete,
  onExtract,
  onSave,
  onSectionChange,
  onSelectSection,
  onSelectSource,
  sections,
  selectedSectionID,
  selectedSource,
  selectedSourceID,
  sources,
}: {
  busyAction: string;
  onDetect: () => void;
  onDelete: (id: number) => void;
  onExtract: () => void;
  onSave: (section: SourceSection) => void;
  onSectionChange: (section: SourceSection) => void;
  onSelectSection: (id: number) => void;
  onSelectSource: (id: number) => void;
  sections: SourceSection[];
  selectedSectionID: number;
  selectedSource?: CandidateSource;
  selectedSourceID: number;
  sources: CandidateSource[];
}) {
  const selectedSection = sections.find((section) => section.id === selectedSectionID) ?? sections[0];

  return (
    <div className="grid gap-4 lg:grid-cols-[340px_1fr]">
      <Panel icon={<Layers3 size={18} />} title="Source sections" subtitle="Split raw sources into editable chunks.">
        <div className="space-y-4">
          <SelectInput
            label="Source"
            value={String(selectedSourceID || '')}
            onChange={(value) => onSelectSource(Number(value))}
            options={sources.map((source) => [String(source.id), source.title])}
          />
          <IconButton label="Detect sections" onClick={onDetect} full disabled={!selectedSourceID || busyAction === 'detect-sections'}>
            <Sparkles size={16} />
          </IconButton>
          <div className="space-y-2">
            {sections.length === 0 ? (
              <EmptyState text={selectedSource ? 'No sections detected for this source.' : 'Import a source first.'} />
            ) : (
              sections.map((section) => (
                <div
                  key={section.id}
                  className={`flex items-center gap-2 rounded-md border px-3 py-2 text-sm ${section.id === selectedSection?.id ? 'border-sky-300 bg-sky-50 text-sky-950' : 'border-slate-200 bg-slate-50 text-slate-800'}`}
                >
                  <button type="button" onClick={() => onSelectSection(section.id)} className="min-w-0 flex-1 text-left">
                    <span className="font-semibold">{section.heading}</span>
                    <span className="mt-1 block text-xs text-slate-500">{sectionTypeLabel(section.section_type)}</span>
                  </button>
                  <IconOnlyButton label="Delete section" onClick={() => onDelete(section.id)}>
                    <Trash2 size={16} />
                  </IconOnlyButton>
                </div>
              ))
            )}
          </div>
        </div>
      </Panel>

      <Panel icon={<FileText size={18} />} title="Section editor" subtitle="Fix obvious section mistakes before extracting facts.">
        {!selectedSection ? (
          <EmptyState text="Select or detect a section to edit." />
        ) : (
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <TextInput label="Heading" value={selectedSection.heading} onChange={(value) => onSectionChange({...selectedSection, heading: value})} />
              <SelectInput
                label="Section type"
                value={selectedSection.section_type}
                onChange={(value) => onSectionChange({...selectedSection, section_type: value})}
                options={[
                  ['summary', 'Summary'],
                  ['skills', 'Skills'],
                  ['experience', 'Experience'],
                  ['project', 'Project'],
                  ['education', 'Education'],
                  ['certification', 'Certification'],
                  ['misc', 'Misc'],
                ]}
              />
            </div>
            <TextArea label="Content" rows={16} value={selectedSection.content} onChange={(value) => onSectionChange({...selectedSection, content: value})} />
            <div className="grid gap-3 md:grid-cols-2">
              <IconButton label="Save section" onClick={() => onSave(selectedSection)} disabled={busyAction === `save-section-${selectedSection.id}`}>
                <Save size={16} />
              </IconButton>
              <IconButton label="Extract facts" onClick={onExtract} disabled={busyAction === 'extract-facts'}>
                <Sparkles size={16} />
              </IconButton>
              <div className="md:col-span-2">
                <DangerButton label="Delete section" onClick={() => onDelete(selectedSection.id)} />
              </div>
            </div>
          </div>
        )}
      </Panel>
    </div>
  );
}

function FactsView({
  busyAction,
  facts,
  onChange,
  onDelete,
  onDeleteAll,
  onReview,
}: {
  busyAction: string;
  facts: EvidenceFact[];
  onChange: (fact: EvidenceFact) => void;
  onDelete: (id: number) => void;
  onDeleteAll: () => void | Promise<void>;
  onReview: (fact: EvidenceFact, status: string) => void | Promise<void>;
}) {
  const [filter, setFilter] = useState('needs_review');
  const counts = {
    all: facts.length,
    needs_review: facts.filter((fact) => fact.status === 'needs_review').length,
    approved: facts.filter((fact) => fact.status === 'approved').length,
    rejected: facts.filter((fact) => fact.status === 'rejected').length,
  };
  const filteredFacts = facts.filter((fact) => filter === 'all' || fact.status === filter);
  const visibleFacts = filteredFacts.slice(0, 40);
  const approveVisible = async () => {
    for (const fact of visibleFacts.filter((item) => item.status === 'needs_review')) {
      await onReview(fact, 'approved');
    }
  };

  return (
    <Panel icon={<ListChecks size={18} />} title="Fact review queue" subtitle="Approve the compact atoms; keep the quote as source evidence. Notes are optional.">
      <div className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-slate-200 bg-slate-50 p-3">
          <div className="flex flex-wrap gap-2">
            {[
              ['needs_review', `Needs review ${counts.needs_review}`],
              ['approved', `Approved ${counts.approved}`],
              ['rejected', `Rejected ${counts.rejected}`],
              ['all', `All ${counts.all}`],
            ].map(([value, label]) => (
              <button
                key={value}
                type="button"
                onClick={() => setFilter(value)}
                className={`h-9 rounded-md border px-3 text-sm font-medium ${filter === value ? 'border-slate-950 bg-slate-950 text-white' : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-100'}`}
              >
                {label}
              </button>
            ))}
          </div>
          <div className="flex flex-wrap gap-2">
            <SecondaryButton label="Approve visible" onClick={approveVisible} />
            <DangerButton label="Delete all facts" onClick={() => {
              if (facts.length > 0 && busyAction !== 'delete-all-facts') {
                void onDeleteAll();
              }
            }} />
          </div>
        </div>

        {facts.length === 0 ? (
          <EmptyState text="No evidence facts extracted yet." />
        ) : filteredFacts.length === 0 ? (
          <EmptyState text="Nothing in this queue." />
        ) : (
          <>
          {filteredFacts.length > visibleFacts.length && (
            <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
              Showing first {visibleFacts.length} of {filteredFacts.length}. Approve or delete some to continue through the queue.
            </p>
          )}
          {visibleFacts.map((fact) => (
            <div key={fact.id} className="rounded-md border border-slate-200 bg-slate-50 p-4">
              <div className="mb-3 flex items-start justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge text={fact.status} />
                  <StatusBadge text={fact.confidence} />
                  {fact.auto_approved && <StatusBadge text="auto approved" />}
                  {fact.origin_type && <StatusBadge text={fact.origin_type} />}
                  {asStringArray(fact.risk_flags).map((flag) => <StatusBadge key={flag} text={flag} />)}
                </div>
                <IconOnlyButton label="Delete fact" onClick={() => onDelete(fact.id)}>
                  <Trash2 size={16} />
                </IconOnlyButton>
              </div>
              {(fact.origin_heading || asStringArray(fact.context).length > 0) && (
                <div className="mb-3 rounded-md border border-slate-200 bg-white px-3 py-2">
                  <p className="text-xs font-semibold uppercase text-slate-500">Origin</p>
                  {fact.origin_heading && <p className="mt-1 text-sm font-medium text-slate-800">{fact.origin_heading}</p>}
                  {asStringArray(fact.context).length > 0 && (
                    <p className="mt-1 text-xs text-slate-500">{asStringArray(fact.context).join(' | ')}</p>
                  )}
                </div>
              )}
              <div className="grid gap-3 lg:grid-cols-2">
                <TextArea label="Evidence atoms" rows={3} value={fact.fact_text} onChange={(value) => onChange({...fact, fact_text: value})} />
                <TextArea label="Evidence quote" rows={3} value={fact.evidence_quote} onChange={(value) => onChange({...fact, evidence_quote: value})} />
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-3">
                <TextInput label="Technologies" value={asStringArray(fact.technologies).join(', ')} onChange={(value) => onChange({...fact, technologies: splitList(value)})} />
                <SelectInput
                  label="Confidence"
                  value={fact.confidence}
                  onChange={(value) => onChange({...fact, confidence: value})}
                  options={[
                    ['high', 'High'],
                    ['medium', 'Medium'],
                    ['low', 'Low'],
                  ]}
                />
                <TextInput label="Review note (optional)" placeholder="Why changed/rejected, or leave blank." value={fact.review_note} onChange={(value) => onChange({...fact, review_note: value})} />
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-3">
                <IconButton label="Approve" onClick={() => onReview(fact, 'approved')} disabled={busyAction === `review-fact-${fact.id}`}>
                  <CheckCircle2 size={16} />
                </IconButton>
                <SecondaryButton label="Needs review" onClick={() => onReview(fact, 'needs_review')} />
                <DangerButton label="Reject" onClick={() => onReview(fact, 'rejected')} />
              </div>
            </div>
          ))}
          </>
        )}
      </div>
    </Panel>
  );
}

function SettingsView({
  apiConfigured,
  apiKey,
  busyAction,
  events,
  health,
  llmResult,
  onAPIKeyChange,
  onInstallTectonic,
  onPromptRuleChange,
  onRenderSamplePDF,
  onRunLLMTest,
  onSaveAPIKey,
  onSavePromptRule,
  onSaveSettings,
  pdfResult,
  promptRules,
  promptSources,
  settings,
  setSettings,
  tectonicStatus,
  toolStatus,
}: {
  apiConfigured: boolean;
  apiKey: string;
  busyAction: string;
  events: AppEvent[];
  health: Health | null;
  llmResult: LLMTestResult | null;
  onAPIKeyChange: (value: string) => void;
  onInstallTectonic: () => void;
  onPromptRuleChange: (rule: PromptRule) => void;
  onRenderSamplePDF: () => void;
  onRunLLMTest: () => void;
  onSaveAPIKey: (event: FormEvent<HTMLFormElement>) => void;
  onSavePromptRule: (rule: PromptRule) => void;
  onSaveSettings: (event: FormEvent<HTMLFormElement>) => void;
  pdfResult: RenderPDFResult | null;
  promptRules: PromptRule[];
  promptSources: PromptResearchSource[];
  settings: Settings;
  setSettings: (settings: Settings) => void;
  tectonicStatus: string;
  toolStatus: ToolStatus | null;
}) {
  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_1fr]">
      <Panel icon={<SettingsIcon size={18} />} title="LLM" subtitle="Provider and key used for fact extraction.">
        <div className="space-y-4">
          <form className="space-y-4" onSubmit={onSaveSettings}>
            <SelectInput
              label="Provider"
              value={settings.provider}
              onChange={(value) => setSettings({...settings, provider: value})}
              options={[
                ['openrouter', 'OpenRouter'],
                ['openai', 'OpenAI direct'],
              ]}
            />
            <TextInput
              label="Model"
              value={settings.model}
              placeholder={modelPlaceholder(settings.provider)}
              onChange={(value) => setSettings({...settings, model: value})}
            />
            <IconButton label="Save settings" submit full disabled={busyAction === 'save-settings'}>
              <Save size={16} />
            </IconButton>
          </form>

          <form className="space-y-3 border-t border-slate-200 pt-4" onSubmit={onSaveAPIKey}>
            <MetricRow label="API key" value={apiConfigured ? `configured${toolStatus?.api_key_source ? ` (${toolStatus.api_key_source})` : ''}` : 'missing'} />
            <TextInput
              label="Update key"
              type="password"
              value={apiKey}
              placeholder={apiKeyPlaceholder(settings.provider)}
              onChange={onAPIKeyChange}
            />
            <IconButton label="Save key" submit full disabled={busyAction === 'save-key' || apiKey.trim() === ''}>
              <KeyRound size={16} />
            </IconButton>
          </form>

          <div className="space-y-3 border-t border-slate-200 pt-4">
            <IconButton label="Test call" onClick={onRunLLMTest} full disabled={busyAction === 'test-llm'}>
              <Play size={16} />
            </IconButton>
            <ResultBox
              ok={llmResult?.success}
              title={llmResult ? `${llmResult.model} - ${llmResult.latency_ms}ms` : 'Last result'}
              text={llmResult ? (llmResult.success ? llmResult.text : llmResult.error) : 'No test run yet.'}
            />
          </div>
        </div>
      </Panel>

      <div className="space-y-4">
        <Panel icon={<ListChecks size={18} />} title="Prompt rules" subtitle="Compact rule digests used by JD, fit, strategy, and resume prompts.">
          <div className="space-y-3">
            {promptRules.length === 0 ? (
              <EmptyState text="No prompt rules seeded yet." />
            ) : (
              promptRules.map((rule) => (
                <div key={rule.id} className="rounded-md border border-slate-200 bg-slate-50 p-3">
                  <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <StatusBadge text={rule.category} />
                      <StatusBadge text={`v${rule.version}`} />
                      <span className="text-sm font-semibold text-slate-950">{rule.title}</span>
                    </div>
                    <CheckInput
                      checked={rule.enabled}
                      label="Enabled"
                      onChange={(checked) => onPromptRuleChange({...rule, enabled: checked})}
                    />
                  </div>
                  <TextArea rows={3} label="Rule" value={rule.content} onChange={(value) => onPromptRuleChange({...rule, content: value})} />
                  <div className="mt-3">
                    <IconButton label="Save rule" onClick={() => onSavePromptRule(rule)} disabled={busyAction === `save-rule-${rule.id}`}>
                      <Save size={16} />
                    </IconButton>
                  </div>
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel icon={<FileText size={18} />} title="Prompt research" subtitle="Distilled source patterns used to shape app-specific prompts.">
          <div className="space-y-2">
            {promptSources.length === 0 ? (
              <EmptyState text="No research sources seeded yet." />
            ) : (
              promptSources.map((source) => (
                <div key={source.id} className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm">
                  <div className="mb-2 flex flex-wrap items-center gap-2">
                    <StatusBadge text={source.trust_tier} />
                    <StatusBadge text={source.source_type} />
                    <span className="font-semibold text-slate-950">{source.title}</span>
                  </div>
                  <p className="text-slate-700">{source.app_adaptation}</p>
                </div>
              ))
            )}
          </div>
        </Panel>

        <Panel icon={<Database size={18} />} title="Storage and PDF" subtitle="Foundation paths and render gate.">
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <PathRow label="Database" value={health?.db_path} icon={<Database size={16} />} />
              <PathRow label="Log file" value={health?.log_path} icon={<FileText size={16} />} />
              <PathRow label="Generated" value={health?.generated_path} icon={<Folder size={16} />} />
              <PathRow label="Tectonic" value={toolStatus?.tectonic_path} icon={<Cpu size={16} />} />
            </div>
            <MetricRow label="Renderer" value={tectonicStatus} />
            <div className="grid gap-3 md:grid-cols-2">
              <IconButton label="Install" onClick={onInstallTectonic} disabled={busyAction === 'install-tectonic'}>
                <Wrench size={16} />
              </IconButton>
              <IconButton label="Render sample" onClick={onRenderSamplePDF} disabled={busyAction === 'render-pdf'}>
                <FileText size={16} />
              </IconButton>
            </div>
            <ResultBox
              ok={pdfResult?.success}
              title="Sample PDF"
              text={pdfResult ? (pdfResult.success ? pdfResult.pdf_path : pdfResult.error) : 'No render yet.'}
            />
          </div>
        </Panel>

        <Panel icon={<Activity size={18} />} title="Recent events" subtitle="Backend actions and failures.">
          <div className="overflow-hidden rounded-md border border-slate-200">
            {events.length === 0 ? (
              <p className="px-4 py-5 text-sm text-slate-500">No events yet.</p>
            ) : (
              <ul className="divide-y divide-slate-200">
                {events.map((event) => (
                  <li key={event.id} className="grid gap-2 px-4 py-3 sm:grid-cols-[88px_1fr_180px]">
                    <span className="text-xs font-semibold uppercase text-slate-500">
                      {event.level}
                    </span>
                    <span className="text-sm text-slate-900">{event.message}</span>
                    <time className="text-xs text-slate-500">{formatDate(event.created_at)}</time>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Panel>
      </div>
    </div>
  );
}

function RecordEditor({
  onChange,
  onRemove,
  record,
}: {
  onChange: (record: CandidateProfileRecord) => void;
  onRemove: () => void;
  record: CandidateProfileRecord;
}) {
  const locked = record.verified;
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <div className="mb-3 flex items-center justify-between gap-3">
        <span className="text-sm font-semibold text-slate-950">{recordTypeLabel(record.record_type)}</span>
        <button
          type="button"
          onClick={onRemove}
          disabled={locked}
          className="text-sm font-medium text-red-700 hover:text-red-900 disabled:cursor-not-allowed disabled:text-slate-400"
        >
          Remove
        </button>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <SelectInput
          label="Type"
          value={record.record_type}
          onChange={(value) => onChange({...record, record_type: value})}
          disabled={locked}
          options={[
            ['education', 'Education'],
            ['employment', 'Employment'],
            ['project', 'Project'],
            ['allowed_alias', 'Allowed alias'],
            ['blocked_alias', 'Blocked alias'],
          ]}
        />
        <TextInput label="Label" value={record.label} onChange={(value) => onChange({...record, label: value})} disabled={locked} />
        <TextInput label="Organization" value={record.organization} onChange={(value) => onChange({...record, organization: value})} disabled={locked} />
        <TextInput label={locked ? 'Tailored role/title' : 'Role/title'} value={record.role} onChange={(value) => onChange({...record, role: value})} />
        <TextInput label="Start date" value={record.start_date} onChange={(value) => onChange({...record, start_date: value})} disabled={locked} />
        <TextInput label="End date" value={record.end_date} onChange={(value) => onChange({...record, end_date: value})} disabled={locked} />
        <div className="md:col-span-2">
          <TextArea label="Locked value" rows={3} value={record.value} onChange={(value) => onChange({...record, value})} disabled={locked} />
        </div>
        <div className="md:col-span-2">
          <CheckInput
            checked={record.verified}
            label="Verified locked record"
            onChange={(checked) => onChange({...record, verified: checked})}
          />
        </div>
      </div>
    </div>
  );
}

function StatusPill({state, text}: {state: LoadState; text: string}) {
  const styles = {
    loading: 'border-amber-200 bg-amber-50 text-amber-800',
    ready: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    error: 'border-red-200 bg-red-50 text-red-800',
  }[state];

  return (
    <span className={`inline-flex h-10 items-center gap-2 rounded-md border px-3 text-sm font-medium ${styles}`}>
      {state === 'ready' ? <CheckCircle2 size={16} /> : <AlertCircle size={16} />}
      {text}
    </span>
  );
}

function SummaryBlock({empty, items, title}: {empty: string; items: string[]; title: string}) {
  const visible = asStringArray(items).slice(0, 6);
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <p className="text-sm font-semibold text-slate-950">{title}</p>
      {visible.length === 0 ? (
        <p className="mt-2 text-sm text-slate-500">{empty}</p>
      ) : (
        <div className="mt-2 flex flex-wrap gap-2">
          {visible.map((item) => <StatusBadge key={item} text={item} />)}
        </div>
      )}
    </div>
  );
}

function TabButton({active, icon, label, onClick}: {active: boolean; icon: JSX.Element; label: string; onClick: () => void}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex h-10 shrink-0 items-center gap-2 rounded-md border px-3 text-sm font-medium ${active ? 'border-slate-950 bg-slate-950 text-white' : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-50'}`}
    >
      {icon}
      {label}
    </button>
  );
}

function IconButton({
  children,
  disabled,
  full,
  label,
  onClick,
  submit,
}: {
  children: JSX.Element;
  disabled?: boolean;
  full?: boolean;
  label: string;
  onClick?: () => void;
  submit?: boolean;
}) {
  return (
    <button
      type={submit ? 'submit' : 'button'}
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex h-10 items-center justify-center gap-2 rounded-md bg-slate-950 px-3 text-sm font-medium text-white shadow-sm hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400 ${full ? 'w-full' : ''}`}
      title={label}
    >
      {children}
      {label}
    </button>
  );
}

function SecondaryButton({label, onClick}: {label: string; onClick: () => void}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex h-10 items-center justify-center rounded-md border border-slate-300 bg-white px-3 text-sm font-medium text-slate-800 shadow-sm hover:bg-slate-50"
    >
      {label}
    </button>
  );
}

function IconOnlyButton({children, label, onClick}: {children: JSX.Element; label: string; onClick: () => void}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-slate-300 bg-white text-slate-700 shadow-sm hover:bg-slate-50"
    >
      {children}
    </button>
  );
}

function DangerButton({label, onClick}: {label: string; onClick: () => void}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex h-10 items-center justify-center rounded-md border border-red-200 bg-red-50 px-3 text-sm font-medium text-red-800 shadow-sm hover:bg-red-100"
    >
      {label}
    </button>
  );
}

function CheckInput({checked, label, onChange}: {checked: boolean; label: string; onChange: (checked: boolean) => void}) {
  return (
    <label className="inline-flex min-h-10 items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-medium text-slate-700">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="h-4 w-4 rounded border-slate-300"
      />
      {label}
    </label>
  );
}

function Panel({
  icon,
  title,
  subtitle,
  children,
}: {
  icon: JSX.Element;
  title: string;
  subtitle: string;
  children: JSX.Element;
}) {
  return (
    <section className="rounded-md border border-slate-200 bg-white p-4 shadow-sm">
      <div className="mb-4 flex items-start gap-3">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-100 text-slate-700">
          {icon}
        </div>
        <div>
          <h2 className="text-base font-semibold text-slate-950">{title}</h2>
          <p className="mt-1 text-sm text-slate-500">{subtitle}</p>
        </div>
      </div>
      {children}
    </section>
  );
}

function TextInput({
  disabled,
  label,
  onChange,
  placeholder,
  type,
  value,
}: {
  disabled?: boolean;
  label: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: string;
  value: string;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        placeholder={placeholder}
        type={type ?? 'text'}
        className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100 disabled:bg-slate-100 disabled:text-slate-500"
      />
    </label>
  );
}

function SelectInput({
  disabled,
  label,
  onChange,
  options,
  value,
}: {
  disabled?: boolean;
  label: string;
  onChange: (value: string) => void;
  options: string[][];
  value: string;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        className="mt-1 h-10 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100 disabled:bg-slate-100 disabled:text-slate-500"
      >
        {options.length === 0 && <option value="">None</option>}
        {options.map(([optionValue, labelText]) => (
          <option key={optionValue} value={optionValue}>{labelText}</option>
        ))}
      </select>
    </label>
  );
}

function TextArea({
  disabled,
  label,
  onChange,
  rows,
  value,
}: {
  disabled?: boolean;
  label: string;
  onChange: (value: string) => void;
  rows: number;
  value: string;
}) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-700">{label}</span>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        disabled={disabled}
        rows={rows}
        className="mt-1 w-full resize-y rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-950 outline-none focus:border-sky-500 focus:ring-2 focus:ring-sky-100 disabled:bg-slate-100 disabled:text-slate-500"
      />
    </label>
  );
}

function EmptyState({text}: {text: string}) {
  return (
    <p className="rounded-md border border-dashed border-slate-300 bg-slate-50 px-4 py-6 text-center text-sm text-slate-500">
      {text}
    </p>
  );
}

function PathRow({label, value, icon}: {label: string; value?: string; icon: JSX.Element}) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-slate-500">
        {icon}
        {label}
      </div>
      <p className="break-all text-sm text-slate-900">{value ?? 'checking'}</p>
    </div>
  );
}

function MetricRow({label, value}: {label: string; value: string}) {
  return (
    <div className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2">
      <dt className="text-sm font-medium text-slate-600">{label}</dt>
      <dd className="text-right text-sm font-semibold text-slate-950">{value}</dd>
    </div>
  );
}

function ResultBox({ok, text, title}: {ok?: boolean; text: string; title: string}) {
  const tone = ok === undefined ? 'border-slate-200 bg-slate-50' : ok ? 'border-emerald-200 bg-emerald-50' : 'border-red-200 bg-red-50';
  return (
    <div className={`rounded-md border px-3 py-3 ${tone}`}>
      <p className="text-xs font-semibold uppercase text-slate-500">{title}</p>
      <p className="mt-1 break-all text-sm text-slate-900">{text}</p>
    </div>
  );
}

function StatusBadge({text}: {text: string}) {
  return (
    <span className="rounded-md border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-600">
      {text.replaceAll('_', ' ')}
    </span>
  );
}

function normalizeProfile(value: CandidateProfile): CandidateProfile {
  const contact = {...emptyProfile.contact, ...(value?.contact ?? {})};
  return {
    contact: {
      ...contact,
      links: asStringArray(contact.links),
      verified: Boolean(contact.verified),
    },
    records: asArray(value?.records).map((record) => ({
      ...newRecord(record.record_type || 'project'),
      ...record,
      verified: Boolean(record.verified),
    })),
  };
}

function newRecord(recordType: string): CandidateProfileRecord {
  return {
    id: 0,
    record_type: recordType,
    label: '',
    organization: '',
    role: '',
    start_date: '',
    end_date: '',
    value: '',
    verified: false,
    created_at: '',
    updated_at: '',
  };
}

function mergeDraftRecords(current: CandidateProfileRecord[], draft: CandidateProfileRecord[]) {
  const currentRecords = asArray(current);
  const draftRecords = asArray(draft);
  const seen = new Set(currentRecords.map((record) => `${record.record_type}|${record.organization}|${record.role}|${record.start_date}|${record.end_date}|${record.value}`.toLowerCase()));
  const next = [...currentRecords];
  for (const record of draftRecords) {
    const key = `${record.record_type}|${record.organization}|${record.role}|${record.start_date}|${record.end_date}|${record.value}`.toLowerCase();
    if (!seen.has(key)) {
      next.push({...record, verified: false});
      seen.add(key);
    }
  }
  return next;
}

function normalizeFacts(value?: EvidenceFact[] | null) {
  return asArray(value).map((fact) => ({
    ...fact,
    technologies: asStringArray(fact.technologies),
    risk_flags: asStringArray(fact.risk_flags),
    origin_heading: fact.origin_heading || '',
    origin_type: fact.origin_type || '',
    context: asStringArray(fact.context),
  }));
}

function normalizeRequirements(value?: JobRequirement[] | null) {
  return asArray(value).map((requirement) => ({
    ...requirement,
    keywords: asStringArray(requirement.keywords),
  }));
}

function normalizeMatches(value?: JobFactMatch[] | null) {
  return asArray(value).map((match) => ({
    ...match,
    risk_flags: asStringArray(match.risk_flags),
  }));
}

function normalizeDrafts(value?: TailoredBulletDraft[] | null) {
  return asArray(value).map((draft) => ({
    ...draft,
    fact_ids: asArray(draft.fact_ids),
    risk_flags: asStringArray(draft.risk_flags),
  }));
}

function normalizeJobAnalysis(value?: JobAnalysis | null): JobAnalysis | null {
  if (!value) return null;
  return {
    ...value,
    top_pain_points: asStringArray(value.top_pain_points),
    required_skills: asStringArray(value.required_skills),
    preferred_skills: asStringArray(value.preferred_skills),
    responsibilities: asStringArray(value.responsibilities),
    keywords: asStringArray(value.keywords),
    risk_flags: asStringArray(value.risk_flags),
  };
}

function normalizeFitAnalysis(value?: JobFitAnalysis | null): JobFitAnalysis | null {
  if (!value) return null;
  return {
    ...value,
    strengths: asStringArray(value.strengths),
    critical_gaps: asStringArray(value.critical_gaps),
    analysis: asArray(value.analysis).map((item) => ({
      ...item,
      matching_fact_ids: asArray(item.matching_fact_ids),
    })),
  };
}

function normalizeApplicationStrategy(value?: ApplicationStrategy | null): ApplicationStrategy | null {
  if (!value) return null;
  return {
    ...value,
    approved_fact_ids: asArray(value.approved_fact_ids),
    rejected_fact_ids: asArray(value.rejected_fact_ids),
    weak_or_missing_requirements: asStringArray(value.weak_or_missing_requirements),
    experience_titles: value.experience_titles ?? {},
    keywords: asStringArray(value.keywords),
    do_not_overclaim: asStringArray(value.do_not_overclaim),
  };
}

function asArray<T>(value?: T[] | null): T[] {
  return Array.isArray(value) ? value : [];
}

function asStringArray(value?: string[] | null): string[] {
  return Array.isArray(value) ? value : [];
}

function splitList(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean);
}

function inferJobDetailsFromText(rawText: string) {
  const lines = rawText
    .split('\n')
    .map(cleanJobDetailLine)
    .filter((line) => line && !line.toLowerCase().startsWith('http'))
    .slice(0, 12);
  let company = '';
  let title = '';
  for (const line of lines) {
    const lower = line.toLowerCase();
    for (const prefix of ['company:', 'employer:', 'organisation:', 'organization:']) {
      if (lower.startsWith(prefix)) {
        company = line.slice(prefix.length).trim();
      }
    }
    for (const prefix of ['job title:', 'role:', 'title:', 'position:']) {
      if (lower.startsWith(prefix)) {
        title = line.slice(prefix.length).trim();
      }
    }
  }
  if (!title) {
    title = lines.find(looksLikeRoleTitle) ?? '';
  }
  if (!company) {
    company = lines.find((line) => {
      const lower = line.toLowerCase();
      return line !== title &&
        !looksLikeRoleTitle(line) &&
        !lower.includes('responsibilities') &&
        !lower.includes('requirements') &&
        !lower.includes('about the role') &&
        line.split(/\s+/).length <= 5;
    }) ?? '';
  }
  if (title.includes(' at ') && !company) {
    const [role, org] = title.split(/\s+at\s+/, 2);
    title = role.trim();
    company = org.trim();
  }
  return {company, title};
}

function cleanJobDetailLine(line: string) {
  return line.trim().replace(/^[-•]\s*/, '').replace(/^[#*_`]+|[#*_`]+$/g, '').replace(/\s+/g, ' ').trim();
}

function looksLikeRoleTitle(line: string) {
  if (line.split(/\s+/).length > 9) {
    return false;
  }
  return /engineer|developer|manager|analyst|designer|architect|consultant|specialist|lead|intern|graduate|backend|frontend|full stack|software|data|devops|platform/i.test(line);
}

function sourceTypeLabel(value: string) {
  const labels: Record<string, string> = {
    current_resume: 'Current resume',
    extended_resume: 'Extended resume',
    old_resume: 'Old resume',
    project_notes: 'Project notes',
    readme: 'README',
    architecture_notes: 'Architecture notes',
    interview_notes: 'Interview notes',
    manual_notes: 'Manual notes',
  };
  return labels[value] ?? value;
}

function sourceName(sources: CandidateSource[], selectedSourceID: number) {
  return sources.find((source) => source.id === selectedSourceID)?.title ?? 'Select a source in Sources or Sections first.';
}

function sectionTypeLabel(value: string) {
  const labels: Record<string, string> = {
    summary: 'Summary',
    skills: 'Skills',
    experience: 'Experience',
    project: 'Project',
    education: 'Education',
    certification: 'Certification',
    misc: 'Misc',
  };
  return labels[value] ?? value;
}

function recordTypeLabel(value: string) {
  const labels: Record<string, string> = {
    education: 'Education',
    employment: 'Employment',
    project: 'Project',
    allowed_alias: 'Allowed alias',
    blocked_alias: 'Blocked alias',
  };
  return labels[value] ?? value;
}

function modelPlaceholder(provider: string) {
  return provider === 'openai' ? 'gpt-5.4-mini' : 'deepseek/deepseek-v4-flash';
}

function apiKeyPlaceholder(provider: string) {
  return provider === 'openai' ? 'OPENAI_API_KEY' : 'OPENROUTER_API_KEY';
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export default App;
