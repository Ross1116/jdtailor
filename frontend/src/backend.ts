import {
  AnalyzeJobDescription as WailsAnalyzeJobDescription,
  BuildJobMatchMap as WailsBuildJobMatchMap,
  CreateCandidateSource as WailsCreateCandidateSource,
  CreateJobDescription as WailsCreateJobDescription,
  DeleteCandidateSource as WailsDeleteCandidateSource,
  DeleteAllEvidenceFacts as WailsDeleteAllEvidenceFacts,
  DeleteEvidenceFact as WailsDeleteEvidenceFact,
  DeleteJobDescription as WailsDeleteJobDescription,
  DeleteSourceSection as WailsDeleteSourceSection,
  DeleteTailoredBulletDraft as WailsDeleteTailoredBulletDraft,
  DetectSourceSections as WailsDetectSourceSections,
  DraftCandidateProfileFromSource as WailsDraftCandidateProfileFromSource,
  ExtractEvidenceFacts as WailsExtractEvidenceFacts,
  GenerateTailoredBulletDrafts as WailsGenerateTailoredBulletDrafts,
  GenerateApplicationStrategy as WailsGenerateApplicationStrategy,
  GenerateFitAnalysis as WailsGenerateFitAnalysis,
  GetApplicationStrategy as WailsGetApplicationStrategy,
  GetCandidateProfile as WailsGetCandidateProfile,
  GetFitAnalysis as WailsGetFitAnalysis,
  GetHealth as WailsGetHealth,
  GetJobAnalysis as WailsGetJobAnalysis,
  GetRecentEvents as WailsGetRecentEvents,
  GetSettings as WailsGetSettings,
  GetToolStatus as WailsGetToolStatus,
  ImportCandidateSourceFile as WailsImportCandidateSourceFile,
  InstallTectonic as WailsInstallTectonic,
  ListJobDescriptions as WailsListJobDescriptions,
  ListJobFactMatches as WailsListJobFactMatches,
  ListJobRequirements as WailsListJobRequirements,
  ListPromptResearchSources as WailsListPromptResearchSources,
  ListPromptRules as WailsListPromptRules,
  ListCandidateSources as WailsListCandidateSources,
  ListEvidenceFacts as WailsListEvidenceFacts,
  ListSourceSections as WailsListSourceSections,
  ListTailoredBulletDrafts as WailsListTailoredBulletDrafts,
  ParseJobDescription as WailsParseJobDescription,
  RenderSamplePDF as WailsRenderSamplePDF,
  SaveAPIKey as WailsSaveAPIKey,
  SaveCandidateProfile as WailsSaveCandidateProfile,
  SaveSettings as WailsSaveSettings,
  TestLLM as WailsTestLLM,
  UpdateEvidenceFactReview as WailsUpdateEvidenceFactReview,
  UpdateJobDescription as WailsUpdateJobDescription,
  UpdatePromptRule as WailsUpdatePromptRule,
  UpdateSourceSection as WailsUpdateSourceSection,
  UpdateTailoredBulletDraft as WailsUpdateTailoredBulletDraft,
} from '../wailsjs/go/main/App';

export type CandidateProfile = {
  contact: CandidateContact;
  records: CandidateProfileRecord[];
};

export type CandidateContact = {
  full_name: string;
  email: string;
  phone: string;
  location: string;
  linkedin: string;
  github: string;
  portfolio: string;
  links: string[];
  verified: boolean;
  updated_at: string;
};

export type CandidateProfileRecord = {
  id: number;
  record_type: string;
  label: string;
  organization: string;
  role: string;
  start_date: string;
  end_date: string;
  value: string;
  verified: boolean;
  created_at: string;
  updated_at: string;
};

export type CandidateSource = {
  id: number;
  source_type: string;
  title: string;
  raw_text: string;
  file_path: string;
  imported_at: string;
  updated_at: string;
};

export type SourceSection = {
  id: number;
  source_id: number;
  heading: string;
  section_type: string;
  content: string;
  sort_order: number;
  start_char: number;
  end_char: number;
  created_at: string;
  updated_at: string;
};

export type EvidenceFact = {
  id: number;
  source_id: number;
  section_id: number;
  fact_text: string;
  evidence_quote: string;
  technologies: string[];
  confidence: string;
  risk_flags: string[];
  origin_heading: string;
  origin_type: string;
  context: string[];
  status: string;
  auto_approved: boolean;
  review_note: string;
  created_at: string;
  updated_at: string;
};

export type JobDescription = {
  id: number;
  company: string;
  title: string;
  url: string;
  raw_text: string;
  created_at: string;
  updated_at: string;
};

export type JobRequirement = {
  id: number;
  job_id: number;
  category: string;
  requirement_text: string;
  keywords: string[];
  priority: string;
  source_quote: string;
  created_at: string;
  updated_at: string;
};

export type JobFactMatch = {
  id: number;
  job_id: number;
  requirement_id: number;
  fact_id: number;
  score: number;
  rationale: string;
  coverage_status: string;
  fact_status: string;
  fact_text: string;
  evidence_quote: string;
  risk_flags: string[];
  created_at: string;
  updated_at: string;
};

export type TailoredBulletDraft = {
  id: number;
  job_id: number;
  requirement_id: number;
  fact_ids: number[];
  draft_text: string;
  rationale: string;
  status: string;
  risk_flags: string[];
  created_at: string;
  updated_at: string;
};

export type JobAnalysis = {
  job_id: number;
  company: string;
  role_title: string;
  location: string;
  work_arrangement: string;
  salary: string;
  top_pain_points: string[];
  required_skills: string[];
  preferred_skills: string[];
  responsibilities: string[];
  seniority_level: string;
  role_archetype: string;
  keywords: string[];
  risk_flags: string[];
  job_poster: string;
  company_url: string;
  created_at: string;
  updated_at: string;
};

export type FitNeedAnalysis = {
  requirement_id: number;
  jd_need: string;
  matching_fact_ids: number[];
  evidence_strength: string;
  gap_level: string;
  confidence: string;
  risk: string;
};

export type JobFitAnalysis = {
  job_id: number;
  overall_score: number;
  recommendation: string;
  strengths: string[];
  critical_gaps: string[];
  reality_check: string;
  analysis: FitNeedAnalysis[];
  created_at: string;
  updated_at: string;
};

export type ApplicationStrategy = {
  job_id: number;
  approved_fact_ids: number[];
  rejected_fact_ids: number[];
  weak_or_missing_requirements: string[];
  resume_headline: string;
  experience_titles: Record<string, string>;
  positioning_strategy: string;
  keywords: string[];
  do_not_overclaim: string[];
  fit_summary: string;
  created_at: string;
  updated_at: string;
};

export type PromptRule = {
  id: number;
  rule_key: string;
  category: string;
  title: string;
  content: string;
  enabled: boolean;
  version: number;
  source: string;
  created_at: string;
  updated_at: string;
};

export type PromptResearchSource = {
  id: number;
  source_type: string;
  trust_tier: string;
  title: string;
  url: string;
  extracted_pattern: string;
  app_adaptation: string;
  accessed_at: string;
  created_at: string;
};

const hasWailsBackend = () => Boolean(window.go?.main?.App);

const now = () => new Date().toISOString();

const mockSettings = {
  provider: 'openrouter',
  model: 'deepseek/deepseek-v4-flash',
  api_key_configured: false,
};

let mockProfile: CandidateProfile = {
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
    updated_at: now(),
  },
  records: [],
};

let mockSources: CandidateSource[] = [];
let mockSections: SourceSection[] = [];
let mockFacts: EvidenceFact[] = [];
let mockJobs: JobDescription[] = [];
let mockRequirements: JobRequirement[] = [];
let mockMatches: JobFactMatch[] = [];
let mockDrafts: TailoredBulletDraft[] = [];
let mockAnalyses: JobAnalysis[] = [];
let mockFitAnalyses: JobFitAnalysis[] = [];
let mockStrategies: ApplicationStrategy[] = [];
let mockPromptRules: PromptRule[] = defaultMockPromptRules();
const mockPromptSources: PromptResearchSource[] = defaultMockPromptSources();

