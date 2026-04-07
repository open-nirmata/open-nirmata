"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { listPromptFlows } from "@/lib/api/prompt-flows";
import { validateAgent } from "@/lib/api/agents";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { Agent, AgentPayload } from "@/lib/types";
import { cn } from "@/lib/utils";

const agentTypeOptions = [
    {
        value: "chat",
        label: "Chat",
        help: "Interactive conversation agent that runs through a selected prompt flow.",
    },
] as const;

const agentSchema = z.object({
    name: z.string().trim().min(1, "Agent name is required."),
    description: z.string(),
    enabled: z.boolean(),
    type: z.string().trim().min(1, "Agent type is required."),
    prompt_flow_id: z.string().trim().min(1, "Prompt flow is required."),
});

type AgentFormValues = z.infer<typeof agentSchema>;

type AgentFormProps = {
    mode: "create" | "edit";
    initialValue?: Agent;
    isPending?: boolean;
    onSubmit: (payload: AgentPayload) => Promise<void>;
    backHref?: string;
};

type ValidationState = {
    message?: string;
    warnings: string[];
} | null;

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

function trimOrUndefined(value: string) {
    const trimmed = value.trim();
    return trimmed || undefined;
}

function toFormValues(initialValue?: Agent): AgentFormValues {
    return {
        name: initialValue?.name ?? "",
        description: initialValue?.description ?? "",
        enabled: initialValue?.enabled ?? true,
        type: initialValue?.type ?? "chat",
        prompt_flow_id: initialValue?.prompt_flow_id ?? "",
    };
}

function toPayload(values: AgentFormValues): AgentPayload {
    return {
        name: values.name.trim(),
        description: trimOrUndefined(values.description),
        enabled: values.enabled,
        type: (values.type.trim() || "chat") as AgentPayload["type"],
        prompt_flow_id: values.prompt_flow_id.trim(),
    };
}

