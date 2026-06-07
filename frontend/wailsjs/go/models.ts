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

}

