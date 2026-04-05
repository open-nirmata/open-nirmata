"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Search } from "lucide-react";
import { toast } from "sonner";

import { ToolTable } from "@/components/tools/tool-table";
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
import { deleteTool, listTools } from "@/lib/api/tools";
import type { Tool, ToolType } from "@/lib/types";
import { cn } from "@/lib/utils";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function ToolsPage() {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [typeFilter, setTypeFilter] = useState<"all" | ToolType>("all");
    const [enabledFilter, setEnabledFilter] = useState<"all" | "true" | "false">("all");

    const toolsQuery = useQuery({
        queryKey: ["tools", search, typeFilter, enabledFilter],
        queryFn: async () => {
            const response = await listTools({
                q: search.trim() || undefined,
                type: typeFilter === "all" ? undefined : typeFilter,
                enabled: enabledFilter === "all" ? undefined : enabledFilter,
            });

            return response.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: deleteTool,
        onSuccess: (response) => {
            toast.success(response.message || "Tool deleted successfully.");
            queryClient.invalidateQueries({ queryKey: ["tools"] });
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const tools = toolsQuery.data ?? [];

    const stats = {
        total: tools.length,
        enabled: tools.filter((tool) => tool.enabled).length,
        auth: tools.filter((tool) => tool.auth_configured).length,
    };

    const handleDelete = async (tool: Tool) => {
        const confirmed = window.confirm(`Delete \"${tool.name}\"?`);
        if (!confirmed) {
            return;
        }

        await deleteMutation.mutateAsync(tool.id);
    };

    return (
        <div className="space-y-6">
            <section className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">Tools registry</h2>
                    <p className="text-sm text-muted-foreground">
                        Manage the MCP and HTTP tools available to your AI agents.
                    </p>
                </div>
                <Link
                    href="/tools/new"
                    className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                >
                    <Plus className="mr-1 size-4" />
                    New tool
                </Link>
            </section>

            <section className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader>
                        <CardDescription>Total tools</CardDescription>
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
                        The UI uses the backend’s `q`, `type`, and `enabled` filters directly.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 lg:grid-cols-[1.4fr_0.8fr_0.8fr_auto]">
                    <div className="relative">
                        <Search className="absolute top-2.5 left-3 size-4 text-muted-foreground" />
                        <Input
                            value={search}
                            onChange={(event) => setSearch(event.target.value)}
                            placeholder="Search by name or description"
                            className="pl-9"
                        />
                    </div>

                    <Select value={typeFilter} onValueChange={(value) => setTypeFilter(value as "all" | ToolType)}>
                        <SelectTrigger className="w-full">
                            <SelectValue placeholder="All types" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">All types</SelectItem>
                            <SelectItem value="mcp">MCP</SelectItem>
                            <SelectItem value="http">HTTP</SelectItem>
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
                            setTypeFilter("all");
                            setEnabledFilter("all");
                        }}
                    >
                        Reset
                    </Button>
                </CardContent>
            </Card>

            <ToolTable
                items={tools}
                isLoading={toolsQuery.isLoading}
                error={toolsQuery.isError ? getErrorMessage(toolsQuery.error) : null}
                onDelete={handleDelete}
            />
        </div>
    );
}