const mockEvents = [
  {
    id: 1,
    level: 'info',
    message: 'frontend mock backend active',
    created_at: now(),
  },
];

export async function GetHealth() {
  if (hasWailsBackend()) {
    return WailsGetHealth();
  }
  return {
    version: 'frontend-dev',
    storage_status: 'mock',
    db_path: 'mock://data/app.db',
    log_path: 'mock://logs/app.log',
    generated_path: 'mock://generated',
    pdf_renderer: 'mock',
  };
}

export async function GetSettings() {
  if (hasWailsBackend()) {
    return WailsGetSettings();
  }
  return mockSettings;
}

export async function SaveSettings(input: {provider: string; model: string}) {
  if (hasWailsBackend()) {
    return WailsSaveSettings(input);
  }
  mockSettings.provider = input.provider;
  mockSettings.model = input.model || modelDefault(input.provider);
  mockEvents.unshift(mockEvent('info', 'mock settings saved'));
  return mockSettings;
}

export async function GetToolStatus() {
  if (hasWailsBackend()) {
    return WailsGetToolStatus();
  }
  return {
    api_key_configured: mockSettings.api_key_configured,
    api_key_source: mockSettings.api_key_configured ? 'mock' : '',
    env_local_path: 'mock://.env.local',
    tectonic_status: 'mock',
    tectonic_path: 'mock://tools/tectonic/tectonic.exe',
    generated_path: 'mock://generated',
  };
}

export async function SaveAPIKey(input: {api_key: string; provider: string}) {
  if (hasWailsBackend()) {
    return WailsSaveAPIKey(input);
  }
  mockSettings.provider = input.provider;
  mockSettings.api_key_configured = input.api_key.trim() !== '';
  mockEvents.unshift(mockEvent('info', 'mock API key saved'));
  return GetToolStatus();
}

export async function TestLLM() {
  if (hasWailsBackend()) {
    return WailsTestLLM();
  }
  const success = mockSettings.api_key_configured;
  mockEvents.unshift(mockEvent(success ? 'info' : 'error', success ? 'mock LLM smoke test succeeded' : 'mock LLM smoke test failed: missing API key'));
  return {
    success,
    provider: mockSettings.provider,
    model: mockSettings.model,
    text: success ? 'JD Tailor LLM check' : '',
    latency_ms: 12,
    status_code: success ? 200 : 0,
    error: success ? '' : 'OPENROUTER_API_KEY is missing',
  };
}

export async function InstallTectonic() {
  if (hasWailsBackend()) {
    return WailsInstallTectonic();
  }
  mockEvents.unshift(mockEvent('info', 'mock Tectonic installed'));
  return {
    success: true,
    status: 'mock',
    executable_path: 'mock://tools/tectonic/tectonic.exe',
    error: '',
  };
}

export async function RenderSamplePDF() {
  if (hasWailsBackend()) {
    return WailsRenderSamplePDF();
  }
  mockEvents.unshift(mockEvent('info', 'mock sample PDF rendered'));
  return {
    success: true,
    tex_path: 'mock://generated/sample-pdf/sample.tex',
    pdf_path: 'mock://generated/sample-pdf/sample.pdf',
    error: '',
  };
}

export async function GetRecentEvents() {
  if (hasWailsBackend()) {
    return WailsGetRecentEvents();
  }
  return mockEvents;
}

export async function GetCandidateProfile() {
  if (hasWailsBackend()) {
    return WailsGetCandidateProfile();
  }
  return mockProfile;
}

export async function SaveCandidateProfile(input: CandidateProfile) {
  if (hasWailsBackend()) {
    return WailsSaveCandidateProfile(input as any);
  }
  const timestamp = now();
  mockProfile = {
    contact: {...input.contact, updated_at: timestamp},
    records: input.records.map((record, index) => ({
      ...record,
      id: index + 1,
      created_at: record.created_at || timestamp,
      updated_at: timestamp,
    })),
  };
  mockEvents.unshift(mockEvent('info', 'mock candidate profile saved'));
  return mockProfile;
}

export async function DraftCandidateProfileFromSource(sourceID: number) {
  if (hasWailsBackend()) {
    return WailsDraftCandidateProfileFromSource(sourceID);
  }
  const source = mockSources.find((item) => item.id === sourceID);
  const text = normalizeRawSourceText(source?.raw_text ?? '');
  const email = text.match(/[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}/)?.[0] ?? '';
  const lines = text.split('\n').map((line) => line.trim()).filter(Boolean);
  const nameLine = lines.find((line) => !isKnownSectionHeading(line) && !line.startsWith('- ') && !line.includes('|')) ?? '';
  return {
    contact: {
      full_name: nameLine,
      email,
      phone: text.match(/\+?\d[\d\s-]{8,}/)?.[0]?.trim() ?? '',
      location: lines.find((line) => line.toLowerCase().includes('melbourne')) ?? '',
      linkedin: text.match(/(?:https?:\/\/)?(?:www\.)?linkedin\.com\/in\/[A-Za-z0-9_\-./]+/)?.[0] ?? '',
      github: text.match(/(?:https?:\/\/)?(?:www\.)?github\.com\/[A-Za-z0-9_\-./]+/)?.[0] ?? '',
      portfolio: '',
      links: [],
      verified: false,
      updated_at: now(),
    },
    records: [
      ...draftRecordsFromText(text, 'Experience', 'employment'),
      ...draftRecordsFromText(text, 'Projects', 'project'),
      ...draftRecordsFromText(text, 'Education', 'education'),
    ],
  };
}

export async function ListCandidateSources() {
  if (hasWailsBackend()) {
    return WailsListCandidateSources();
  }
  return mockSources;
}

export async function DeleteCandidateSource(input: {id: number}) {
  if (hasWailsBackend()) {
    return WailsDeleteCandidateSource(input);
  }
  mockSources = mockSources.filter((source) => source.id !== input.id);
  mockSections = mockSections.filter((section) => section.source_id !== input.id);
  mockFacts = mockFacts.filter((fact) => fact.source_id !== input.id);
  mockEvents.unshift(mockEvent('info', 'mock source deleted'));
}

export async function CreateCandidateSource(input: {source_type: string; title: string; raw_text: string}) {
  if (hasWailsBackend()) {
    return WailsCreateCandidateSource(input);
  }
  const timestamp = now();
  const source = {
    id: Date.now(),
    source_type: input.source_type || 'manual_notes',
    title: input.title || 'Untitled source',
    raw_text: normalizeRawSourceText(input.raw_text),
    file_path: '',
    imported_at: timestamp,
    updated_at: timestamp,
  };
  mockSources = [source, ...mockSources];
  mockEvents.unshift(mockEvent('info', 'mock source imported'));
  return source;
}

export async function ImportCandidateSourceFile(input: {path: string; source_type: string; title: string}) {
  if (hasWailsBackend()) {
    return WailsImportCandidateSourceFile(input);
  }
  return CreateCandidateSource({
    source_type: input.source_type,
    title: input.title || input.path,
    raw_text: `Mock file import from ${input.path}`,
  });
}

export async function ListSourceSections(sourceID: number) {
  if (hasWailsBackend()) {
    return WailsListSourceSections(sourceID);
  }
  return sourceID > 0 ? mockSections.filter((section) => section.source_id === sourceID) : mockSections;
}

export async function DetectSourceSections(sourceID: number) {
  if (hasWailsBackend()) {
    return WailsDetectSourceSections(sourceID);
  }
  const source = mockSources.find((item) => item.id === sourceID);
  if (!source) {
    return [];
  }
  const timestamp = now();
  const normalizedText = normalizeRawSourceText(source.raw_text);
  const parts = splitSourceSections(normalizedText);
  mockSections = mockSections.filter((section) => section.source_id !== sourceID);
  const sections = (parts.length ? parts : [{heading: source.title, section_type: 'misc', content: normalizedText}]).map((part, index) => {
    return {
      id: Date.now() + index,
      source_id: sourceID,
      heading: part.heading,
      section_type: part.section_type,
      content: part.content,
      sort_order: index,
      start_char: 0,
      end_char: part.content.length,
      created_at: timestamp,
      updated_at: timestamp,
    };
  });
  mockSections = [...sections, ...mockSections];
  mockEvents.unshift(mockEvent('info', 'mock sections detected'));
  return sections;
}

