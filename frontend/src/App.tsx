import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
  Activity,
  AlertCircle,
  CheckCircle2,
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
  CandidateProfile,
  CandidateProfileRecord,
  CandidateSource,
  CreateCandidateSource,
  DeleteCandidateSource,
  DeleteEvidenceFact,
  DeleteSourceSection,
  DetectSourceSections,
  DraftCandidateProfileFromSource,
  EvidenceFact,
  ExtractEvidenceFacts,
  GetCandidateProfile,
  GetHealth,
  GetRecentEvents,
  GetSettings,
  GetToolStatus,
  ListCandidateSources,
  ListEvidenceFacts,
  ListSourceSections,
  RenderSamplePDF,
  SaveAPIKey,
  SaveCandidateProfile,
  SaveSettings,
  SourceSection,
  TestLLM,
  UpdateEvidenceFactReview,
  UpdateSourceSection,
  InstallTectonic,
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
type Tab = 'profile' | 'sources' | 'sections' | 'facts' | 'settings';

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
  const [activeTab, setActiveTab] = useState<Tab>('sources');
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
  const [profile, setProfile] = useState<CandidateProfile>(emptyProfile);
  const [sources, setSources] = useState<CandidateSource[]>([]);
  const [sections, setSections] = useState<SourceSection[]>([]);
  const [facts, setFacts] = useState<EvidenceFact[]>([]);
  const [selectedSourceID, setSelectedSourceID] = useState<number>(0);
  const [selectedSectionID, setSelectedSectionID] = useState<number>(0);
  const [sourceDraft, setSourceDraft] = useState({
    source_type: 'current_resume',
    title: '',
    raw_text: '',
  });
  const [apiKey, setAPIKey] = useState('');
  const [llmResult, setLLMResult] = useState<LLMTestResult | null>(null);
  const [pdfResult, setPDFResult] = useState<RenderPDFResult | null>(null);
  const [busyAction, setBusyAction] = useState('');

  const selectedSource = sources.find((source) => source.id === selectedSourceID);
  const selectedSection = sections.find((section) => section.id === selectedSectionID);
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
        nextProfile,
        nextSources,
        nextSections,
        nextFacts,
      ] = await Promise.all([
        GetHealth(),
        GetSettings(),
        GetToolStatus(),
        GetRecentEvents(),
        GetCandidateProfile(),
        ListCandidateSources(),
        ListSourceSections(0),
        ListEvidenceFacts('all'),
      ]);
      setHealth(nextHealth as Health);
      setSettings(nextSettings as Settings);
      setToolStatus(nextStatus as ToolStatus);
      setEvents((nextEvents ?? []) as AppEvent[]);
      setProfile(normalizeProfile(nextProfile as CandidateProfile));
      setSources((nextSources ?? []) as CandidateSource[]);
      setSections((nextSections ?? []) as SourceSection[]);
      setFacts((nextFacts ?? []) as EvidenceFact[]);
      const firstSource = (nextSources as CandidateSource[] | undefined)?.[0];
      if (!selectedSourceID && firstSource) {
        setSelectedSourceID(firstSource.id);
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
    const [nextSources, nextSections, nextFacts] = await Promise.all([
      ListCandidateSources(),
      ListSourceSections(0),
      ListEvidenceFacts('all'),
    ]);
    setSources((nextSources ?? []) as CandidateSource[]);
    setSections((nextSections ?? []) as SourceSection[]);
    setFacts((nextFacts ?? []) as EvidenceFact[]);
    await refreshEvents();
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
      setFacts(nextFacts);
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
      setFacts((previous) => previous.map((item) => item.id === saved.id ? saved : item));
      await refreshEvents();
    });
  }

  async function deleteFact(factID: number) {
    await runAction(`delete-fact-${factID}`, async () => {
      await DeleteEvidenceFact({id: factID});
      const nextFacts = (await ListEvidenceFacts('all')) as EvidenceFact[];
      setFacts(nextFacts);
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
            onChange={(fact) => setFacts((previous) => previous.map((item) => item.id === fact.id ? fact : item))}
            onDelete={deleteFact}
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
            onSaveAPIKey={saveAPIKey}
            onSaveSettings={saveSettings}
            pdfResult={pdfResult}
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
  onReview,
}: {
  busyAction: string;
  facts: EvidenceFact[];
  onChange: (fact: EvidenceFact) => void;
  onDelete: (id: number) => void;
  onReview: (fact: EvidenceFact, status: string) => void;
}) {
  return (
    <Panel icon={<ListChecks size={18} />} title="Fact review queue" subtitle="Only risky or uncertain facts need attention.">
      <div className="space-y-4">
        {facts.length === 0 ? (
          <EmptyState text="No evidence facts extracted yet." />
        ) : (
          facts.map((fact) => (
            <div key={fact.id} className="rounded-md border border-slate-200 bg-slate-50 p-4">
              <div className="mb-3 flex items-start justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge text={fact.status} />
                  <StatusBadge text={fact.confidence} />
                  {fact.auto_approved && <StatusBadge text="auto approved" />}
                  {fact.risk_flags.map((flag) => <StatusBadge key={flag} text={flag} />)}
                </div>
                <IconOnlyButton label="Delete fact" onClick={() => onDelete(fact.id)}>
                  <Trash2 size={16} />
                </IconOnlyButton>
              </div>
              <div className="grid gap-3 lg:grid-cols-2">
                <TextArea label="Fact" rows={5} value={fact.fact_text} onChange={(value) => onChange({...fact, fact_text: value})} />
                <TextArea label="Evidence quote" rows={5} value={fact.evidence_quote} onChange={(value) => onChange({...fact, evidence_quote: value})} />
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-3">
                <TextInput label="Technologies" value={fact.technologies.join(', ')} onChange={(value) => onChange({...fact, technologies: splitList(value)})} />
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
                <TextInput label="Review note" value={fact.review_note} onChange={(value) => onChange({...fact, review_note: value})} />
              </div>
              <div className="mt-3 grid gap-3 md:grid-cols-3">
                <IconButton label="Approve" onClick={() => onReview(fact, 'approved')} disabled={busyAction === `review-fact-${fact.id}`}>
                  <CheckCircle2 size={16} />
                </IconButton>
                <SecondaryButton label="Needs review" onClick={() => onReview(fact, 'needs_review')} />
                <DangerButton label="Reject" onClick={() => onReview(fact, 'rejected')} />
              </div>
            </div>
          ))
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
  onRenderSamplePDF,
  onRunLLMTest,
  onSaveAPIKey,
  onSaveSettings,
  pdfResult,
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
  onRenderSamplePDF: () => void;
  onRunLLMTest: () => void;
  onSaveAPIKey: (event: FormEvent<HTMLFormElement>) => void;
  onSaveSettings: (event: FormEvent<HTMLFormElement>) => void;
  pdfResult: RenderPDFResult | null;
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
  return {
    contact: {...emptyProfile.contact, ...(value?.contact ?? {})},
    records: value?.records ?? [],
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
  const seen = new Set(current.map((record) => `${record.record_type}|${record.organization}|${record.role}|${record.start_date}|${record.end_date}|${record.value}`.toLowerCase()));
  const next = [...current];
  for (const record of draft) {
    const key = `${record.record_type}|${record.organization}|${record.role}|${record.start_date}|${record.end_date}|${record.value}`.toLowerCase();
    if (!seen.has(key)) {
      next.push({...record, verified: false});
      seen.add(key);
    }
  }
  return next;
}

function splitList(value: string) {
  return value.split(',').map((item) => item.trim()).filter(Boolean);
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
