import { apiFetch } from "@/lib/api/client";
import type {
    LLMProviderListResponse,
    LLMProviderPayload,
    LLMProviderResponse,
} from "@/lib/types";

export type LLMProviderFilters = {
    q?: string;
    provider?: string;
    enabled?: string;
};

export async function listProviders(filters: LLMProviderFilters = {}) {
    const params = new URLSearchParams();

    if (filters.q) {
        params.set("q", filters.q);
    }

    if (filters.provider) {
        params.set("provider", filters.provider);
    }

    if (filters.enabled) {
        params.set("enabled", filters.enabled);
    }

    const query = params.toString();
    const path = query ? `/llm-providers?${query}` : "/llm-providers";

    return apiFetch<LLMProviderListResponse>(path);
}

export async function getProvider(id: string) {
    return apiFetch<LLMProviderResponse>(`/llm-providers/${id}`);
}

export async function createProvider(payload: LLMProviderPayload) {
    return apiFetch<LLMProviderResponse>("/llm-providers", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updateProvider(id: string, payload: Partial<LLMProviderPayload>) {
    return apiFetch<LLMProviderResponse>(`/llm-providers/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function deleteProvider(id: string) {
    return apiFetch<LLMProviderResponse>(`/llm-providers/${id}`, {
        method: "DELETE",
    });
}
