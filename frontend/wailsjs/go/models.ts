export namespace app {
	
	export class BookmarkFolderDTO {
	    id: number;
	    name: string;
	    key: string;
	    parentKey: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BookmarkFolderDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.key = source["key"];
	        this.parentKey = source["parentKey"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class ModuleDetails {
	    module: string;
	    tree: mib.Node[];
	    stats: mib.ModuleStats;
	    missingImports: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModuleDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.module = source["module"];
	        this.tree = this.convertValues(source["tree"], mib.Node);
	        this.stats = this.convertValues(source["stats"], mib.ModuleStats);
	        this.missingImports = source["missingImports"];
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
	export class TableColumn {
	    key: string;
	    label: string;
	    oid: string;
	    type: string;
	    syntax?: string;
	    access?: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.oid = source["oid"];
	        this.type = source["type"];
	        this.syntax = source["syntax"];
	        this.access = source["access"];
	        this.description = source["description"];
	    }
	}
	export class TableDataResponse {
	    tableOid: string;
	    entryOid: string;
	    columns: TableColumn[];
	    rows: any[];
	
	    static createFrom(source: any = {}) {
	        return new TableDataResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tableOid = source["tableOid"];
	        this.entryOid = source["entryOid"];
	        this.columns = this.convertValues(source["columns"], TableColumn);
	        this.rows = source["rows"];
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

export namespace mib {
	
	export class HostConfig {
	    address: string;
	    port: number;
	    community: string;
	    writeCommunity: string;
	    version: string;
	    lastUsedAt: string;
	    createdAt: string;
	    contextName?: string;
	    securityLevel?: string;
	    securityUsername?: string;
	    authProtocol?: string;
	    authPassword?: string;
	    privProtocol?: string;
	    privPassword?: string;
	
	    static createFrom(source: any = {}) {
	        return new HostConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.address = source["address"];
	        this.port = source["port"];
	        this.community = source["community"];
	        this.writeCommunity = source["writeCommunity"];
	        this.version = source["version"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.createdAt = source["createdAt"];
	        this.contextName = source["contextName"];
	        this.securityLevel = source["securityLevel"];
	        this.securityUsername = source["securityUsername"];
	        this.authProtocol = source["authProtocol"];
	        this.authPassword = source["authPassword"];
	        this.privProtocol = source["privProtocol"];
	        this.privPassword = source["privPassword"];
	    }
	}
	export class ModuleStats {
	    nodeCount: number;
	    scalarCount: number;
	    tableCount: number;
	    columnCount: number;
	    typeCount: number;
	    skippedNodes: number;
	    missingCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ModuleStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeCount = source["nodeCount"];
	        this.scalarCount = source["scalarCount"];
	        this.tableCount = source["tableCount"];
	        this.columnCount = source["columnCount"];
	        this.typeCount = source["typeCount"];
	        this.skippedNodes = source["skippedNodes"];
	        this.missingCount = source["missingCount"];
	    }
	}
	export class ModuleSummary {
	    name: string;
	    filePath: string;
	    nodeCount: number;
	    scalarCount: number;
	    tableCount: number;
	    columnCount: number;
	    typeCount: number;
	    skippedNodes: number;
	    missingImports: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModuleSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.filePath = source["filePath"];
	        this.nodeCount = source["nodeCount"];
	        this.scalarCount = source["scalarCount"];
	        this.tableCount = source["tableCount"];
	        this.columnCount = source["columnCount"];
	        this.typeCount = source["typeCount"];
	        this.skippedNodes = source["skippedNodes"];
	        this.missingImports = source["missingImports"];
	    }
	}
	export class Node {
	    id: number;
	    oid: string;
	    name: string;
	    parentOid: string;
	    type: string;
	    syntax: string;
	    access: string;
	    status: string;
	    description: string;
	    module: string;
	    children?: Node[];
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.oid = source["oid"];
	        this.name = source["name"];
	        this.parentOid = source["parentOid"];
	        this.type = source["type"];
	        this.syntax = source["syntax"];
	        this.access = source["access"];
	        this.status = source["status"];
	        this.description = source["description"];
	        this.module = source["module"];
	        this.children = this.convertValues(source["children"], Node);
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

export namespace services {
	
	export class InfoSistema {
	    go_version: string;
	    go_os: string;
	    go_arch: string;
	
	    static createFrom(source: any = {}) {
	        return new InfoSistema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.go_version = source["go_version"];
	        this.go_os = source["go_os"];
	        this.go_arch = source["go_arch"];
	    }
	}

}

export namespace snmp {
	
	export class Config {
	    host: string;
	    port: number;
	    community: string;
	    writeCommunity?: string;
	    version: string;
	    contextName?: string;
	    securityLevel?: string;
	    securityUsername?: string;
	    authProtocol?: string;
	    authPassword?: string;
	    privProtocol?: string;
	    privPassword?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.community = source["community"];
	        this.writeCommunity = source["writeCommunity"];
	        this.version = source["version"];
	        this.contextName = source["contextName"];
	        this.securityLevel = source["securityLevel"];
	        this.securityUsername = source["securityUsername"];
	        this.authProtocol = source["authProtocol"];
	        this.authPassword = source["authPassword"];
	        this.privProtocol = source["privProtocol"];
	        this.privPassword = source["privPassword"];
	    }
	}
	export class Result {
	    oid: string;
	    value: string;
	    type: string;
	    status: string;
	    responseTime: number;
	    timestamp: string;
	    resolvedName: string;
	    rawValue?: string;
	    displayValue?: string;
	    syntax?: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.oid = source["oid"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.responseTime = source["responseTime"];
	        this.timestamp = source["timestamp"];
	        this.resolvedName = source["resolvedName"];
	        this.rawValue = source["rawValue"];
	        this.displayValue = source["displayValue"];
	        this.syntax = source["syntax"];
	    }
	}

}

