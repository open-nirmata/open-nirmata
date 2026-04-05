"use client";

import { useQuery } from "@tanstack/react-query";

import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { API_BASE_URL } from "@/lib/api/client";
import { getHealth } from "@/lib/api/health";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function HealthPage() {
    const healthQuery = useQuery({
        queryKey: ["health"],
        queryFn: getHealth,
        retry: 0,
    });

    return (
        <div className="space-y-6">
            <div>
                <h2 className="text-xl font-semibold tracking-tight">API health</h2>
                <p className="text-sm text-muted-foreground">
                    Quick verification that the frontend can reach your Go backend.
                </p>
            </div>

            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader>
                        <CardDescription>Status</CardDescription>
                        <CardTitle className="text-2xl">
                            {healthQuery.isSuccess && healthQuery.data.success ? "Healthy" : "Waiting"}
                        </CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader>
                        <CardDescription>Version</CardDescription>
                        <CardTitle className="text-2xl">{healthQuery.data?.version || "—"}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader>
                        <CardDescription>API base URL</CardDescription>
                        <CardTitle className="text-base">{API_BASE_URL}</CardTitle>
                    </CardHeader>
                </Card>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Connection details</CardTitle>
                    <CardDescription>
                        This page calls `GET /health` from the configured API base URL.
                    </CardDescription>
                </CardHeader>
                <CardContent className="text-sm text-muted-foreground">
                    {healthQuery.isLoading ? (
                        <p>Checking the backend…</p>
                    ) : null}

                    {healthQuery.isError ? (
                        <p className="text-destructive">{getErrorMessage(healthQuery.error)}</p>
                    ) : null}

                    {healthQuery.isSuccess ? (
                        <pre className="overflow-x-auto rounded-lg bg-muted p-3 text-xs text-foreground">
                            {JSON.stringify(healthQuery.data, null, 2)}
                        </pre>
                    ) : null}
                </CardContent>
            </Card>
        </div>
    );
}
