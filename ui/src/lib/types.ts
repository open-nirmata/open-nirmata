export type ToolType = "mcp" | "http";
export type LLMProviderKind =
    | "openai"
    | "ollama"
    | "anthropic"
    | "groq"
    | "openrouter"
    | "gemini";
export type KnowledgebaseProviderKind =
    | "milvus"
    | "mixedbread"
    | "zeroentropy"
    | "algolia"
    | "qdrant";

export type JsonObject = Record<string, unknown>;

export interface Tool {
    id: string;
    name: string;
    type: ToolType | string;
    provider?: string;
    description?: string;
    enabled: boolean;
    tags?: string[];
    config?: JsonObject;
    auth_configured: boolean;
    created_at?: string | null;
    updated_at?: string | null;
}

export interface ToolPayload {
    name: string;
    type: ToolType;
    provider?: string;
    description?: string;
    enabled?: boolean;
    tags?: string[];
    config?: JsonObject;
    auth?: JsonObject;
}

export interface ToolListResponse {
    success: boolean;
    data: Tool[];
    count: number;
}

export interface ToolResponse {
    success: boolean;
    data?: Tool;
    message?: string;
}

export interface MCPServerInfo {
    name?: string;
    version?: string;
}

export interface MCPDiscoveredTool {
    name: string;
    description?: string;
    input_schema?: JsonObject;
    annotations?: JsonObject;
}

export interface TestMCPToolPayload {
    config: JsonObject;
    timeout_seconds?: number;
}

export interface TestMCPToolResult {
    transport: string;
    server_url?: string;
    server_info?: MCPServerInfo;
    tools: MCPDiscoveredTool[];
    count: number;
}

export interface TestMCPToolResponse {
    success: boolean;
    data?: TestMCPToolResult;
    message?: string;
}

export interface LLMProvider {
    id: string;
    name: string;
    provider: LLMProviderKind | string;
    description?: string;
    enabled: boolean;
    base_url?: string;
    default_model?: string;
    organization?: string;
    project_id?: string;
    auth_configured: boolean;
    created_at?: string | null;
    updated_at?: string | null;
}

export interface LLMProviderPayload {
    name: string;
    provider: LLMProviderKind;
    description?: string;
    enabled?: boolean;
    base_url?: string;
    default_model?: string;
    api_key?: string;
    organization?: string;
    project_id?: string;
}

export interface LLMProviderListResponse {
    success: boolean;
    data: LLMProvider[];
    count: number;
}

export interface LLMProviderResponse {
    success: boolean;
    data?: LLMProvider;
    message?: string;
}

export interface LLMModel {
    id: string;
    name: string;
    provider: string;
    description?: string;
    owned_by?: string;
    context_window?: number;
    input_token_limit?: number;
    output_token_limit?: number;
    capabilities?: string[];
    raw?: JsonObject;
}

export interface ListLLMProviderModelsPayload {
    llm_provider_id?: string;
    provider?: LLMProviderKind;
    api_key?: string;
    base_url?: string;
    organization?: string;
    project_id?: string;
    timeout_seconds?: number;
}

export interface ListLLMProviderModelsResponse {
    success: boolean;
    data: LLMModel[];
    count: number;
    message?: string;
}

export interface Knowledgebase {
    id: string;
    name: string;
    provider: KnowledgebaseProviderKind | string;
    description?: string;
    enabled: boolean;
    base_url?: string;
    index_name?: string;
    namespace?: string;
    embedding_model?: string;
    config?: JsonObject;
    auth_configured: boolean;
    created_at?: string | null;
    updated_at?: string | null;
}

export interface KnowledgebasePayload {
    name: string;
    provider: KnowledgebaseProviderKind;
    description?: string;
    enabled?: boolean;
    base_url?: string;
    index_name?: string;
    namespace?: string;
    embedding_model?: string;
    api_key?: string;
    config?: JsonObject;
    auth?: JsonObject;
}

export interface KnowledgebaseListResponse {
    success: boolean;
    data: Knowledgebase[];
    count: number;
}

export interface KnowledgebaseResponse {
    success: boolean;
    data?: Knowledgebase;
    message?: string;
}

export type AgentType = "chat";

export interface Agent {
    id: string;
    name: string;
    description?: string;
    enabled: boolean;
    type: AgentType | string;
    prompt_flow_id: string;
    created_at?: string | null;
    updated_at?: string | null;
}

export interface AgentPayload {
    name: string;
    description?: string;
    enabled?: boolean;
    type: AgentType;
    prompt_flow_id: string;
}

export interface AgentListResponse {
    success: boolean;
    data: Agent[];
    count: number;
}

export interface AgentResponse {
    success: boolean;
    data?: Agent;
    message?: string;
    warnings?: string[];
}

export interface HealthResponse {
    success: boolean;
    version?: string;
    message?: string;
}

// ─── Executions ──────────────────────────────────────────────────────────────

export type ExecutionStatus = "running" | "completed" | "failed" | "cancelled";

