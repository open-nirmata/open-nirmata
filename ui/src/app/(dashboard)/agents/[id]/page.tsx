"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlayCircle } from "lucide-react";
import { toast } from "sonner";

import { AgentForm } from "@/components/agents/agent-form";
import { Button, buttonVariants } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { deleteAgent, getAgent, updateAgent } from "@/lib/api/agents";
import type { AgentPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function AgentDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const queryClient = useQueryClient();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    const agentQuery = useQuery({
        queryKey: ["agent", id],
        queryFn: async () => {
            const response = await getAgent(id);
            if (!response.data) {
                throw new Error("Agent not found.");
            }
            return response.data;
        },
    });

    const updateMutation = useMutation({
        mutationFn: (payload: Partial<AgentPayload>) => updateAgent(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Agent updated successfully.");
            if (response.warnings?.length) {
                toast.warning(`${response.warnings.length} warning${response.warnings.length === 1 ? "" : "s"} returned.`);
            }

            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["agents"] }),
                queryClient.invalidateQueries({ queryKey: ["agent", id] }),
            ]);
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteAgent(id),
        onSuccess: async (response) => {
            toast.success(response.message || "Agent deleted successfully.");
            await queryClient.invalidateQueries({ queryKey: ["agents"] });
            router.push("/agents");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    if (agentQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading agent…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (agentQuery.isError || !agentQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load the agent</CardTitle>
                    <CardDescription>{getErrorMessage(agentQuery.error)}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    const agent = agentQuery.data;

    const handleDelete = async () => {
        const confirmed = window.confirm(`Delete \"${agent.name}\"?`);
        if (!confirmed) {
            return;
        }

        await deleteMutation.mutateAsync();
    };

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                        <div>
                            <CardTitle>{agent.name}</CardTitle>
                            <CardDescription>
                                Type: {agent.type} • Prompt flow: {agent.prompt_flow_id}
                            </CardDescription>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            <Link
                                href={`/agents/${agent.id}/execute`}
                                className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                            >
                                <PlayCircle className="mr-1 size-4" />
                                Test agent
                            </Link>
                            <Button
                                type="button"
                                variant="destructive"
                                onClick={handleDelete}
                                disabled={deleteMutation.isPending}
                            >
                                {deleteMutation.isPending ? "Deleting…" : "Delete agent"}
                            </Button>
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <p>
                        <span className="font-medium text-foreground">Created:</span> {agent.created_at || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Updated:</span> {agent.updated_at || "—"}
                    </p>
                </CardContent>
            </Card>

            <AgentForm
                mode="edit"
                initialValue={agent}
                isPending={updateMutation.isPending}
                onSubmit={async (payload) => {
                    await updateMutation.mutateAsync(payload);
                }}
            />
        </div>
    );
}
