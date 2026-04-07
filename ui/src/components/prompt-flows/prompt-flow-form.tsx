"use client";

import React, { useCallback, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import {
    AlertTriangle,
    CheckCircle2,
    ChevronDown,
    ChevronRight,
    List,
    Loader2,
    Network,
    Plus,
    Trash2,
    XCircle,
} from "lucide-react";
import {
    Controller,
    useFieldArray,
    useForm,
    useWatch,
} from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { validatePromptFlow } from "@/lib/api/prompt-flows";
import { listProviderModels, listProviders } from "@/lib/api/llm-providers";
import { listTools } from "@/lib/api/tools";
import { listKnowledgebases } from "@/lib/api/knowledgebases";
import { FlowGraph } from "@/components/prompt-flows/flow-graph";
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
import type {
    Knowledgebase,
    LLMProvider,
    PromptFlow,
    PromptFlowPayload,
    PromptFlowResources,
    PromptFlowStage,
    PromptFlowValidateResult,
    Tool,
} from "@/lib/types";
import { cn } from "@/lib/utils";

// ─── Zod Schema ─────────────────────────────────────────────────────────────

const transitionSchema = z.object({
    label: z.string(),
    condition: z.string(),
    target_stage_id: z.string(),
});

const stageSchema = z.object({
    id: z.string().trim().min(1, "Stage ID is required."),
    name: z.string().trim().min(1, "Stage name is required."),
    type: z.string(),
    enabled: z.boolean(),
    description: z.string(),
    prompt: z.string(),
    // Overrides flags + values
    ov_llm_provider: z.boolean(),
    ov_llm_provider_value: z.string(),
    ov_model: z.boolean(),
    ov_model_value: z.string(),
    ov_system_prompt: z.boolean(),
    ov_system_prompt_value: z.string(),
    ov_temperature: z.boolean(),
    ov_temperature_value: z.string(),
    ov_tool_ids: z.boolean(),
    ov_tool_ids_value: z.array(z.string()),
    ov_knowledgebase_ids: z.boolean(),
    ov_knowledgebase_ids_value: z.array(z.string()),
    transitions: z.array(transitionSchema),
    on_success: z.string().optional(),
});

const flowSchema = z
    .object({
        name: z.string().trim().min(1, "Flow name is required."),
        description: z.string(),
        enabled: z.boolean(),
        include_conversation_history: z.boolean(),
        entry_stage_id: z.string(),
        def_llm_provider_id: z.string(),
        def_model: z.string(),
        def_system_prompt: z.string(),
        def_temperature: z.string(),
        def_tool_ids: z.array(z.string()),
        def_knowledgebase_ids: z.array(z.string()),
        stages: z.array(stageSchema),
    })
    .superRefine((values, ctx) => {
        // Validate stage IDs are unique
        const ids = values.stages.map((s) => s.id.trim());
        const seen = new Set<string>();
        ids.forEach((id, index) => {
            if (seen.has(id)) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["stages", index, "id"],
                    message: `Stage ID "${id}" is already used.`,
                });
            }
            if (id) seen.add(id);
        });

        // Validate transitions reference existing stages
        const stageIds = new Set(ids.filter(Boolean));
        values.stages.forEach((stage, si) => {
            if (stage.type === "router" && stage.transitions.length === 0) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["stages", si, "transitions"],
                    message: "Router stages must have at least one transition.",
                });
            }
            stage.transitions.forEach((t, ti) => {
                if (!t.target_stage_id.trim()) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        path: ["stages", si, "transitions", ti, "target_stage_id"],
                        message: "Target stage is required.",
                    });
                } else if (!stageIds.has(t.target_stage_id.trim())) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        path: ["stages", si, "transitions", ti, "target_stage_id"],
                        message: `Stage "${t.target_stage_id}" does not exist.`,
                    });
                }
            });
        });
    });

type FlowFormValues = z.infer<typeof flowSchema>;
type StageFormValues = FlowFormValues["stages"][number];

// ─── Helpers ─────────────────────────────────────────────────────────────────

function trimOrUndefined(v: string) {
    const t = v.trim();
    return t || undefined;
}

function parseTemperature(v: string): number | undefined {
    const n = parseFloat(v);
    if (Number.isNaN(n)) return undefined;
    return n;
}

function buildResources(
    llm_provider_id: string,
    model: string,
    system_prompt: string,
    temperature: string,
    tool_ids: string[],
    knowledgebase_ids: string[],
): PromptFlowResources | undefined {
    const r: PromptFlowResources = {};
    if (llm_provider_id.trim()) r.llm_provider_id = llm_provider_id.trim();
    if (model.trim()) r.model = model.trim();
    if (system_prompt.trim()) r.system_prompt = system_prompt.trim();
    const temp = parseTemperature(temperature);
    if (temp !== undefined) r.temperature = temp;
    if (tool_ids.length > 0) r.tool_ids = tool_ids;
    if (knowledgebase_ids.length > 0) r.knowledgebase_ids = knowledgebase_ids;
    return Object.keys(r).length > 0 ? r : undefined;
}

function buildStageOverrides(stage: StageFormValues): PromptFlowResources | undefined {
    const r: PromptFlowResources = {};
    if (stage.ov_llm_provider) r.llm_provider_id = stage.ov_llm_provider_value.trim();
    if (stage.ov_model) r.model = stage.ov_model_value.trim();
    if (stage.ov_system_prompt) r.system_prompt = stage.ov_system_prompt_value.trim();
    if (stage.ov_temperature) {
        const t = parseTemperature(stage.ov_temperature_value);
        if (t !== undefined) r.temperature = t;
    }
    // tool_ids and knowledgebase_ids: include even if empty array (semantics = clear inherited)
    if (stage.ov_tool_ids) r.tool_ids = stage.ov_tool_ids_value;
    if (stage.ov_knowledgebase_ids) r.knowledgebase_ids = stage.ov_knowledgebase_ids_value;
    return Object.keys(r).length > 0 ? r : undefined;
}

