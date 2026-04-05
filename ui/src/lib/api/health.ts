import { apiFetch } from "@/lib/api/client";
import type { HealthResponse } from "@/lib/types";

export async function getHealth() {
    return apiFetch<HealthResponse>("/health");
}
