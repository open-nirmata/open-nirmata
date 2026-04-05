"use client";

import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ToolForm } from "@/components/tools/tool-form";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { deleteTool, getTool, updateTool } from "@/lib/api/tools";
import type { ToolPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function ToolDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const queryClient = useQueryClient();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    const toolQuery = useQuery({
        queryKey: ["tool", id],
        queryFn: async () => {
            const response = await getTool(id);
            if (!response.data) {
                throw new Error("Tool not found.");
            }
            return response.data;
        },
    });

    const updateMutation = useMutation({
        mutationFn: (payload: Partial<ToolPayload>) => updateTool(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Tool updated successfully.");
            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["tools"] }),
                queryClient.invalidateQueries({ queryKey: ["tool", id] }),
            ]);
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteTool(id),
        onSuccess: async (response) => {
            toast.success(response.message || "Tool deleted successfully.");
            await queryClient.invalidateQueries({ queryKey: ["tools"] });
            router.push("/tools");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    if (toolQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading tool…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (toolQuery.isError || !toolQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load the tool</CardTitle>
                    <CardDescription>{getErrorMessage(toolQuery.error)}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    const tool = toolQuery.data;

    const handleDelete = async () => {
        const confirmed = window.confirm(`Delete \"${tool.name}\"?`);
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
                            <CardTitle>{tool.name}</CardTitle>
                            <CardDescription>
                                Type: {tool.type} • Auth configured: {tool.auth_configured ? "yes" : "no"}
                            </CardDescription>
                        </div>
                        <Button
                            type="button"
                            variant="destructive"
                            onClick={handleDelete}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? "Deleting…" : "Delete tool"}
                        </Button>
                    </div>
                </CardHeader>
                <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <p>
                        <span className="font-medium text-foreground">Created:</span> {tool.created_at || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Updated:</span> {tool.updated_at || "—"}
                    </p>
                </CardContent>
            </Card>

            <ToolForm
                mode="edit"
                initialValue={tool}
                isPending={updateMutation.isPending}
                onSubmit={async (payload) => {
                    await updateMutation.mutateAsync(payload);
                }}
            />
        </div>
    );
}