function buildPayload(values: FlowFormValues): PromptFlowPayload {
    const defaults = buildResources(
        values.def_llm_provider_id,
        values.def_model,
        values.def_system_prompt,
        values.def_temperature,
        values.def_tool_ids,
        values.def_knowledgebase_ids,
    );

    return {
        name: values.name.trim(),
        description: trimOrUndefined(values.description),
        enabled: values.enabled,
        include_conversation_history: values.include_conversation_history,
        defaults,
        entry_stage_id: trimOrUndefined(values.entry_stage_id),
        stages: values.stages.map((stage): PromptFlowStage => ({
            id: stage.id.trim(),
            name: stage.name.trim(),
            type: stage.type,
            description: trimOrUndefined(stage.description),
            prompt: trimOrUndefined(stage.prompt),
            enabled: stage.enabled,
            overrides: buildStageOverrides(stage),
            on_success: stage.on_success && stage.type !== "router" && stage.type !== "result" ? trimOrUndefined(stage.on_success) : undefined,
            transitions:
                stage.transitions.length > 0
                    ? stage.transitions.map((t) => ({
                        label: trimOrUndefined(t.label),
                        condition: trimOrUndefined(t.condition),
                        target_stage_id: t.target_stage_id.trim(),
                    }))
                    : undefined,
        })),
    };
}

function flowToFormValues(flow: PromptFlow): FlowFormValues {
    const d = flow.defaults ?? {};
    return {
        name: flow.name,
        description: flow.description ?? "",
        enabled: flow.enabled,
        include_conversation_history: flow.include_conversation_history ?? false,
        entry_stage_id: flow.entry_stage_id ?? "",
        def_llm_provider_id: d.llm_provider_id ?? "",
        def_model: d.model ?? "",
        def_system_prompt: d.system_prompt ?? "",
        def_temperature: d.temperature !== undefined ? String(d.temperature) : "",
        def_tool_ids: d.tool_ids ?? [],
        def_knowledgebase_ids: d.knowledgebase_ids ?? [],
        stages: (flow.stages ?? []).map((stage): StageFormValues => {
            const ov = stage.overrides ?? {};
            return {
                id: stage.id,
                name: stage.name,
                type: stage.type,
                enabled: stage.enabled !== false,
                on_success: stage.on_success && stage.type !== "router" && stage.type !== "result" ? trimOrUndefined(stage.on_success) : undefined,
                description: stage.description ?? "",
                prompt: stage.prompt ?? "",
                ov_llm_provider: ov.llm_provider_id !== undefined,
                ov_llm_provider_value: ov.llm_provider_id ?? "",
                ov_model: ov.model !== undefined,
                ov_model_value: ov.model ?? "",
                ov_system_prompt: ov.system_prompt !== undefined,
                ov_system_prompt_value: ov.system_prompt ?? "",
                ov_temperature: ov.temperature !== undefined,
                ov_temperature_value: ov.temperature !== undefined ? String(ov.temperature) : "",
                ov_tool_ids: ov.tool_ids !== undefined,
                ov_tool_ids_value: ov.tool_ids ?? [],
                ov_knowledgebase_ids: ov.knowledgebase_ids !== undefined,
                ov_knowledgebase_ids_value: ov.knowledgebase_ids ?? [],
                transitions: (stage.transitions ?? []).map((t) => ({
                    label: t.label ?? "",
                    condition: t.condition ?? "",
                    target_stage_id: t.target_stage_id,
                })),
            };
        }),
    };
}

const defaultStageFormValues = (): StageFormValues => ({
    id: "",
    name: "",
    type: "llm",
    enabled: true,
    description: "",
    prompt: "",
    ov_llm_provider: false,
    ov_llm_provider_value: "",
    ov_model: false,
    ov_model_value: "",
    ov_system_prompt: false,
    ov_system_prompt_value: "",
    ov_temperature: false,
    ov_temperature_value: "",
    ov_tool_ids: false,
    ov_tool_ids_value: [],
    ov_knowledgebase_ids: false,
    ov_knowledgebase_ids_value: [],
    transitions: [],
});

const defaultFormValues = (): FlowFormValues => ({
    name: "",
    description: "",
    enabled: true,
    include_conversation_history: false,
    entry_stage_id: "",
    def_llm_provider_id: "",
    def_model: "",
    def_system_prompt: "",
    def_temperature: "",
    def_tool_ids: [],
    def_knowledgebase_ids: [],
    stages: [],
});

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

// ─── Sub-components ──────────────────────────────────────────────────────────

function MultiSelect({
    label,
    available,
    selected,
    onChange,
    getLabel,
    getId,
    placeholder = "None selected",
}: {
    label: string;
    available: { id: string; name: string }[];
    selected: string[];
    onChange: (ids: string[]) => void;
    getLabel?: (item: { id: string; name: string }) => string;
    getId?: (item: { id: string; name: string }) => string;
    placeholder?: string;
}) {
    const resolveId = getId ?? ((item) => item.id);
    const resolveLabel = getLabel ?? ((item) => item.name);

    const toggle = useCallback(
        (id: string) => {
            onChange(
                selected.includes(id)
                    ? selected.filter((s) => s !== id)
                    : [...selected, id],
            );
        },
        [selected, onChange],
    );

    return (
        <div className="space-y-2">
            <Label>{label}</Label>
            {available.length === 0 ? (
                <p className="text-xs text-muted-foreground">No {label.toLowerCase()} available.</p>
            ) : (
                <div className="max-h-36 overflow-y-auto rounded-lg border divide-y">
                    {available.map((item) => {
                        const id = resolveId(item);
                        const name = resolveLabel(item);
                        const checked = selected.includes(id);
                        return (
                            <label
                                key={id}
                                className={cn(
                                    "flex cursor-pointer items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-muted",
                                    checked && "bg-muted/50",
                                )}
                            >
                                <input
                                    type="checkbox"
                                    checked={checked}
                                    onChange={() => toggle(id)}
                                    className="size-4 accent-primary"
                                />
                                <span className="truncate">{name}</span>
                                <span className="ml-auto font-mono text-xs text-muted-foreground truncate max-w-32">
                                    {id}
                                </span>
                            </label>
                        );
                    })}
                </div>
            )}
            {selected.length > 0 && (
                <div className="flex flex-wrap gap-1">
                    {selected.map((id) => {
                        const item = available.find((a) => resolveId(a) === id);
                        return (
                            <Badge
                                key={id}
                                variant="secondary"
                                className="cursor-pointer gap-1"
                                onClick={() => toggle(id)}
                            >
                                {item ? resolveLabel(item) : id}
                                <XCircle className="size-3" />
                            </Badge>
                        );
                    })}
                </div>
            )}
            {selected.length === 0 && (
                <p className="text-xs text-muted-foreground">{placeholder}</p>
            )}
        </div>
    );
}

