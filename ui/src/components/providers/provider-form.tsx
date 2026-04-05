"use client";

import Link from "next/link";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { listProviderModels } from "@/lib/api/llm-providers";
import { Badge } from "@/components/ui/badge";
import { Button, buttonVariants } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
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
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import type {
    LLMModel,
    LLMProvider,
    LLMProviderKind,
    LLMProviderPayload,
    ListLLMProviderModelsPayload,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const providerSchemaBase = z.object({
    name: z.string().trim().min(1, "Name is required."),
    provider: z.enum(["openai", "ollama", "anthropic", "groq", "openrouter", "gemini"]),
    description: z.string(),
    enabled: z.boolean(),
    base_url: z.string(),
    default_model: z.string(),
    api_key: z.string(),
    organization: z.string(),
    project_id: z.string(),
});

type ProviderFormValues = z.infer<typeof providerSchemaBase>;

type ProviderFormProps = {
    mode: "create" | "edit";
    initialValue?: LLMProvider;
    isPending?: boolean;
    onSubmit: (payload: LLMProviderPayload) => Promise<void>;
    backHref?: string;
};

const providerOptions: Array<{
    value: LLMProviderKind;
    label: string;
    help: string;
}> = [
        { value: "openai", label: "OpenAI", help: "Hosted API provider" },
        { value: "ollama", label: "Ollama", help: "Local or self-hosted via base URL" },
        { value: "anthropic", label: "Anthropic", help: "Claude models" },
        { value: "groq", label: "Groq", help: "Fast hosted inference" },
        { value: "openrouter", label: "OpenRouter", help: "Unified hosted routing" },
        { value: "gemini", label: "Gemini", help: "Google Gemini models" },
    ];

function createProviderSchema(mode: "create" | "edit", hasExistingAuth: boolean) {
    return providerSchemaBase.superRefine((values, ctx) => {
        if (values.provider === "ollama" && !values.base_url.trim()) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["base_url"],
                message: "Base URL is required for Ollama.",
            });
        }

        if (
            values.provider !== "ollama" &&
            !values.api_key.trim() &&
            (mode === "create" || !hasExistingAuth)
        ) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["api_key"],
                message: "API key is required for hosted providers.",
            });
        }
    });
}

function normalizeProvider(value?: string): LLMProviderKind {
    if (
        value === "openai" ||
        value === "ollama" ||
        value === "anthropic" ||
        value === "groq" ||
        value === "openrouter" ||
        value === "gemini"
    ) {
        return value;
    }

    return "openai";
}

function trimOrUndefined(value: string) {
    const trimmed = value.trim();
    return trimmed ? trimmed : undefined;
}

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong.";
}

function formatOptionalNumber(value?: number) {
    return typeof value === "number" && Number.isFinite(value) && value > 0
        ? new Intl.NumberFormat().format(value)
        : "—";
}

