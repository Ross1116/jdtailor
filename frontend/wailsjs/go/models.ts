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
	        this.title = source["title"];
	        this.raw_text = source["raw_text"];
	        this.file_path = source["file_path"];
	        this.imported_at = source["imported_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CreateCandidateSourceInput {
	    source_type: string;
	    title: string;
	    raw_text: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateCandidateSourceInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.title = source["title"];
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
	    status: string;
	    auto_approved: boolean;
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
	        this.status = source["status"];
	        this.auto_approved = source["auto_approved"];
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
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportCandidateSourceFileInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.source_type = source["source_type"];
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
	export class RenderPDFResult {
	    success: boolean;
	    tex_path: string;
	    pdf_path: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new RenderPDFResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.tex_path = source["tex_path"];
	        this.pdf_path = source["pdf_path"];
	        this.error = source["error"];
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
	
	    static createFrom(source: any = {}) {
	        return new SaveSettingsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	    }
	}
	export class Settings {
	    provider: string;
	    model: string;
	    api_key_configured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
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

}