export async function UpdateSourceSection(input: {id: number; heading: string; section_type: string; content: string}) {
  if (hasWailsBackend()) {
    return WailsUpdateSourceSection(input);
  }
  const timestamp = now();
  mockSections = mockSections.map((section) => section.id === input.id ? {...section, ...input, updated_at: timestamp} : section);
  return mockSections.find((section) => section.id === input.id);
}

export async function DeleteSourceSection(input: {id: number}) {
  if (hasWailsBackend()) {
    return WailsDeleteSourceSection(input);
  }
  mockSections = mockSections.filter((section) => section.id !== input.id);
  mockFacts = mockFacts.filter((fact) => fact.section_id !== input.id);
  mockEvents.unshift(mockEvent('info', 'mock section deleted'));
}

export async function ExtractEvidenceFacts(input: {source_id: number; section_id: number}) {
  if (hasWailsBackend()) {
    return WailsExtractEvidenceFacts(input);
  }
  const section = mockSections.find((item) => item.id === input.section_id);
  if (!section) {
    return [];
  }
  const timestamp = now();
  const facts = fallbackFactsFromSection(section).map((item, index) => ({
    id: Date.now() + index,
    source_id: section.source_id,
    section_id: section.id,
    fact_text: item.fact_text,
    evidence_quote: item.evidence_quote,
    technologies: item.technologies,
    confidence: item.confidence,
    risk_flags: item.risk_flags,
    origin_heading: section.heading,
    origin_type: section.section_type,
    context: factContextAtoms(section),
    status: 'needs_review',
    auto_approved: false,
    review_note: '',
    created_at: timestamp,
    updated_at: timestamp,
  }));
  mockFacts = [...facts, ...mockFacts];
  mockEvents.unshift(mockEvent('info', 'mock evidence facts extracted'));
  return facts;
}

export async function ListEvidenceFacts(status: string) {
  if (hasWailsBackend()) {
    return WailsListEvidenceFacts(status);
  }
  return status && status !== 'all' ? mockFacts.filter((fact) => fact.status === status) : mockFacts;
}

export async function UpdateEvidenceFactReview(input: {
  id: number;
  fact_text: string;
  evidence_quote: string;
  technologies: string[];
  confidence: string;
  risk_flags: string[];
  status: string;
  review_note: string;
}) {
  if (hasWailsBackend()) {
    return WailsUpdateEvidenceFactReview(input);
  }
  const timestamp = now();
  mockFacts = mockFacts.map((fact) => fact.id === input.id ? {...fact, ...input, auto_approved: false, updated_at: timestamp} : fact);
  return mockFacts.find((fact) => fact.id === input.id);
}

export async function DeleteEvidenceFact(input: {id: number}) {
  if (hasWailsBackend()) {
    return WailsDeleteEvidenceFact(input);
  }
  mockFacts = mockFacts.filter((fact) => fact.id !== input.id);
  mockEvents.unshift(mockEvent('info', 'mock fact deleted'));
}

export async function DeleteAllEvidenceFacts() {
  if (hasWailsBackend()) {
    return WailsDeleteAllEvidenceFacts();
  }
  mockFacts = [];
  mockMatches = [];
  mockDrafts = [];
  mockFitAnalyses = [];
  mockStrategies = [];
  mockEvents.unshift(mockEvent('info', 'mock all evidence facts deleted'));
}

export async function ListJobDescriptions() {
  if (hasWailsBackend()) {
    return WailsListJobDescriptions();
  }
  return mockJobs;
}

export async function CreateJobDescription(input: {company: string; title: string; url: string; raw_text: string}) {
  if (hasWailsBackend()) {
    return WailsCreateJobDescription(input);
  }
  const timestamp = now();
  const details = inferJobDetailsFromText(input.raw_text);
  const job: JobDescription = {
    id: Date.now(),
    company: input.company.trim() || details.company,
    title: input.title.trim() || details.title || 'Untitled job',
    url: input.url.trim(),
    raw_text: input.raw_text.trim(),
    created_at: timestamp,
    updated_at: timestamp,
  };
  mockJobs = [job, ...mockJobs];
  mockEvents.unshift(mockEvent('info', 'mock job saved'));
  return job;
}

export async function UpdateJobDescription(input: {id: number; company: string; title: string; url: string; raw_text: string}) {
  if (hasWailsBackend()) {
    return WailsUpdateJobDescription(input);
  }
  const timestamp = now();
  const details = inferJobDetailsFromText(input.raw_text);
  mockJobs = mockJobs.map((job) => job.id === input.id ? {
    ...job,
    company: input.company.trim() || details.company,
    title: input.title.trim() || details.title || 'Untitled job',
    url: input.url.trim(),
    raw_text: input.raw_text.trim(),
    updated_at: timestamp,
  } : job);
  return mockJobs.find((job) => job.id === input.id);
}

export async function DeleteJobDescription(input: {id: number}) {
  if (hasWailsBackend()) {
    return WailsDeleteJobDescription(input);
  }
  mockJobs = mockJobs.filter((job) => job.id !== input.id);
  mockRequirements = mockRequirements.filter((req) => req.job_id !== input.id);
  mockMatches = mockMatches.filter((match) => match.job_id !== input.id);
  mockDrafts = mockDrafts.filter((draft) => draft.job_id !== input.id);
  mockEvents.unshift(mockEvent('info', 'mock job deleted'));
}

export async function ParseJobDescription(jobID: number) {
  if (hasWailsBackend()) {
    return WailsParseJobDescription(jobID);
  }
  const job = mockJobs.find((item) => item.id === jobID);
  if (!job) return [];
  const timestamp = now();
  const lines = job.raw_text
    .split(/\n|\.|;/)
    .map((line) => line.trim())
    .filter((line) => line.length > 10 && !isIrrelevantJobRequirementLine(line));
  const reqs = lines.slice(0, 16).map((line, index): JobRequirement => ({
    id: Date.now() + index,
    job_id: jobID,
    category: mockRequirementCategory(line),
    requirement_text: line.replace(/^[-•]\s*/, ''),
    keywords: extractKeywords(line),
    priority: index < 3 ? 'high' : 'medium',
    source_quote: line,
    created_at: timestamp,
    updated_at: timestamp,
  })).filter((req) => extractKeywords(req.requirement_text).length > 0);
  mockRequirements = [...reqs, ...mockRequirements.filter((req) => req.job_id !== jobID)];
  mockMatches = mockMatches.filter((match) => match.job_id !== jobID);
  mockDrafts = mockDrafts.filter((draft) => draft.job_id !== jobID);
  mockAnalyses = [buildMockJobAnalysis(job, reqs), ...mockAnalyses.filter((analysis) => analysis.job_id !== jobID)];
  mockEvents.unshift(mockEvent('info', 'mock job requirements parsed'));
  return reqs;
}

export async function ListJobRequirements(jobID: number) {
  if (hasWailsBackend()) {
    return WailsListJobRequirements(jobID);
  }
  return mockRequirements.filter((req) => req.job_id === jobID);
}

export async function BuildJobMatchMap(jobID: number) {
  if (hasWailsBackend()) {
    return WailsBuildJobMatchMap(jobID);
  }
  const timestamp = now();
  const requirements = mockRequirements.filter((req) => req.job_id === jobID);
  const matches: JobFactMatch[] = [];
  for (const req of requirements) {
    const reqTerms = new Set(req.keywords.map((word) => word.toLowerCase()));
    for (const fact of mockFacts) {
      const factText = `${fact.fact_text} ${fact.technologies.join(' ')}`.toLowerCase();
      const overlap = [...reqTerms].filter((term) => factText.includes(term));
      if (!overlap.length) continue;
      const score = Math.min(1, 0.45 + overlap.length * 0.18);
      if (score <= 0) continue;
      matches.push({
        id: Date.now() + matches.length,
        job_id: jobID,
        requirement_id: req.id,
        fact_id: fact.id,
        score,
        rationale: `Keyword overlap: ${overlap.join(', ')}`,
        coverage_status: overlap.length >= 2 ? 'strong' : 'partial',
        fact_status: fact.status,
        fact_text: fact.fact_text,
        evidence_quote: fact.evidence_quote,
        risk_flags: fact.risk_flags,
        created_at: timestamp,
        updated_at: timestamp,
      });
    }
  }
  mockMatches = [...matches, ...mockMatches.filter((match) => match.job_id !== jobID)];
  mockEvents.unshift(mockEvent('info', 'mock match map built'));
  return matches;
}

