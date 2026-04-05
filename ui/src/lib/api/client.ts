const DEFAULT_API_BASE_URL = "http://localhost:4050";

export const API_BASE_URL = (
    process.env.NEXT_PUBLIC_API_BASE_URL || DEFAULT_API_BASE_URL
).replace(/\/$/, "");

type ErrorPayload = {
    message?: string;
};

export async function apiFetch<T>(
    path: string,
    init: RequestInit = {},
): Promise<T> {
    const headers = new Headers(init.headers);

    if (init.body && !headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
    }

    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...init,
        headers,
    });

    const raw = await response.text();
    let payload: unknown = null;

    if (raw) {
        try {
            payload = JSON.parse(raw) as unknown;
        } catch {
            payload = raw;
        }
    }

    if (!response.ok) {
        const message =
            typeof payload === "object" &&
                payload !== null &&
                "message" in payload &&
                typeof (payload as ErrorPayload).message === "string"
                ? (payload as ErrorPayload).message
                : `Request failed with status ${response.status}`;

        throw new Error(message);
    }

    return payload as T;
}
