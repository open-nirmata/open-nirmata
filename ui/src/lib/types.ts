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
