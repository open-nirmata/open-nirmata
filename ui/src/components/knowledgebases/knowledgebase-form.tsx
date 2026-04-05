"use client";

import Link from "next/link";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";

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
    JsonObject,
    Knowledgebase,
    KnowledgebasePayload,
    KnowledgebaseProviderKind,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const knowledgebaseSchema = z.object({
    name: z.string().trim().min(1, "Name is required."),
    provider: z.enum(["milvus", "mixedbread", "zeroentropy", "algolia", "qdrant"]),
    description: z.string(),
    enabled: z.boolean(),
    base_url: z.string(),
    index_name: z.string(),
    namespace: z.string(),
    embedding_model: z.string(),
    api_key: z.string(),
    configText: z
        .string()
        .refine((value) => isValidObjectJson(value), "Config must be a valid JSON object."),
});

type KnowledgebaseFormValues = z.infer<typeof knowledgebaseSchema>;

type KnowledgebaseFormProps = {
    mode: "create" | "edit";
    initialValue?: Knowledgebase;
    isPending?: boolean;
    onSubmit: (payload: KnowledgebasePayload) => Promise<void>;
    backHref?: string;
};

const providerOptions: Array<{
    value: KnowledgebaseProviderKind;
    label: string;
    help: string;
}> = [
        {
            value: "milvus",
            label: "Milvus",
            help: "Vector database deployments, often using a collection or index per knowledge base.",
        },
        {
            value: "mixedbread",
            label: "Mixedbread",
            help: "Hosted retrieval and embedding APIs for document search workloads.",
        },
        {
            value: "zeroentropy",
            label: "ZeroEntropy",
            help: "Managed search and retrieval provider for AI assistants.",
        },
        {
            value: "algolia",
            label: "Algolia",
            help: "Managed search indices with API-based access and ranking controls.",
        },
        {
            value: "qdrant",
            label: "Qdrant",
            help: "Vector database collections backed by a configurable endpoint and namespace.",
        },
    ];

function isValidObjectJson(value: string) {
    const trimmed = value.trim();

    if (!trimmed) {
        return true;
    }

    try {
        const parsed = JSON.parse(trimmed) as unknown;
        return typeof parsed === "object" && parsed !== null && !Array.isArray(parsed);
    } catch {
        return false;
    }
}

function parseObject(value: string): JsonObject | undefined {
    const trimmed = value.trim();

    if (!trimmed) {
        return undefined;
    }

    return JSON.parse(trimmed) as JsonObject;
}

function formatJson(value?: JsonObject) {
    return value ? JSON.stringify(value, null, 2) : "";
}

function normalizeProvider(value?: string): KnowledgebaseProviderKind {
    if (
        value === "milvus" ||
        value === "mixedbread" ||
        value === "zeroentropy" ||
        value === "algolia" ||
        value === "qdrant"
    ) {
        return value;
    }

    return "milvus";
}

function trimOrUndefined(value: string) {
    const trimmed = value.trim();
    return trimmed ? trimmed : undefined;
}

