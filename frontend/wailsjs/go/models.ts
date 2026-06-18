export namespace main {
	
	export class AppEvent {
	    id: number;
	    level: string;
	    message: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new AppEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.level = source["level"];
	        this.message = source["message"];
	        this.created_at = source["created_at"];
	    }
	}
	export class Application {
	    id: number;
	    job_id: number;
	    status: string;
	    fit_score: number;
	    resume_version_id: number;
	    cover_letter_version_id: number;
	    notes: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Application(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.status = source["status"];
	        this.fit_score = source["fit_score"];
	        this.resume_version_id = source["resume_version_id"];
	        this.cover_letter_version_id = source["cover_letter_version_id"];
	        this.notes = source["notes"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ApplicationStrategy {
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
	
	    static createFrom(source: any = {}) {
	        return new ApplicationStrategy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.approved_fact_ids = source["approved_fact_ids"];
	        this.rejected_fact_ids = source["rejected_fact_ids"];
	        this.weak_or_missing_requirements = source["weak_or_missing_requirements"];
	        this.resume_headline = source["resume_headline"];
	        this.experience_titles = source["experience_titles"];
	        this.positioning_strategy = source["positioning_strategy"];
	        this.keywords = source["keywords"];
	        this.do_not_overclaim = source["do_not_overclaim"];
	        this.fit_summary = source["fit_summary"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class BlockedClaim {
	    id: number;
	    pattern: string;
	    reason: string;
	    severity: string;
	    source: string;
	    enabled: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new BlockedClaim(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pattern = source["pattern"];
	        this.reason = source["reason"];
	        this.severity = source["severity"];
	        this.source = source["source"];
	        this.enabled = source["enabled"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class BulletGenerationEvent {
	    id: number;
	    job_id: number;
	    origin_heading: string;
	    stage: string;
	    status: string;
	    reason: string;
	    draft_text: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new BulletGenerationEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.origin_heading = source["origin_heading"];
	        this.stage = source["stage"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.draft_text = source["draft_text"];
	        this.created_at = source["created_at"];
	    }
	}
	export class CandidateClaim {
	    id: number;
	    claim_text: string;
	    claim_type: string;
	    source_fact_ids: number[];
	    evidence_quotes: string[];
	    technologies: string[];
	    actions: string[];
	    capabilities: string[];
	    objects: string[];
	    domains: string[];
	    artifacts: string[];
	    scope: string[];
	    metrics: string[];
	    outcomes: string[];
	    profile_context: string[];
	    evidence_strength: string;
	    strength: string;
	    allowed_use: string[];
	    allowed_contexts: string[];
	    blocked_contexts: string[];
	    safe_phrasings: string[];
	    unsafe_phrasings: string[];
	    origin_heading: string;
	    origin_type: string;
	    status: string;
	    risk_flags: string[];
	    similarity_key: string;
	    similarity_score: number;
	    duplicate_of_id: number;
	    review_note: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new CandidateClaim(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.claim_text = source["claim_text"];
	        this.claim_type = source["claim_type"];
	        this.source_fact_ids = source["source_fact_ids"];
	        this.evidence_quotes = source["evidence_quotes"];
	        this.technologies = source["technologies"];
	        this.actions = source["actions"];
	        this.capabilities = source["capabilities"];
	        this.objects = source["objects"];
	        this.domains = source["domains"];
	        this.artifacts = source["artifacts"];
	        this.scope = source["scope"];
	        this.metrics = source["metrics"];
	        this.outcomes = source["outcomes"];
	        this.profile_context = source["profile_context"];
	        this.evidence_strength = source["evidence_strength"];
	        this.strength = source["strength"];
	        this.allowed_use = source["allowed_use"];
	        this.allowed_contexts = source["allowed_contexts"];
	        this.blocked_contexts = source["blocked_contexts"];
	        this.safe_phrasings = source["safe_phrasings"];
	        this.unsafe_phrasings = source["unsafe_phrasings"];
	        this.origin_heading = source["origin_heading"];
	        this.origin_type = source["origin_type"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	        this.similarity_key = source["similarity_key"];
	        this.similarity_score = source["similarity_score"];
	        this.duplicate_of_id = source["duplicate_of_id"];
	        this.review_note = source["review_note"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CandidateContact {
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
	
	    static createFrom(source: any = {}) {
	        return new CandidateContact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.full_name = source["full_name"];
	        this.email = source["email"];
	        this.phone = source["phone"];
	        this.location = source["location"];
	        this.linkedin = source["linkedin"];
	        this.github = source["github"];
	        this.portfolio = source["portfolio"];
	        this.links = source["links"];
	        this.verified = source["verified"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CandidateProfileRecord {
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
	
	    static createFrom(source: any = {}) {
	        return new CandidateProfileRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.record_type = source["record_type"];
	        this.label = source["label"];
	        this.organization = source["organization"];
	        this.role = source["role"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.value = source["value"];
	        this.verified = source["verified"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CandidateProfile {
	    contact: CandidateContact;
	    records: CandidateProfileRecord[];
	
	    static createFrom(source: any = {}) {
	        return new CandidateProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contact = this.convertValues(source["contact"], CandidateContact);
	        this.records = this.convertValues(source["records"], CandidateProfileRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class CandidateSource {
	    id: number;
	    source_type: string;
	    trust_tier: string;
	    title: string;
	    raw_text: string;
	    file_path: string;
	    imported_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new CandidateSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_type = source["source_type"];
	        this.trust_tier = source["trust_tier"];
	        this.title = source["title"];
	        this.raw_text = source["raw_text"];
	        this.file_path = source["file_path"];
	        this.imported_at = source["imported_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ContextAgentRun {
	    id: number;
	    source_id: number;
	    status: string;
	    started_at: string;
	    finished_at: string;
	    error: string;
	    facts_created: number;
	    claims_created: number;
	
	    static createFrom(source: any = {}) {
	        return new ContextAgentRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_id = source["source_id"];
	        this.status = source["status"];
	        this.started_at = source["started_at"];
	        this.finished_at = source["finished_at"];
	        this.error = source["error"];
	        this.facts_created = source["facts_created"];
	        this.claims_created = source["claims_created"];
	    }
	}
	export class ContextAgentStep {
	    id: number;
	    run_id: number;
	    stage: string;
	    status: string;
	    message: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ContextAgentStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.run_id = source["run_id"];
	        this.stage = source["stage"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.created_at = source["created_at"];
	    }
	}
	export class CorrectionLog {
	    id: number;
	    application_id: number;
	    resume_version_id: number;
	    original_bullet_text: string;
	    corrected_bullet_text: string;
	    claim_ids: number[];
	    reason: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new CorrectionLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.application_id = source["application_id"];
	        this.resume_version_id = source["resume_version_id"];
	        this.original_bullet_text = source["original_bullet_text"];
	        this.corrected_bullet_text = source["corrected_bullet_text"];
	        this.claim_ids = source["claim_ids"];
	        this.reason = source["reason"];
	        this.created_at = source["created_at"];
	    }
	}
	export class CreateBlockedClaimInput {
	    pattern: string;
	    reason: string;
	    severity: string;
	    source: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateBlockedClaimInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.reason = source["reason"];
	        this.severity = source["severity"];
	        this.source = source["source"];
	        this.enabled = source["enabled"];
	    }
	}
	export class CreateCandidateSourceInput {
	    source_type: string;
	    trust_tier: string;
	    title: string;
	    raw_text: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateCandidateSourceInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.trust_tier = source["trust_tier"];
	        this.title = source["title"];
	        this.raw_text = source["raw_text"];
	    }
	}
	export class CreateJobDescriptionInput {
	    company: string;
	    title: string;
	    url: string;
	    raw_text: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateJobDescriptionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.company = source["company"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.raw_text = source["raw_text"];
	    }
	}
	export class DeleteInput {
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class EvidenceFact {
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
	    similarity_key: string;
	    similarity_score: number;
	    duplicate_of_id: number;
	    review_note: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceFact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_id = source["source_id"];
	        this.section_id = source["section_id"];
	        this.fact_text = source["fact_text"];
	        this.evidence_quote = source["evidence_quote"];
	        this.technologies = source["technologies"];
	        this.confidence = source["confidence"];
	        this.risk_flags = source["risk_flags"];
	        this.origin_heading = source["origin_heading"];
	        this.origin_type = source["origin_type"];
	        this.context = source["context"];
	        this.status = source["status"];
	        this.auto_approved = source["auto_approved"];
	        this.similarity_key = source["similarity_key"];
	        this.similarity_score = source["similarity_score"];
	        this.duplicate_of_id = source["duplicate_of_id"];
	        this.review_note = source["review_note"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ExtractEvidenceFactsInput {
	    source_id: number;
	    section_id: number;
	
	    static createFrom(source: any = {}) {
	        return new ExtractEvidenceFactsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_id = source["source_id"];
	        this.section_id = source["section_id"];
	    }
	}
	export class FactualityCheck {
	    bullet_index: number;
	    bullet: string;
	    has_claims: boolean;
	    all_approved: boolean;
	    issues?: string[];
	
	    static createFrom(source: any = {}) {
	        return new FactualityCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bullet_index = source["bullet_index"];
	        this.bullet = source["bullet"];
	        this.has_claims = source["has_claims"];
	        this.all_approved = source["all_approved"];
	        this.issues = source["issues"];
	    }
	}
	export class FitNeedAnalysis {
	    requirement_id: number;
	    jd_need: string;
	    matching_fact_ids: number[];
	    evidence_strength: string;
	    gap_level: string;
	    confidence: string;
	    risk: string;
	
	    static createFrom(source: any = {}) {
	        return new FitNeedAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requirement_id = source["requirement_id"];
	        this.jd_need = source["jd_need"];
	        this.matching_fact_ids = source["matching_fact_ids"];
	        this.evidence_strength = source["evidence_strength"];
	        this.gap_level = source["gap_level"];
	        this.confidence = source["confidence"];
	        this.risk = source["risk"];
	    }
	}
	export class GenerateResumeJSONInput {
	    job_id: number;
	    selected_bullet_ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new GenerateResumeJSONInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.selected_bullet_ids = source["selected_bullet_ids"];
	    }
	}
	export class Health {
	    version: string;
	    storage_status: string;
	    db_path: string;
	    log_path: string;
	    generated_path: string;
	    pdf_renderer: string;
	
	    static createFrom(source: any = {}) {
	        return new Health(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.storage_status = source["storage_status"];
	        this.db_path = source["db_path"];
	        this.log_path = source["log_path"];
	        this.generated_path = source["generated_path"];
	        this.pdf_renderer = source["pdf_renderer"];
	    }
	}
	export class ImportCandidateSourceFileInput {
	    path: string;
	    source_type: string;
	    trust_tier: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportCandidateSourceFileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.source_type = source["source_type"];
	        this.trust_tier = source["trust_tier"];
	        this.title = source["title"];
	    }
	}
	export class InstallTectonicResult {
	    success: boolean;
	    status: string;
	    executable_path: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallTectonicResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.status = source["status"];
	        this.executable_path = source["executable_path"];
	        this.error = source["error"];
	    }
	}
	export class JobAgentWorkflowInput {
	    job: CreateJobDescriptionInput;
	    job_id: number;
	    auto_select_bullets: boolean;
	    build_resume: boolean;
	    min_selected_bullets: number;
	    max_selected_bullets: number;
	    require_resume_review: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JobAgentWorkflowInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job = this.convertValues(source["job"], CreateJobDescriptionInput);
	        this.job_id = source["job_id"];
	        this.auto_select_bullets = source["auto_select_bullets"];
	        this.build_resume = source["build_resume"];
	        this.min_selected_bullets = source["min_selected_bullets"];
	        this.max_selected_bullets = source["max_selected_bullets"];
	        this.require_resume_review = source["require_resume_review"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ValidationResult {
	    passed: boolean;
	    errors: string[];
	    warnings: string[];
	    factuality_checks: FactualityCheck[];
	    style_issues: string[];
	    immutable_issues: string[];
	    title_issues: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.passed = source["passed"];
	        this.errors = source["errors"];
	        this.warnings = source["warnings"];
	        this.factuality_checks = this.convertValues(source["factuality_checks"], FactualityCheck);
	        this.style_issues = source["style_issues"];
	        this.immutable_issues = source["immutable_issues"];
	        this.title_issues = source["title_issues"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResumeEducation {
	    organization: string;
	    degree: string;
	    location: string;
	    end_date: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeEducation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.organization = source["organization"];
	        this.degree = source["degree"];
	        this.location = source["location"];
	        this.end_date = source["end_date"];
	    }
	}
	export class ResumeEntry {
	    company: string;
	    url?: string;
	    title: string;
	    location: string;
	    start_date: string;
	    end_date: string;
	    bullets: string[];
	    claim_ids: number[];
	    bullet_ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new ResumeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.company = source["company"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.location = source["location"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.bullets = source["bullets"];
	        this.claim_ids = source["claim_ids"];
	        this.bullet_ids = source["bullet_ids"];
	    }
	}
	export class ResumeSkill {
	    category: string;
	    items: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResumeSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.category = source["category"];
	        this.items = source["items"];
	    }
	}
	export class ResumeJSON {
	    headline: string;
	    summary: string;
	    contact_line: string;
	    skills_line: string;
	    skills: ResumeSkill[];
	    experience: ResumeEntry[];
	    projects: ResumeEntry[];
	    education: ResumeEducation[];
	    tex_source: string;
	    generated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeJSON(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headline = source["headline"];
	        this.summary = source["summary"];
	        this.contact_line = source["contact_line"];
	        this.skills_line = source["skills_line"];
	        this.skills = this.convertValues(source["skills"], ResumeSkill);
	        this.experience = this.convertValues(source["experience"], ResumeEntry);
	        this.projects = this.convertValues(source["projects"], ResumeEntry);
	        this.education = this.convertValues(source["education"], ResumeEducation);
	        this.tex_source = source["tex_source"];
	        this.generated_at = source["generated_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobFitAnalysis {
	    job_id: number;
	    overall_score: number;
	    recommendation: string;
	    strengths: string[];
	    critical_gaps: string[];
	    reality_check: string;
	    analysis: FitNeedAnalysis[];
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new JobFitAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.overall_score = source["overall_score"];
	        this.recommendation = source["recommendation"];
	        this.strengths = source["strengths"];
	        this.critical_gaps = source["critical_gaps"];
	        this.reality_check = source["reality_check"];
	        this.analysis = this.convertValues(source["analysis"], FitNeedAnalysis);
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobAnalysis {
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
	
	    static createFrom(source: any = {}) {
	        return new JobAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.company = source["company"];
	        this.role_title = source["role_title"];
	        this.location = source["location"];
	        this.work_arrangement = source["work_arrangement"];
	        this.salary = source["salary"];
	        this.top_pain_points = source["top_pain_points"];
	        this.required_skills = source["required_skills"];
	        this.preferred_skills = source["preferred_skills"];
	        this.responsibilities = source["responsibilities"];
	        this.seniority_level = source["seniority_level"];
	        this.role_archetype = source["role_archetype"];
	        this.keywords = source["keywords"];
	        this.risk_flags = source["risk_flags"];
	        this.job_poster = source["job_poster"];
	        this.company_url = source["company_url"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class TailoredBulletDraft {
	    id: number;
	    job_id: number;
	    requirement_id: number;
	    fact_ids: number[];
	    claim_ids: number[];
	    origin_heading: string;
	    origin_type: string;
	    value_theme: string;
	    draft_text: string;
	    rationale: string;
	    status: string;
	    risk_flags: string[];
	    selection_score: number;
	    resume_value_score: number;
	    jd_relevance_score: number;
	    origin_weight: number;
	    risk_penalty: number;
	    unsupported_context_penalty: number;
	    selection_reason: string;
	    display_order: number;
	    selected_for_resume: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new TailoredBulletDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.requirement_id = source["requirement_id"];
	        this.fact_ids = source["fact_ids"];
	        this.claim_ids = source["claim_ids"];
	        this.origin_heading = source["origin_heading"];
	        this.origin_type = source["origin_type"];
	        this.value_theme = source["value_theme"];
	        this.draft_text = source["draft_text"];
	        this.rationale = source["rationale"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	        this.selection_score = source["selection_score"];
	        this.resume_value_score = source["resume_value_score"];
	        this.jd_relevance_score = source["jd_relevance_score"];
	        this.origin_weight = source["origin_weight"];
	        this.risk_penalty = source["risk_penalty"];
	        this.unsupported_context_penalty = source["unsupported_context_penalty"];
	        this.selection_reason = source["selection_reason"];
	        this.display_order = source["display_order"];
	        this.selected_for_resume = source["selected_for_resume"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class JobFactMatch {
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
	
	    static createFrom(source: any = {}) {
	        return new JobFactMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.requirement_id = source["requirement_id"];
	        this.fact_id = source["fact_id"];
	        this.score = source["score"];
	        this.rationale = source["rationale"];
	        this.coverage_status = source["coverage_status"];
	        this.fact_status = source["fact_status"];
	        this.fact_text = source["fact_text"];
	        this.evidence_quote = source["evidence_quote"];
	        this.risk_flags = source["risk_flags"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class JobRequirement {
	    id: number;
	    job_id: number;
	    category: string;
	    requirement_text: string;
	    keywords: string[];
	    priority: string;
	    source_quote: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new JobRequirement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.category = source["category"];
	        this.requirement_text = source["requirement_text"];
	        this.keywords = source["keywords"];
	        this.priority = source["priority"];
	        this.source_quote = source["source_quote"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class JobAgentWorkflowStage {
	    key: string;
	    label: string;
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new JobAgentWorkflowStage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class JobDescription {
	    id: number;
	    company: string;
	    title: string;
	    url: string;
	    raw_text: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new JobDescription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.company = source["company"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.raw_text = source["raw_text"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class JobAgentWorkflowResult {
	    job: JobDescription;
	    stages: JobAgentWorkflowStage[];
	    requirements: JobRequirement[];
	    matches: JobFactMatch[];
	    drafts: TailoredBulletDraft[];
	    analysis: JobAnalysis;
	    fit: JobFitAnalysis;
	    strategy: ApplicationStrategy;
	    resume: ResumeJSON;
	    validation: ValidationResult;
	    resume_generated: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new JobAgentWorkflowResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job = this.convertValues(source["job"], JobDescription);
	        this.stages = this.convertValues(source["stages"], JobAgentWorkflowStage);
	        this.requirements = this.convertValues(source["requirements"], JobRequirement);
	        this.matches = this.convertValues(source["matches"], JobFactMatch);
	        this.drafts = this.convertValues(source["drafts"], TailoredBulletDraft);
	        this.analysis = this.convertValues(source["analysis"], JobAnalysis);
	        this.fit = this.convertValues(source["fit"], JobFitAnalysis);
	        this.strategy = this.convertValues(source["strategy"], ApplicationStrategy);
	        this.resume = this.convertValues(source["resume"], ResumeJSON);
	        this.validation = this.convertValues(source["validation"], ValidationResult);
	        this.resume_generated = source["resume_generated"];
	        this.created_at = source["created_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	export class LLMTestResult {
	    success: boolean;
	    provider: string;
	    model: string;
	    text: string;
	    latency_ms: number;
	    status_code: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.text = source["text"];
	        this.latency_ms = source["latency_ms"];
	        this.status_code = source["status_code"];
	        this.error = source["error"];
	    }
	}
	export class PromptResearchSource {
	    id: number;
	    source_type: string;
	    trust_tier: string;
	    title: string;
	    url: string;
	    extracted_pattern: string;
	    app_adaptation: string;
	    accessed_at: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new PromptResearchSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_type = source["source_type"];
	        this.trust_tier = source["trust_tier"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.extracted_pattern = source["extracted_pattern"];
	        this.app_adaptation = source["app_adaptation"];
	        this.accessed_at = source["accessed_at"];
	        this.created_at = source["created_at"];
	    }
	}
	export class PromptRule {
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
	
	    static createFrom(source: any = {}) {
	        return new PromptRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rule_key = source["rule_key"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.enabled = source["enabled"];
	        this.version = source["version"];
	        this.source = source["source"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class RenderPDFResult {
	    success: boolean;
	    tex_path: string;
	    pdf_path: string;
	    output_dir: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new RenderPDFResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.tex_path = source["tex_path"];
	        this.pdf_path = source["pdf_path"];
	        this.output_dir = source["output_dir"];
	        this.error = source["error"];
	    }
	}
	export class ResumeContextClaim {
	    id: number;
	    label: string;
	    source_fact_ids: number[];
	    actions: string[];
	    capabilities: string[];
	    objects: string[];
	    technologies: string[];
	    domains: string[];
	    artifacts: string[];
	    scope: string[];
	    metrics: string[];
	    outcomes: string[];
	    evidence_strength: string;
	    status: string;
	    risk_flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResumeContextClaim(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.source_fact_ids = source["source_fact_ids"];
	        this.actions = source["actions"];
	        this.capabilities = source["capabilities"];
	        this.objects = source["objects"];
	        this.technologies = source["technologies"];
	        this.domains = source["domains"];
	        this.artifacts = source["artifacts"];
	        this.scope = source["scope"];
	        this.metrics = source["metrics"];
	        this.outcomes = source["outcomes"];
	        this.evidence_strength = source["evidence_strength"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	    }
	}
	export class ResumeContextFact {
	    id: number;
	    atoms: string;
	    technologies: string[];
	    status: string;
	    risk_flags: string[];
	    evidence_quote: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeContextFact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.atoms = source["atoms"];
	        this.technologies = source["technologies"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	        this.evidence_quote = source["evidence_quote"];
	    }
	}
	export class ResumeContextOrigin {
	    origin_heading: string;
	    origin_type: string;
	    facts: ResumeContextFact[];
	    claims: ResumeContextClaim[];
	    keywords: string[];
	    risk_flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResumeContextOrigin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.origin_heading = source["origin_heading"];
	        this.origin_type = source["origin_type"];
	        this.facts = this.convertValues(source["facts"], ResumeContextFact);
	        this.claims = this.convertValues(source["claims"], ResumeContextClaim);
	        this.keywords = source["keywords"];
	        this.risk_flags = source["risk_flags"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResumeContext {
	    source_id: number;
	    origins: ResumeContextOrigin[];
	
	    static createFrom(source: any = {}) {
	        return new ResumeContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_id = source["source_id"];
	        this.origins = this.convertValues(source["origins"], ResumeContextOrigin);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	export class ResumeVersion {
	    id: number;
	    job_id: number;
	    resume_json: ResumeJSON;
	    tex_source: string;
	    pdf_path: string;
	    validation_result: ValidationResult;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ResumeVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.job_id = source["job_id"];
	        this.resume_json = this.convertValues(source["resume_json"], ResumeJSON);
	        this.tex_source = source["tex_source"];
	        this.pdf_path = source["pdf_path"];
	        this.validation_result = this.convertValues(source["validation_result"], ValidationResult);
	        this.created_at = source["created_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SaveAPIKeyInput {
	    api_key: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveAPIKeyInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.provider = source["provider"];
	    }
	}
	export class SaveSettingsInput {
	    provider: string;
	    model: string;
	    embedding_model: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveSettingsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.embedding_model = source["embedding_model"];
	    }
	}
	export class SelectTailoredBulletDraftInput {
	    id: number;
	    selected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SelectTailoredBulletDraftInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.selected = source["selected"];
	    }
	}
	export class Settings {
	    provider: string;
	    model: string;
	    embedding_model: string;
	    api_key_configured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.embedding_model = source["embedding_model"];
	        this.api_key_configured = source["api_key_configured"];
	    }
	}
	export class SourceSection {
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
	
	    static createFrom(source: any = {}) {
	        return new SourceSection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_id = source["source_id"];
	        this.heading = source["heading"];
	        this.section_type = source["section_type"];
	        this.content = source["content"];
	        this.sort_order = source["sort_order"];
	        this.start_char = source["start_char"];
	        this.end_char = source["end_char"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	
	export class ToolStatus {
	    api_key_configured: boolean;
	    api_key_source: string;
	    env_local_path: string;
	    tectonic_status: string;
	    tectonic_path: string;
	    generated_path: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key_configured = source["api_key_configured"];
	        this.api_key_source = source["api_key_source"];
	        this.env_local_path = source["env_local_path"];
	        this.tectonic_status = source["tectonic_status"];
	        this.tectonic_path = source["tectonic_path"];
	        this.generated_path = source["generated_path"];
	    }
	}
	export class UpdateBlockedClaimInput {
	    id: number;
	    pattern: string;
	    reason: string;
	    severity: string;
	    source: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateBlockedClaimInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pattern = source["pattern"];
	        this.reason = source["reason"];
	        this.severity = source["severity"];
	        this.source = source["source"];
	        this.enabled = source["enabled"];
	    }
	}
	export class UpdateCandidateClaimReviewInput {
	    id: number;
	    claim_text: string;
	    claim_type: string;
	    actions: string[];
	    capabilities: string[];
	    objects: string[];
	    technologies: string[];
	    domains: string[];
	    artifacts: string[];
	    scope: string[];
	    metrics: string[];
	    outcomes: string[];
	    profile_context: string[];
	    evidence_strength: string;
	    strength: string;
	    allowed_use: string[];
	    allowed_contexts: string[];
	    blocked_contexts: string[];
	    safe_phrasings: string[];
	    unsafe_phrasings: string[];
	    status: string;
	    risk_flags: string[];
	    review_note: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCandidateClaimReviewInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.claim_text = source["claim_text"];
	        this.claim_type = source["claim_type"];
	        this.actions = source["actions"];
	        this.capabilities = source["capabilities"];
	        this.objects = source["objects"];
	        this.technologies = source["technologies"];
	        this.domains = source["domains"];
	        this.artifacts = source["artifacts"];
	        this.scope = source["scope"];
	        this.metrics = source["metrics"];
	        this.outcomes = source["outcomes"];
	        this.profile_context = source["profile_context"];
	        this.evidence_strength = source["evidence_strength"];
	        this.strength = source["strength"];
	        this.allowed_use = source["allowed_use"];
	        this.allowed_contexts = source["allowed_contexts"];
	        this.blocked_contexts = source["blocked_contexts"];
	        this.safe_phrasings = source["safe_phrasings"];
	        this.unsafe_phrasings = source["unsafe_phrasings"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	        this.review_note = source["review_note"];
	    }
	}
	export class UpdateEvidenceFactReviewInput {
	    id: number;
	    fact_text: string;
	    evidence_quote: string;
	    technologies: string[];
	    confidence: string;
	    risk_flags: string[];
	    status: string;
	    review_note: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateEvidenceFactReviewInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fact_text = source["fact_text"];
	        this.evidence_quote = source["evidence_quote"];
	        this.technologies = source["technologies"];
	        this.confidence = source["confidence"];
	        this.risk_flags = source["risk_flags"];
	        this.status = source["status"];
	        this.review_note = source["review_note"];
	    }
	}
	export class UpdateJobDescriptionInput {
	    id: number;
	    company: string;
	    title: string;
	    url: string;
	    raw_text: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateJobDescriptionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.company = source["company"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.raw_text = source["raw_text"];
	    }
	}
	export class UpdatePromptRuleInput {
	    id: number;
	    content: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdatePromptRuleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.enabled = source["enabled"];
	    }
	}
	export class UpdateSourceSectionInput {
	    id: number;
	    heading: string;
	    section_type: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSourceSectionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.heading = source["heading"];
	        this.section_type = source["section_type"];
	        this.content = source["content"];
	    }
	}
	export class UpdateTailoredBulletDraftInput {
	    id: number;
	    draft_text: string;
	    rationale: string;
	    status: string;
	    risk_flags: string[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateTailoredBulletDraftInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.draft_text = source["draft_text"];
	        this.rationale = source["rationale"];
	        this.status = source["status"];
	        this.risk_flags = source["risk_flags"];
	    }
	}

}