export async function ListJobFactMatches(jobID: number) {
  if (hasWailsBackend()) {
    return WailsListJobFactMatches(jobID);
  }
  return mockMatches.filter((match) => match.job_id === jobID);
}

export async function GenerateTailoredBulletDrafts(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGenerateTailoredBulletDrafts(jobID);
  }
  const timestamp = now();
  const grouped = new Map<number, JobFactMatch[]>();
  mockMatches.filter((match) => match.job_id === jobID).forEach((match) => {
    grouped.set(match.requirement_id, [...(grouped.get(match.requirement_id) ?? []), match]);
  });
  const drafts: TailoredBulletDraft[] = [...grouped.entries()].map(([requirementID, matches], index) => {
    const topMatches = matches.sort((a, b) => b.score - a.score).slice(0, 2);
    const risky = topMatches.flatMap((match) => match.fact_status === 'approved' ? match.risk_flags : [match.fact_status, ...match.risk_flags]);
    return {
      id: Date.now() + index,
      job_id: jobID,
      requirement_id: requirementID,
      fact_ids: topMatches.map((match) => match.fact_id),
      draft_text: `- ${topMatches[0]?.fact_text ?? 'Tailored bullet needs evidence.'}`,
      rationale: topMatches.map((match) => match.rationale).join('; '),
      status: 'needs_review',
      risk_flags: [...new Set(risky)].filter(Boolean),
      created_at: timestamp,
      updated_at: timestamp,
    };
  });
  mockDrafts = [...drafts, ...mockDrafts.filter((draft) => draft.job_id !== jobID)];
  mockEvents.unshift(mockEvent('info', 'mock tailored bullet drafts generated'));
  return drafts;
}

export async function ListTailoredBulletDrafts(jobID: number) {
  if (hasWailsBackend()) {
    return WailsListTailoredBulletDrafts(jobID);
  }
  return mockDrafts.filter((draft) => draft.job_id === jobID);
}

export async function UpdateTailoredBulletDraft(input: {id: number; draft_text: string; rationale: string; status: string; risk_flags: string[]}) {
  if (hasWailsBackend()) {
    return WailsUpdateTailoredBulletDraft(input);
  }
  const timestamp = now();
  mockDrafts = mockDrafts.map((draft) => draft.id === input.id ? {...draft, ...input, updated_at: timestamp} : draft);
  return mockDrafts.find((draft) => draft.id === input.id);
}

export async function DeleteTailoredBulletDraft(input: {id: number}) {
  if (hasWailsBackend()) {
    return WailsDeleteTailoredBulletDraft(input);
  }
  mockDrafts = mockDrafts.filter((draft) => draft.id !== input.id);
  mockEvents.unshift(mockEvent('info', 'mock bullet draft deleted'));
}

export async function ListPromptRules() {
  if (hasWailsBackend()) {
    return WailsListPromptRules();
  }
  return mockPromptRules;
}

export async function UpdatePromptRule(input: {id: number; content: string; enabled: boolean}) {
  if (hasWailsBackend()) {
    return WailsUpdatePromptRule(input);
  }
  const timestamp = now();
  mockPromptRules = mockPromptRules.map((rule) => rule.id === input.id ? {
    ...rule,
    content: input.content.trim(),
    enabled: input.enabled,
    version: rule.version + 1,
    updated_at: timestamp,
  } : rule);
  return mockPromptRules.find((rule) => rule.id === input.id);
}

export async function ListPromptResearchSources() {
  if (hasWailsBackend()) {
    return WailsListPromptResearchSources();
  }
  return mockPromptSources;
}

export async function AnalyzeJobDescription(jobID: number) {
  if (hasWailsBackend()) {
    return WailsAnalyzeJobDescription(jobID);
  }
  const job = mockJobs.find((item) => item.id === jobID);
  if (!job) throw new Error('job not found');
  const requirements = mockRequirements.filter((req) => req.job_id === jobID);
  const analysis = buildMockJobAnalysis(job, requirements);
  mockAnalyses = [analysis, ...mockAnalyses.filter((item) => item.job_id !== jobID)];
  return analysis;
}

export async function GetJobAnalysis(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGetJobAnalysis(jobID);
  }
  return mockAnalyses.find((analysis) => analysis.job_id === jobID);
}

export async function GenerateFitAnalysis(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGenerateFitAnalysis(jobID);
  }
  const fit = buildMockFitAnalysis(jobID);
  mockFitAnalyses = [fit, ...mockFitAnalyses.filter((item) => item.job_id !== jobID)];
  return fit;
}

export async function GetFitAnalysis(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGetFitAnalysis(jobID);
  }
  return mockFitAnalyses.find((fit) => fit.job_id === jobID);
}

export async function GenerateApplicationStrategy(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGenerateApplicationStrategy(jobID);
  }
  const job = mockJobs.find((item) => item.id === jobID);
  if (!job) throw new Error('job not found');
  const analysis = mockAnalyses.find((item) => item.job_id === jobID) ?? buildMockJobAnalysis(job, mockRequirements.filter((req) => req.job_id === jobID));
  const fit = mockFitAnalyses.find((item) => item.job_id === jobID) ?? buildMockFitAnalysis(jobID);
  const approved = mockMatches.filter((match) => match.job_id === jobID && match.fact_status === 'approved').map((match) => match.fact_id);
  const rejected = mockMatches.filter((match) => match.job_id === jobID && match.fact_status !== 'approved').map((match) => match.fact_id);
  const strategy: ApplicationStrategy = {
    job_id: jobID,
    approved_fact_ids: [...new Set(approved)],
    rejected_fact_ids: [...new Set(rejected)],
    weak_or_missing_requirements: fit.critical_gaps,
    resume_headline: `${analysis.role_archetype || 'Software engineer'} for ${job.title}`.trim(),
    experience_titles: {default: job.title},
    positioning_strategy: 'Lead with strongest approved evidence for the top JD pain points; avoid unsupported gaps.',
    keywords: analysis.keywords.slice(0, 12),
    do_not_overclaim: fit.critical_gaps,
    fit_summary: fit.reality_check,
    created_at: now(),
    updated_at: now(),
  };
  mockStrategies = [strategy, ...mockStrategies.filter((item) => item.job_id !== jobID)];
  return strategy;
}

export async function GetApplicationStrategy(jobID: number) {
  if (hasWailsBackend()) {
    return WailsGetApplicationStrategy(jobID);
  }
  return mockStrategies.find((strategy) => strategy.job_id === jobID);
}

function mockEvent(level: string, message: string) {
  return {
    id: Date.now(),
    level,
    message,
    created_at: now(),
  };
}

function modelDefault(provider: string) {
  return provider === 'openai' ? 'gpt-5.4-mini' : 'deepseek/deepseek-v4-flash';
}

function normalizeRawSourceText(rawText: string) {
  const text = rawText.trim();
  if (!looksLikeLatex(text)) {
    return text;
  }
  return cleanLatexText(text);
}

function looksLikeLatex(text: string) {
  return text.includes('\\documentclass') || text.includes('\\begin{document}') || text.includes('\\section{') || text.includes('\\resumeItem');
}

