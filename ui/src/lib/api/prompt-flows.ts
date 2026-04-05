import { apiFetch } from "@/lib/api/client";
import type {
    PromptFlowListResponse,
    PromptFlowPayload,
    PromptFlowResponse,
    PromptFlowValidateResponse,
} from "@/lib/types";

export type PromptFlowFilters = {
    q?: string;
    enabled?: string;
};

export async function listPromptFlows(filters: PromptFlowFilters = {}) {
    const params = new URLSearchParams();

    if (filters.q) {
        params.set("q", filters.q);
    }

    if (filters.enabled) {
        params.set("enabled", filters.enabled);
    }

    const query = params.toString();
    const path = query ? `/prompt-flows?${query}` : "/prompt-flows";

    return apiFetch<PromptFlowListResponse>(path);
}

export async function getPromptFlow(id: string) {
    return apiFetch<PromptFlowResponse>(`/prompt-flows/${id}`);
}

export async function createPromptFlow(payload: PromptFlowPayload) {
    return apiFetch<PromptFlowResponse>("/prompt-flows", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updatePromptFlow(id: string, payload: PromptFlowPayload) {
    return apiFetch<PromptFlowResponse>(`/prompt-flows/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function deletePromptFlow(id: string) {
    return apiFetch<PromptFlowResponse>(`/prompt-flows/${id}`, {
        method: "DELETE",
    });
}

export async function validatePromptFlow(payload: PromptFlowPayload) {
    return apiFetch<PromptFlowValidateResponse>("/prompt-flows/validate", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}