export function ProviderForm({
    mode,
    initialValue,
    isPending = false,
    onSubmit,
    backHref = "/providers",
}: ProviderFormProps) {
    const hasExistingAuth = Boolean(initialValue?.auth_configured);
    const [isModelsDialogOpen, setIsModelsDialogOpen] = useState(false);

    const form = useForm<ProviderFormValues>({
        resolver: zodResolver(createProviderSchema(mode, hasExistingAuth)),
        defaultValues: {
            name: initialValue?.name ?? "",
            provider: normalizeProvider(initialValue?.provider),
            description: initialValue?.description ?? "",
            enabled: initialValue?.enabled ?? true,
            base_url: initialValue?.base_url ?? "",
            default_model: initialValue?.default_model ?? "",
            api_key: "",
            organization: initialValue?.organization ?? "",
            project_id: initialValue?.project_id ?? "",
        },
    });

    const currentProvider =
        useWatch({ control: form.control, name: "provider" }) ??
        normalizeProvider(initialValue?.provider);
    const currentDefaultModel = useWatch({ control: form.control, name: "default_model" }) ?? "";
    const requiresBaseUrl = currentProvider === "ollama";
    const requiresApiKey = currentProvider !== "ollama";
    const canUseSavedProviderAuth = mode === "edit" && Boolean(initialValue?.id) && hasExistingAuth;
    const submitLabel = mode === "create" ? "Create provider" : "Save changes";

    const modelTestMutation = useMutation({
        mutationFn: listProviderModels,
        onSuccess: (response) => {
            const count = response.count ?? response.data.length;
            setIsModelsDialogOpen(true);
            toast.success(
                response.message ||
                `Connection successful. ${count} model${count === 1 ? "" : "s"} available.`,
            );
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const handleFetchModels = async () => {
        const fieldsToValidate: Array<keyof ProviderFormValues> = ["provider"];

        if (requiresBaseUrl) {
            fieldsToValidate.push("base_url");
        }

        if (requiresApiKey && !canUseSavedProviderAuth) {
            fieldsToValidate.push("api_key");
        }

        const isValid = await form.trigger(fieldsToValidate);
        if (!isValid) {
            return;
        }

        const values = form.getValues();
        const apiKey = trimOrUndefined(values.api_key);
        const baseUrl = trimOrUndefined(values.base_url);
        const organization = trimOrUndefined(values.organization);
        const projectId = trimOrUndefined(values.project_id);

        if (requiresApiKey && !apiKey && !canUseSavedProviderAuth) {
            form.setError("api_key", {
                type: "manual",
                message: "API key is required to fetch models.",
            });
            return;
        }

        const payload: ListLLMProviderModelsPayload = {
            provider: values.provider,
        };

        if (initialValue?.id) {
            payload.llm_provider_id = initialValue.id;
        }

        if (apiKey) {
            payload.api_key = apiKey;
        }

        if (baseUrl) {
            payload.base_url = baseUrl;
        }

        if (organization) {
            payload.organization = organization;
        }

        if (projectId) {
            payload.project_id = projectId;
        }

        modelTestMutation.mutate(payload);
    };

    const fetchedModels = modelTestMutation.data?.data ?? [];
    const availableModelOptions: Array<{ value: string; label: string }> = [];
    const seenModelValues = new Set<string>();

    for (const model of fetchedModels) {
        if (!model.id || seenModelValues.has(model.id)) {
            continue;
        }

        availableModelOptions.push({
            value: model.id,
            label: model.name || model.id,
        });
        seenModelValues.add(model.id);
    }

    if (currentDefaultModel && !seenModelValues.has(currentDefaultModel)) {
        availableModelOptions.push({
            value: currentDefaultModel,
            label: currentDefaultModel,
        });
    }

    const handleSubmit = form.handleSubmit(async (values) => {
        const payload: LLMProviderPayload = {
            name: values.name.trim(),
            provider: values.provider,
            enabled: values.enabled,
        };

        const description = trimOrUndefined(values.description);
        const baseUrl = trimOrUndefined(values.base_url);
        const defaultModel = trimOrUndefined(values.default_model);
        const apiKey = trimOrUndefined(values.api_key);
        const organization = trimOrUndefined(values.organization);
        const projectId = trimOrUndefined(values.project_id);

        if (description) {
            payload.description = description;
        }

        if (baseUrl) {
            payload.base_url = baseUrl;
        }

        if (defaultModel) {
            payload.default_model = defaultModel;
        }

        if (apiKey) {
            payload.api_key = apiKey;
        }

        if (organization) {
            payload.organization = organization;
        }

        if (projectId) {
            payload.project_id = projectId;
        }

        await onSubmit(payload);
    });

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">
                        {mode === "create"
                            ? "Add an LLM provider"
                            : `Edit ${initialValue?.name ?? "provider"}`}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                        Save provider credentials and defaults for model access through the Go API.
                    </p>
                </div>
                <Link
                    href={backHref}
                    className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                >
                    Back to providers
                </Link>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Basic details</CardTitle>
                    <CardDescription>
                        Choose a provider and define the connection details your workspace should use.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="name">Name</Label>
                        <Input id="name" placeholder="Primary OpenAI" {...form.register("name")} />
                        {form.formState.errors.name ? (
                            <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
                        ) : null}
                    </div>

                    <div className="space-y-2">
                        <Label>Provider</Label>
                        <Controller
                            control={form.control}
                            name="provider"
                            render={({ field }) => (
                                <Select
                                    value={field.value}
                                    onValueChange={(value) => field.onChange(value as LLMProviderKind)}
                                >
                                    <SelectTrigger className="w-full">
                                        <SelectValue placeholder="Select a provider" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {providerOptions.map((option) => (
                                            <SelectItem key={option.value} value={option.value}>
                                                {option.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            )}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label>Enabled</Label>
                        <div className="flex min-h-9 items-center justify-between rounded-lg border px-3">
                            <span className="text-sm text-muted-foreground">Allow this provider to be used</span>
                            <Controller
                                control={form.control}
                                name="enabled"
                                render={({ field }) => (
                                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                                )}
                            />
                        </div>
                    </div>

                    <div className="space-y-2 md:col-span-2 rounded-lg border bg-muted/30 p-3 text-sm text-muted-foreground">
                        <p className="font-medium text-foreground">
                            {providerOptions.find((option) => option.value === currentProvider)?.label}
                        </p>
                        <p>
                            {requiresBaseUrl
                                ? "Ollama requires a reachable base URL, for example http://localhost:11434."
                                : providerOptions.find((option) => option.value === currentProvider)?.help}
                        </p>
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="description">Description</Label>
                        <Textarea
                            id="description"
                            rows={4}
                            placeholder="Shared production provider for chat completions."
                            {...form.register("description")}
                        />
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Connection settings</CardTitle>
                    <CardDescription>
                        Configure the endpoint, model defaults, and credentials required by the selected provider.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2">
                        <Label htmlFor="base_url">
                            Base URL {requiresBaseUrl ? <span className="text-destructive">*</span> : null}
                        </Label>
                        <Input
                            id="base_url"
                            placeholder={requiresBaseUrl ? "http://localhost:11434" : "Optional custom endpoint"}
                            {...form.register("base_url")}
                        />
                        {form.formState.errors.base_url ? (
                            <p className="text-xs text-destructive">
                                {form.formState.errors.base_url.message}
                            </p>
                        ) : (
                            <p className="text-xs text-muted-foreground">
                                {requiresBaseUrl
                                    ? "Required for Ollama deployments."
                                    : "Optional unless you are using a custom-compatible endpoint."}
                            </p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label>Default model</Label>
                        <Controller
                            control={form.control}
                            name="default_model"
                            render={({ field }) => (
                                <Select value={field.value || undefined} onValueChange={field.onChange}>
                                    <SelectTrigger className="w-full" disabled={availableModelOptions.length === 0}>
                                        <SelectValue placeholder="Fetch models to choose a default" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {availableModelOptions.length ? (
                                            availableModelOptions.map((option) => (
                                                <SelectItem key={option.value} value={option.value}>
                                                    {option.label}
                                                </SelectItem>
                                            ))
                                        ) : (
                                            <SelectItem value="__no_models" disabled>
                                                Fetch models first
                                            </SelectItem>
                                        )}
                                    </SelectContent>
                                </Select>
                            )}
                        />
                        <p className="text-xs text-muted-foreground">
                            {availableModelOptions.length
                                ? "Choose the default model from the fetched provider list."
                                : "Use “Fetch models” to populate this list for the current credentials."}
                        </p>
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="api_key">
                            API key {requiresApiKey ? <span className="text-destructive">*</span> : null}
                        </Label>
                        <Input
                            id="api_key"
                            type="password"
                            autoComplete="off"
                            placeholder={
                                requiresApiKey
                                    ? mode === "edit" && hasExistingAuth
                                        ? "Leave blank to keep the current key"
                                        : "Paste the provider API key"
                                    : "Not usually required for Ollama"
                            }
                            {...form.register("api_key")}
                        />
                        {form.formState.errors.api_key ? (
                            <p className="text-xs text-destructive">
                                {form.formState.errors.api_key.message}
                            </p>
                        ) : (
                            <p className="text-xs text-muted-foreground">
                                {mode === "edit" && hasExistingAuth
                                    ? "Credentials are write-only. Leave this blank to keep the existing secret; model fetch can reuse the saved key."
                                    : "Credentials are never returned by the API after saving."}
                            </p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="organization">Organization</Label>
                        <Input id="organization" placeholder="org_123" {...form.register("organization")} />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="project_id">Project ID</Label>
                        <Input id="project_id" placeholder="proj_abc" {...form.register("project_id")} />
                    </div>

                    <div className="space-y-4 rounded-lg border bg-muted/20 p-4 md:col-span-2">
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                            <div className="space-y-1">
                                <p className="font-medium text-foreground">Fetch available models</p>
                                <p className="text-xs text-muted-foreground">
                                    Test the current unsaved credentials and list models without saving this provider.
                                </p>
                                {mode === "edit" && hasExistingAuth && requiresApiKey ? (
                                    <p className="text-xs text-muted-foreground">
                                        Leave the API key blank to use the saved credential for this provider, or enter a new one here to override it just for this test.
                                    </p>
                                ) : null}
                            </div>
                            <Button
                                type="button"
                                variant="outline"
                                onClick={handleFetchModels}
                                disabled={isPending || form.formState.isSubmitting || modelTestMutation.isPending}
                            >
                                {modelTestMutation.isPending ? (
                                    <>
                                        <Loader2 className="size-4 animate-spin" />
                                        Fetching…
                                    </>
                                ) : (
                                    "Fetch models"
                                )}
                            </Button>
                        </div>

                        {modelTestMutation.isError ? (
                            <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                                {getErrorMessage(modelTestMutation.error)}
                            </div>
                        ) : null}

                        {modelTestMutation.data ? (
                            <div className="flex flex-col gap-2 rounded-md border bg-background p-3 text-sm sm:flex-row sm:items-center sm:justify-between">
                                <div>
                                    <p className="font-medium text-foreground">
                                        {modelTestMutation.data.count} model{modelTestMutation.data.count === 1 ? "" : "s"} fetched
                                    </p>
                                    <p className="text-xs text-muted-foreground">
                                        {modelTestMutation.data.message || "Connection successful. View the results in the popup."}
                                    </p>
                                </div>
                                <Button
                                    type="button"
                                    variant="secondary"
                                    onClick={() => setIsModelsDialogOpen(true)}
                                >
                                    View models
                                </Button>
                            </div>
                        ) : null}
                    </div>
                </CardContent>
            </Card>

            <Dialog open={isModelsDialogOpen} onOpenChange={setIsModelsDialogOpen}>
                <DialogContent className="max-h-[90vh] w-[95vw] max-w-350 overflow-hidden p-0">
                    <DialogHeader className="p-4 pb-0">
                        <DialogTitle>Available models</DialogTitle>
                        <DialogDescription>
                            Models returned for the current unsaved {providerOptions.find((option) => option.value === currentProvider)?.label ?? currentProvider} credentials.
                        </DialogDescription>
                    </DialogHeader>

                    <div className="max-h-[70vh] overflow-auto px-4 pb-4">
                        {fetchedModels.length ? (
                            <div className="overflow-x-auto rounded-lg border bg-background">
                                <Table className="min-w-240">
                                    <TableHeader>
                                        <TableRow>
                                            <TableHead>Model</TableHead>
                                            <TableHead>Owner</TableHead>
                                            <TableHead>Token limits</TableHead>
                                            <TableHead>Capabilities</TableHead>
                                        </TableRow>
                                    </TableHeader>
                                    <TableBody>
                                        {fetchedModels.map((model: LLMModel) => (
                                            <TableRow key={`${model.provider}-${model.id}`}>
                                                <TableCell className="align-top whitespace-normal">
                                                    <div className="space-y-1">
                                                        <p className="font-medium text-foreground">
                                                            {model.name || model.id}
                                                        </p>
                                                        <p className="break-all text-xs text-muted-foreground">
                                                            {model.id}
                                                        </p>
                                                        {model.description ? (
                                                            <p className="text-xs text-muted-foreground">
                                                                {model.description}
                                                            </p>
                                                        ) : null}
                                                    </div>
                                                </TableCell>
                                                <TableCell className="align-top whitespace-normal text-sm text-muted-foreground">
                                                    {model.owned_by || "—"}
                                                </TableCell>
                                                <TableCell className="align-top whitespace-normal text-sm text-muted-foreground">
                                                    <div>
                                                        <span className="font-medium text-foreground">Context:</span>{" "}
                                                        {formatOptionalNumber(model.context_window)}
                                                    </div>
                                                    <div>
                                                        <span className="font-medium text-foreground">Input:</span>{" "}
                                                        {formatOptionalNumber(model.input_token_limit)}
                                                    </div>
                                                    <div>
                                                        <span className="font-medium text-foreground">Output:</span>{" "}
                                                        {formatOptionalNumber(model.output_token_limit)}
                                                    </div>
                                                </TableCell>
                                                <TableCell className="align-top whitespace-normal">
                                                    {model.capabilities?.length ? (
                                                        <div className="flex flex-wrap gap-2">
                                                            {model.capabilities.map((capability) => (
                                                                <Badge
                                                                    key={`${model.id}-${capability}`}
                                                                    variant="outline"
                                                                >
                                                                    {capability}
                                                                </Badge>
                                                            ))}
                                                        </div>
                                                    ) : (
                                                        <span className="text-sm text-muted-foreground">—</span>
                                                    )}
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </div>
                        ) : (
                            <div className="rounded-md border bg-background p-3 text-sm text-muted-foreground">
                                The provider responded successfully, but no models were returned for the current credentials.
                            </div>
                        )}
                    </div>

                    <DialogFooter showCloseButton />
                </DialogContent>
            </Dialog>

            <div className="flex flex-col gap-3 sm:flex-row sm:justify-end">
                <Link
                    href={backHref}
                    className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                >
                    Cancel
                </Link>
                <Button type="submit" disabled={isPending || form.formState.isSubmitting}>
                    {isPending ? "Saving…" : submitLabel}
                </Button>
            </div>
        </form>
    );
}