function cleanLatexText(rawText: string) {
  let text = rawText.replace(/\r\n/g, '\n');
  const docStart = text.indexOf('\\begin{document}');
  if (docStart >= 0) {
    text = text.slice(docStart + '\\begin{document}'.length);
  }
  const docEnd = text.indexOf('\\end{document}');
  if (docEnd >= 0) {
    text = text.slice(0, docEnd);
  }
  text = text
    .split('\n')
    .map(stripLatexComment)
    .map((line) => line.trim())
    .filter((line) => line !== '' && !line.startsWith('%'))
    .join('\n');

  text = expandLatexCommand(text, 'section', (args) => `\n${args[0]}\n`);
  text = expandLatexCommand(text, 'resumeSubheading', (args) => `\n${args[0]} | ${args[1]}\n${args[2]} | ${args[3]}\n`);
  text = expandLatexCommand(text, 'resumeProjectHeading', (args) => `\n${args[0]} | ${args[1]}\n`);
  text = expandLatexCommand(text, 'resumeItem', (args) => `- ${args[0]}\n`);
  text = expandLatexCommand(text, 'href', (args) => args[1] ?? args[0] ?? '');
  for (const command of ['textbf', 'textit', 'emph', 'small', 'scshape']) {
    text = expandLatexCommand(text, command, (args) => args.join(''));
  }

  text = text
    .replace(/\\begin\{[^}]+\}(\[[^\]]+\])?/g, '')
    .replace(/\\end\{[^}]+\}/g, '');
  const replacements: Record<string, string> = {
    '$|$': ' | ',
    '\\\\': '\n',
    '\\&': '&',
    '\\%': '%',
    '\\_': '_',
    '\\#': '#',
    '\\$': '$',
    '\\resumeSubHeadingListStart': '',
    '\\resumeSubHeadingListEnd': '',
    '\\resumeItemListStart': '',
    '\\resumeItemListEnd': '',
    '\\begin{center}': '',
    '\\end{center}': '',
    '\\begin{itemize}': '',
    '\\end{itemize}': '',
  };
  for (const [from, to] of Object.entries(replacements)) {
    text = text.split(from).join(to);
  }
  return ensureKnownSectionHeadingsOnLines(text
    .replace(/\\vspace\{[^}]*\}/g, '')
    .replace(/^[ \t]*\[[^\]\n]*(?:leftmargin|label)[^\]\n]*\][ \t]*$/gm, '')
    .replace(/\\[a-zA-Z]+(\[[^\]]+\])?/g, '')
    .replace(/[{}]/g, '')
    .replace(/[ \t]+/g, ' ')
    .replace(/^\s+/gm, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim());
}

function stripLatexComment(line: string) {
  for (let index = 0; index < line.length; index += 1) {
    if (line[index] === '%' && (index === 0 || line[index - 1] !== '\\')) {
      return line.slice(0, index);
    }
  }
  return line;
}

function expandLatexCommand(text: string, command: string, format: (args: string[]) => string) {
  let next = text;
  let searchStart = 0;
  while (true) {
    const index = next.indexOf(`\\${command}`, searchStart);
    if (index < 0) {
      return next;
    }
    const commandEnd = index + command.length + 1;
    if (/[A-Za-z]/.test(next[commandEnd] ?? '')) {
      searchStart = commandEnd;
      continue;
    }
    const parsed = readLatexCommandArgs(next, index + command.length + 1);
    if (!parsed || parsed.args.length === 0) {
      next = `${next.slice(0, index)}${next.slice(index + command.length + 1)}`;
      searchStart = index;
      continue;
    }
    next = `${next.slice(0, index)}${format(parsed.args)}${next.slice(parsed.end)}`;
    searchStart = index;
  }
}

function readLatexCommandArgs(text: string, startIndex: number): {args: string[]; end: number} | null {
  const args: string[] = [];
  let index = startIndex;
  while (index < text.length) {
    while (index < text.length && /\s/.test(text[index])) {
      index += 1;
    }
    if (text[index] !== '{') {
      break;
    }
    let depth = 0;
    const start = index + 1;
    while (index < text.length) {
      if (text[index] === '{') {
        depth += 1;
      } else if (text[index] === '}') {
        depth -= 1;
        if (depth === 0) {
          args.push(cleanLatexFragment(text.slice(start, index)));
          index += 1;
          break;
        }
      }
      index += 1;
    }
  }
  return args.length ? {args, end: index} : null;
}

function cleanLatexFragment(value: string) {
  let text = value;
  text = expandLatexCommand(text, 'href', (args) => args[1] ?? args[0] ?? '');
  for (const command of ['textbf', 'textit', 'emph', 'small']) {
    text = expandLatexCommand(text, command, (args) => args.join(''));
  }
  return text
    .replace(/\$\|\$/g, ' | ')
    .replace(/\\&/g, '&')
    .replace(/\\%/g, '%')
    .replace(/\\[a-zA-Z]+/g, '')
    .replace(/[{}]/g, '')
    .replace(/[ \t]+/g, ' ')
    .trim();
}

type MockSectionPart = {
  heading: string;
  section_type: string;
  content: string;
};

