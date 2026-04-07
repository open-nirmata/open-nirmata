import { apiFetch } from "@/lib/api/client";
import type { AgentListResponse, AgentPayload, AgentResponse } from "@/lib/types";

export type AgentFilters = {
    q?: string;
    enabled?: string;
};

export async function listAgents(filters: AgentFilters = {}) {
    const params = new URLSearchParams();

    if (filters.q) {
        params.set("q", filters.q);
    }

    if (filters.enabled) {
        params.set("enabled", filters.enabled);
    }

    const query = params.toString();
    const path = query ? `/agents?${query}` : "/agents";

    return apiFetch<AgentListResponse>(path);
}

export async function getAgent(id: string) {
    return apiFetch<AgentResponse>(`/agents/${id}`);
}

export async function createAgent(payload: AgentPayload) {
    return apiFetch<AgentResponse>("/agents", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updateAgent(id: string, payload: Partial<AgentPayload>) {
    return apiFetch<AgentResponse>(`/agents/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function validateAgent(payload: AgentPayload) {
    return apiFetch<AgentResponse>("/agents/validate", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function deleteAgent(id: string) {
    return apiFetch<AgentResponse>(`/agents/${id}`, {
        method: "DELETE",
    });
}
