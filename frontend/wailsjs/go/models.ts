export namespace main {
	
	export class Settings {
	    autoReconnect: boolean;
	    keepaliveSeconds: number;
	    startOnBoot: boolean;
	    theme: string;
	    defaultSshPort: number;
	    defaultSshUser: string;
	    defaultKeyPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoReconnect = source["autoReconnect"];
	        this.keepaliveSeconds = source["keepaliveSeconds"];
	        this.startOnBoot = source["startOnBoot"];
	        this.theme = source["theme"];
	        this.defaultSshPort = source["defaultSshPort"];
	        this.defaultSshUser = source["defaultSshUser"];
	        this.defaultKeyPath = source["defaultKeyPath"];
	    }
	}
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
	    bastionHost?: string;
	    bastionPort?: number;
	    bastionUser?: string;
	    bastionAuthType?: string;
	    bastionPassword?: string;
	    bastionKeyPath?: string;
	
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
	        this.bastionHost = source["bastionHost"];
	        this.bastionPort = source["bastionPort"];
	        this.bastionUser = source["bastionUser"];
	        this.bastionAuthType = source["bastionAuthType"];
	        this.bastionPassword = source["bastionPassword"];
	        this.bastionKeyPath = source["bastionKeyPath"];
	    }
	}
	export class TunnelStatus {
	    id: string;
	    active: boolean;
	    error: string;
	    uptime: string;
	    reconnecting: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TunnelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.active = source["active"];
	        this.error = source["error"];
	        this.uptime = source["uptime"];
	        this.reconnecting = source["reconnecting"];
	    }
	}

}