function splitSourceSections(text: string): MockSectionPart[] {
  const lines = ensureKnownSectionHeadingsOnLines(text).split('\n');
  const headings: number[] = [];
  lines.forEach((line, index) => {
    if (isKnownSectionHeading(line.trim()) || /^[A-Z][A-Z ]{3,}:?$/.test(line.trim()) || /^#{1,3}\s+/.test(line.trim())) {
      headings.push(index);
    }
  });
  if (!headings.length) {
    return [{heading: 'Imported source', section_type: 'misc', content: text.trim()}];
  }
  return headings.flatMap((start, index) => {
    const end = headings[index + 1] ?? lines.length;
    const heading = lines[start]?.replace(/^#+\s*/, '').replace(/:$/, '').trim() || 'Untitled section';
    const content = lines.slice(start + 1, end).join('\n').trim() || heading;
    return splitResumeEntryParts({
      heading,
      section_type: sectionTypeFromHeading(heading),
      content,
    });
  });
}

function splitResumeEntryParts(part: MockSectionPart): MockSectionPart[] {
  if (!['experience', 'project', 'education'].includes(part.section_type)) {
    return [part];
  }
  const lines = part.content.split('\n').map((line) => line.trim()).filter(Boolean);
  const parts: MockSectionPart[] = [];
  for (let index = 0; index < lines.length;) {
    while (index < lines.length && lines[index].startsWith('- ')) {
      index += 1;
    }
    if (index >= lines.length) {
      break;
    }
    const start = index;
    index += 1;
    if (part.section_type !== 'project' && index < lines.length && isEntryHeaderLine(lines[index])) {
      index += 1;
    }
    while (index < lines.length && !isResumeEntryStart(lines, index, part.section_type)) {
      index += 1;
    }
    const entryLines = lines.slice(start, index);
    if (entryLines.length) {
      parts.push({
        ...part,
        heading: resumeEntryHeading(part.section_type, entryLines, part.heading),
        content: entryLines.join('\n'),
      });
    }
  }
  return parts.length ? parts : [part];
}

function isResumeEntryStart(lines: string[], index: number, sectionType: string) {
  const line = lines[index]?.trim() ?? '';
  if (!line || line.startsWith('- ') || !isEntryHeaderLine(line)) {
    return false;
  }
  if (sectionType === 'project') {
    return true;
  }
  return index + 1 < lines.length && isEntryHeaderLine(lines[index + 1]);
}

function isEntryHeaderLine(line: string) {
  return line.trim() !== '' && !line.trim().startsWith('- ') && line.includes('|');
}

function resumeEntryHeading(sectionType: string, lines: string[], fallback: string) {
  const firstParts = splitPipeLine(lines[0] ?? '');
  const name = firstParts[0] || lines[0] || fallback;
  if (sectionType === 'project') {
    return name.trim();
  }
  const detailParts = splitPipeLine(lines[1] ?? '');
  const role = detailParts[0] ?? '';
  return role ? `${name.trim()} - ${role}` : name.trim();
}

function fallbackFactsFromSection(section: SourceSection) {
  const lines = section.content.split('\n').map((line) => line.trim()).filter(Boolean);
  const metadataCount = section.section_type === 'experience' || section.section_type === 'education' ? 2 : section.section_type === 'project' ? 1 : 0;
  return lines
    .filter((line, index) => index >= metadataCount && !isKnownSectionHeading(line) && line.replace(/^- /, '').trim().length >= 12)
    .flatMap((line) => atomicFactsFromLine(line));
}

function factContextAtoms(section: SourceSection) {
  const atoms = [
    keyValueFact('origin_heading', section.heading),
    keyValueFact('section_type', section.section_type),
    ...sectionMetadataContext(section),
  ];
  return [...new Set(atoms.filter(Boolean))];
}

function sectionMetadataContext(section: SourceSection) {
  const lines = section.content.split('\n').map((line) => line.trim()).filter(Boolean);
  const atoms: string[] = [];
  const firstParts = splitPipeLine(lines[0] ?? '');
  const secondParts = splitPipeLine(lines[1] ?? '');
  if (section.section_type === 'experience') {
    if (firstParts[0]) atoms.push(keyValueFact('organization', firstParts[0]));
    if (firstParts.length > 1) atoms.push(keyValueFact('location', firstParts.slice(1).join(' | ')));
    if (secondParts[0]) atoms.push(keyValueFact('role', secondParts[0]));
    if (secondParts.length > 1) atoms.push(keyValueFact('dates', secondParts[secondParts.length - 1]));
  } else if (section.section_type === 'project') {
    if (firstParts[0]) atoms.push(keyValueFact('project', firstParts[0]));
    if (firstParts.length > 1) atoms.push(keyValueFact('project_context', firstParts.slice(1).join(' | ')));
  } else if (section.section_type === 'education') {
    if (firstParts[0]) atoms.push(keyValueFact('organization', firstParts[0]));
    if (secondParts[0]) atoms.push(keyValueFact('credential', secondParts[0]));
    if (secondParts.length > 1) atoms.push(keyValueFact('dates', secondParts[secondParts.length - 1]));
  }
  return atoms;
}

function ensureKnownSectionHeadingsOnLines(text: string) {
  const headings = [
    'Professional Summary',
    'Technical Skills',
    'Work Experience',
    'Other Projects',
    'Experience',
    'Projects',
    'Education',
    'Certifications',
  ];
  const lines: string[] = [];
  for (const line of text.split('\n')) {
    let remaining = line.trim();
    if (!remaining) {
      lines.push('');
      continue;
    }
    while (remaining) {
      let matchedHeading = '';
      let matchedIndex = -1;
      for (const heading of headings) {
        const index = remaining.indexOf(heading);
        if (index < 0 || !headingBoundary(remaining, index, heading)) {
          continue;
        }
        if (matchedIndex < 0 || index < matchedIndex || (index === matchedIndex && heading.length > matchedHeading.length)) {
          matchedHeading = heading;
          matchedIndex = index;
        }
      }
      if (matchedIndex < 0) {
        lines.push(remaining);
        break;
      }
      const prefix = remaining.slice(0, matchedIndex).trim();
      if (prefix) {
        lines.push(prefix);
      }
      lines.push(matchedHeading);
      remaining = remaining.slice(matchedIndex + matchedHeading.length).trim();
    }
  }
  return lines.join('\n').replace(/\n{3,}/g, '\n\n').trim();
}

function headingBoundary(text: string, index: number, heading: string) {
  const after = text[index + heading.length] ?? '';
  return after === '' || after === ':' || after === '.' || after === ' ' || after === '-' || /[A-Z0-9]/.test(after);
}

function isKnownSectionHeading(value: string) {
  return ['professional summary', 'summary', 'technical skills', 'skills', 'experience', 'projects', 'education'].includes(value.trim().toLowerCase());
}

function sectionTypeFromHeading(heading: string) {
  const lower = heading.toLowerCase();
  if (lower.includes('summary')) return 'summary';
  if (lower.includes('skill')) return 'skills';
  if (lower.includes('experience')) return 'experience';
  if (lower.includes('project')) return 'project';
  if (lower.includes('education')) return 'education';
  return 'misc';
}

function draftRecordsFromText(text: string, heading: string, recordType: string): CandidateProfileRecord[] {
  const section = extractNamedSection(text, heading);
  if (!section) {
    return [];
  }
  const lines = section.split('\n').map((line) => line.trim()).filter(Boolean);
  const records: CandidateProfileRecord[] = [];
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (line.startsWith('- ') || isKnownSectionHeading(line)) {
      continue;
    }
    if (!line.includes('|') && recordType !== 'education') {
      continue;
    }
    const parts = splitPipeLine(line);
    const record: CandidateProfileRecord = {
      id: 0,
      record_type: recordType,
      label: parts[0] ?? '',
      organization: parts[0] ?? '',
      role: '',
      start_date: '',
      end_date: '',
      value: parts.slice(1).join(' | '),
      verified: false,
      created_at: '',
      updated_at: '',
    };
    if (index + 1 < lines.length && lines[index + 1].includes('|') && !lines[index + 1].startsWith('- ')) {
      const detailParts = splitPipeLine(lines[index + 1]);
      record.role = detailParts[0] ?? '';
      const [start, end] = splitDateRange(detailParts[detailParts.length - 1] ?? '');
      record.start_date = start;
      record.end_date = end;
      index += 1;
    }
    if (recordType === 'education' && !record.role && parts.length > 1) {
      record.role = parts[1];
    }
    if (record.organization || record.role || record.value) {
      records.push(record);
    }
  }
  return records;
}

function extractNamedSection(text: string, heading: string) {
  const lines = text.split('\n');
  const start = lines.findIndex((line) => line.trim().toLowerCase() === heading.toLowerCase());
  if (start < 0) {
    return '';
  }
  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    if (isKnownSectionHeading(lines[index])) {
      end = index;
      break;
    }
  }
  return lines.slice(start + 1, end).join('\n');
}

function splitPipeLine(line: string) {
  return line.split('|').map((part) => part.trim()).filter(Boolean);
}

function splitDateRange(value: string): [string, string] {
  const parts = value.includes('--') ? value.split('--') : value.split('-');
  if (parts.length >= 2) {
    return [parts[0].trim(), parts.slice(1).join('-').trim()];
  }
  return ['', value.trim()];
}

function compactEvidenceFact(value: string) {
  return atomicFactsFromLine(value)[0]?.fact_text ?? compactCoreEvidenceFact(value);
}

function atomicFactsFromLine(value: string) {
  const text = value.replace(/^- /, '').replace(/\.$/, '').trim();
  const technologies = extractTechnologies(text);
  const riskFlags = inferRiskFlags(text);
  const confidence = inferConfidence(text, technologies, riskFlags);
  const base = {
    evidence_quote: value,
    technologies,
    confidence,
    risk_flags: riskFlags,
  };
  const facts = [
    compactCoreEvidenceFact(text),
  ];
  const scope = inferScope(text);
  if (scope) facts.push(keyValueFact('scope', scope, 'tools', technologies.join(', ')));
  const environment = inferEnvironment(text);
  if (environment) facts.push(keyValueFact('environment', environment, 'tools', technologies.join(', ')));
  const figures = extractFigures(text);
  const outcome = inferOutcome(text);
  if (figures.length || outcome) facts.push(keyValueFact('metric', figures.join(', '), 'outcome', outcome));
  return [...new Set(facts.filter(Boolean))].map((factText) => ({
    ...base,
    fact_text: factText,
  }));
}

function compactCoreEvidenceFact(value: string) {
  const text = value.replace(/^- /, '').replace(/\.$/, '').trim();
  const technologies = extractTechnologies(text);
  return keyValueFact(
    'actions', extractActions(text).join(', '),
    'artifact', inferArtifact(text, technologies),
    'tools', technologies.join(', '),
  ) || `evidence=${text}`;
}

function inferAction(text: string) {
  return extractActions(text)[0] ?? '';
}

function extractActions(text: string) {
  const lower = text.toLowerCase();
  return [...new Set(['built', 'shipped', 'added', 'implemented', 'designed', 'developed', 'created', 'improved', 'reduced', 'migrated', 'automated', 'supported', 'tested', 'integrated', 'delivered', 'optimized', 'deployed']
    .filter((action) => lower.includes(`${action} `) || lower.startsWith(action)))];
}

