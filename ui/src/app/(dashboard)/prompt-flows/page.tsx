"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { PromptFlowTable } from "@/components/prompt-flows/prompt-flow-table";
import { buttonVariants } from "@/components/ui/button";
import {
    Card,
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
import { copyPromptFlow, deletePromptFlow, listPromptFlows } from "@/lib/api/prompt-flows";
import type { PromptFlow, PromptFlowCopyPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function PromptFlowsPage() {
    const router = useRouter();
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");

    const flowsQuery = useQuery({
        queryKey: ["prompt-flows", search, enabledFilter],
        queryFn: async () => {
            const response = await listPromptFlows({
                q: search.trim() || undefined,
                enabled: enabledFilter === "all" ? undefined : enabledFilter,
            });
            return response.data;
        },
    });

    const copyMutation = useMutation({
        mutationFn: ({ id, payload }: { id: string; payload?: PromptFlowCopyPayload }) =>
            copyPromptFlow(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Prompt flow copied successfully.");
            await queryClient.invalidateQueries({ queryKey: ["prompt-flows"] });

            if (response.data?.id) {
                router.push(`/prompt-flows/${response.data.id}`);
            }
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: deletePromptFlow,
        onSuccess: (response) => {
            toast.success(response.message || "Prompt flow deleted successfully.");
            queryClient.invalidateQueries({ queryKey: ["prompt-flows"] });
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const flows = flowsQuery.data ?? [];

    const stats = {
        total: flows.length,
        enabled: flows.filter((f) => f.enabled).length,
        stages: flows.reduce((sum, f) => sum + (f.stages?.length ?? 0), 0),
    };

    const handleCopy = async (flow: PromptFlow) => {
        const suggestedName = flow.name?.trim() ? `${flow.name} (Copy)` : "Copied flow";
        const name = window.prompt("Name for the copied prompt flow:", suggestedName);

        if (name === null) {
            return;
        }

        await copyMutation.mutateAsync({
            id: flow.id,
            payload: {
                name: name.trim() || undefined,
                description: flow.description?.trim() || undefined,
            },
        });
    };

    const handleDelete = async (flow: PromptFlow) => {
        const confirmed = window.confirm(`Delete "${flow.name}"?`);
        if (!confirmed) return;
        await deleteMutation.mutateAsync(flow.id);
    };

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">Prompt flows</h2>
                    <p className="text-sm text-muted-foreground">
                        Design multi-stage conversational workflows for your AI agents.
                    </p>
                </div>
                <Link
                    href="/prompt-flows/new"
                    className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                >
                    <Plus className="mr-1 size-4" />
                    New flow
                </Link>
            </section>

            <section className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="pb-2">
                        <CardDescription>Total flows</CardDescription>
                        <CardTitle className="text-3xl">{stats.total}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardDescription>Enabled</CardDescription>
                        <CardTitle className="text-3xl">{stats.enabled}</CardTitle>
                    </CardHeader>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardDescription>Total stages</CardDescription>
                        <CardTitle className="text-3xl">{stats.stages}</CardTitle>
                    </CardHeader>
                </Card>
            </section>

            <section className="flex flex-col gap-3 sm:flex-row sm:items-center">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        placeholder="Search flows…"
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="pl-8"
                    />
                </div>
                <Select
                    value={enabledFilter}
                    onValueChange={(v) => setEnabledFilter(v as typeof enabledFilter)}
                >
                    <SelectTrigger className="w-40">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">All statuses</SelectItem>
                        <SelectItem value="true">Enabled only</SelectItem>
                        <SelectItem value="false">Disabled only</SelectItem>
                    </SelectContent>
                </Select>
            </section>

            <PromptFlowTable
                items={flows}
                isLoading={flowsQuery.isLoading}
                error={flowsQuery.isError ? getErrorMessage(flowsQuery.error) : null}
                onCopy={handleCopy}
                onDelete={handleDelete}
                isActionPending={copyMutation.isPending || deleteMutation.isPending}
            />
        </div>
    );
}
