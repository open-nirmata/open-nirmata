"use client";

import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ToolForm } from "@/components/tools/tool-form";
import { createTool } from "@/lib/api/tools";
import type { ToolPayload } from "@/lib/types";

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

export default function NewToolPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    const createMutation = useMutation({
        mutationFn: createTool,
        onSuccess: async (response) => {
            toast.success(response.message || "Tool created successfully.");
            await queryClient.invalidateQueries({ queryKey: ["tools"] });

            if (response.data?.id) {
                router.push(`/tools/${response.data.id}`);
                return;
            }

            router.push("/tools");
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleSubmit = async (payload: ToolPayload) => {
        await createMutation.mutateAsync(payload);
    };

    return <ToolForm mode="create" isPending={createMutation.isPending} onSubmit={handleSubmit} />;
}