function ModelSelectField({
    control,
    name,
    providerId,
    label,
    placeholder,
}: {
    control: ReturnType<typeof useForm<FlowFormValues>>["control"];
    name: "def_model" | `stages.${number}.ov_model_value`;
    providerId?: string;
    label: string;
    placeholder?: string;
}) {
    const selectedModel = useWatch({ control, name }) ?? "";
    const trimmedProviderId = providerId?.trim() ?? "";

    const modelsQuery = useQuery({
        queryKey: ["provider-models", trimmedProviderId],
        queryFn: () => listProviderModels({ llm_provider_id: trimmedProviderId }),
        enabled: Boolean(trimmedProviderId),
        select: (response) => response.data,
        staleTime: 60_000,
        retry: false,
        refetchOnWindowFocus: false,
    });

    const options = new Map<string, string>();
    for (const model of modelsQuery.data ?? []) {
        if (model.id) {
            options.set(model.id, model.name || model.id);
        }
    }
    if (selectedModel && !options.has(selectedModel)) {
        options.set(selectedModel, selectedModel);
    }

    return (
        <div className="space-y-1.5">
            <div className="flex items-center gap-2">
                <Label>{label}</Label>
                {modelsQuery.isFetching && <Loader2 className="size-3.5 animate-spin text-muted-foreground" />}
            </div>
            <Controller
                control={control}
                name={name}
                render={({ field }) => (
                    <Select
                        value={field.value || undefined}
                        onValueChange={field.onChange}
                        disabled={!trimmedProviderId || modelsQuery.isLoading}
                    >
                        <SelectTrigger className="w-full">
                            <SelectValue
                                placeholder={
                                    !trimmedProviderId
                                        ? "Select provider first"
                                        : modelsQuery.isLoading
                                            ? "Loading models..."
                                            : placeholder ?? "Select model"
                                }
                            />
                        </SelectTrigger>
                        <SelectContent>
                            {options.size > 0 ? (
                                Array.from(options.entries()).map(([value, optionLabel]) => (
                                    <SelectItem key={value} value={value}>
                                        {optionLabel}
                                    </SelectItem>
                                ))
                            ) : (
                                <div className="px-2 py-1.5 text-sm text-muted-foreground">
                                    {trimmedProviderId
                                        ? "No models available for this provider"
                                        : "Select a provider to load models"}
                                </div>
                            )}
                        </SelectContent>
                    </Select>
                )}
            />
            {modelsQuery.isError && (
                <p className="text-xs text-amber-600">
                    Couldn’t load models for the selected provider.
                </p>
            )}
        </div>
    );
}

// ─── Stage Card ───────────────────────────────────────────────────────────────

type StageCardProps = {
    index: number;
    control: ReturnType<typeof useForm<FlowFormValues>>["control"];
    register: ReturnType<typeof useForm<FlowFormValues>>["register"];
    errors: ReturnType<typeof useForm<FlowFormValues>>["formState"]["errors"];
    allStageIds: string[];
    providers: LLMProvider[];
    tools: Tool[];
    knowledgebases: Knowledgebase[];
    isExpanded: boolean;
    isOverridesOpen: boolean;
    onToggleExpand: () => void;
    onToggleOverrides: () => void;
    onRemove: () => void;
    setValue: ReturnType<typeof useForm<FlowFormValues>>["setValue"];
    getValues: ReturnType<typeof useForm<FlowFormValues>>["getValues"];
};

