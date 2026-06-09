import {
  BuildJobMatchMap as WailsBuildJobMatchMap,
  CreateCandidateSource as WailsCreateCandidateSource,
  CreateJobDescription as WailsCreateJobDescription,
  DeleteCandidateSource as WailsDeleteCandidateSource,
  DeleteEvidenceFact as WailsDeleteEvidenceFact,
  DeleteJobDescription as WailsDeleteJobDescription,
  DeleteSourceSection as WailsDeleteSourceSection,
  DeleteTailoredBulletDraft as WailsDeleteTailoredBulletDraft,
  DetectSourceSections as WailsDetectSourceSections,
  DraftCandidateProfileFromSource as WailsDraftCandidateProfileFromSource,
  ExtractEvidenceFacts as WailsExtractEvidenceFacts,
  GenerateTailoredBulletDrafts as WailsGenerateTailoredBulletDrafts,
  GetCandidateProfile as WailsGetCandidateProfile,
  GetHealth as WailsGetHealth,
  GetRecentEvents as WailsGetRecentEvents,
  GetSettings as WailsGetSettings,
  GetToolStatus as WailsGetToolStatus,
  ImportCandidateSourceFile as WailsImportCandidateSourceFile,
  InstallTectonic as WailsInstallTectonic,
  ListJobDescriptions as WailsListJobDescriptions,
  ListJobFactMatches as WailsListJobFactMatches,
  ListJobRequirements as WailsListJobRequirements,
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
  const job: JobDescription = {
    id: Date.now(),
    company: input.company.trim(),
    title: input.title.trim() || 'Untitled job',
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
  mockJobs = mockJobs.map((job) => job.id === input.id ? {
    ...job,
    company: input.company.trim(),
    title: input.title.trim() || 'Untitled job',
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
  const lines = job.raw_text.split(/\n|\.|;/).map((line) => line.trim()).filter((line) => line.length > 10);
  const reqs = lines.slice(0, 12).map((line, index): JobRequirement => ({
    id: Date.now() + index,
    job_id: jobID,
    category: mockRequirementCategory(line),
    requirement_text: line.replace(/^[-•]\s*/, ''),
    keywords: extractKeywords(line),
    priority: index < 3 ? 'high' : 'medium',
    source_quote: line,
    created_at: timestamp,
    updated_at: timestamp,
  }));
  mockRequirements = [...reqs, ...mockRequirements.filter((req) => req.job_id !== jobID)];
  mockMatches = mockMatches.filter((match) => match.job_id !== jobID);
  mockDrafts = mockDrafts.filter((draft) => draft.job_id !== jobID);
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
      matches.push({
        id: Date.now() + matches.length,
        job_id: jobID,
        requirement_id: req.id,
        fact_id: fact.id,
        score: Math.min(1, 0.45 + overlap.length * 0.18),
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
