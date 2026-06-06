export namespace browser_pref {
	
	export class RuntimeNoticeResult {
	    show_notice: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeNoticeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.show_notice = source["show_notice"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace controlseal {
	
	export class Settings {
	    rotate_every_successful_turns: number;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rotate_every_successful_turns = source["rotate_every_successful_turns"];
	    }
	}

}

export namespace credential {
	
	export class MigrationStatus {
	    ready: boolean;
	    required: boolean;
	    provider_id: string;
	    disabled: boolean;
	    error: string;
	    credential_path: string;
	
	    static createFrom(source: any = {}) {
	        return new MigrationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.required = source["required"];
	        this.provider_id = source["provider_id"];
	        this.disabled = source["disabled"];
	        this.error = source["error"];
	        this.credential_path = source["credential_path"];
	    }
	}

}

export namespace dag {
	
	export class DAGNode {
	    id: string;
	    title?: string;
	    operation: string;
	    action?: string;
	    action_code?: string;
	    target?: string;
	    executor_type?: string;
	    risk_class: string;
	    model_risk_class?: string;
	    status: string;
	    dependencies: string[];
	    parallel_root?: boolean;
	    block_reason: string;
	    error: string;
	    started_at: string;
	    completed_at: string;
	    review_id?: string;
	    result_summary?: string;
	    output_ref?: string;
	    trace_hash?: string;
	    retry_count: number;
	    max_retries: number;
	    approved_by?: string;
	    approved_at?: string;
	    app_session_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new DAGNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.operation = source["operation"];
	        this.action = source["action"];
	        this.action_code = source["action_code"];
	        this.target = source["target"];
	        this.executor_type = source["executor_type"];
	        this.risk_class = source["risk_class"];
	        this.model_risk_class = source["model_risk_class"];
	        this.status = source["status"];
	        this.dependencies = source["dependencies"];
	        this.parallel_root = source["parallel_root"];
	        this.block_reason = source["block_reason"];
	        this.error = source["error"];
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	        this.review_id = source["review_id"];
	        this.result_summary = source["result_summary"];
	        this.output_ref = source["output_ref"];
	        this.trace_hash = source["trace_hash"];
	        this.retry_count = source["retry_count"];
	        this.max_retries = source["max_retries"];
	        this.approved_by = source["approved_by"];
	        this.approved_at = source["approved_at"];
	        this.app_session_id = source["app_session_id"];
	    }
	}
	export class TaskPlanNode {
	    id: string;
	    title: string;
	    executor_type: string;
	    action_code: string;
	    action: string;
	    target: string;
	    risk_class: string;
	    model_risk_class?: string;
	    dependencies: string[];
	    parallel_root?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TaskPlanNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.executor_type = source["executor_type"];
	        this.action_code = source["action_code"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.risk_class = source["risk_class"];
	        this.model_risk_class = source["model_risk_class"];
	        this.dependencies = source["dependencies"];
	        this.parallel_root = source["parallel_root"];
	    }
	}
	export class TaskPlan {
	    title: string;
	    nodes: TaskPlanNode[];
	
	    static createFrom(source: any = {}) {
	        return new TaskPlan(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.nodes = this.convertValues(source["nodes"], TaskPlanNode);
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
	export class PlannerMetadata {
	    normalized_plan?: TaskPlan;
	    raw_model_plan?: string;
	    raw_model_plan_truncated?: boolean;
	    repair_attempt_count: number;
	    planner_adapter_id?: string;
	    planner_model_id?: string;
	    validation_warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new PlannerMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.normalized_plan = this.convertValues(source["normalized_plan"], TaskPlan);
	        this.raw_model_plan = source["raw_model_plan"];
	        this.raw_model_plan_truncated = source["raw_model_plan_truncated"];
	        this.repair_attempt_count = source["repair_attempt_count"];
	        this.planner_adapter_id = source["planner_adapter_id"];
	        this.planner_model_id = source["planner_model_id"];
	        this.validation_warnings = source["validation_warnings"];
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
	export class DAGRun {
	    id: string;
	    status: string;
	    title?: string;
	    nodes: DAGNode[];
	    created_at: string;
	    updated_at: string;
	    guard_hash: string;
	    hook_run_id?: string;
	    outline_id?: string;
	    active_node_id?: string;
	    active_trace_id?: string;
	    interrupt_reason?: string;
	    planner?: PlannerMetadata;
	    schema?: string;
	
	    static createFrom(source: any = {}) {
	        return new DAGRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.title = source["title"];
	        this.nodes = this.convertValues(source["nodes"], DAGNode);
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.guard_hash = source["guard_hash"];
	        this.hook_run_id = source["hook_run_id"];
	        this.outline_id = source["outline_id"];
	        this.active_node_id = source["active_node_id"];
	        this.active_trace_id = source["active_trace_id"];
	        this.interrupt_reason = source["interrupt_reason"];
	        this.planner = this.convertValues(source["planner"], PlannerMetadata);
	        this.schema = source["schema"];
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
	export class DAGRunSummary {
	    run_id: string;
	    status: string;
	    started_at: string;
	    ended_at?: string;
	    duration_ms?: number;
	    node_count: number;
	    failed_node_count: number;
	    sub_id?: string;
	    error_summary?: string;
	
	    static createFrom(source: any = {}) {
	        return new DAGRunSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.status = source["status"];
	        this.started_at = source["started_at"];
	        this.ended_at = source["ended_at"];
	        this.duration_ms = source["duration_ms"];
	        this.node_count = source["node_count"];
	        this.failed_node_count = source["failed_node_count"];
	        this.sub_id = source["sub_id"];
	        this.error_summary = source["error_summary"];
	    }
	}
	export class GuardCheckResult {
	    safe: boolean;
	    changed_fields: string[];
	    block_reason: string;
	
	    static createFrom(source: any = {}) {
	        return new GuardCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.safe = source["safe"];
	        this.changed_fields = source["changed_fields"];
	        this.block_reason = source["block_reason"];
	    }
	}
	
	

}

export namespace debugtrace {
	
	export class LinkSnapshot {
	    url: string;
	    addr: string;
	    started: boolean;
	    version: number;
	    updated_at: string;
	    last_error?: string;
	
	    static createFrom(source: any = {}) {
	        return new LinkSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.addr = source["addr"];
	        this.started = source["started"];
	        this.version = source["version"];
	        this.updated_at = source["updated_at"];
	        this.last_error = source["last_error"];
	    }
	}

}

export namespace health {
	
	export class ConfigPublic {
	    app_version: string;
	    spec_version: string;
	    data_root: string;
	    dev_mode: boolean;
	    first_run_done: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPublic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_version = source["app_version"];
	        this.spec_version = source["spec_version"];
	        this.data_root = source["data_root"];
	        this.dev_mode = source["dev_mode"];
	        this.first_run_done = source["first_run_done"];
	    }
	}
	export class MemoryHealth {
	    heap_alloc_mb: number;
	    num_goroutines: number;
	    data_dir_size_mb: number;
	    last_checked: string;
	
	    static createFrom(source: any = {}) {
	        return new MemoryHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.heap_alloc_mb = source["heap_alloc_mb"];
	        this.num_goroutines = source["num_goroutines"];
	        this.data_dir_size_mb = source["data_dir_size_mb"];
	        this.last_checked = source["last_checked"];
	    }
	}

}

export namespace llm_context {
	
	export class ContentBlock {
	    source: string;
	    content: string;
	    role: string;
	
	    static createFrom(source: any = {}) {
	        return new ContentBlock(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.content = source["content"];
	        this.role = source["role"];
	    }
	}
	export class WarningToken {
	    type: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new WarningToken(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.detail = source["detail"];
	    }
	}
	export class SourceToken {
	    hostname: string;
	    rank: number;
	    auth_ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SourceToken(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hostname = source["hostname"];
	        this.rank = source["rank"];
	        this.auth_ok = source["auth_ok"];
	    }
	}
	export class ContextPayload {
	    source_tokens: SourceToken[];
	    warning_tokens: WarningToken[];
	    content_blocks: ContentBlock[];
	    is_high_impact: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContextPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_tokens = this.convertValues(source["source_tokens"], SourceToken);
	        this.warning_tokens = this.convertValues(source["warning_tokens"], WarningToken);
	        this.content_blocks = this.convertValues(source["content_blocks"], ContentBlock);
	        this.is_high_impact = source["is_high_impact"];
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

export namespace main {
	
	export class ConsoleState {
	    greeting: string;
	    statusRail: statusrail.View;
	    adapters: string[];
	    haoras: string[];
	    messages: string[];
	
	    static createFrom(source: any = {}) {
	        return new ConsoleState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.greeting = source["greeting"];
	        this.statusRail = this.convertValues(source["statusRail"], statusrail.View);
	        this.adapters = source["adapters"];
	        this.haoras = source["haoras"];
	        this.messages = source["messages"];
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
	export class CreatedSubagent {
	    id: string;
	    name: string;
	    sub_dir: string;
	    memory_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new CreatedSubagent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sub_dir = source["sub_dir"];
	        this.memory_dir = source["memory_dir"];
	    }
	}
	export class DestructiveReviewExecutionResult {
	    review_id: string;
	    mode: string;
	    operation: string;
	    retry_count: number;
	    card_pending: boolean;
	    message: string;
	    backup?: project_lifecycle.PurgeBackupManifest;
	    purge?: any;
	
	    static createFrom(source: any = {}) {
	        return new DestructiveReviewExecutionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.review_id = source["review_id"];
	        this.mode = source["mode"];
	        this.operation = source["operation"];
	        this.retry_count = source["retry_count"];
	        this.card_pending = source["card_pending"];
	        this.message = source["message"];
	        this.backup = this.convertValues(source["backup"], project_lifecycle.PurgeBackupManifest);
	        this.purge = source["purge"];
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
	export class DocumentImportResult {
	    doc_id: string;
	    display_name: string;
	    format: string;
	    word_count: number;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.doc_id = source["doc_id"];
	        this.display_name = source["display_name"];
	        this.format = source["format"];
	        this.word_count = source["word_count"];
	        this.encoding = source["encoding"];
	    }
	}
	export class DocumentPreview {
	    display_name: string;
	    format: string;
	    word_count: number;
	    preview: string;
	    doc_id: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.display_name = source["display_name"];
	        this.format = source["format"];
	        this.word_count = source["word_count"];
	        this.preview = source["preview"];
	        this.doc_id = source["doc_id"];
	    }
	}
	export class EmbedModelInfo {
	    id: string;
	    providerId: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new EmbedModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.label = source["label"];
	    }
	}
	export class EmbedPullJob {
	    jobId: string;
	    modelId: string;
	
	    static createFrom(source: any = {}) {
	        return new EmbedPullJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.modelId = source["modelId"];
	    }
	}
	export class FloatingCandidate {
	    id: string;
	    label: string;
	    draft: string;
	
	    static createFrom(source: any = {}) {
	        return new FloatingCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.draft = source["draft"];
	    }
	}
	export class ImportSubResult {
	    new_system_code: string;
	    sub_dir: string;
	    tool_conflicts: subexport.ToolConflict[];
	    installed_tools: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportSubResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.new_system_code = source["new_system_code"];
	        this.sub_dir = source["sub_dir"];
	        this.tool_conflicts = this.convertValues(source["tool_conflicts"], subexport.ToolConflict);
	        this.installed_tools = source["installed_tools"];
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
	export class NativePersonaDragExportResult {
	    status: string;
	    export_path: string;
	    landed_path: string;
	    platform: string;
	    fallback_required: boolean;
	    message: string;
	    persona_id: string;
	    display_name: string;
	    drop_target_kind: string;
	    drop_target_dir: string;
	    state?: any;
	
	    static createFrom(source: any = {}) {
	        return new NativePersonaDragExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.export_path = source["export_path"];
	        this.landed_path = source["landed_path"];
	        this.platform = source["platform"];
	        this.fallback_required = source["fallback_required"];
	        this.message = source["message"];
	        this.persona_id = source["persona_id"];
	        this.display_name = source["display_name"];
	        this.drop_target_kind = source["drop_target_kind"];
	        this.drop_target_dir = source["drop_target_dir"];
	        this.state = source["state"];
	    }
	}
	export class NativeReferenceFileDragResult {
	    status: string;
	    source_path: string;
	    landed_path: string;
	    platform: string;
	    fallback_required: boolean;
	    message: string;
	    display_name: string;
	    drop_target_kind: string;
	    drop_target_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeReferenceFileDragResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.source_path = source["source_path"];
	        this.landed_path = source["landed_path"];
	        this.platform = source["platform"];
	        this.fallback_required = source["fallback_required"];
	        this.message = source["message"];
	        this.display_name = source["display_name"];
	        this.drop_target_kind = source["drop_target_kind"];
	        this.drop_target_dir = source["drop_target_dir"];
	    }
	}
	export class NativeSubDragExportResult {
	    status: string;
	    export_dir: string;
	    landed_path: string;
	    platform: string;
	    fallback_required: boolean;
	    message: string;
	    sub_id: string;
	    display_name: string;
	    new_system_code: string;
	    drop_target_kind: string;
	    drop_target_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new NativeSubDragExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.export_dir = source["export_dir"];
	        this.landed_path = source["landed_path"];
	        this.platform = source["platform"];
	        this.fallback_required = source["fallback_required"];
	        this.message = source["message"];
	        this.sub_id = source["sub_id"];
	        this.display_name = source["display_name"];
	        this.new_system_code = source["new_system_code"];
	        this.drop_target_kind = source["drop_target_kind"];
	        this.drop_target_dir = source["drop_target_dir"];
	    }
	}
	export class OllamaState {
	    binaryFound: boolean;
	    daemonRunning: boolean;
	    binaryPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new OllamaState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.binaryFound = source["binaryFound"];
	        this.daemonRunning = source["daemonRunning"];
	        this.binaryPath = source["binaryPath"];
	    }
	}
	export class ReadinessGateState {
	    risk_tier: string;
	    missing_slots: string[];
	    floating_candidates: FloatingCandidate[];
	    clarification_count: number;
	    max_clarifications: number;
	    retrieval_scanning: boolean;
	    retrieval_sources: string[];
	    impact_explanation: string;
	    low_confidence_output: boolean;
	    assumption_used: boolean;
	    auto_output_allowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReadinessGateState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.risk_tier = source["risk_tier"];
	        this.missing_slots = source["missing_slots"];
	        this.floating_candidates = this.convertValues(source["floating_candidates"], FloatingCandidate);
	        this.clarification_count = source["clarification_count"];
	        this.max_clarifications = source["max_clarifications"];
	        this.retrieval_scanning = source["retrieval_scanning"];
	        this.retrieval_sources = source["retrieval_sources"];
	        this.impact_explanation = source["impact_explanation"];
	        this.low_confidence_output = source["low_confidence_output"];
	        this.assumption_used = source["assumption_used"];
	        this.auto_output_allowed = source["auto_output_allowed"];
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
	export class ReferenceFile {
	    name: string;
	    path: string;
	    source?: string;
	    status?: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReferenceFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.source = source["source"];
	        this.status = source["status"];
	        this.detail = source["detail"];
	    }
	}
	export class SessionAnalysis {
	    total_actions: number;
	    main_direct_actions: number;
	    delegated_actions: number;
	    direct_ratio: number;
	    should_prompt: boolean;
	    has_content: boolean;
	    suggested_name: string;
	    mode?: string;
	    agent_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_actions = source["total_actions"];
	        this.main_direct_actions = source["main_direct_actions"];
	        this.delegated_actions = source["delegated_actions"];
	        this.direct_ratio = source["direct_ratio"];
	        this.should_prompt = source["should_prompt"];
	        this.has_content = source["has_content"];
	        this.suggested_name = source["suggested_name"];
	        this.mode = source["mode"];
	        this.agent_id = source["agent_id"];
	    }
	}
	export class SkillDraftSaveResult {
	    manifest?: skill_step.SkillManifest;
	    problems: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillDraftSaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifest = this.convertValues(source["manifest"], skill_step.SkillManifest);
	        this.problems = source["problems"];
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
	export class SkillExecutionDecision {
	    decision: string;
	    resolve_id: string;
	    skill_id?: string;
	    status: string;
	    injected: boolean;
	    executed: boolean;
	    action_target?: string;
	    response?: skill_step.CLIResponse;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillExecutionDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decision = source["decision"];
	        this.resolve_id = source["resolve_id"];
	        this.skill_id = source["skill_id"];
	        this.status = source["status"];
	        this.injected = source["injected"];
	        this.executed = source["executed"];
	        this.action_target = source["action_target"];
	        this.response = this.convertValues(source["response"], skill_step.CLIResponse);
	        this.message = source["message"];
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
	export class SubExportCapabilities {
	    platform: string;
	    native_drag_supported: boolean;
	    native_drag_strategy: string;
	    fallback_supported: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SubExportCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.native_drag_supported = source["native_drag_supported"];
	        this.native_drag_strategy = source["native_drag_strategy"];
	        this.fallback_supported = source["fallback_supported"];
	        this.message = source["message"];
	    }
	}
	export class SubPackagePreview {
	    export_dir: string;
	    display_name: string;
	    source_system_code: string;
	    tool_count: number;
	    tools: subexport.ManifestTool[];
	
	    static createFrom(source: any = {}) {
	        return new SubPackagePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.export_dir = source["export_dir"];
	        this.display_name = source["display_name"];
	        this.source_system_code = source["source_system_code"];
	        this.tool_count = source["tool_count"];
	        this.tools = this.convertValues(source["tools"], subexport.ManifestTool);
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
	export class SummaryModelOption {
	    provider: string;
	    id: string;
	    label: string;
	    endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new SummaryModelOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.id = source["id"];
	        this.label = source["label"];
	        this.endpoint = source["endpoint"];
	    }
	}
	export class SummaryModelScanResult {
	    options: SummaryModelOption[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SummaryModelScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.options = this.convertValues(source["options"], SummaryModelOption);
	        this.message = source["message"];
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

export namespace memory {
	
	export class PipelineState {
	    talk_full_size: number;
	    rotation_action: string;
	    main_memory_size: number;
	    deep_memory_size: number;
	    threat_entries: number;
	    manifest_hash: string;
	    last_rotation: string;
	
	    static createFrom(source: any = {}) {
	        return new PipelineState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.talk_full_size = source["talk_full_size"];
	        this.rotation_action = source["rotation_action"];
	        this.main_memory_size = source["main_memory_size"];
	        this.deep_memory_size = source["deep_memory_size"];
	        this.threat_entries = source["threat_entries"];
	        this.manifest_hash = source["manifest_hash"];
	        this.last_rotation = source["last_rotation"];
	    }
	}
	export class ValidationResult {
	    valid: boolean;
	    status: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace onboarding {
	
	export class Step {
	    id: string;
	    title: string;
	    description: string;
	    completed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.completed = source["completed"];
	    }
	}
	export class State {
	    is_first_run: boolean;
	    read_only_mode: boolean;
	    steps: Step[];
	    current_step: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_first_run = source["is_first_run"];
	        this.read_only_mode = source["read_only_mode"];
	        this.steps = this.convertValues(source["steps"], Step);
	        this.current_step = source["current_step"];
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

export namespace persona_avatar {
	
	export class CropRect {
	    x: number;
	    y: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new CropRect(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class AvatarConfig {
	    avatar_provider: string;
	    persona_id: string;
	    pixel_pack?: string;
	    static_avatar_path?: string;
	    original_image_path?: string;
	    crop?: CropRect;
	    output_size: number;
	    updated_at: string;
	    style_preset_id?: string;
	    credential_ref?: string;
	    api_endpoint?: string;
	
	    static createFrom(source: any = {}) {
	        return new AvatarConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avatar_provider = source["avatar_provider"];
	        this.persona_id = source["persona_id"];
	        this.pixel_pack = source["pixel_pack"];
	        this.static_avatar_path = source["static_avatar_path"];
	        this.original_image_path = source["original_image_path"];
	        this.crop = this.convertValues(source["crop"], CropRect);
	        this.output_size = source["output_size"];
	        this.updated_at = source["updated_at"];
	        this.style_preset_id = source["style_preset_id"];
	        this.credential_ref = source["credential_ref"];
	        this.api_endpoint = source["api_endpoint"];
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
	
	export class GenerateAvatarRequest {
	    prompt: string;
	    api_endpoint: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateAvatarRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt = source["prompt"];
	        this.api_endpoint = source["api_endpoint"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class StylePreset {
	    style_preset_id: string;
	    name: string;
	    prompt_template: string;
	    state_prompts: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new StylePreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.style_preset_id = source["style_preset_id"];
	        this.name = source["name"];
	        this.prompt_template = source["prompt_template"];
	        this.state_prompts = source["state_prompts"];
	    }
	}

}

export namespace project_lifecycle {
	
	export class BackupEntry {
	    source_path: string;
	    backup_path: string;
	    size: number;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_path = source["source_path"];
	        this.backup_path = source["backup_path"];
	        this.size = source["size"];
	        this.action = source["action"];
	    }
	}
	export class PurgeBackupManifest {
	    backup_id: string;
	    project_id: string;
	    timestamp: string;
	    root: string;
	    entries: BackupEntry[];
	
	    static createFrom(source: any = {}) {
	        return new PurgeBackupManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backup_id = source["backup_id"];
	        this.project_id = source["project_id"];
	        this.timestamp = source["timestamp"];
	        this.root = source["root"];
	        this.entries = this.convertValues(source["entries"], BackupEntry);
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

export namespace remote_bridge {
	
	export class DetectResult {
	    channel: string;
	    matched: boolean;
	    url_type: string;
	    hint_label: string;
	
	    static createFrom(source: any = {}) {
	        return new DetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.channel = source["channel"];
	        this.matched = source["matched"];
	        this.url_type = source["url_type"];
	        this.hint_label = source["hint_label"];
	    }
	}

}

export namespace review {
	
	export class ArchivedCard {
	    id: string;
	    risk_class: string;
	    level: string;
	    status: string;
	    source_type: string;
	    source_id: string;
	    plain_reason: string;
	    engineer_reason: string;
	    reject_reason?: string;
	    created_at: string;
	    archived_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ArchivedCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.risk_class = source["risk_class"];
	        this.level = source["level"];
	        this.status = source["status"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.plain_reason = source["plain_reason"];
	        this.engineer_reason = source["engineer_reason"];
	        this.reject_reason = source["reject_reason"];
	        this.created_at = source["created_at"];
	        this.archived_at = source["archived_at"];
	    }
	}
	export class DualStepState {
	    step1_confirmed_at?: string;
	    step2_executed_at?: string;
	    review_id_at_step1?: string;
	    scope_hash_at_step1?: string;
	    risk_policy_hash?: string;
	    tool_registry_hash?: string;
	    target_hash_set?: string;
	    invalidated?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DualStepState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step1_confirmed_at = source["step1_confirmed_at"];
	        this.step2_executed_at = source["step2_executed_at"];
	        this.review_id_at_step1 = source["review_id_at_step1"];
	        this.scope_hash_at_step1 = source["scope_hash_at_step1"];
	        this.risk_policy_hash = source["risk_policy_hash"];
	        this.tool_registry_hash = source["tool_registry_hash"];
	        this.target_hash_set = source["target_hash_set"];
	        this.invalidated = source["invalidated"];
	    }
	}
	export class Card {
	    review_id: string;
	    created_at: string;
	    risk_class: string;
	    operation: string;
	    target: string;
	    reason: string;
	    accept_label: string;
	    reject_label: string;
	    accept_effect: string;
	    reject_effect: string;
	    rollback_available: boolean;
	    backup_available: boolean;
	    requires_dual_step: boolean;
	    cooldown_seconds: number;
	    source_type?: string;
	    source_id?: string;
	    engineer_reason?: string;
	    log_location?: string;
	    resolved: boolean;
	    resolved_at?: string;
	    dual_step_state?: DualStepState;
	    level?: string;
	
	    static createFrom(source: any = {}) {
	        return new Card(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.review_id = source["review_id"];
	        this.created_at = source["created_at"];
	        this.risk_class = source["risk_class"];
	        this.operation = source["operation"];
	        this.target = source["target"];
	        this.reason = source["reason"];
	        this.accept_label = source["accept_label"];
	        this.reject_label = source["reject_label"];
	        this.accept_effect = source["accept_effect"];
	        this.reject_effect = source["reject_effect"];
	        this.rollback_available = source["rollback_available"];
	        this.backup_available = source["backup_available"];
	        this.requires_dual_step = source["requires_dual_step"];
	        this.cooldown_seconds = source["cooldown_seconds"];
	        this.source_type = source["source_type"];
	        this.source_id = source["source_id"];
	        this.engineer_reason = source["engineer_reason"];
	        this.log_location = source["log_location"];
	        this.resolved = source["resolved"];
	        this.resolved_at = source["resolved_at"];
	        this.dual_step_state = this.convertValues(source["dual_step_state"], DualStepState);
	        this.level = source["level"];
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
	
	export class LightweightCard {
	    review_id: string;
	    operation: string;
	    target: string;
	    accept_label: string;
	    reject_label: string;
	    details?: Record<string, any>;
	    created_at: string;
	    resolved: boolean;
	    resolved_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new LightweightCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.review_id = source["review_id"];
	        this.operation = source["operation"];
	        this.target = source["target"];
	        this.accept_label = source["accept_label"];
	        this.reject_label = source["reject_label"];
	        this.details = source["details"];
	        this.created_at = source["created_at"];
	        this.resolved = source["resolved"];
	        this.resolved_at = source["resolved_at"];
	    }
	}
	export class ReviewExecutionContext {
	    review_id: string;
	    scope_hash: string;
	    risk_policy_hash: string;
	    tool_registry_hash: string;
	    target_hash_set: string;
	
	    static createFrom(source: any = {}) {
	        return new ReviewExecutionContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.review_id = source["review_id"];
	        this.scope_hash = source["scope_hash"];
	        this.risk_policy_hash = source["risk_policy_hash"];
	        this.tool_registry_hash = source["tool_registry_hash"];
	        this.target_hash_set = source["target_hash_set"];
	    }
	}

}

export namespace scheduler {
	
	export class Job {
	    id: string;
	    name: string;
	    cron_expr: string;
	    enabled: boolean;
	    action_type: string;
	    action_payload: string;
	    last_fired: string;
	    next_fire: string;
	    created_at: string;
	    consecutive_failures: number;
	    risk_class: string;
	    payload_hash: string;
	    project_id: string;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.cron_expr = source["cron_expr"];
	        this.enabled = source["enabled"];
	        this.action_type = source["action_type"];
	        this.action_payload = source["action_payload"];
	        this.last_fired = source["last_fired"];
	        this.next_fire = source["next_fire"];
	        this.created_at = source["created_at"];
	        this.consecutive_failures = source["consecutive_failures"];
	        this.risk_class = source["risk_class"];
	        this.payload_hash = source["payload_hash"];
	        this.project_id = source["project_id"];
	    }
	}
	export class JobExecution {
	    job_id: string;
	    fired_at: string;
	    duration_ms: number;
	    status: string;
	    error?: string;
	    retried: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JobExecution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job_id = source["job_id"];
	        this.fired_at = source["fired_at"];
	        this.duration_ms = source["duration_ms"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.retried = source["retried"];
	    }
	}

}

export namespace settings {
	
	export class EmbeddingConfig {
	    providerId?: string;
	    modelId?: string;
	    dimension?: number;
	    pickerDismissed?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EmbeddingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	        this.dimension = source["dimension"];
	        this.pickerDismissed = source["pickerDismissed"];
	    }
	}
	export class PanelSettings {
	    panelLanguage: string;
	    roleLanguage: string;
	    fontPreset: string;
	    fontScale: string;
	    panelStyle: string;
	
	    static createFrom(source: any = {}) {
	        return new PanelSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.panelLanguage = source["panelLanguage"];
	        this.roleLanguage = source["roleLanguage"];
	        this.fontPreset = source["fontPreset"];
	        this.fontScale = source["fontScale"];
	        this.panelStyle = source["panelStyle"];
	    }
	}
	export class Persona {
	    id: string;
	    name: string;
	    icon: string;
	    avatarUrl: string;
	    identity: string;
	    replyStrategy: string;
	    roleStrength: string;
	    personality: string;
	    scenario: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new Persona(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.avatarUrl = source["avatarUrl"];
	        this.identity = source["identity"];
	        this.replyStrategy = source["replyStrategy"];
	        this.roleStrength = source["roleStrength"];
	        this.personality = source["personality"];
	        this.scenario = source["scenario"];
	        this.description = source["description"];
	    }
	}
	export class SummaryModelSettings {
	    source: string;
	    modelId: string;
	    endpoint: string;
	    alwaysUse: boolean;
	    manualModel: string;
	    manualEndpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new SummaryModelSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.modelId = source["modelId"];
	        this.endpoint = source["endpoint"];
	        this.alwaysUse = source["alwaysUse"];
	        this.manualModel = source["manualModel"];
	        this.manualEndpoint = source["manualEndpoint"];
	    }
	}
	export class State {
	    panel: PanelSettings;
	    personas: Persona[];
	    activePersonaId: string;
	    summaryModel: SummaryModelSettings;
	    controlSeal: controlseal.Settings;
	    adapterModelChoices?: Record<string, string>;
	    embeddingConfig?: EmbeddingConfig;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.panel = this.convertValues(source["panel"], PanelSettings);
	        this.personas = this.convertValues(source["personas"], Persona);
	        this.activePersonaId = source["activePersonaId"];
	        this.summaryModel = this.convertValues(source["summaryModel"], SummaryModelSettings);
	        this.controlSeal = this.convertValues(source["controlSeal"], controlseal.Settings);
	        this.adapterModelChoices = source["adapterModelChoices"];
	        this.embeddingConfig = this.convertValues(source["embeddingConfig"], EmbeddingConfig);
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

export namespace skill_step {
	
	export class CLIResponse {
	    text: string;
	    error?: string;
	    auth_required?: boolean;
	    auth_url?: string;
	    adapter_id?: string;
	    action?: string;
	    target?: string;
	    next?: string;
	
	    static createFrom(source: any = {}) {
	        return new CLIResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.error = source["error"];
	        this.auth_required = source["auth_required"];
	        this.auth_url = source["auth_url"];
	        this.adapter_id = source["adapter_id"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.next = source["next"];
	    }
	}
	export class Candidate {
	    SkillID: string;
	    Score: number;
	    Risk: string;
	    Reason: string;
	
	    static createFrom(source: any = {}) {
	        return new Candidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SkillID = source["SkillID"];
	        this.Score = source["Score"];
	        this.Risk = source["Risk"];
	        this.Reason = source["Reason"];
	    }
	}
	export class ExpectedStep {
	    action: string;
	    target: string;
	    next?: string;
	    code: string;
	    requirement: string;
	
	    static createFrom(source: any = {}) {
	        return new ExpectedStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.target = source["target"];
	        this.next = source["next"];
	        this.code = source["code"];
	        this.requirement = source["requirement"];
	    }
	}
	export class ExpectedChain {
	    schema: string;
	    max_steps: number;
	    steps: ExpectedStep[];
	
	    static createFrom(source: any = {}) {
	        return new ExpectedChain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.max_steps = source["max_steps"];
	        this.steps = this.convertValues(source["steps"], ExpectedStep);
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
	
	export class Injection {
	    injection_id: string;
	    skill_id: string;
	    reason: string;
	    summary_hash: string;
	    risk: string;
	    allowed_use: string[];
	    blocked_use: string[];
	    resource_refs: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new Injection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.injection_id = source["injection_id"];
	        this.skill_id = source["skill_id"];
	        this.reason = source["reason"];
	        this.summary_hash = source["summary_hash"];
	        this.risk = source["risk"];
	        this.allowed_use = source["allowed_use"];
	        this.blocked_use = source["blocked_use"];
	        this.resource_refs = source["resource_refs"];
	    }
	}
	export class Lifecycle {
	    status: string;
	    visible_in_toolbar: boolean;
	    route_as_candidate: boolean;
	    auto_execute: boolean;
	    created_from_trace?: string;
	    user_confirmed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Lifecycle(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.visible_in_toolbar = source["visible_in_toolbar"];
	        this.route_as_candidate = source["route_as_candidate"];
	        this.auto_execute = source["auto_execute"];
	        this.created_from_trace = source["created_from_trace"];
	        this.user_confirmed = source["user_confirmed"];
	    }
	}
	export class ResolveResult {
	    ResolveID: string;
	    SessionID: string;
	    Status: string;
	    SelectedSkillID: string;
	    Candidates: Candidate[];
	    ReviewRequired: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResolveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ResolveID = source["ResolveID"];
	        this.SessionID = source["SessionID"];
	        this.Status = source["Status"];
	        this.SelectedSkillID = source["SelectedSkillID"];
	        this.Candidates = this.convertValues(source["Candidates"], Candidate);
	        this.ReviewRequired = source["ReviewRequired"];
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
	export class SkillRouting {
	    action_patterns: string[];
	    target_aliases: string[];
	    minimum_auto_score: number;
	
	    static createFrom(source: any = {}) {
	        return new SkillRouting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action_patterns = source["action_patterns"];
	        this.target_aliases = source["target_aliases"];
	        this.minimum_auto_score = source["minimum_auto_score"];
	    }
	}
	export class SkillResources {
	    examples: string[];
	    programs: string[];
	    cli_md: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillResources(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.examples = source["examples"];
	        this.programs = source["programs"];
	        this.cli_md = source["cli_md"];
	    }
	}
	export class SkillPermissions {
	    network: string;
	    filesystem: string;
	    execution: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillPermissions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.network = source["network"];
	        this.filesystem = source["filesystem"];
	        this.execution = source["execution"];
	    }
	}
	export class SkillTags {
	    purpose_tag: string[];
	    action_tag: string[];
	    domain_tag: string[];
	    risk_tag: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillTags(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purpose_tag = source["purpose_tag"];
	        this.action_tag = source["action_tag"];
	        this.domain_tag = source["domain_tag"];
	        this.risk_tag = source["risk_tag"];
	    }
	}
	export class SkillSource {
	    source_type: string;
	    original_path_hash: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_type = source["source_type"];
	        this.original_path_hash = source["original_path_hash"];
	    }
	}
	export class SkillManifest {
	    schema_version: string;
	    skill_id: string;
	    display_name: string;
	    version: string;
	    description_doc: string;
	    source: SkillSource;
	    tags: SkillTags;
	    permissions: SkillPermissions;
	    resources: SkillResources;
	    routing: SkillRouting;
	    lifecycle?: Lifecycle;
	    expected_chain?: ExpectedChain;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.skill_id = source["skill_id"];
	        this.display_name = source["display_name"];
	        this.version = source["version"];
	        this.description_doc = source["description_doc"];
	        this.source = this.convertValues(source["source"], SkillSource);
	        this.tags = this.convertValues(source["tags"], SkillTags);
	        this.permissions = this.convertValues(source["permissions"], SkillPermissions);
	        this.resources = this.convertValues(source["resources"], SkillResources);
	        this.routing = this.convertValues(source["routing"], SkillRouting);
	        this.lifecycle = this.convertValues(source["lifecycle"], Lifecycle);
	        this.expected_chain = this.convertValues(source["expected_chain"], ExpectedChain);
	        this.hash = source["hash"];
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

export namespace source_trust {
	
	export class SourceTrustEvidence {
	    source_url: string;
	    canonical_hostname: string;
	    domain_class: string;
	    source_trust_label: string;
	    visual_flags?: string[];
	    content_flags?: string[];
	    allowlist_status: string;
	    ranking_score: number;
	    auth_ok: boolean;
	    review_required: boolean;
	    review_reason?: string;
	    is_high_impact: boolean;
	    warning_tokens?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SourceTrustEvidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_url = source["source_url"];
	        this.canonical_hostname = source["canonical_hostname"];
	        this.domain_class = source["domain_class"];
	        this.source_trust_label = source["source_trust_label"];
	        this.visual_flags = source["visual_flags"];
	        this.content_flags = source["content_flags"];
	        this.allowlist_status = source["allowlist_status"];
	        this.ranking_score = source["ranking_score"];
	        this.auth_ok = source["auth_ok"];
	        this.review_required = source["review_required"];
	        this.review_reason = source["review_reason"];
	        this.is_high_impact = source["is_high_impact"];
	        this.warning_tokens = source["warning_tokens"];
	    }
	}

}

export namespace statusrail {
	
	export class View {
	    text: string;
	    layer: string;
	    degraded: boolean;
	    lockedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new View(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.layer = source["layer"];
	        this.degraded = source["degraded"];
	        this.lockedCount = source["lockedCount"];
	    }
	}

}

export namespace stop_recovery {
	
	export class ActionOption {
	    action: string;
	    label: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.label = source["label"];
	        this.description = source["description"];
	    }
	}
	export class ResumeCondition {
	    description: string;
	    met: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResumeCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.met = source["met"];
	    }
	}
	export class StopRecoveryCard {
	    id: string;
	    stop_reason: string;
	    detected_signal: string;
	    safe_next_actions: ActionOption[];
	    resume_conditions: ResumeCondition[];
	    user_message: string;
	    created_at: string;
	    resolved: boolean;
	    resolved_action: string;
	    resolved_at: string;
	
	    static createFrom(source: any = {}) {
	        return new StopRecoveryCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.stop_reason = source["stop_reason"];
	        this.detected_signal = source["detected_signal"];
	        this.safe_next_actions = this.convertValues(source["safe_next_actions"], ActionOption);
	        this.resume_conditions = this.convertValues(source["resume_conditions"], ResumeCondition);
	        this.user_message = source["user_message"];
	        this.created_at = source["created_at"];
	        this.resolved = source["resolved"];
	        this.resolved_action = source["resolved_action"];
	        this.resolved_at = source["resolved_at"];
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

export namespace subexport {
	
	export class ExportResult {
	    ExportDir: string;
	    NewSystemCode: string;
	    ManifestPath: string;
	    DelegationLogOp: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ExportDir = source["ExportDir"];
	        this.NewSystemCode = source["NewSystemCode"];
	        this.ManifestPath = source["ManifestPath"];
	        this.DelegationLogOp = source["DelegationLogOp"];
	    }
	}
	export class ManifestTool {
	    type: string;
	    path: string;
	    original_id: string;
	
	    static createFrom(source: any = {}) {
	        return new ManifestTool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.path = source["path"];
	        this.original_id = source["original_id"];
	    }
	}
	export class ToolConflict {
	    original_id: string;
	    type: string;
	    export_path: string;
	    system_path: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original_id = source["original_id"];
	        this.type = source["type"];
	        this.export_path = source["export_path"];
	        this.system_path = source["system_path"];
	    }
	}

}

export namespace taborder {
	
	export class TabOrder {
	    main_handler: string;
	    sub_order: string[];
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new TabOrder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.main_handler = source["main_handler"];
	        this.sub_order = source["sub_order"];
	        this.updated_at = source["updated_at"];
	    }
	}

}

export namespace tools {
	
	export class ActionResult {
	    toolId: string;
	    ok: boolean;
	    message: string;
	    kind: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.toolId = source["toolId"];
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.kind = source["kind"];
	        this.target = source["target"];
	    }
	}
	export class Tool {
	    id: string;
	    icon: string;
	    title: string;
	    detail: string;
	    kind: string;
	    target: string;
	    enabled: boolean;
	    action_tags?: string[];
	    available: boolean;
	    needs_reauth: boolean;
	    unavailable_reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new Tool(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.icon = source["icon"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.kind = source["kind"];
	        this.target = source["target"];
	        this.enabled = source["enabled"];
	        this.action_tags = source["action_tags"];
	        this.available = source["available"];
	        this.needs_reauth = source["needs_reauth"];
	        this.unavailable_reason = source["unavailable_reason"];
	    }
	}

}

export namespace visual_learning {
	
	export class BBox {
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	
	    static createFrom(source: any = {}) {
	        return new BBox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	    }
	}
	export class CanonicalLabel {
	    element_type: string;
	    action_semantic: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new CanonicalLabel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.element_type = source["element_type"];
	        this.action_semantic = source["action_semantic"];
	        this.description = source["description"];
	    }
	}
	export class RegionProposal {
	    bbox: BBox;
	    raw_score: number;
	    proposal_id: string;
	
	    static createFrom(source: any = {}) {
	        return new RegionProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bbox = this.convertValues(source["bbox"], BBox);
	        this.raw_score = source["raw_score"];
	        this.proposal_id = source["proposal_id"];
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
	export class DetectorResult {
	    proposals: RegionProposal[];
	    degraded: boolean;
	    reason?: string;
	    backend: string;
	
	    static createFrom(source: any = {}) {
	        return new DetectorResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proposals = this.convertValues(source["proposals"], RegionProposal);
	        this.degraded = source["degraded"];
	        this.reason = source["reason"];
	        this.backend = source["backend"];
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
	export class InferenceStatus {
	    available: boolean;
	    backend: string;
	    model_path?: string;
	    degraded: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new InferenceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.backend = source["backend"];
	        this.model_path = source["model_path"];
	        this.degraded = source["degraded"];
	        this.reason = source["reason"];
	    }
	}
	export class OCRResult {
	    text: string;
	    confidence: number;
	    source: string;
	    bbox?: number[];
	
	    static createFrom(source: any = {}) {
	        return new OCRResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	        this.bbox = source["bbox"];
	    }
	}
	export class OCRStatus {
	    available: boolean;
	    platform: string;
	    source: string;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new OCRStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.platform = source["platform"];
	        this.source = source["source"];
	        this.reason = source["reason"];
	    }
	}
	export class PixelBBox {
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	
	    static createFrom(source: any = {}) {
	        return new PixelBBox(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	    }
	}
	export class PixelPoint {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new PixelPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	
	export class WindowsButtonCandidate {
	    id: string;
	    source: string;
	    bbox: PixelBBox;
	    confidence: number;
	    contains_click: boolean;
	    click_distance: number;
	    center_distance: number;
	    area_reasonable: boolean;
	    selection_score: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowsButtonCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.bbox = this.convertValues(source["bbox"], PixelBBox);
	        this.confidence = source["confidence"];
	        this.contains_click = source["contains_click"];
	        this.click_distance = source["click_distance"];
	        this.center_distance = source["center_distance"];
	        this.area_reasonable = source["area_reasonable"];
	        this.selection_score = source["selection_score"];
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
	export class WindowsClickAnchorResult {
	    platform: string;
	    ok: boolean;
	    mode: string;
	    reason?: string;
	    click: PixelPoint;
	    execution_point: PixelPoint;
	    execution_hint: string;
	    anchor_bbox: PixelBBox;
	    crop_bbox: PixelBBox;
	    crop_png_base64?: string;
	    candidates?: WindowsButtonCandidate[];
	    ocr_status: string;
	    ocr_note: string;
	    detector_backend?: string;
	    detector_degraded: boolean;
	    needs_review: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowsClickAnchorResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.ok = source["ok"];
	        this.mode = source["mode"];
	        this.reason = source["reason"];
	        this.click = this.convertValues(source["click"], PixelPoint);
	        this.execution_point = this.convertValues(source["execution_point"], PixelPoint);
	        this.execution_hint = source["execution_hint"];
	        this.anchor_bbox = this.convertValues(source["anchor_bbox"], PixelBBox);
	        this.crop_bbox = this.convertValues(source["crop_bbox"], PixelBBox);
	        this.crop_png_base64 = source["crop_png_base64"];
	        this.candidates = this.convertValues(source["candidates"], WindowsButtonCandidate);
	        this.ocr_status = source["ocr_status"];
	        this.ocr_note = source["ocr_note"];
	        this.detector_backend = source["detector_backend"];
	        this.detector_degraded = source["detector_degraded"];
	        this.needs_review = source["needs_review"];
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

export namespace voice {
	
	export class CommandRoute {
	    matched: boolean;
	    action: string;
	    transcript: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandRoute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched = source["matched"];
	        this.action = source["action"];
	        this.transcript = source["transcript"];
	        this.reason = source["reason"];
	    }
	}
	export class Settings {
	    debugMode: boolean;
	    languageMode: string;
	    manualLanguage: string;
	    commandMode: boolean;
	    whisperBinPath: string;
	    modelPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.debugMode = source["debugMode"];
	        this.languageMode = source["languageMode"];
	        this.manualLanguage = source["manualLanguage"];
	        this.commandMode = source["commandMode"];
	        this.whisperBinPath = source["whisperBinPath"];
	        this.modelPath = source["modelPath"];
	    }
	}
	export class State {
	    settings: Settings;
	    whisperBinPath: string;
	    modelPath: string;
	    managedModelPath: string;
	    whisperAvailable: boolean;
	    modelAvailable: boolean;
	    language: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.whisperBinPath = source["whisperBinPath"];
	        this.modelPath = source["modelPath"];
	        this.managedModelPath = source["managedModelPath"];
	        this.whisperAvailable = source["whisperAvailable"];
	        this.modelAvailable = source["modelAvailable"];
	        this.language = source["language"];
	        this.status = source["status"];
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
	export class TranscriptResult {
	    text: string;
	    language: string;
	    debugSaved: boolean;
	    warning?: string;
	    duration_seconds?: number;
	    audioPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new TranscriptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.language = source["language"];
	        this.debugSaved = source["debugSaved"];
	        this.warning = source["warning"];
	        this.duration_seconds = source["duration_seconds"];
	        this.audioPath = source["audioPath"];
	    }
	}

}

export namespace w3a_media {
	
	export class PollutionReport {
	    high_freq_score: number;
	    histogram_score: number;
	    lsb_score: number;
	    weighted_total: number;
	    is_pollution_risk: boolean;
	    details?: string;
	
	    static createFrom(source: any = {}) {
	        return new PollutionReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.high_freq_score = source["high_freq_score"];
	        this.histogram_score = source["histogram_score"];
	        this.lsb_score = source["lsb_score"];
	        this.weighted_total = source["weighted_total"];
	        this.is_pollution_risk = source["is_pollution_risk"];
	        this.details = source["details"];
	    }
	}
	export class TransferGuidance {
	    recommended: string[];
	    not_recommended: string[];
	    ui_message: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferGuidance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recommended = source["recommended"];
	        this.not_recommended = source["not_recommended"];
	        this.ui_message = source["ui_message"];
	    }
	}

}

