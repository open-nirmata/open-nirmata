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

export interface HealthResponse {
    success: boolean;
    version?: string;
    message?: string;
}

// ─── Prompt Flows ────────────────────────────────────────────────────────────

export type PromptFlowStageType = "chat" | "tool" | "retrieval" | "router";

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
    stages: PromptFlowStage[];
}

export interface PromptFlowUpdatePayload {
    name?: string;
    description?: string;
    enabled?: boolean;
    defaults?: PromptFlowResources;
    entry_stage_id?: string;
    stages?: PromptFlowStage[];
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
