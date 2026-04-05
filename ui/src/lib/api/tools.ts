import { apiFetch } from "@/lib/api/client";
import type {
    TestMCPToolPayload,
    TestMCPToolResponse,
    ToolListResponse,
    ToolPayload,
    ToolResponse,
} from "@/lib/types";

export type ToolFilters = {
    q?: string;
    type?: string;
    enabled?: string;
};

export async function listTools(filters: ToolFilters = {}) {
    const params = new URLSearchParams();

    if (filters.q) {
        params.set("q", filters.q);
    }

    if (filters.type) {
        params.set("type", filters.type);
    }

    if (filters.enabled) {
        params.set("enabled", filters.enabled);
    }

    const query = params.toString();
    const path = query ? `/tools?${query}` : "/tools";

    return apiFetch<ToolListResponse>(path);
}

export async function getTool(id: string) {
    return apiFetch<ToolResponse>(`/tools/${id}`);
}

export async function createTool(payload: ToolPayload) {
    return apiFetch<ToolResponse>("/tools", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updateTool(id: string, payload: Partial<ToolPayload>) {
    return apiFetch<ToolResponse>(`/tools/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function testMCPTool(payload: TestMCPToolPayload) {
    return apiFetch<TestMCPToolResponse>("/tools/test", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function deleteTool(id: string) {
    return apiFetch<ToolResponse>(`/tools/${id}`, {
        method: "DELETE",
    });
}