export interface ExecutionToolCallItem {
    id: string;
    tool_name: string;
    arguments: JsonObject;
}

export interface ExecutionMessageItem {
    role: string;
    content?: string;
    tool_calls?: ExecutionToolCallItem[];
    tool_call_id?: string;
    name?: string;
}

export interface ExecuteAgentPayload {
    message?: string;
    messages?: ExecutionMessageItem[];
    stream?: boolean;
    metadata?: JsonObject;
}

export interface ExecutionToolCallResult {
    id: string;
    tool_id?: string;
    tool_name: string;
    tool_type?: string;
    arguments: JsonObject;
    result?: string;
    error?: string;
    started_at?: string | null;
    completed_at?: string | null;
    latency_ms?: number;
}

export interface ExecutionLLMMetadataItem {
    provider_id: string;
    provider: string;
    model: string;
    temperature?: number | null;
    max_tokens?: number | null;
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
    latency_ms?: number;
    finish_reason?: string;
}

export interface ExecutionStepItem {
    id: string;
    stage_id: string;
    stage_name: string;
    stage_type: string;
    started_at?: string | null;
    completed_at?: string | null;
    status: string;
    input_messages?: ExecutionMessageItem[];
    output_message?: ExecutionMessageItem | null;
    llm_calls?: ExecutionLLMMetadataItem[];
    tool_calls?: ExecutionToolCallResult[];
    retrieved_context?: string[];
    next_stage_id?: string;
    transition_reason?: string;
    error?: string;
    metadata?: JsonObject;
}

export interface ExecutionItem {
    id: string;
    agent_id: string;
    prompt_flow_id: string;
    status: ExecutionStatus | string;
    input: JsonObject;
    steps?: ExecutionStepItem[];
    final_output?: string;
    error?: string;
    created_at?: string | null;
    updated_at?: string | null;
    completed_at?: string | null;
    total_latency_ms?: number;
    metadata?: JsonObject;
}

export interface ExecuteAgentResponse {
    success: boolean;
    data?: ExecutionItem;
    job_id?: string;
    message?: string;
}

export interface GetExecutionResponse {
    success: boolean;
    data?: ExecutionItem;
    message?: string;
}

export interface ExecutionListResponse {
    success: boolean;
    data: ExecutionItem[];
    count: number;
    message?: string;
}

export type ExecutionStreamEventType =
    | "stage_start"
    | "llm_token"
    | "llm_complete"
    | "tool_call"
    | "tool_result"
    | "stage_complete"
    | "execution_complete"
    | "error";

export interface ExecutionStreamEvent<TData = unknown> {
    event: ExecutionStreamEventType | string;
    data: TData;
}

// ─── Prompt Flows ────────────────────────────────────────────────────────────

export type PromptFlowStageType = "llm" | "tool" | "retrieval" | "router" | "result";

export interface PromptFlowResources {
    llm_provider_id?: string;
    model?: string;
    system_prompt?: string;
    temperature?: number;
    tool_ids?: string[];
    knowledgebase_ids?: string[];
}

export interface PromptFlowTransition {
    label?: string;
    condition?: string;
    target_stage_id: string;
}

export interface PromptFlowStage {
    id: string;
    name: string;
    type: PromptFlowStageType | string;
    description?: string;
    prompt?: string;
    enabled?: boolean;
    on_success?: string;
    overrides?: PromptFlowResources;
    transitions?: PromptFlowTransition[];
}

export interface PromptFlow {
    id: string;
    name: string;
    description?: string;
    enabled: boolean;
    defaults?: PromptFlowResources;
    entry_stage_id?: string;
    include_conversation_history?: boolean;
    stages?: PromptFlowStage[];
    created_at?: string | null;
    updated_at?: string | null;
}

export interface PromptFlowPayload {
    name: string;
    description?: string;
    enabled?: boolean;
    defaults?: PromptFlowResources;
    entry_stage_id?: string;
    include_conversation_history?: boolean;
    stages: PromptFlowStage[];
}

export interface PromptFlowUpdatePayload {
    name?: string;
    description?: string;
    enabled?: boolean;
    defaults?: PromptFlowResources;
    entry_stage_id?: string;
    include_conversation_history?: boolean;
    stages?: PromptFlowStage[];
}

export interface PromptFlowCopyPayload {
    name?: string;
    description?: string;
}

export interface PromptFlowListResponse {
    success: boolean;
    data: PromptFlow[];
    count: number;
}

export interface PromptFlowResponse {
    success: boolean;
    data?: PromptFlow;
    message?: string;
}

export interface PromptFlowResolvedStage {
    id: string;
    name: string;
    type: string;
    enabled: boolean;
    effective?: PromptFlowResources;
    transition_count: number;
}

export interface PromptFlowValidateResult {
    valid: boolean;
    entry_stage_id?: string;
    stages: PromptFlowResolvedStage[];
    warnings?: string[];
}

export interface PromptFlowValidateResponse {
    success: boolean;
    data?: PromptFlowValidateResult;
    message?: string;
}