function StageCard({
    index,
    control,
    register,
    errors,
    allStageIds,
    providers,
    tools,
    knowledgebases,
    isExpanded,
    isOverridesOpen,
    onToggleExpand,
    onToggleOverrides,
    onRemove,
    setValue,
    getValues,
}: StageCardProps) {
    const stageErrors = errors.stages?.[index];
    const prefix = `stages.${index}` as const;

    const stageType = useWatch({ control, name: `stages.${index}.type` });
    const stageEnabled = useWatch({ control, name: `stages.${index}.enabled` });
    const stageName = useWatch({ control, name: `stages.${index}.name` });
    const stageId = useWatch({ control, name: `stages.${index}.id` });
    const onSuccess = useWatch({ control, name: `stages.${index}.on_success` });

    const defaultLLMProviderId = useWatch({ control, name: "def_llm_provider_id" });
    const ovLLMProvider = useWatch({ control, name: `stages.${index}.ov_llm_provider` });
    const ovLLMProviderValue = useWatch({ control, name: `stages.${index}.ov_llm_provider_value` });
    const ovModel = useWatch({ control, name: `stages.${index}.ov_model` });
    const ovSystemPrompt = useWatch({ control, name: `stages.${index}.ov_system_prompt` });
    const ovTemperature = useWatch({ control, name: `stages.${index}.ov_temperature` });
    const ovToolIds = useWatch({ control, name: `stages.${index}.ov_tool_ids` });
    const ovKnowledgebaseIds = useWatch({ control, name: `stages.${index}.ov_knowledgebase_ids` });
    const ovToolIdsValue = useWatch({ control, name: `stages.${index}.ov_tool_ids_value` });
    const ovKnowledgebaseIdsValue = useWatch({ control, name: `stages.${index}.ov_knowledgebase_ids_value` });

    const { fields: transFields, append: appendTrans, remove: removeTrans } = useFieldArray({
        control,
        name: `stages.${index}.transitions`,
    });

    return (
        <div className="rounded-xl border bg-card">
            {/* Stage header */}
            <button
                type="button"
                onClick={onToggleExpand}
                className="flex w-full items-center justify-between p-4 text-left"
            >
                <div className="flex items-center gap-3 min-w-0">
                    {isExpanded ? (
                        <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
                    ) : (
                        <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
                    )}
                    <div className="min-w-0">
                        <p className="font-medium truncate">
                            {stageName || <span className="italic text-muted-foreground">Unnamed stage</span>}
                        </p>
                        <p className="text-xs text-muted-foreground font-mono truncate">
                            {stageId || "—"}
                        </p>
                    </div>
                    <div className="flex items-center gap-2 ml-2 shrink-0">
                        <Badge variant="outline" className="uppercase text-xs">
                            {stageType}
                        </Badge>
                        <Badge variant={stageEnabled ? "default" : "secondary"} className="text-xs">
                            {stageEnabled ? "Enabled" : "Disabled"}
                        </Badge>
                    </div>
                </div>
                <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); onRemove(); }}
                    className="ml-3 shrink-0 rounded p-1 text-destructive hover:bg-destructive/10"
                    title="Remove stage"
                >
                    <Trash2 className="size-4" />
                </button>
            </button>

            {isExpanded && (
                <div className="border-t p-4 space-y-4">
                    {/* Basic fields */}
                    <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-1.5">
                            <Label htmlFor={`${prefix}.id`}>
                                Stage ID <span className="text-destructive">*</span>
                            </Label>
                            <Input
                                id={`${prefix}.id`}
                                {...register(`stages.${index}.id`)}
                                placeholder="e.g. triage"
                                className={cn(stageErrors?.id && "border-destructive")}
                            />
                            {stageErrors?.id && (
                                <p className="text-xs text-destructive">{stageErrors.id.message}</p>
                            )}
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor={`${prefix}.name`}>
                                Name <span className="text-destructive">*</span>
                            </Label>
                            <Input
                                id={`${prefix}.name`}
                                {...register(`stages.${index}.name`)}
                                placeholder="e.g. Initial Triage"
                                className={cn(stageErrors?.name && "border-destructive")}
                            />
                            {stageErrors?.name && (
                                <p className="text-xs text-destructive">{stageErrors.name.message}</p>
                            )}
                        </div>
                    </div>

                    <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-1.5">
                            <Label>Type</Label>
                            <Controller
                                control={control}
                                name={`stages.${index}.type`}
                                render={({ field }) => (
                                    <Select value={field.value} onValueChange={field.onChange}>
                                        <SelectTrigger className="w-full">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="llm">LLM</SelectItem>
                                            <SelectItem value="router">Router</SelectItem>
                                            <SelectItem value="tool">Tool</SelectItem>
                                            <SelectItem value="retrieval">Retrieval</SelectItem>
                                            <SelectItem value="result">Result</SelectItem>
                                        </SelectContent>
                                    </Select>
                                )}
                            />
                        </div>
                        <div className="flex items-center gap-3 pt-6">
                            <Controller
                                control={control}
                                name={`stages.${index}.enabled`}
                                render={({ field }) => (
                                    <Switch
                                        checked={field.value}
                                        onCheckedChange={field.onChange}
                                    />
                                )}
                            />
                            <Label>Enabled</Label>
                        </div>
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor={`${prefix}.description`}>Description</Label>
                        <Input
                            id={`${prefix}.description`}
                            {...register(`stages.${index}.description`)}
                            placeholder="Optional stage description"
                        />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor={`${prefix}.prompt`}>Prompt</Label>
                        <Textarea
                            id={`${prefix}.prompt`}
                            {...register(`stages.${index}.prompt`)}
                            placeholder="Stage-specific prompt instructions"
                            rows={3}
                        />
                    </div>

                    {/* Overrides section */}
                    <div className="rounded-lg border">
                        <button
                            type="button"
                            onClick={onToggleOverrides}
                            className="flex w-full items-center gap-2 p-3 text-left text-sm font-medium hover:bg-muted/50"
                        >
                            {isOverridesOpen ? (
                                <ChevronDown className="size-4 text-muted-foreground" />
                            ) : (
                                <ChevronRight className="size-4 text-muted-foreground" />
                            )}
                            Stage Overrides
                            <span className="ml-auto text-xs text-muted-foreground font-normal">
                                Override flow-level defaults for this stage
                            </span>
                        </button>

                        {isOverridesOpen && (
                            <div className="border-t p-4 space-y-4">
                                {/* LLM Provider override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_llm_provider`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_llm_provider`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_llm_provider`}>
                                            Override LLM Provider
                                        </Label>
                                    </div>
                                    {ovLLMProvider && (
                                        <div className="pl-6">
                                            <Controller
                                                control={control}
                                                name={`stages.${index}.ov_llm_provider_value`}
                                                render={({ field }) => (
                                                    <Select value={field.value} onValueChange={field.onChange}>
                                                        <SelectTrigger className="w-full">
                                                            <SelectValue placeholder="Select provider" />
                                                        </SelectTrigger>
                                                        <SelectContent>
                                                            {providers.map((p) => (
                                                                <SelectItem key={p.id} value={p.id}>
                                                                    {p.name}
                                                                </SelectItem>
                                                            ))}
                                                        </SelectContent>
                                                    </Select>
                                                )}
                                            />
                                        </div>
                                    )}
                                </div>

                                {/* Model override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_model`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_model`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_model`}>Override Model</Label>
                                    </div>
                                    {ovModel && (
                                        <div className="pl-6">
                                            <ModelSelectField
                                                control={control}
                                                name={`stages.${index}.ov_model_value`}
                                                providerId={
                                                    ovLLMProvider
                                                        ? ovLLMProviderValue
                                                        : defaultLLMProviderId
                                                }
                                                label="Model"
                                                placeholder="Select model"
                                            />
                                        </div>
                                    )}
                                </div>

                                {/* System prompt override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_system_prompt`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_system_prompt`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_system_prompt`}>
                                            Override System Prompt
                                        </Label>
                                    </div>
                                    {ovSystemPrompt && (
                                        <div className="pl-6">
                                            <Textarea
                                                {...register(`stages.${index}.ov_system_prompt_value`)}
                                                placeholder="System prompt for this stage"
                                                rows={2}
                                            />
                                        </div>
                                    )}
                                </div>

                                {/* Temperature override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_temperature`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_temperature`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_temperature`}>
                                            Override Temperature
                                        </Label>
                                    </div>
                                    {ovTemperature && (
                                        <div className="pl-6">
                                            <Input
                                                {...register(`stages.${index}.ov_temperature_value`)}
                                                placeholder="0.0 – 2.0"
                                                type="number"
                                                min={0}
                                                max={2}
                                                step={0.1}
                                            />
                                        </div>
                                    )}
                                </div>

                                {/* Tool IDs override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_tool_ids`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_tool_ids`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_tool_ids`}>
                                            Override Tools
                                        </Label>
                                        {ovToolIds && (
                                            <span className="text-xs text-amber-600 dark:text-amber-400">
                                                Empty selection clears inherited tools
                                            </span>
                                        )}
                                    </div>
                                    {ovToolIds && (
                                        <div className="pl-6">
                                            <MultiSelect
                                                label="Tools"
                                                available={tools}
                                                selected={ovToolIdsValue ?? []}
                                                onChange={(ids) =>
                                                    setValue(`stages.${index}.ov_tool_ids_value`, ids, {
                                                        shouldDirty: true,
                                                    })
                                                }
                                            />
                                        </div>
                                    )}
                                </div>

                                {/* Knowledgebase IDs override */}
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.ov_knowledgebase_ids`}
                                            render={({ field }) => (
                                                <input
                                                    type="checkbox"
                                                    checked={field.value}
                                                    onChange={field.onChange}
                                                    className="size-4 accent-primary"
                                                    id={`${prefix}.ov_knowledgebase_ids`}
                                                />
                                            )}
                                        />
                                        <Label htmlFor={`${prefix}.ov_knowledgebase_ids`}>
                                            Override Knowledge Bases
                                        </Label>
                                        {ovKnowledgebaseIds && (
                                            <span className="text-xs text-amber-600 dark:text-amber-400">
                                                Empty selection clears inherited knowledge bases
                                            </span>
                                        )}
                                    </div>
                                    {ovKnowledgebaseIds && (
                                        <div className="pl-6">
                                            <MultiSelect
                                                label="Knowledge Bases"
                                                available={knowledgebases}
                                                selected={ovKnowledgebaseIdsValue ?? []}
                                                onChange={(ids) =>
                                                    setValue(
                                                        `stages.${index}.ov_knowledgebase_ids_value`,
                                                        ids,
                                                        { shouldDirty: true },
                                                    )
                                                }
                                            />
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Transitions */}
                    {stageType == "router" && <div className="space-y-3">
                        <div className="flex items-center justify-between">
                            <Label>
                                Transitions
                                <span className="ml-1 text-destructive">*</span>
                            </Label>
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() =>
                                    appendTrans({ label: "", condition: "", target_stage_id: "" })
                                }
                            >
                                <Plus className="mr-1 size-3" />
                                Add transition
                            </Button>
                        </div>
                        {(errors.stages?.[index] as { transitions?: { message?: string } })?.transitions?.message && (
                            <p className="text-xs text-destructive">
                                {(errors.stages?.[index] as { transitions?: { message?: string } }).transitions!.message}
                            </p>
                        )}
                        {transFields.length === 0 && (
                            <p className="text-xs text-muted-foreground">
                                {stageType === "router"
                                    ? "Add at least one transition for a router stage."
                                    : "Optionally add transitions to define next stages."}
                            </p>
                        )}
                        {transFields.map((trans, ti) => {
                            const transErrors = errors.stages?.[index]?.transitions?.[ti];
                            return (
                                <div
                                    key={trans.id}
                                    className="grid gap-2 sm:grid-cols-3 items-start rounded-lg border p-3"
                                >
                                    <div className="space-y-1">
                                        <Label className="text-xs">Label</Label>
                                        <Input
                                            {...register(`stages.${index}.transitions.${ti}.label`)}
                                            placeholder="e.g. Billing"
                                        />
                                    </div>
                                    <div className="space-y-1">
                                        <Label className="text-xs">
                                            Target Stage <span className="text-destructive">*</span>
                                        </Label>
                                        <Controller
                                            control={control}
                                            name={`stages.${index}.transitions.${ti}.target_stage_id`}
                                            render={({ field }) => (
                                                <Select value={field.value} onValueChange={field.onChange}>
                                                    <SelectTrigger
                                                        className={cn(
                                                            "w-full",
                                                            transErrors?.target_stage_id &&
                                                            "border-destructive",
                                                        )}
                                                    >
                                                        <SelectValue placeholder="Select stage" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {allStageIds.filter(Boolean).map((sid) => (
                                                            <SelectItem key={sid} value={sid}>
                                                                {sid}
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>
                                            )}
                                        />
                                        {transErrors?.target_stage_id && (
                                            <p className="text-xs text-destructive">
                                                {transErrors.target_stage_id.message}
                                            </p>
                                        )}
                                    </div>
                                    <div className="flex items-end gap-2">
                                        <div className="flex-1 space-y-1">
                                            <Label className="text-xs">Condition</Label>
                                            <Input
                                                {...register(
                                                    `stages.${index}.transitions.${ti}.condition`,
                                                )}
                                                placeholder="Optional"
                                            />
                                        </div>
                                        <Button
                                            type="button"
                                            variant="ghost"
                                            size="sm"
                                            className="shrink-0 text-destructive hover:text-destructive"
                                            onClick={() => removeTrans(ti)}
                                        >
                                            <Trash2 className="size-4" />
                                        </Button>
                                    </div>
                                </div>
                            );
                        })}
                    </div>}

                    {stageType != "router" && stageType != "result" && <div className="space-y-3">
                        <div className="flex items-center justify-between">
                            <Label>
                                On Success
                                {stageType === "router" && (
                                    <span className="ml-1 text-destructive">*</span>
                                )}
                            </Label>
                        </div>
                        {(errors.stages?.[index] as { on_success?: { message?: string } })?.on_success?.message && (
                            <p className="text-xs text-destructive">
                                {(errors.stages?.[index] as { on_success?: { message?: string } }).on_success!.message}
                            </p>
                        )}
                        <div className="space-y-1">
                            <Label className="text-xs">
                                Target Stage <span className="text-destructive">*</span>
                            </Label>
                            <Controller
                                control={control}
                                name={`stages.${index}.on_success`}
                                render={({ field }) => (
                                    <Select value={field.value} onValueChange={field.onChange}>
                                        <SelectTrigger
                                            className={cn(
                                                "w-full",
                                                (errors.stages?.[index] as { on_success?: { message?: string } })?.on_success?.message &&
                                                "border-destructive",
                                            )}
                                        >
                                            <SelectValue placeholder="Select stage" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {allStageIds.filter(Boolean).map((sid) => (
                                                <SelectItem key={sid} value={sid}>
                                                    {sid}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                )}
                            />
                        </div>
                    </div>}
                </div>
            )}
        </div>
    );
}

// ─── Main Form ────────────────────────────────────────────────────────────────

type PromptFlowFormProps = {
    mode: "create" | "edit";
    initialValue?: PromptFlow;
    isPending?: boolean;
    onSubmit: (payload: PromptFlowPayload) => Promise<void>;
    onDelete?: () => void;
    backHref?: string;
};

export function PromptFlowForm({
    mode,
    initialValue,
    isPending = false,
    onSubmit,
    onDelete,
    backHref = "/prompt-flows",
}: PromptFlowFormProps) {
    // Remote data
    const providersQuery = useQuery({
        queryKey: ["providers"],
        queryFn: () => listProviders({ enabled: "true" }),
        select: (r) => r.data,
    });
    const toolsQuery = useQuery({
        queryKey: ["tools"],
        queryFn: () => listTools({ enabled: "true" }),
        select: (r) => r.data,
    });
    const kbsQuery = useQuery({
        queryKey: ["knowledgebases"],
        queryFn: () => listKnowledgebases({ enabled: "true" }),
        select: (r) => r.data,
    });

    const providers = providersQuery.data ?? [];
    const tools = toolsQuery.data ?? [];
    const knowledgebases = kbsQuery.data ?? [];

    // Validate mutation
    const validateMutation = useMutation({
        mutationFn: validatePromptFlow,
    });

    // Form state
    const {
        control,
        register,
        handleSubmit,
        setValue,
        getValues,
        watch,
        formState: { errors },
    } = useForm<FlowFormValues>({
        resolver: zodResolver(flowSchema),
        defaultValues: initialValue ? flowToFormValues(initialValue) : defaultFormValues(),
    });

    const { fields: stageFields, append: appendStage, remove: removeStage } = useFieldArray({
        control,
        name: "stages",
    });

    // Expanded/collapsed state for stages and their overrides
    const [expandedStages, setExpandedStages] = useState<Set<number>>(new Set());
    const [openOverrides, setOpenOverrides] = useState<Set<number>>(new Set());
    const [validateResult, setValidateResult] = useState<PromptFlowValidateResult | null>(
        null,
    );
    const [validateError, setValidateError] = useState<string | null>(null);
    const [viewMode, setViewMode] = useState<"list" | "graph">("list");

    const allStageIds = watch("stages").map((s) => s.id.trim());
    const defToolIds = watch("def_tool_ids");
    const defKnowledgebaseIds = watch("def_knowledgebase_ids");
    const enabled = watch("enabled");
    const includeConversationHistory = watch("include_conversation_history");

    const toggleStageExpand = (i: number) => {
        setExpandedStages((prev) => {
            const next = new Set(prev);
            next.has(i) ? next.delete(i) : next.add(i);
            return next;
        });
    };

    const toggleOverrides = (i: number) => {
        setOpenOverrides((prev) => {
            const next = new Set(prev);
            next.has(i) ? next.delete(i) : next.add(i);
            return next;
        });
    };

    const handleAddStage = () => {
        const newIndex = stageFields.length;
        appendStage(defaultStageFormValues());
        setExpandedStages((prev) => new Set(prev).add(newIndex));
    };

    const handleRemoveStage = (i: number) => {
        removeStage(i);
        setExpandedStages((prev) => {
            const next = new Set<number>();
            prev.forEach((n) => {
                if (n < i) next.add(n);
                else if (n > i) next.add(n - 1);
            });
            return next;
        });
        setOpenOverrides((prev) => {
            const next = new Set<number>();
            prev.forEach((n) => {
                if (n < i) next.add(n);
                else if (n > i) next.add(n - 1);
            });
            return next;
        });
    };

    const handleNodeClick = (index: number) => {
        // Expand the clicked stage
        setExpandedStages((prev) => new Set(prev).add(index));
        // Scroll to stage after a short delay to allow expansion
        setTimeout(() => {
            const element = document.getElementById(`stage-${index}`);
            element?.scrollIntoView({ behavior: "smooth", block: "center" });
        }, 100);
    };

    const handleValidate = async () => {
        setValidateResult(null);
        setValidateError(null);
        const values = getValues();
        const payload = buildPayload(values);
        try {
            const result = await validateMutation.mutateAsync(payload);
            if (result.data) {
                setValidateResult(result.data);
            } else {
                setValidateError(result.message ?? "Validation failed.");
            }
        } catch (err) {
            setValidateError(getErrorMessage(err));
        }
    };

    const handleFormSubmit = handleSubmit(async (values) => {
        const payload = buildPayload(values);
        await onSubmit(payload);
    });

    return (
        <form onSubmit={handleFormSubmit} noValidate>
            <div className="space-y-6">
                {/* Header */}
                <div className="flex items-center justify-between gap-4">
                    <div>
                        <h2 className="text-xl font-semibold tracking-tight">
                            {mode === "create" ? "Create prompt flow" : "Edit prompt flow"}
                        </h2>
                        <p className="text-sm text-muted-foreground">
                            {mode === "create"
                                ? "Define a multi-stage conversational workflow."
                                : "Update the flow configuration."}
                        </p>
                    </div>
                    <div className="flex items-center gap-2">
                        {mode === "edit" && onDelete && (
                            <Button
                                type="button"
                                variant="destructive"
                                onClick={onDelete}
                                disabled={isPending}
                            >
                                Delete
                            </Button>
                        )}
                        <Link
                            href={backHref}
                            className={cn(buttonVariants({ variant: "outline" }))}
                        >
                            Cancel
                        </Link>
                        <Button
                            type="button"
                            variant="outline"
                            onClick={handleValidate}
                            disabled={validateMutation.isPending}
                        >
                            {validateMutation.isPending ? (
                                <Loader2 className="mr-2 size-4 animate-spin" />
                            ) : null}
                            Validate
                        </Button>
                        <Button type="submit" disabled={isPending}>
                            {isPending && <Loader2 className="mr-2 size-4 animate-spin" />}
                            {mode === "create" ? "Create flow" : "Save changes"}
                        </Button>
                    </div>
                </div>

                {/* ── Flow Metadata ───────────────────────────────────────────────── */}
                <Card>
                    <CardHeader>
                        <CardTitle>Flow details</CardTitle>
                        <CardDescription>
                            Basic information about this prompt flow.
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="grid gap-4 sm:grid-cols-2">
                            <div className="space-y-1.5">
                                <Label htmlFor="name">
                                    Name <span className="text-destructive">*</span>
                                </Label>
                                <Input
                                    id="name"
                                    {...register("name")}
                                    placeholder="Customer Support Flow"
                                    className={cn(errors.name && "border-destructive")}
                                />
                                {errors.name && (
                                    <p className="text-xs text-destructive">{errors.name.message}</p>
                                )}
                            </div>
                            <div className="space-y-3 pt-1 sm:pt-6">
                                <div className="flex items-center gap-3">
                                    <Controller
                                        control={control}
                                        name="enabled"
                                        render={({ field }) => (
                                            <Switch
                                                checked={field.value}
                                                onCheckedChange={field.onChange}
                                            />
                                        )}
                                    />
                                    <Label>
                                        {enabled ? "Enabled" : "Disabled"}
                                    </Label>
                                </div>
                                <div className="flex items-center gap-3">
                                    <Controller
                                        control={control}
                                        name="include_conversation_history"
                                        render={({ field }) => (
                                            <Switch
                                                checked={field.value}
                                                onCheckedChange={field.onChange}
                                            />
                                        )}
                                    />
                                    <div>
                                        <Label>Include conversation history</Label>
                                        <p className="text-xs text-muted-foreground">
                                            {includeConversationHistory
                                                ? "Previous messages will be included during execution."
                                                : "Only the current request will be sent by default."}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="space-y-1.5">
                            <Label htmlFor="description">Description</Label>
                            <Textarea
                                id="description"
                                {...register("description")}
                                placeholder="Optional description of what this flow does"
                                rows={2}
                            />
                        </div>

                        <div className="space-y-1.5">
                            <Label htmlFor="entry_stage_id">Entry stage</Label>
                            <Controller
                                control={control}
                                name="entry_stage_id"
                                render={({ field }) => (
                                    <Select value={field.value} onValueChange={field.onChange}>
                                        <SelectTrigger className="w-full sm:w-72">
                                            <SelectValue placeholder="Defaults to first stage" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {allStageIds.filter(Boolean).map((sid) => (
                                                <SelectItem key={sid} value={sid}>
                                                    {sid}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                )}
                            />
                            <p className="text-xs text-muted-foreground">
                                Leave empty to use the first stage as the entry point.
                            </p>
                        </div>
                    </CardContent>
                </Card>

                {/* ── Flow Defaults ──────────────────────────────────────────────── */}
                <Card>
                    <CardHeader>
                        <CardTitle>Flow defaults</CardTitle>
                        <CardDescription>
                            Shared settings inherited by all stages unless overridden.
                        </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="grid gap-4 sm:grid-cols-2">
                            <div className="space-y-1.5">
                                <Label>LLM Provider</Label>
                                <Controller
                                    control={control}
                                    name="def_llm_provider_id"
                                    render={({ field }) => (
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder="Select provider" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {providers.map((p) => (
                                                    <SelectItem key={p.id} value={p.id}>
                                                        {p.name}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    )}
                                />
                            </div>
                            <ModelSelectField
                                control={control}
                                name="def_model"
                                providerId={watch("def_llm_provider_id")}
                                label="Model"
                                placeholder="Select model"
                            />
                        </div>

                        <div className="space-y-1.5">
                            <Label htmlFor="def_system_prompt">System prompt</Label>
                            <Textarea
                                id="def_system_prompt"
                                {...register("def_system_prompt")}
                                placeholder="You are a helpful assistant."
                                rows={3}
                            />
                        </div>

                        <div className="space-y-1.5">
                            <Label htmlFor="def_temperature">
                                Temperature{" "}
                                <span className="font-normal text-muted-foreground">(0 – 2)</span>
                            </Label>
                            <Input
                                id="def_temperature"
                                {...register("def_temperature")}
                                type="number"
                                step={0.1}
                                min={0}
                                max={2}
                                placeholder="e.g. 0.7"
                                className="w-32"
                            />
                        </div>

                        <MultiSelect
                            label="Tools"
                            available={tools}
                            selected={defToolIds}
                            onChange={(ids) =>
                                setValue("def_tool_ids", ids, { shouldDirty: true })
                            }
                        />

                        <MultiSelect
                            label="Knowledge Bases"
                            available={knowledgebases}
                            selected={defKnowledgebaseIds}
                            onChange={(ids) =>
                                setValue("def_knowledgebase_ids", ids, { shouldDirty: true })
                            }
                        />
                    </CardContent>
                </Card>

                {/* ── Stage Builder ──────────────────────────────────────────────── */}
                <Card>
                    <CardHeader>
                        <div className="flex items-center justify-between gap-4">
                            <div className="flex-1">
                                <CardTitle>Stages</CardTitle>
                                <CardDescription>
                                    Define the stages of this prompt flow. Stages execute in order
                                    unless transitions are defined.
                                </CardDescription>
                            </div>
                            <div className="flex items-center gap-2 shrink-0">
                                <div className="flex rounded-lg border">
                                    <button
                                        type="button"
                                        onClick={() => setViewMode("list")}
                                        className={cn(
                                            "flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors rounded-l-md",
                                            viewMode === "list"
                                                ? "bg-primary text-primary-foreground"
                                                : "hover:bg-muted",
                                        )}
                                        title="List view"
                                    >
                                        <List className="size-4" />
                                        List
                                    </button>
                                    <button
                                        type="button"
                                        onClick={() => setViewMode("graph")}
                                        className={cn(
                                            "flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors rounded-r-md border-l",
                                            viewMode === "graph"
                                                ? "bg-primary text-primary-foreground"
                                                : "hover:bg-muted",
                                        )}
                                        title="Graph view"
                                    >
                                        <Network className="size-4" />
                                        Graph
                                    </button>
                                </div>
                                <Button type="button" variant="outline" onClick={handleAddStage}>
                                    <Plus className="mr-1 size-4" />
                                    Add stage
                                </Button>
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent className="space-y-3">
                        {viewMode === "graph" && stageFields.length > 0 && (
                            <FlowGraph
                                stages={watch("stages")}
                                entryStageId={watch("entry_stage_id")}
                                onStageClick={handleNodeClick}
                                className="mb-6"
                            />
                        )}
                        {stageFields.length === 0 ? (
                            <div className="rounded-lg border-2 border-dashed p-8 text-center">
                                <p className="text-sm text-muted-foreground">
                                    No stages yet. Click &ldquo;Add stage&rdquo; to begin.
                                </p>
                            </div>
                        ) : (
                            stageFields.map((field, index) => (
                                <div key={field.id} id={`stage-${index}`}>
                                    <StageCard
                                        index={index}
                                        control={control}
                                        register={register}
                                        errors={errors}
                                        allStageIds={allStageIds}
                                        providers={providers}
                                        tools={tools}
                                        knowledgebases={knowledgebases}
                                        isExpanded={expandedStages.has(index)}
                                        isOverridesOpen={openOverrides.has(index)}
                                        onToggleExpand={() => toggleStageExpand(index)}
                                        onToggleOverrides={() => toggleOverrides(index)}
                                        onRemove={() => handleRemoveStage(index)}
                                        setValue={setValue}
                                        getValues={getValues}
                                    />
                                </div>
                            ))
                        )}
                    </CardContent>
                </Card>

                {/* ── Validate Result ────────────────────────────────────────────── */}
                {(validateResult || validateError) && (
                    <Card
                        className={cn(
                            "border-2",
                            validateResult?.valid
                                ? "border-green-500/50"
                                : "border-destructive/50",
                        )}
                    >
                        <CardHeader>
                            <div className="flex items-center gap-2">
                                {validateResult?.valid ? (
                                    <CheckCircle2 className="size-5 text-green-500" />
                                ) : (
                                    <XCircle className="size-5 text-destructive" />
                                )}
                                <CardTitle className="text-base">
                                    {validateResult?.valid ? "Flow is valid" : "Validation failed"}
                                </CardTitle>
                            </div>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {validateError && (
                                <p className="text-sm text-destructive">{validateError}</p>
                            )}

                            {validateResult?.warnings && validateResult.warnings.length > 0 && (
                                <div className="space-y-1">
                                    {validateResult.warnings.map((w, i) => (
                                        <div
                                            key={i}
                                            className="flex items-start gap-2 text-sm text-amber-600 dark:text-amber-400"
                                        >
                                            <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                                            {w}
                                        </div>
                                    ))}
                                </div>
                            )}

                            {validateResult?.stages && validateResult.stages.length > 0 && (
                                <div className="space-y-2">
                                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                        Resolved stages
                                    </p>
                                    {validateResult.stages.map((stage) => (
                                        <div
                                            key={stage.id}
                                            className="rounded-lg border p-3 space-y-2"
                                        >
                                            <div className="flex items-center gap-2">
                                                <span className="font-medium text-sm">{stage.name}</span>
                                                <Badge variant="outline" className="uppercase text-xs">
                                                    {stage.type}
                                                </Badge>
                                                <Badge
                                                    variant={stage.enabled ? "default" : "secondary"}
                                                    className="text-xs"
                                                >
                                                    {stage.enabled ? "Enabled" : "Disabled"}
                                                </Badge>
                                                {stage.transition_count > 0 && (
                                                    <Badge variant="secondary" className="text-xs">
                                                        {stage.transition_count} transition
                                                        {stage.transition_count !== 1 ? "s" : ""}
                                                    </Badge>
                                                )}
                                            </div>
                                            {stage.effective && (
                                                <div className="text-xs text-muted-foreground space-y-0.5">
                                                    {stage.effective.llm_provider_id && (
                                                        <p>Provider: {stage.effective.llm_provider_id}</p>
                                                    )}
                                                    {stage.effective.model && (
                                                        <p>Model: {stage.effective.model}</p>
                                                    )}
                                                    {stage.effective.tool_ids && (
                                                        <p>
                                                            Tools:{" "}
                                                            {stage.effective.tool_ids.length === 0
                                                                ? "none"
                                                                : stage.effective.tool_ids.join(", ")}
                                                        </p>
                                                    )}
                                                    {stage.effective.knowledgebase_ids && (
                                                        <p>
                                                            Knowledge bases:{" "}
                                                            {stage.effective.knowledgebase_ids.length === 0
                                                                ? "none"
                                                                : stage.effective.knowledgebase_ids.join(", ")}
                                                        </p>
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                )}

                {/* Bottom action bar */}
                <div className="flex justify-end gap-2">
                    <Link
                        href={backHref}
                        className={cn(buttonVariants({ variant: "outline" }))}
                    >
                        Cancel
                    </Link>
                    <Button
                        type="button"
                        variant="outline"
                        onClick={handleValidate}
                        disabled={validateMutation.isPending}
                    >
                        {validateMutation.isPending ? (
                            <Loader2 className="mr-2 size-4 animate-spin" />
                        ) : null}
                        Validate
                    </Button>
                    <Button type="submit" disabled={isPending}>
                        {isPending && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {mode === "create" ? "Create flow" : "Save changes"}
                    </Button>
                </div>
            </div>
        </form>
    );
}
