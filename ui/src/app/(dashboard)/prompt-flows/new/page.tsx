"use client";

import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { PromptFlowForm } from "@/components/prompt-flows/prompt-flow-form";
import { createPromptFlow } from "@/lib/api/prompt-flows";
import type { PromptFlowPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function NewPromptFlowPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    const createMutation = useMutation({
        mutationFn: createPromptFlow,
        onSuccess: async (response) => {
            toast.success(response.message || "Prompt flow created successfully.");
            await queryClient.invalidateQueries({ queryKey: ["prompt-flows"] });

            if (response.data?.id) {
                router.push(`/prompt-flows/${response.data.id}`);
                return;
            }

            router.push("/prompt-flows");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleSubmit = async (payload: PromptFlowPayload) => {
        await createMutation.mutateAsync(payload);
    };

    return (
        <PromptFlowForm
            mode="create"
            isPending={createMutation.isPending}
            onSubmit={handleSubmit}
        />
    );
}
