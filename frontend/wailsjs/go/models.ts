export namespace main {
	
	export class FileFilter {
	    pattern: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.name = source["name"];
	    }
	}
	export class RequestData {
	    method: string;
	    url: string;
	    data: string;
	    cookie: string;
	    headers: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RequestData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.data = source["data"];
	        this.cookie = source["cookie"];
	        this.headers = source["headers"];
	    }
	}
	export class ResponseData {
	    status: number;
	    headers: Record<string, Array<string>>;
	    data: number[];
	
	    static createFrom(source: any = {}) {
	        return new ResponseData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.headers = source["headers"];
	        this.data = source["data"];
	    }
	}

}

