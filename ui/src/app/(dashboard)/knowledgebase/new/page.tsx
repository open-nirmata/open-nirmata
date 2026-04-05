"use client";

import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { KnowledgebaseForm } from "@/components/knowledgebases/knowledgebase-form";
import { createKnowledgebase } from "@/lib/api/knowledgebases";
import type { KnowledgebasePayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function NewKnowledgebasePage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    const createMutation = useMutation({
        mutationFn: createKnowledgebase,
        onSuccess: async (response) => {
            toast.success(response.message || "Knowledge base created successfully.");
            await queryClient.invalidateQueries({ queryKey: ["knowledgebases"] });

            if (response.data?.id) {
                router.push(`/knowledgebase/${response.data.id}`);
                return;
            }

            router.push("/knowledgebase");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleSubmit = async (payload: KnowledgebasePayload) => {
        await createMutation.mutateAsync(payload);
    };

    return (
        <KnowledgebaseForm
            mode="create"
            isPending={createMutation.isPending}
            onSubmit={handleSubmit}
        />
    );
}
