import { apiFetch } from "@/lib/api/client";
import type {
    KnowledgebaseListResponse,
    KnowledgebasePayload,
    KnowledgebaseResponse,
} from "@/lib/types";

export type KnowledgebaseFilters = {
    q?: string;
    provider?: string;
    enabled?: string;
};

export async function listKnowledgebases(filters: KnowledgebaseFilters = {}) {
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
    const path = query ? `/knowledgebases?${query}` : "/knowledgebases";

    return apiFetch<KnowledgebaseListResponse>(path);
}

export async function getKnowledgebase(id: string) {
    return apiFetch<KnowledgebaseResponse>(`/knowledgebases/${id}`);
}

export async function createKnowledgebase(payload: KnowledgebasePayload) {
    return apiFetch<KnowledgebaseResponse>("/knowledgebases", {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function updateKnowledgebase(
    id: string,
    payload: Partial<KnowledgebasePayload>,
) {
    return apiFetch<KnowledgebaseResponse>(`/knowledgebases/${id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
    });
}

export async function deleteKnowledgebase(id: string) {
    return apiFetch<KnowledgebaseResponse>(`/knowledgebases/${id}`, {
        method: "DELETE",
    });
}
