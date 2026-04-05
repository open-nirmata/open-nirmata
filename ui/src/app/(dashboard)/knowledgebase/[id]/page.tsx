"use client";

import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { KnowledgebaseForm } from "@/components/knowledgebases/knowledgebase-form";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    deleteKnowledgebase,
    getKnowledgebase,
    updateKnowledgebase,
} from "@/lib/api/knowledgebases";
import type { KnowledgebasePayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function KnowledgebaseDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const queryClient = useQueryClient();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    const knowledgebaseQuery = useQuery({
        queryKey: ["knowledgebase", id],
        queryFn: async () => {
            const response = await getKnowledgebase(id);
            if (!response.data) {
                throw new Error("Knowledge base not found.");
            }
            return response.data;
        },
    });

    const updateMutation = useMutation({
        mutationFn: (payload: Partial<KnowledgebasePayload>) => updateKnowledgebase(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Knowledge base updated successfully.");
            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["knowledgebases"] }),
                queryClient.invalidateQueries({ queryKey: ["knowledgebase", id] }),
            ]);
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteKnowledgebase(id),
        onSuccess: async (response) => {
            toast.success(response.message || "Knowledge base deleted successfully.");
            await queryClient.invalidateQueries({ queryKey: ["knowledgebases"] });
            router.push("/knowledgebase");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    if (knowledgebaseQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading knowledge base…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (knowledgebaseQuery.isError || !knowledgebaseQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load the knowledge base</CardTitle>
                    <CardDescription>{getErrorMessage(knowledgebaseQuery.error)}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    const knowledgebase = knowledgebaseQuery.data;

    const handleDelete = async () => {
        const confirmed = window.confirm(`Delete \"${knowledgebase.name}\"?`);
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
                            <CardTitle>{knowledgebase.name}</CardTitle>
                            <CardDescription>
                                Provider: {knowledgebase.provider} • Auth configured: {knowledgebase.auth_configured ? "yes" : "no"}
                            </CardDescription>
                        </div>
                        <Button
                            type="button"
                            variant="destructive"
                            onClick={handleDelete}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? "Deleting…" : "Delete knowledge base"}
                        </Button>
                    </div>
                </CardHeader>
                <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-2">
                    <p>
                        <span className="font-medium text-foreground">Base URL:</span> {knowledgebase.base_url || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Embedding model:</span> {knowledgebase.embedding_model || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Index name:</span> {knowledgebase.index_name || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Namespace:</span> {knowledgebase.namespace || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Created:</span> {knowledgebase.created_at || "—"}
                    </p>
                    <p>
                        <span className="font-medium text-foreground">Updated:</span> {knowledgebase.updated_at || "—"}
                    </p>
                </CardContent>
            </Card>

            <KnowledgebaseForm
                mode="edit"
                initialValue={knowledgebase}
                isPending={updateMutation.isPending}
                onSubmit={async (payload) => {
                    await updateMutation.mutateAsync(payload);
                }}
            />
        </div>
    );
}
