export namespace main {
	
	export class TunnelConfig {
	    id: string;
	    name: string;
	    sshHost: string;
	    sshPort: number;
	    user: string;
	    authType: string;
	    password?: string;
	    keyPath?: string;
	    localPort: number;
	    remoteHost: string;
	    remotePort: number;
	
	    static createFrom(source: any = {}) {
	        return new TunnelConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sshHost = source["sshHost"];
	        this.sshPort = source["sshPort"];
	        this.user = source["user"];
	        this.authType = source["authType"];
	        this.password = source["password"];
	        this.keyPath = source["keyPath"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	    }
	}
	export class TunnelStatus {
	    id: string;
	    active: boolean;
	    error: string;
	    uptime: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.active = source["active"];
	        this.error = source["error"];
	        this.uptime = source["uptime"];
	    }
	}

}

