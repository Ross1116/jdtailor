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

}
