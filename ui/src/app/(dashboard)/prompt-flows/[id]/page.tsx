"use client";

import { useParams, useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PromptFlowForm } from "@/components/prompt-flows/prompt-flow-form";
import {
    Card,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    deletePromptFlow,
    getPromptFlow,
    updatePromptFlow,
} from "@/lib/api/prompt-flows";
import type { PromptFlowPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function PromptFlowDetailPage() {
    const params = useParams<{ id: string }>();
    const router = useRouter();
    const queryClient = useQueryClient();
    const id = Array.isArray(params.id) ? params.id[0] : params.id;

    const flowQuery = useQuery({
        queryKey: ["prompt-flow", id],
        queryFn: async () => {
            const response = await getPromptFlow(id);
            if (!response.data) {
                throw new Error("Prompt flow not found.");
            }
            return response.data;
        },
    });

    const updateMutation = useMutation({
        mutationFn: (payload: PromptFlowPayload) => updatePromptFlow(id, payload),
        onSuccess: async (response) => {
            toast.success(response.message || "Prompt flow updated successfully.");
            await Promise.all([
                queryClient.invalidateQueries({ queryKey: ["prompt-flows"] }),
                queryClient.invalidateQueries({ queryKey: ["prompt-flow", id] }),
            ]);
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const deleteMutation = useMutation({
        mutationFn: () => deletePromptFlow(id),
        onSuccess: async (response) => {
            toast.success(response.message || "Prompt flow deleted successfully.");
            await queryClient.invalidateQueries({ queryKey: ["prompt-flows"] });
            router.push("/prompt-flows");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleDelete = async () => {
        const confirmed = window.confirm(
            `Delete "${flowQuery.data?.name ?? "this flow"}"? This cannot be undone.`,
        );
        if (!confirmed) return;
        await deleteMutation.mutateAsync();
    };

    if (flowQuery.isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Loading prompt flow…</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    if (flowQuery.isError || !flowQuery.data) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Prompt flow not found</CardTitle>
                </CardHeader>
            </Card>
        );
    }

    return (
        <PromptFlowForm
            mode="edit"
            initialValue={flowQuery.data}
            isPending={updateMutation.isPending || deleteMutation.isPending}
            onSubmit={async (payload) => { await updateMutation.mutateAsync(payload); }}
            onDelete={handleDelete}
        />
    );
}