export function AgentForm({
    mode,
    initialValue,
    isPending = false,
    onSubmit,
    backHref = "/agents",
}: AgentFormProps) {
    const [validationState, setValidationState] = useState<ValidationState>(null);

    const {
        control,
        register,
        handleSubmit,
        formState: { errors },
        getValues,
    } = useForm<AgentFormValues>({
        resolver: zodResolver(agentSchema),
        defaultValues: toFormValues(initialValue),
    });

    const selectedPromptFlowId = useWatch({ control, name: "prompt_flow_id" });
    const selectedType = useWatch({ control, name: "type" });

    const promptFlowsQuery = useQuery({
        queryKey: ["prompt-flows", "agent-form"],
        queryFn: async () => {
            const response = await listPromptFlows();
            return response.data;
        },
    });

    const validateMutation = useMutation({
        mutationFn: validateAgent,
        onSuccess: (response) => {
            const warnings = response.warnings ?? [];
            setValidationState({
                message: response.message || (warnings.length ? "Validation passed with warnings." : "Agent is valid."),
                warnings,
            });

            if (warnings.length > 0) {
                toast.warning(`Validation returned ${warnings.length} warning${warnings.length === 1 ? "" : "s"}.`);
            } else {
                toast.success(response.message || "Agent is valid.");
            }
        },
        onError: (error) => {
            setValidationState(null);
            toast.error(getErrorMessage(error));
        },
    });

    const promptFlows = promptFlowsQuery.data ?? [];
    const selectedPromptFlow = useMemo(
        () => promptFlows.find((flow) => flow.id === selectedPromptFlowId),
        [promptFlows, selectedPromptFlowId],
    );

    const handleValidate = async () => {
        const values = getValues();
        const payload = toPayload(values);
        await validateMutation.mutateAsync(payload);
    };

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>{mode === "create" ? "Create agent" : "Agent settings"}</CardTitle>
                    <CardDescription>
                        Save a chat agent definition and point it at a prompt flow. Validation checks required
                        fields and the referenced prompt flow without writing to the database.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <form className="space-y-6" onSubmit={handleSubmit(async (values) => onSubmit(toPayload(values)))}>
                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="space-y-2 md:col-span-2">
                                <Label htmlFor="name">Name</Label>
                                <Input id="name" placeholder="Customer Support Agent" {...register("name")} />
                                {errors.name ? (
                                    <p className="text-xs text-destructive">{errors.name.message}</p>
                                ) : null}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="type">Type</Label>
                                <Controller
                                    name="type"
                                    control={control}
                                    render={({ field }) => (
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger id="type">
                                                <SelectValue placeholder="Select agent type" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {agentTypeOptions.map((option) => (
                                                    <SelectItem key={option.value} value={option.value}>
                                                        {option.label}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    )}
                                />
                                <p className="text-xs text-muted-foreground">
                                    {agentTypeOptions.find((option) => option.value === selectedType)?.help}
                                </p>
                                {errors.type ? (
                                    <p className="text-xs text-destructive">{errors.type.message}</p>
                                ) : null}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="prompt_flow_id">Prompt flow</Label>
                                <Controller
                                    name="prompt_flow_id"
                                    control={control}
                                    render={({ field }) => (
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger id="prompt_flow_id">
                                                <SelectValue placeholder={promptFlowsQuery.isLoading ? "Loading flows…" : "Select a prompt flow"} />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {promptFlows.map((flow) => (
                                                    <SelectItem key={flow.id} value={flow.id}>
                                                        {flow.name}{flow.enabled ? "" : " (disabled)"}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    )}
                                />
                                {errors.prompt_flow_id ? (
                                    <p className="text-xs text-destructive">{errors.prompt_flow_id.message}</p>
                                ) : null}
                                {promptFlowsQuery.isError ? (
                                    <p className="text-xs text-destructive">{getErrorMessage(promptFlowsQuery.error)}</p>
                                ) : null}
                            </div>

                            <div className="space-y-2 md:col-span-2">
                                <Label htmlFor="description">Description</Label>
                                <Textarea
                                    id="description"
                                    rows={4}
                                    placeholder="Describe what this agent is responsible for."
                                    {...register("description")}
                                />
                            </div>

                            <div className="rounded-xl border bg-muted/20 p-4 md:col-span-2">
                                <div className="flex items-center justify-between gap-3">
                                    <div>
                                        <p className="text-sm font-medium">Enabled</p>
                                        <p className="text-xs text-muted-foreground">
                                            Disable the agent to keep it configured but unavailable.
                                        </p>
                                    </div>
                                    <Controller
                                        name="enabled"
                                        control={control}
                                        render={({ field }) => (
                                            <Switch checked={field.value} onCheckedChange={field.onChange} />
                                        )}
                                    />
                                </div>
                            </div>
                        </div>

                        {promptFlows.length === 0 && !promptFlowsQuery.isLoading ? (
                            <div className="rounded-xl border border-dashed bg-muted/20 p-4 text-sm text-muted-foreground">
                                No prompt flows are available yet. Create one before saving an agent. {" "}
                                <Link href="/prompt-flows/new" className="font-medium text-foreground underline underline-offset-4">
                                    Create prompt flow
                                </Link>
                            </div>
                        ) : null}

                        {selectedPromptFlow ? (
                            <Card className="border-dashed">
                                <CardHeader>
                                    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                                        <div>
                                            <CardTitle className="text-base">Selected prompt flow</CardTitle>
                                            <CardDescription>
                                                {selectedPromptFlow.description?.trim() || "No description provided."}
                                            </CardDescription>
                                        </div>
                                        <Badge variant={selectedPromptFlow.enabled ? "default" : "secondary"}>
                                            {selectedPromptFlow.enabled ? "Enabled" : "Disabled"}
                                        </Badge>
                                    </div>
                                </CardHeader>
                                <CardContent className="grid gap-2 text-sm text-muted-foreground md:grid-cols-2">
                                    <p>
                                        <span className="font-medium text-foreground">Name:</span> {selectedPromptFlow.name}
                                    </p>
                                    <p>
                                        <span className="font-medium text-foreground">Entry stage:</span> {selectedPromptFlow.entry_stage_id || "—"}
                                    </p>
                                    <p className="md:col-span-2 font-mono text-[11px]">{selectedPromptFlow.id}</p>
                                </CardContent>
                            </Card>
                        ) : null}

                        {validationState ? (
                            <Card
                                className={cn(
                                    "border-dashed",
                                    validationState.warnings.length > 0 ? "border-amber-500/50" : "border-emerald-500/50",
                                )}
                            >
                                <CardHeader>
                                    <div className="flex items-start gap-3">
                                        {validationState.warnings.length > 0 ? (
                                            <AlertTriangle className="mt-0.5 size-5 text-amber-500" />
                                        ) : (
                                            <CheckCircle2 className="mt-0.5 size-5 text-emerald-500" />
                                        )}
                                        <div>
                                            <CardTitle className="text-base">Validation result</CardTitle>
                                            <CardDescription>{validationState.message}</CardDescription>
                                        </div>
                                    </div>
                                </CardHeader>
                                {validationState.warnings.length > 0 ? (
                                    <CardContent>
                                        <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                                            {validationState.warnings.map((warning) => (
                                                <li key={warning}>{warning}</li>
                                            ))}
                                        </ul>
                                    </CardContent>
                                ) : null}
                            </Card>
                        ) : null}

                        <div className="flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
                            <Link href={backHref} className={cn(buttonVariants({ variant: "ghost" }), "justify-start px-0")}>
                                ← Back to agents
                            </Link>
                            <div className="flex flex-col gap-2 sm:flex-row">
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={handleValidate}
                                    disabled={validateMutation.isPending || isPending}
                                >
                                    {validateMutation.isPending ? (
                                        <>
                                            <Loader2 className="mr-2 size-4 animate-spin" />
                                            Validating…
                                        </>
                                    ) : (
                                        "Validate"
                                    )}
                                </Button>
                                <Button type="submit" disabled={isPending}>
                                    {isPending ? (
                                        <>
                                            <Loader2 className="mr-2 size-4 animate-spin" />
                                            Saving…
                                        </>
                                    ) : mode === "create" ? (
                                        "Create agent"
                                    ) : (
                                        "Save changes"
                                    )}
                                </Button>
                            </div>
                        </div>
                    </form>
                </CardContent>
            </Card>
        </div>
    );
}
