import { API_BASE_URL, apiFetch } from "@/lib/api/client";
import type {
    ExecuteAgentPayload,
    ExecuteAgentResponse,
    ExecutionListResponse,
    ExecutionStreamEvent,
    GetExecutionResponse,
} from "@/lib/types";

export type ExecutionFilters = {
    agent_id?: string;
    status?: string;
    limit?: number;
};

export type StreamAgentExecutionOptions = {
    signal?: AbortSignal;
    onEvent?: (event: ExecutionStreamEvent<unknown>) => void;
};

export async function executeAgent(
    agentId: string,
    payload: ExecuteAgentPayload,
    options: { async?: boolean } = {},
) {
    const params = new URLSearchParams();

    if (options.async) {
        params.set("async", "true");
    }

    if (payload.stream) {
        params.set("stream", "true");
    }

    const query = params.toString();
    const path = query ? `/agents/${agentId}/execute?${query}` : `/agents/${agentId}/execute`;

    return apiFetch<ExecuteAgentResponse>(path, {
        method: "POST",
        body: JSON.stringify(payload),
    });
}

export async function getExecution(id: string) {
    return apiFetch<GetExecutionResponse>(`/executions/${id}`);
}

export async function listExecutions(filters: ExecutionFilters = {}) {
    const params = new URLSearchParams();

    if (filters.agent_id) {
        params.set("agent_id", filters.agent_id);
    }

    if (filters.status) {
        params.set("status", filters.status);
    }

    if (typeof filters.limit === "number") {
        params.set("limit", String(filters.limit));
    }

    const query = params.toString();
    const path = query ? `/executions?${query}` : "/executions";

    return apiFetch<ExecutionListResponse>(path);
}

export async function streamAgentExecution(
    agentId: string,
    payload: ExecuteAgentPayload,
    options: StreamAgentExecutionOptions = {},
) {
    const params = new URLSearchParams({ stream: "true" });
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/execute?${params.toString()}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ ...payload, stream: true }),
        signal: options.signal,
    });

    if (!response.ok) {
        const raw = await response.text();
        let message = `Request failed with status ${response.status}`;

        if (raw) {
            try {
                const payload = JSON.parse(raw) as { message?: string };
                message = payload.message || raw;
            } catch {
                message = raw;
            }
        }

        throw new Error(message);
    }

    if (!response.body) {
        throw new Error("Streaming response did not include a readable body.");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
        const { done, value } = await reader.read();
        buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });

        const chunks = buffer.split(/\r?\n\r?\n/);
        buffer = chunks.pop() ?? "";

        for (const chunk of chunks) {
            const event = parseSseChunk(chunk);
            if (event) {
                options.onEvent?.(event);
            }
        }

        if (done) {
            break;
        }
    }

    if (buffer.trim()) {
        const event = parseSseChunk(buffer);
        if (event) {
            options.onEvent?.(event);
        }
    }
}

function parseSseChunk(chunk: string): ExecutionStreamEvent<unknown> | null {
    const trimmed = chunk.trim();
    if (!trimmed) {
        return null;
    }

    let eventName = "message";
    const dataLines: string[] = [];

    for (const line of trimmed.split(/\r?\n/)) {
        if (line.startsWith("event:")) {
            eventName = line.slice("event:".length).trim() || eventName;
        }

        if (line.startsWith("data:")) {
            dataLines.push(line.slice("data:".length).trim());
        }
    }

    const rawData = dataLines.join("\n");
    if (!rawData) {
        return { event: eventName, data: null };
    }

    try {
        const parsed = JSON.parse(rawData) as ExecutionStreamEvent<unknown>;

        if (parsed && typeof parsed === "object" && "event" in parsed && "data" in parsed) {
            return parsed;
        }

        return { event: eventName, data: parsed };
    } catch {
        return { event: eventName, data: rawData };
    }
}