function inferArtifact(text: string, technologies: string[]) {
  let cleaned = text.replace(/^- /, '').replace(/\.$/, '').trim();
  const actions = extractActions(cleaned);
  if (actions.length) {
    const index = cleaned.toLowerCase().indexOf(actions[0]);
    if (index >= 0) cleaned = cleaned.slice(index + actions[0].length).trim();
  }
  while (/^(and|shipped|built)\s+/i.test(cleaned)) {
    cleaned = cleaned.replace(/^(and|shipped|built)\s+/i, '').trim();
  }
  for (const stop of [' for ', ' across ', ' against ', ' using ', ' with ', ' by ', ' through ', ' to ', ', reducing ', ', improving ']) {
    const index = cleaned.toLowerCase().indexOf(stop);
    if (index > 0) cleaned = cleaned.slice(0, index);
  }
  for (const tech of technologies) {
    cleaned = cleaned.replace(new RegExp(`\\b${escapeRegExp(tech)}\\b`, 'ig'), '');
  }
  return cleaned.replace(/[ /]+/g, ' ').trim();
}

function inferScope(text: string) {
  const cleaned = text.replace(/^- /, '').replace(/\.$/, '').trim();
  const lower = cleaned.toLowerCase();
  for (const marker of [' for ', ' across ', ' covering ', ' coverage for ']) {
    const index = lower.indexOf(marker);
    if (index < 0) continue;
    let scope = cleaned.slice(index + marker.length);
    for (const stop of [' against ', ' using ', ' with ', ' by ', ' through ', ' to ']) {
      const stopIndex = scope.toLowerCase().indexOf(stop);
      if (stopIndex > 0) scope = scope.slice(0, stopIndex);
    }
    return scope.trim();
  }
  return '';
}

function inferEnvironment(text: string) {
  const cleaned = text.replace(/^- /, '').replace(/\.$/, '').trim();
  const lower = cleaned.toLowerCase();
  for (const marker of [' against ', ' in ', ' on ']) {
    const index = lower.indexOf(marker);
    if (index < 0) continue;
    const candidate = cleaned.slice(index + marker.length).trim();
    const candidateLower = candidate.toLowerCase();
    if (candidateLower.includes('workflow') || candidateLower.includes('production') || candidateLower.includes('staging') || candidateLower.includes('linux') || candidateLower.includes('internal system')) {
      return candidate;
    }
  }
  return '';
}

function inferOutcome(text: string) {
  const cleaned = text.replace(/^- /, '').replace(/\.$/, '').trim();
  const lower = cleaned.toLowerCase();
  for (const marker of [' reducing ', ' reduced ', ' improving ', ' improved ', ' delivering ', ' delivered ', ' enabling ', ' enabled ']) {
    const index = lower.indexOf(marker);
    if (index >= 0) return cleaned.slice(index + 1).trim();
  }
  if (lower.includes('coverage')) return 'test coverage';
  if (lower.includes('production')) return 'production delivery';
  return '';
}

function extractFigures(text: string) {
  return [...new Set(text.match(/\b\d+(?:\.\d+)?\s*(?:%|percent|ms|s|sec|seconds|min|minutes|x|k|m|hours?|days?|weeks?|months?|years?)\b/gi) ?? [])];
}

function extractTechnologies(text: string) {
  const known = ['FastAPI', 'PostgreSQL', 'React', 'TypeScript', 'JavaScript', 'Python', 'Go', 'Golang', 'Java', 'C#', 'C++', 'Node.js', 'Node', 'Express', 'Linux', 'Docker', 'Kubernetes', 'AWS', 'Azure', 'GCP', 'Terraform', 'Postgres', 'SQLite', 'MySQL', 'Redis', 'Locust', 'Playwright', 'Tailwind', 'Vite', 'Wails', 'GitHub Actions', 'CI/CD', 'REST', 'GraphQL', 'RBAC', 'SQL', 'NoSQL', 'MongoDB', 'DynamoDB'];
  const lower = text.toLowerCase();
  return [...new Set(known
    .filter((tech) => lower.includes(tech.toLowerCase()))
    .map((tech) => tech === 'Golang' ? 'Go' : tech === 'Postgres' ? 'PostgreSQL' : tech))];
}

function inferRiskFlags(text: string) {
  const lower = text.toLowerCase();
  const flags: string[] = [];
  if (lower.includes('approximately') || lower.includes('around ') || lower.includes('~')) flags.push('unclear_metric');
  if (lower.includes('supported') && !lower.includes('built') && !lower.includes('implemented')) flags.push('unclear_ownership');
  if (lower.includes('staging-style') || lower.includes('prototype')) flags.push('production_vs_project_ambiguity');
  return [...new Set(flags)];
}

function inferConfidence(text: string, technologies: string[], riskFlags: string[]) {
  const lower = text.toLowerCase();
  if (riskFlags.length) return 'medium';
  if (technologies.length || extractFigures(text).length) return 'high';
  if (lower.includes('built') || lower.includes('shipped') || lower.includes('implemented')) return 'high';
  return 'medium';
}

