"use client";

import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ProviderForm } from "@/components/providers/provider-form";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    deleteProvider,
    getProvider,
    updateProvider,
} from "@/lib/api/llm-providers";
import type { LLMProviderPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function ProviderDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const queryClient = useQueryClient();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    const providerQuery = useQuery({
        queryKey: ["provider", id],
        queryFn: async () => {
            const response = await getProvider(id);
            if (!response.data) {
                throw new Error("Provider not found.");
            }
            return response.data;
        },
    });

    const updateMutation = useMutation({
        mutationFn: (payload: Partial<LLMProviderPayload>) => updateProvider(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Provider updated successfully.");
            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["providers"] }),
                queryClient.invalidateQueries({ queryKey: ["provider", id] }),
            ]);
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteProvider(id),
        onSuccess: async (response) => {
            toast.success(response.message || "Provider deleted successfully.");
            await queryClient.invalidateQueries({ queryKey: ["providers"] });
            router.push("/providers");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    if (providerQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading provider…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (providerQuery.isError || !providerQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load the provider</CardTitle>
                    <CardDescription>{getErrorMessage(providerQuery.error)}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    const provider = providerQuery.data;

    const handleDelete = async () => {
        const confirmed = window.confirm(`Delete \"${provider.name}\"?`);
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
                            <CardTitle>{provider.name}</CardTitle>
                            <CardDescription>
                                Provider: {provider.provider} • Auth configured: {provider.auth_configured ? "yes" : "no"}
                            </CardDescription>
                        </div>
                        <Button
                            type="button"
                            variant="destructive"
                            onClick={handleDelete}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? "Deleting…" : "Delete provider"}
                        </Button>
                    </div>
                </CardHeader>
                <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <p>
                        <span className="font-medium text-foreground">Base URL:</span> {provider.base_url || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Default model:</span> {provider.default_model || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Created:</span> {provider.created_at || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Updated:</span> {provider.updated_at || "—"}
                    </p>
                </CardContent>
            </Card>

            <ProviderForm
                mode="edit"
                initialValue={provider}
                isPending={updateMutation.isPending}
                onSubmit={async (payload) => {
                    await updateMutation.mutateAsync(payload);
                }}
            />
        </div>
    );
}
