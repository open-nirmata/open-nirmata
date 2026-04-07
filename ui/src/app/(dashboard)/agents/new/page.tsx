"use client";

import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { AgentForm } from "@/components/agents/agent-form";
import { createAgent } from "@/lib/api/agents";
import type { AgentPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function NewAgentPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    const createMutation = useMutation({
        mutationFn: createAgent,
        onSuccess: async (response) => {
            toast.success(response.message || "Agent created successfully.");
            if (response.warnings?.length) {
                toast.warning(`${response.warnings.length} warning${response.warnings.length === 1 ? "" : "s"} returned.`);
            }

            await queryClient.invalidateQueries({ queryKey: ["agents"] });

            if (response.data?.id) {
                router.push(`/agents/${response.data.id}`);
                return;
            }

            router.push("/agents");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleSubmit = async (payload: AgentPayload) => {
        await createMutation.mutateAsync(payload);
    };

    return <AgentForm mode="create" isPending={createMutation.isPending} onSubmit={handleSubmit} />;
}
