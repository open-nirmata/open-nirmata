"use client";

import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ProviderForm } from "@/components/providers/provider-form";
import { createProvider } from "@/lib/api/llm-providers";
import type { LLMProviderPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function NewProviderPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    const createMutation = useMutation({
        mutationFn: createProvider,
        onSuccess: async (response) => {
            toast.success(response.message || "Provider created successfully.");
            await queryClient.invalidateQueries({ queryKey: ["providers"] });

            if (response.data?.id) {
                router.push(`/providers/${response.data.id}`);
                return;
            }

            router.push("/providers");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleSubmit = async (payload: LLMProviderPayload) => {
        await createMutation.mutateAsync(payload);
    };

    return (
        <ProviderForm
            mode="create"
            isPending={createMutation.isPending}
            onSubmit={handleSubmit}
        />
    );
}
