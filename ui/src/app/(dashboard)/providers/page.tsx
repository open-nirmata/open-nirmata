"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { ProviderTable } from "@/components/providers/provider-table";
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
import { deleteProvider, listProviders } from "@/lib/api/llm-providers";
import type { LLMProvider, LLMProviderKind } from "@/lib/types";
import { cn } from "@/lib/utils";

const providerOptions: LLMProviderKind[] = [
    "openai",
    "ollama",
    "anthropic",
    "groq",
    "openrouter",
    "gemini",
];

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function ProvidersPage() {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [providerFilter, setProviderFilter] = useState<"all" | LLMProviderKind>("all");
    const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");

    const providersQuery = useQuery({
        queryKey: ["providers", search, providerFilter, enabledFilter],
        queryFn: async () => {
            const response = await listProviders({
                q: search.trim() || undefined,
                provider: providerFilter === "all" ? undefined : providerFilter,
                enabled: enabledFilter === "all" ? undefined : enabledFilter,
            });

            return response.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: deleteProvider,
        onSuccess: (response) => {
            toast.success(response.message || "Provider deleted successfully.");
            queryClient.invalidateQueries({ queryKey: ["providers"] });
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const providers = providersQuery.data ?? [];

    const stats = {
        total: providers.length,
        enabled: providers.filter((provider) => provider.enabled).length,
        auth: providers.filter((provider) => provider.auth_configured).length,
    };

    const handleDelete = async (provider: LLMProvider) => {
        const confirmed = window.confirm(`Delete \"${provider.name}\"?`);
        if (!confirmed) {
            return;
        }

        await deleteMutation.mutateAsync(provider.id);
    };

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">LLM providers</h2>
                    <p className="text-sm text-muted-foreground">
                        Manage hosted and local model providers available to your Open Nirmata setup.
                    </p>
                </div>
                <Link
                    href="/providers/new"
                    className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                >
                    <Plus className="mr-1 size-4" />
                    New provider
                </Link>
            </section>

            <section className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader>
                        <CardDescription>Total providers</CardDescription>
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
                            placeholder="Search by name, description, or model"
                            className="pl-9"
                        />
                    </div>

                    <Select
                        value={providerFilter}
                        onValueChange={(value) => setProviderFilter(value as "all" | LLMProviderKind)}
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

            <ProviderTable
                items={providers}
                isLoading={providersQuery.isLoading}
                error={providersQuery.isError ? getErrorMessage(providersQuery.error) : null}
                onDelete={handleDelete}
            />
        </div>
    );
}
