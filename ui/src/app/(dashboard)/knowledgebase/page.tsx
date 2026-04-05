"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { KnowledgebaseTable } from "@/components/knowledgebases/knowledgebase-table";
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
import {
    deleteKnowledgebase,
    listKnowledgebases,
} from "@/lib/api/knowledgebases";
import type { Knowledgebase, KnowledgebaseProviderKind } from "@/lib/types";
import { cn } from "@/lib/utils";

const providerOptions: KnowledgebaseProviderKind[] = [
    "milvus",
    "mixedbread",
    "zeroentropy",
    "algolia",
    "qdrant",
];

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function KnowledgebasePage() {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [providerFilter, setProviderFilter] = useState<"all" | KnowledgebaseProviderKind>("all");
    const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");

    const knowledgebasesQuery = useQuery({
        queryKey: ["knowledgebases", search, providerFilter, enabledFilter],
        queryFn: async () => {
            const response = await listKnowledgebases({
                q: search.trim() || undefined,
                provider: providerFilter === "all" ? undefined : providerFilter,
                enabled: enabledFilter === "all" ? undefined : enabledFilter,
            });

            return response.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: deleteKnowledgebase,
        onSuccess: (response) => {
            toast.success(response.message || "Knowledge base deleted successfully.");
            queryClient.invalidateQueries({ queryKey: ["knowledgebases"] });
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const knowledgebases = knowledgebasesQuery.data ?? [];

    const stats = {
        total: knowledgebases.length,
        enabled: knowledgebases.filter((knowledgebase) => knowledgebase.enabled).length,
        auth: knowledgebases.filter((knowledgebase) => knowledgebase.auth_configured).length,
    };

    const handleDelete = async (knowledgebase: Knowledgebase) => {
        const confirmed = window.confirm(`Delete \"${knowledgebase.name}\"?`);
        if (!confirmed) {
            return;
        }

        await deleteMutation.mutateAsync(knowledgebase.id);
    };

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">Knowledge bases</h2>
                    <p className="text-sm text-muted-foreground">
                        Manage retrieval backends, index settings, and embedding defaults for agent knowledge.
                    </p>
                </div>
                <Link
                    href="/knowledgebase/new"
                    className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                >
                    <Plus className="mr-1 size-4" />
                    New knowledge base
                </Link>
            </section>

            <section className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader>
                        <CardDescription>Total knowledge bases</CardDescription>
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
                        <CardDescription>Auth configured</CardDescription>
                        <CardTitle className="text-3xl">{stats.auth}</CardTitle>
                    </CardHeader>
                </Card>
            </section>

            <Card>
                <CardHeader>
                    <CardTitle>Filters</CardTitle>
                    <CardDescription>
                        The UI uses the backend’s `q`, `provider`, and `enabled` filters directly.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 lg:grid-cols-[1.4fr_0.9fr_0.8fr_auto]">
                    <div className="relative">
                        <Search className="absolute top-2.5 left-3 size-4 text-muted-foreground" />
                        <Input
                            value={search}
                            onChange={(event) => setSearch(event.target.value)}
                            placeholder="Search by name, description, index, or namespace"
                            className="pl-9"
                        />
                    </div>

                    <Select
                        value={providerFilter}
                        onValueChange={(value) => setProviderFilter(value as "all" | KnowledgebaseProviderKind)}
                    >
                        <SelectTrigger className="w-full">
                            <SelectValue placeholder="All providers" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All providers</SelectItem>
                            {providerOptions.map((provider) => (
                                <SelectItem key={provider} value={provider}>
                                    {provider}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>

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
                            setProviderFilter("all");
                            setEnabledFilter("all");
                        }}
                    >
                        Reset
                    </Button>
                </CardContent>
            </Card>

            <KnowledgebaseTable
                items={knowledgebases}
                isLoading={knowledgebasesQuery.isLoading}
                error={knowledgebasesQuery.isError ? getErrorMessage(knowledgebasesQuery.error) : null}
                onDelete={handleDelete}
            />
        </div>
    );
}