function keyValueFact(...parts: string[]) {
  const pairs: string[] = [];
  for (let index = 0; index + 1 < parts.length; index += 2) {
    const key = parts[index].trim();
    const value = parts[index + 1].replace(/\.$/, '').trim();
    if (key && value) pairs.push(`${key}=${value}`);
  }
  return pairs.join('; ');
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function buildMockJobAnalysis(job: JobDescription, requirements: JobRequirement[]): JobAnalysis {
  const required = requirements.filter((req) => req.priority === 'high' || req.category === 'must_have');
  const preferred = requirements.filter((req) => req.category === 'nice_to_have');
  const responsibilities = requirements.filter((req) => req.category === 'responsibility');
  const keywords = [...new Set(requirements.flatMap((req) => req.keywords))];
  return {
    job_id: job.id,
    company: job.company,
    role_title: job.title,
    location: inferLocationFromJD(job.raw_text),
    work_arrangement: inferArrangementFromJD(job.raw_text),
    salary: '',
    top_pain_points: [...responsibilities, ...required].map((req) => req.requirement_text).slice(0, 3),
    required_skills: [...new Set(required.flatMap((req) => req.keywords))],
    preferred_skills: [...new Set(preferred.flatMap((req) => req.keywords))],
    responsibilities: responsibilities.map((req) => req.requirement_text),
    seniority_level: requirements.some((req) => /senior|lead|mentor/i.test(req.requirement_text)) ? 'senior' : 'mid',
    role_archetype: inferRoleArchetypeFromJD(job.title, keywords),
    keywords,
    risk_flags: requirements.some((req) => req.category === 'domain') ? ['domain_requirement'] : [],
    job_poster: '',
    company_url: '',
    created_at: now(),
    updated_at: now(),
  };
}

function buildMockFitAnalysis(jobID: number): JobFitAnalysis {
  const requirements = mockRequirements.filter((req) => req.job_id === jobID);
  const matches = mockMatches.filter((match) => match.job_id === jobID);
  const rows: FitNeedAnalysis[] = requirements.map((req) => {
    const reqMatches = matches.filter((match) => match.requirement_id === req.id);
    const best = reqMatches.sort((a, b) => b.score - a.score)[0];
    const gap = !best ? 'critical' : best.score >= 0.75 ? 'covered' : best.score >= 0.45 ? 'partial' : 'critical';
    return {
      requirement_id: req.id,
      jd_need: req.requirement_text,
      matching_fact_ids: reqMatches.map((match) => match.fact_id),
      evidence_strength: best?.coverage_status ?? 'gap',
      gap_level: gap,
      confidence: best?.coverage_status === 'strong' ? 'high' : best ? 'medium' : 'low',
      risk: best ? '' : 'no matching evidence',
    };
  });
  const score = rows.length ? Math.round(rows.reduce((sum, row) => sum + (row.gap_level === 'covered' ? 1 : row.gap_level === 'partial' ? 0.55 : 0), 0) / rows.length * 100) : 0;
  return {
    job_id: jobID,
    overall_score: score,
    recommendation: score >= 70 ? 'Apply' : score >= 55 ? 'Apply With Caution' : score >= 40 ? 'Upskill First' : 'Look Elsewhere',
    strengths: rows.filter((row) => row.gap_level === 'covered').map((row) => row.jd_need),
    critical_gaps: rows.filter((row) => row.gap_level === 'critical').map((row) => row.jd_need),
    reality_check: `${score}% evidence-backed fit based on ${requirements.length} parsed requirements.`,
    analysis: rows,
    created_at: now(),
    updated_at: now(),
  };
}

function inferLocationFromJD(rawText: string) {
  return rawText.split('\n').find((line) => /melbourne|sydney|australia/i.test(line))?.trim() ?? '';
}

function inferArrangementFromJD(rawText: string) {
  const lower = rawText.toLowerCase();
  if (lower.includes('remote')) return 'remote';
  if (lower.includes('hybrid')) return 'hybrid';
  if (lower.includes('onsite') || lower.includes('on-site')) return 'onsite';
  return '';
}

function inferRoleArchetypeFromJD(title: string, keywords: string[]) {
  const joined = `${title} ${keywords.join(' ')}`.toLowerCase();
  if (joined.includes('cloud') || joined.includes('azure') || joined.includes('aws')) return 'cloud software engineer';
  if (joined.includes('backend') || joined.includes('api')) return 'backend engineer';
  if (joined.includes('react') || joined.includes('full stack')) return 'full stack engineer';
  return 'software engineer';
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
      if (lower.startsWith(prefix)) company = line.slice(prefix.length).trim();
    }
    for (const prefix of ['job title:', 'role:', 'title:', 'position:']) {
      if (lower.startsWith(prefix)) title = line.slice(prefix.length).trim();
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
  if (line.split(/\s+/).length > 9) return false;
  return /engineer|developer|manager|analyst|designer|architect|consultant|specialist|lead|intern|graduate|backend|frontend|full stack|software|data|devops|platform/i.test(line);
}

function isIrrelevantJobRequirementLine(line: string) {
  const lower = line.toLowerCase();
  const markers = [
    '12-month', '12 month', 'contract', 'max term', 'fixed term', 'salary', 'compensation', 'benefit', 'leave', 'hybrid', 'remote', 'location', 'office',
    'how to apply', 'application process', 'submit your application', 'recruit', 'hiring', 'interview', 'equal opportunity', 'diversity', 'background check', 'sponsorship',
    'leading personal injury', 'class actions law firm', 'about us', 'about the company', 'company is', 'we are a', "we're a", 'our client',
    'logo', 'linkedin', 'promoted by', 'responses managed', 'profile matches', 'is this information helpful', 'personalized tips', 'top applicant', 'retry premium', 'people you can reach out', 'school alumni', 'clicked apply',
  ];
  if (markers.some((marker) => lower.includes(marker))) return true;
  if (isJobHeadingOrMetadata(line)) return true;
  if (!hasTailorableRequirementSignal(line)) return true;
  if (looksLikeRoleTitle(line) && !lower.includes('develop') && !lower.includes('build') && !lower.includes('experience')) return true;
  return false;
}

function isJobHeadingOrMetadata(line: string) {
  const cleaned = line.trim().toLowerCase().replace(/[.:?!]+$/g, '');
  const exact = new Set(['about', 'about the job', 'what are we looking for', 'responsibilities', 'requirements', 'key responsibilities', 'software engineer', 'slater and gordon lawyers', 'slater and gordon lawyers logo']);
  if (exact.has(cleaned)) return true;
  if (cleaned.includes('·') || cleaned.includes(' clicked apply') || cleaned.endsWith(' logo')) return true;
  return cleaned.split(/\s+/).length <= 4 && looksLikeRoleTitle(cleaned);
}

function hasTailorableRequirementSignal(line: string) {
  const lower = line.toLowerCase();
  return [
    'experience', 'hands-on', 'strong', 'deep', 'knowledge', 'understanding', 'programming', 'skills', 'design', 'build', 'develop', 'deliver', 'modernis', 'test', 'support',
    'architecture', 'cloud', 'serverless', 'event-driven', 'distributed', 'scalable', 'resilience', 'observability', 'security', 'networking', 'identity', 'database', 'nosql',
    'devsecops', 'agile', 'solid', 'containers', 'messaging', 'queues', 'topics', 'stakeholder', 'mentor', 'engineering practices', 'technical excellence',
    'fastapi', 'postgresql', 'react', 'typescript', 'javascript', 'python', 'golang', 'java', 'spring', 'mysql', 'node.js', 'node', 'azure', 'cosmos db', 'aws', 'gcp', 'docker', 'kubernetes', 'terraform', 'redis',
  ].some((signal) => lower.includes(signal));
}

function mockRequirementCategory(line: string) {
  const lower = line.toLowerCase();
  if (lower.includes('must') || lower.includes('required') || lower.includes('experience with')) return 'must_have';
  if (lower.includes('nice') || lower.includes('preferred') || lower.includes('bonus')) return 'nice_to_have';
  if (lower.includes('senior') || lower.includes('years') || lower.includes('lead')) return 'seniority';
  if (lower.includes('industry') || lower.includes('domain')) return 'domain';
  return 'responsibility';
}

function extractKeywords(text: string) {
  const stop = new Set(['with', 'and', 'the', 'for', 'you', 'will', 'must', 'have', 'need', 'needs', 'role', 'work', 'build', 'using']);
  return [...new Set(
    text
      .replace(/[^A-Za-z0-9+#. ]/g, ' ')
      .split(/\s+/)
      .map((word) => word.trim())
      .filter((word) => word.length > 2 && !stop.has(word.toLowerCase()))
      .slice(0, 8),
  )];
}

function defaultMockPromptRules(): PromptRule[] {
  const timestamp = now();
  return [
    ['truth.no_fabrication', 'validation', 'No fabrication', 'Use only sourced facts. Never invent tools, metrics, titles, leadership, production scope, cloud platforms, or business impact.'],
    ['jd.top_pain_points', 'jd_parse', 'Top pain points first', 'Prioritize the top 3 employer pain points over exhaustive requirement lists.'],
    ['resume.human_style', 'resume', 'Human technical style', 'Write concise, specific engineering language. Avoid fake enthusiasm, generic praise, and AI-sounding filler.'],
    ['fit.blunt_reality', 'fit_analysis', 'Blunt fit analysis', 'Act like a brutally honest hiring fit analyzer. Output an evidence-backed percentage, strengths alignment, critical gaps, reality check, and a directive recommendation.'],
    ['fit.competitive_market', 'fit_analysis', 'Competitive market calibration', 'Assume competitive roles receive many qualified applicants. A partial transferable match is not enough for Apply unless several high-priority needs are directly supported by approved evidence.'],
  ].map(([ruleKey, category, title, content], index) => ({
    id: index + 1,
    rule_key: ruleKey,
    category,
    title,
    content,
    enabled: true,
    version: 1,
    source: 'mock defaults',
    created_at: timestamp,
    updated_at: timestamp,
  }));
}

function defaultMockPromptSources(): PromptResearchSource[] {
  const timestamp = now();
  return [
    {
      id: 1,
      source_type: 'official_docs',
      trust_tier: 'official',
      title: 'OpenAI prompt engineering',
      url: 'https://developers.openai.com/api/docs/guides/prompt-engineering',
      extracted_pattern: 'Clear task instructions, delimiters, relevant context, and evals.',
      app_adaptation: 'PromptBuilder uses task/schema/rules/tagged input sections.',
      accessed_at: '2026-06-09',
      created_at: timestamp,
    },
    {
      id: 2,
      source_type: 'community',
      trust_tier: 'forum',
      title: 'Reddit resume tailoring cautions',
      url: 'https://www.reddit.com/r/resumes/comments/1lnsrap/chatgpt_for_resume_tailoring/',
      extracted_pattern: 'AI resume tailoring needs manual review and anti-fabrication checks.',
      app_adaptation: 'Unsupported JD terms become do-not-overclaim strategy items.',
      accessed_at: '2026-06-09',
      created_at: timestamp,
    },
  ];
}