export function KnowledgebaseForm({
    mode,
    initialValue,
    isPending = false,
    onSubmit,
    backHref = "/knowledgebase",
}: KnowledgebaseFormProps) {
    const hasExistingAuth = Boolean(initialValue?.auth_configured);

    const form = useForm<KnowledgebaseFormValues>({
        resolver: zodResolver(knowledgebaseSchema),
        defaultValues: {
            name: initialValue?.name ?? "",
            provider: normalizeProvider(initialValue?.provider),
            description: initialValue?.description ?? "",
            enabled: initialValue?.enabled ?? true,
            base_url: initialValue?.base_url ?? "",
            index_name: initialValue?.index_name ?? "",
            namespace: initialValue?.namespace ?? "",
            embedding_model: initialValue?.embedding_model ?? "",
            api_key: "",
            configText: formatJson(initialValue?.config),
        },
    });

    const currentProvider =
        useWatch({ control: form.control, name: "provider" }) ??
        normalizeProvider(initialValue?.provider);
    const submitLabel =
        mode === "create" ? "Create knowledge base" : "Save changes";

    const handleSubmit = form.handleSubmit(async (values) => {
        const payload: KnowledgebasePayload = {
            name: values.name.trim(),
            provider: values.provider,
            enabled: values.enabled,
        };

        const description = trimOrUndefined(values.description);
        const baseUrl = trimOrUndefined(values.base_url);
        const indexName = trimOrUndefined(values.index_name);
        const namespace = trimOrUndefined(values.namespace);
        const embeddingModel = trimOrUndefined(values.embedding_model);
        const apiKey = trimOrUndefined(values.api_key);
        const config = parseObject(values.configText);

        if (description) {
            payload.description = description;
        }

        if (baseUrl) {
            payload.base_url = baseUrl;
        }

        if (indexName) {
            payload.index_name = indexName;
        }

        if (namespace) {
            payload.namespace = namespace;
        }

        if (embeddingModel) {
            payload.embedding_model = embeddingModel;
        }

        if (apiKey) {
            payload.api_key = apiKey;
        }

        if (config && Object.keys(config).length > 0) {
            payload.config = config;
        }

        await onSubmit(payload);
    });

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">
                        {mode === "create"
                            ? "Add a knowledge base"
                            : `Edit ${initialValue?.name ?? "knowledge base"}`}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                        Save retrieval provider settings, index targets, and embedding defaults through the Go API.
                    </p>
                </div>
                <Link
                    href={backHref}
                    className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                >
                    Back to knowledge bases
                </Link>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Basic details</CardTitle>
                    <CardDescription>
                        Choose the retrieval backend and define the knowledge base you want agents to use.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="name">Name</Label>
                        <Input id="name" placeholder="Product docs search" {...form.register("name")} />
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
                                    onValueChange={(value) => field.onChange(value as KnowledgebaseProviderKind)}
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
                            <span className="text-sm text-muted-foreground">Allow agents to query this knowledge base</span>
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
                        <p>{providerOptions.find((option) => option.value === currentProvider)?.help}</p>
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="description">Description</Label>
                        <Textarea
                            id="description"
                            rows={4}
                            placeholder="Search product documentation, runbooks, or internal support content."
                            {...form.register("description")}
                        />
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Retrieval settings</CardTitle>
                    <CardDescription>
                        Configure the endpoint, index naming, embedding model, and optional API key.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-2">
                        <Label htmlFor="base_url">Base URL</Label>
                        <Input
                            id="base_url"
                            placeholder="https://your-provider.example.com"
                            {...form.register("base_url")}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="embedding_model">Embedding model</Label>
                        <Input
                            id="embedding_model"
                            placeholder="text-embedding-3-large"
                            {...form.register("embedding_model")}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="index_name">Index name</Label>
                        <Input
                            id="index_name"
                            placeholder="product-docs"
                            {...form.register("index_name")}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="namespace">Namespace</Label>
                        <Input id="namespace" placeholder="production" {...form.register("namespace")} />
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="api_key">API key</Label>
                        <Input
                            id="api_key"
                            type="password"
                            autoComplete="off"
                            placeholder={
                                mode === "edit" && hasExistingAuth
                                    ? "Leave blank to keep the current key"
                                    : "Optional: paste the provider API key"
                            }
                            {...form.register("api_key")}
                        />
                        <p className="text-xs text-muted-foreground">
                            {mode === "edit" && hasExistingAuth
                                ? "Credentials are write-only. Leave this blank to keep the existing secret."
                                : "Only needed when the selected provider requires authenticated requests."}
                        </p>
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Advanced config</CardTitle>
                    <CardDescription>
                        Optional JSON forwarded directly to the backend as `config`.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                    <Label htmlFor="configText">Config JSON</Label>
                    <Textarea
                        id="configText"
                        rows={12}
                        className="font-mono text-xs"
                        placeholder='{"top_k": 8, "metric": "cosine"}'
                        {...form.register("configText")}
                    />
                    {form.formState.errors.configText ? (
                        <p className="text-xs text-destructive">
                            {form.formState.errors.configText.message}
                        </p>
                    ) : (
                        <p className="text-xs text-muted-foreground">
                            Use this for provider-specific options that don’t yet have dedicated fields.
                        </p>
                    )}
                </CardContent>
            </Card>

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
