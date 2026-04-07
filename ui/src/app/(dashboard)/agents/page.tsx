"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { AgentTable } from "@/components/agents/agent-table";
import { Button, buttonVariants } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { deleteAgent, listAgents } from "@/lib/api/agents";
import { listPromptFlows } from "@/lib/api/prompt-flows";
import type { Agent } from "@/lib/types";
import { cn } from "@/lib/utils";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function AgentsPage() {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");

    const agentsQuery = useQuery({
        queryKey: ["agents", search, enabledFilter],
        queryFn: async () => {
            const response = await listAgents({
                q: search.trim() || undefined,
                enabled: enabledFilter === "all" ? undefined : enabledFilter,
            });

            return response.data;
        },
    });

    const promptFlowsQuery = useQuery({
        queryKey: ["prompt-flows", "agent-list"],
        queryFn: async () => {
            const response = await listPromptFlows();
            return response.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: deleteAgent,
        onSuccess: (response) => {
            toast.success(response.message || "Agent deleted successfully.");
            queryClient.invalidateQueries({ queryKey: ["agents"] });
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const agents = agentsQuery.data ?? [];

    const promptFlowNames = useMemo(
        () => Object.fromEntries((promptFlowsQuery.data ?? []).map((flow) => [flow.id, flow.name])),
        [promptFlowsQuery.data],
    );

    const stats = {
        total: agents.length,
        enabled: agents.filter((agent) => agent.enabled).length,
        flows: new Set(agents.map((agent) => agent.prompt_flow_id).filter(Boolean)).size,
    };

    const handleDelete = async (agent: Agent) => {
        const confirmed = window.confirm(`Delete \"${agent.name}\"?`);
        if (!confirmed) {
            return;
        }

        await deleteMutation.mutateAsync(agent.id);
    };

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">Agents registry</h2>
                    <p className="text-sm text-muted-foreground">
                        Manage saved chat agents and the prompt flows they depend on.
                    </p>
                </div>
                <Link
                    href="/agents/new"
                    className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                >
                    <Plus className="mr-1 size-4" />
                    New agent
                </Link>
            </section>

            <section className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader>
                        <CardDescription>Total agents</CardDescription>
                        <CardTitle className="text-3xl">{stats.total}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader>
                        <CardDescription>Enabled</CardDescription>
                        <CardTitle className="text-3xl">{stats.enabled}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader>
                        <CardDescription>Prompt flows referenced</CardDescription>
                        <CardTitle className="text-3xl">{stats.flows}</CardTitle>
                    </CardHeader>
                </Card>
            </section>

            <Card>
                <CardHeader>
                    <CardTitle>Filters</CardTitle>
                    <CardDescription>
                        The UI uses the backend’s `q` and `enabled` filters directly.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 lg:grid-cols-[1.4fr_0.8fr_auto]">
                    <div className="relative">
                        <Search className="absolute top-2.5 left-3 size-4 text-muted-foreground" />
                        <Input
                            value={search}
                            onChange={(event) => setSearch(event.target.value)}
                            placeholder="Search by name or description"
                            className="pl-9"
                        />
                    </div>

                    <Select
                        value={enabledFilter}
                        onValueChange={(value) => setEnabledFilter(value as "all" | "true" | "false")}
                    >
                        <SelectTrigger className="w-full">
                            <SelectValue placeholder="Any status" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">Any status</SelectItem>
                            <SelectItem value="true">Enabled</SelectItem>
                            <SelectItem value="false">Disabled</SelectItem>
                        </SelectContent>
                    </Select>

                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => {
                            setSearch("");
                            setEnabledFilter("all");
                        }}
                    >
                        Reset
                    </Button>
                </CardContent>
            </Card>

            <AgentTable
                items={agents}
                promptFlowNames={promptFlowNames}
                isLoading={agentsQuery.isLoading}
                error={agentsQuery.isError ? getErrorMessage(agentsQuery.error) : null}
                onDelete={handleDelete}
            />
        </div>
    );
}
