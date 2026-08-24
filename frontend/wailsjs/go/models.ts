export namespace main {
	
	export class AppError {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AppError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class EntryPatch {
	    name: string;
	    aliases: string[];
	    summary: string;
	    body: string;
	    links: string[];
	    tags: string[];
	    alias_weight: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new EntryPatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.aliases = source["aliases"];
	        this.summary = source["summary"];
	        this.body = source["body"];
	        this.links = source["links"];
	        this.tags = source["tags"];
	        this.alias_weight = source["alias_weight"];
	    }
	}
	export class EntryView {
	    id: string;
	    type: string;
	    name: string;
	    aliases: string[];
	    summary: string;
	    body: string;
	    links: string[];
	    tags: string[];
	    alias_weight: Record<string, number>;
	    sovereign: boolean;
	    updated: string;
	
	    static createFrom(source: any = {}) {
	        return new EntryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.aliases = source["aliases"];
	        this.summary = source["summary"];
	        this.body = source["body"];
	        this.links = source["links"];
	        this.tags = source["tags"];
	        this.alias_weight = source["alias_weight"];
	        this.sovereign = source["sovereign"];
	        this.updated = source["updated"];
	    }
	}
	export class EventView {
	    timestamp: string;
	    turn: number;
	    kind: string;
	    text: string;
	    participants: string[];
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new EventView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.turn = source["turn"];
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.participants = source["participants"];
	        this.location = source["location"];
	    }
	}
	export class ImportResult {
	    ok: boolean;
	    message: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.path = source["path"];
	    }
	}
	export class SceneView {
	    now: string;
	    location: string;
	    present: string[];
	    open_threads: string[];
	
	    static createFrom(source: any = {}) {
	        return new SceneView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.now = source["now"];
	        this.location = source["location"];
	        this.present = source["present"];
	        this.open_threads = source["open_threads"];
	    }
	}
	export class SearchHit {
	    id: string;
	    type: string;
	    name: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.summary = source["summary"];
	    }
	}
	export class WorldInfo {
	    name: string;
	    path: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new WorldInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.description = source["description"];
	    }
	}

}

export namespace world {
	
	export class Budget {
	    inject_max_tokens: number;
	    top_k_entries: number;
	    recency_boost_turns: number;
	    dedup_window_turns: number;
	
	    static createFrom(source: any = {}) {
	        return new Budget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inject_max_tokens = source["inject_max_tokens"];
	        this.top_k_entries = source["top_k_entries"];
	        this.recency_boost_turns = source["recency_boost_turns"];
	        this.dedup_window_turns = source["dedup_window_turns"];
	    }
	}
	export class ProviderConfig {
	    backend: string;
	    server_url: string;
	    base_url: string;
	    model_id: string;
	    api_key?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backend = source["backend"];
	        this.server_url = source["server_url"];
	        this.base_url = source["base_url"];
	        this.model_id = source["model_id"];
	        this.api_key = source["api_key"];
	    }
	}
	export class WorldManifest {
	    name: string;
	    description: string;
	    starting_scene: string;
	    narrator_rules: string[];
	    auto_accept_writes: boolean;
	    budget: Budget;
	    provider: ProviderConfig;
	
	    static createFrom(source: any = {}) {
	        return new WorldManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.starting_scene = source["starting_scene"];
	        this.narrator_rules = source["narrator_rules"];
	        this.auto_accept_writes = source["auto_accept_writes"];
	        this.budget = this.convertValues(source["budget"], Budget);
	        this.provider = this.convertValues(source["provider"], ProviderConfig);
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

}

