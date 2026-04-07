"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, PlayCircle } from "lucide-react";

import { AgentChat } from "@/components/agents/agent-chat";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { getAgent } from "@/lib/api/agents";
import { cn } from "@/lib/utils";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function AgentExecutePage() {
    const params = useParams<{ id: string }>();
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

    if (agentQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading agent tester…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (agentQuery.isError || !agentQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load the agent tester</CardTitle>
                    <CardDescription>{getErrorMessage(agentQuery.error)}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    const agent = agentQuery.data;

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={agent.enabled ? "default" : "secondary"}>
                            {agent.enabled ? "Enabled" : "Disabled"}
                        </Badge>
                        <Badge variant="outline">{agent.type}</Badge>
                        <Badge variant="outline">Prompt flow: {agent.prompt_flow_id}</Badge>
                    </div>
                    <div>
                        <h2 className="flex items-center gap-2 text-xl font-semibold tracking-tight">
                            <PlayCircle className="size-5" />
                            Agent tester
                        </h2>
                        <p className="text-sm text-muted-foreground">
                            Run chat conversations against `{agent.name}` and inspect the execution trace.
                        </p>
                    </div>
                </div>
                <Link
                    href={`/agents/${agent.id}`}
                    className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                >
                    <ArrowLeft className="mr-1 size-4" />
                    Back to agent
                </Link>
            </section>

            <Card>
                <CardHeader>
                    <CardTitle>{agent.name}</CardTitle>
                    <CardDescription>
                        {agent.description?.trim() || "No description provided yet."}
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2 xl:grid-cols-4">
                    <p>
                        <span className="font-medium text-foreground">Agent ID:</span> {agent.id}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Prompt flow:</span> {agent.prompt_flow_id}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Created:</span> {agent.created_at || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Updated:</span> {agent.updated_at || "—"}
                    </p>
                </CardContent>
            </Card>

            <AgentChat agent={agent} />
        </div>
    );
}
