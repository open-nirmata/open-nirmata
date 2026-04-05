"use client";

import Link from "next/link";
import { useMutation } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2, PlugZap } from "lucide-react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { testMCPTool } from "@/lib/api/tools";
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
    JsonObject,
    TestMCPToolPayload,
    Tool,
    ToolPayload,
    ToolType,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const toolTypeOptions: Array<{
    value: ToolType;
    label: string;
    help: string;
}> = [
        {
            value: "mcp",
            label: "MCP",
            help: "Model Context Protocol tools can run locally over stdio or connect to a remote MCP server.",
        },
        {
            value: "http",
            label: "HTTP",
            help: "HTTP tools call a concrete endpoint with a required URL and method.",
        },
    ];

const httpMethods = ["GET", "POST", "PUT", "PATCH", "DELETE"] as const;
const mcpTransports = ["stdio", "remote"] as const;

const toolSchema = z
    .object({
        name: z.string().trim().min(1, "Name is required."),
        type: z.string(),
        description: z.string(),
        enabled: z.boolean(),
        tags: z.string(),
        http_url: z.string(),
        http_method: z.string(),
        http_payload_template: z.string(),
        http_timeout_seconds: z.string(),
        headersText: z
            .string()
            .refine((value) => isValidObjectJson(value), "Headers must be a valid JSON object."),
        http_query_params_text: z
            .string()
            .refine((value) => isValidObjectJson(value), "Query params must be a valid JSON object."),
        mcp_transport: z.enum(mcpTransports),
        mcp_command: z.string(),
        mcp_args_text: z.string(),
        mcp_env_text: z
            .string()
            .refine((value) => isValidObjectJson(value), "Environment values must be a valid JSON object."),
        mcp_server_url: z.string(),
        mcp_timeout_seconds: z.string(),
        authText: z
            .string()
            .refine((value) => isValidObjectJson(value), "Auth must be a valid JSON object."),
    })
    .superRefine((values, ctx) => {
        if (!isSupportedToolType(values.type)) {
            ctx.addIssue({
                code: z.ZodIssueCode.custom,
                path: ["type"],
                message: "Choose MCP or HTTP.",
            });
        }

        if (values.type === "http") {
            if (!values.http_url.trim()) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["http_url"],
                    message: "URL is required for HTTP tools.",
                });
            }

            if (!values.http_method.trim()) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["http_method"],
                    message: "Method is required for HTTP tools.",
                });
            }

            if (values.http_timeout_seconds.trim()) {
                const timeout = Number.parseInt(values.http_timeout_seconds.trim(), 10);
                if (Number.isNaN(timeout) || timeout < 1) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        path: ["http_timeout_seconds"],
                        message: "Timeout must be a positive whole number.",
                    });
                }
            }
        }

        if (values.type === "mcp") {
            if (values.mcp_transport === "stdio" && !values.mcp_command.trim()) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["mcp_command"],
                    message: "Command is required for stdio MCP tools.",
                });
            }

            if (values.mcp_transport === "remote" && !values.mcp_server_url.trim()) {
                ctx.addIssue({
                    code: z.ZodIssueCode.custom,
                    path: ["mcp_server_url"],
                    message: "Server URL is required for remote MCP tools.",
                });
            }

            if (values.mcp_timeout_seconds.trim()) {
                const timeout = Number.parseInt(values.mcp_timeout_seconds.trim(), 10);
                if (Number.isNaN(timeout) || timeout < 1) {
                    ctx.addIssue({
                        code: z.ZodIssueCode.custom,
                        path: ["mcp_timeout_seconds"],
                        message: "Timeout must be a positive whole number.",
                    });
                }
            }
        }
    });

type ToolFormValues = z.infer<typeof toolSchema>;

function getErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : "Something went wrong while testing the MCP server.";
}

function formatJsonValue(value: unknown) {
    return JSON.stringify(value, null, 2);
}

function hasObjectEntries(value: unknown): value is JsonObject {
    return Boolean(value && typeof value === "object" && !Array.isArray(value) && Object.keys(value).length > 0);
}

function buildMCPConfig(values: ToolFormValues, headers?: JsonObject): JsonObject {
    const config: JsonObject = {
        transport: values.mcp_transport,
    };

    const timeout = parsePositiveInteger(values.mcp_timeout_seconds);

    if (values.mcp_transport === "stdio") {
        const command = trimOrUndefined(values.mcp_command);
        const args = parseStringList(values.mcp_args_text);
        const env = parseObject(values.mcp_env_text);

        if (command) {
            config.command = command;
        }

        if (args) {
            config.args = args;
        }

        if (env) {
            config.env = env;
        }
    }

    if (values.mcp_transport === "remote") {
        const serverUrl = trimOrUndefined(values.mcp_server_url);

        if (serverUrl) {
            config.server_url = serverUrl;
        }

        if (headers) {
            config.headers = headers;
        }
    }

    if (typeof timeout === "number") {
        config.timeout_seconds = timeout;
    }

    return config;
}

type ToolFormProps = {
    mode: "create" | "edit";
    initialValue?: Tool;
    isPending?: boolean;
    onSubmit: (payload: ToolPayload) => Promise<void>;
    backHref?: string;
};

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

function readObject(value: unknown): JsonObject | undefined {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
        return undefined;
    }

    return value as JsonObject;
}

function readString(value: unknown) {
    return typeof value === "string" ? value : "";
}

function readNumberAsString(value: unknown) {
    return typeof value === "number" ? String(value) : "";
}

function readStringArray(value: unknown) {
    if (!Array.isArray(value)) {
        return [];
    }

    return value.filter((item): item is string => typeof item === "string");
}

function formatJsonObject(value: unknown) {
    const objectValue = readObject(value);
    return objectValue ? JSON.stringify(objectValue, null, 2) : "";
}

function parseStringList(value: string) {
    const items = value
        .split("\n")
        .map((item) => item.trim())
        .filter(Boolean);

    return items.length ? items : undefined;
}

function trimOrUndefined(value: string) {
    const trimmed = value.trim();
    return trimmed ? trimmed : undefined;
}

function parsePositiveInteger(value: string) {
    const trimmed = value.trim();
    if (!trimmed) {
        return undefined;
    }

    const parsed = Number.parseInt(trimmed, 10);
    return Number.isNaN(parsed) ? undefined : parsed;
}

function isSupportedToolType(type?: string): type is ToolType {
    return type === "mcp" || type === "http";
}

function getInitialType(type?: string): ToolType | "" {
    return isSupportedToolType(type) ? type : "";
}

function getInitialTransport(config?: JsonObject): "stdio" | "remote" {
    if (config?.transport === "remote" || typeof config?.server_url === "string") {
        return "remote";
    }

    return "stdio";
}

export function ToolForm({
    mode,
    initialValue,
    isPending = false,
    onSubmit,
    backHref = "/tools",
}: ToolFormProps) {
    const initialConfig = initialValue?.config;
    const hasUnsupportedInitialType = Boolean(
        initialValue?.type && !isSupportedToolType(initialValue.type),
    );
    const hasExistingAuth = Boolean(initialValue?.auth_configured);

    const form = useForm<ToolFormValues>({
        resolver: zodResolver(toolSchema),
        defaultValues: {
            name: initialValue?.name ?? "",
            type: getInitialType(initialValue?.type),
            description: initialValue?.description ?? "",
            enabled: initialValue?.enabled ?? true,
            tags: initialValue?.tags?.join(", ") ?? "",
            http_url: readString(initialConfig?.url),
            http_method: readString(initialConfig?.method) || "GET",
            http_payload_template: readString(initialConfig?.payload_template),
            http_timeout_seconds: readNumberAsString(initialConfig?.timeout_seconds),
            headersText: formatJsonObject(initialConfig?.headers),
            http_query_params_text: formatJsonObject(initialConfig?.query_params),
            mcp_transport: getInitialTransport(initialConfig),
            mcp_command: readString(initialConfig?.command),
            mcp_args_text: readStringArray(initialConfig?.args).join("\n"),
            mcp_env_text: formatJsonObject(initialConfig?.env),
            mcp_server_url: readString(initialConfig?.server_url),
            mcp_timeout_seconds: readNumberAsString(initialConfig?.timeout_seconds),
            authText: "",
        },
    });

    const currentType = useWatch({ control: form.control, name: "type" }) ?? getInitialType(initialValue?.type);
    const currentTransport =
        useWatch({ control: form.control, name: "mcp_transport" }) ?? getInitialTransport(initialConfig);
    const submitLabel = mode === "create" ? "Create tool" : "Save changes";

    const testMutation = useMutation({
        mutationFn: testMCPTool,
        onSuccess: (response) => {
            const count = response.data?.count ?? 0;
            toast.success(
                response.message || `MCP server responded successfully${count ? ` with ${count} discovered tool${count === 1 ? "" : "s"}` : ""}.`,
            );
        },
        onError: (error) => {
            toast.error(getErrorMessage(error));
        },
    });

    const mcpTestResult = testMutation.data?.data;

    const handleTestConnection = async () => {
        if (currentType !== "mcp") {
            return;
        }

        const fieldsToValidate: Array<keyof ToolFormValues> =
            currentTransport === "stdio"
                ? ["type", "mcp_transport", "mcp_command", "mcp_env_text", "mcp_timeout_seconds"]
                : ["type", "mcp_transport", "mcp_server_url", "headersText", "mcp_timeout_seconds"];

        const isValid = await form.trigger(fieldsToValidate);
        if (!isValid) {
            return;
        }

        const values = form.getValues();
        const headers = currentTransport === "remote" ? parseObject(values.headersText) : undefined;
        const timeout = parsePositiveInteger(values.mcp_timeout_seconds);
        const payload: TestMCPToolPayload = {
            config: buildMCPConfig(values, headers),
        };

        if (typeof timeout === "number") {
            payload.timeout_seconds = timeout;
        }

        testMutation.mutate(payload);
    };

    const handleSubmit = form.handleSubmit(async (values) => {
        const payload: ToolPayload = {
            name: values.name.trim(),
            type: values.type as ToolType,
            enabled: values.enabled,
        };

        const description = trimOrUndefined(values.description);
        if (description) {
            payload.description = description;
        }

        const tags = values.tags
            .split(",")
            .map((tag) => tag.trim())
            .filter(Boolean);
        if (tags.length) {
            payload.tags = tags;
        }

        const headers = parseObject(values.headersText);
        const auth = parseObject(values.authText);

        if (values.type === "http") {
            const config: JsonObject = {};
            const url = trimOrUndefined(values.http_url);
            const method = trimOrUndefined(values.http_method);
            const payloadTemplate = trimOrUndefined(values.http_payload_template);
            const timeout = parsePositiveInteger(values.http_timeout_seconds);
            const queryParams = parseObject(values.http_query_params_text);

            if (url) {
                config.url = url;
            }

            if (method) {
                config.method = method.toUpperCase();
            }

            if (payloadTemplate) {
                config.payload_template = payloadTemplate;
            }

            if (headers) {
                config.headers = headers;
            }

            if (queryParams) {
                config.query_params = queryParams;
            }

            if (typeof timeout === "number") {
                config.timeout_seconds = timeout;
            }

            if (Object.keys(config).length > 0) {
                payload.config = config;
            }
        }

        if (values.type === "mcp") {
            const config = buildMCPConfig(values, headers);

            if (Object.keys(config).length > 0) {
                payload.config = config;
            }
        }

        if (auth) {
            payload.auth = auth;
        }

        await onSubmit(payload);
    });

    const selectedTypeHelp = currentType
        ? toolTypeOptions.find((option) => option.value === currentType)?.help
        : "Choose MCP or HTTP to reveal the required configuration fields.";

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <h2 className="text-xl font-semibold tracking-tight">
                        {mode === "create" ? "Create a new tool" : `Edit ${initialValue?.name ?? "tool"}`}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                        Define concrete MCP or HTTP settings that match the Go API contract.
                    </p>
                </div>
                <Link
                    href={backHref}
                    className={cn(buttonVariants({ variant: "outline" }), "inline-flex")}
                >
                    Back to tools
                </Link>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Basic details</CardTitle>
                    <CardDescription>
                        Choose the supported tool type and describe what it does for your agents.
                    </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4 md:grid-cols-2">
                    {hasUnsupportedInitialType ? (
                        <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-900 md:col-span-2 dark:text-amber-200">
                            This tool still uses the legacy <code className="font-mono">{initialValue?.type}</code> type.
                            Choose <strong> MCP</strong> or <strong>HTTP</strong> before saving changes.
                        </div>
                    ) : null}

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="name">Name</Label>
                        <Input id="name" placeholder="Slack MCP" {...form.register("name")} />
                        {form.formState.errors.name ? (
                            <p className="text-xs text-destructive">{form.formState.errors.name.message}</p>
                        ) : null}
                    </div>

                    <div className="space-y-2">
                        <Label>Type</Label>
                        <Controller
                            control={form.control}
                            name="type"
                            render={({ field }) => (
                                <Select value={field.value || undefined} onValueChange={field.onChange}>
                                    <SelectTrigger className="w-full">
                                        <SelectValue placeholder="Select a tool type" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {toolTypeOptions.map((option) => (
                                            <SelectItem key={option.value} value={option.value}>
                                                {option.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            )}
                        />
                        {form.formState.errors.type ? (
                            <p className="text-xs text-destructive">{form.formState.errors.type.message}</p>
                        ) : null}
                    </div>

                    <div className="space-y-2">
                        <Label>Enabled</Label>
                        <div className="flex min-h-9 items-center justify-between rounded-lg border px-3">
                            <span className="text-sm text-muted-foreground">Expose this tool to agents</span>
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
                        <p className="font-medium text-foreground">Configuration guidance</p>
                        <p>{selectedTypeHelp}</p>
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="description">Description</Label>
                        <Textarea
                            id="description"
                            rows={4}
                            placeholder="Search tickets, call external APIs, or connect to your MCP server."
                            {...form.register("description")}
                        />
                    </div>

                    <div className="space-y-2 md:col-span-2">
                        <Label htmlFor="tags">Tags</Label>
                        <Input
                            id="tags"
                            placeholder="support, internal, search"
                            {...form.register("tags")}
                        />
                        <p className="text-xs text-muted-foreground">
                            Separate tags with commas. They will be normalized by the backend.
                        </p>
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Configuration</CardTitle>
                    <CardDescription>
                        Fill in the concrete settings required by the selected tool type.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    {currentType === "http" ? (
                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="space-y-2 md:col-span-2">
                                <Label htmlFor="http_url">URL <span className="text-destructive">*</span></Label>
                                <Input
                                    id="http_url"
                                    placeholder="https://api.example.com/health"
                                    {...form.register("http_url")}
                                />
                                {form.formState.errors.http_url ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.http_url.message}
                                    </p>
                                ) : null}
                            </div>

                            <div className="space-y-2">
                                <Label>Method <span className="text-destructive">*</span></Label>
                                <Controller
                                    control={form.control}
                                    name="http_method"
                                    render={({ field }) => (
                                        <Select value={field.value || "GET"} onValueChange={field.onChange}>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder="Select method" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                {httpMethods.map((method) => (
                                                    <SelectItem key={method} value={method}>
                                                        {method}
                                                    </SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                    )}
                                />
                                {form.formState.errors.http_method ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.http_method.message}
                                    </p>
                                ) : null}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="http_timeout_seconds">Timeout seconds</Label>
                                <Input
                                    id="http_timeout_seconds"
                                    type="number"
                                    min={1}
                                    step={1}
                                    placeholder="15"
                                    {...form.register("http_timeout_seconds")}
                                />
                                {form.formState.errors.http_timeout_seconds ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.http_timeout_seconds.message}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground">
                                        Optional. Must be a positive integer when provided.
                                    </p>
                                )}
                            </div>

                            <div className="space-y-2 md:col-span-2">
                                <Label htmlFor="http_payload_template">Payload template</Label>
                                <Textarea
                                    id="http_payload_template"
                                    rows={5}
                                    className="font-mono text-xs"
                                    placeholder='{"query":"{{input}}"}'
                                    {...form.register("http_payload_template")}
                                />
                                <p className="text-xs text-muted-foreground">
                                    Optional request body template for POST, PUT, or PATCH requests.
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="headersText">Headers JSON</Label>
                                <Textarea
                                    id="headersText"
                                    rows={8}
                                    className="font-mono text-xs"
                                    placeholder='{"Authorization":"Bearer ..."}'
                                    {...form.register("headersText")}
                                />
                                {form.formState.errors.headersText ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.headersText.message}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground">
                                        Optional object map of header names to values.
                                    </p>
                                )}
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="http_query_params_text">Query params JSON</Label>
                                <Textarea
                                    id="http_query_params_text"
                                    rows={8}
                                    className="font-mono text-xs"
                                    placeholder='{"environment":"prod"}'
                                    {...form.register("http_query_params_text")}
                                />
                                {form.formState.errors.http_query_params_text ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.http_query_params_text.message}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground">
                                        Optional object map appended to the request URL.
                                    </p>
                                )}
                            </div>
                        </div>
                    ) : null}

                    {currentType === "mcp" ? (
                        <div className="grid gap-4 md:grid-cols-2">
                            <div className="space-y-2 md:col-span-2">
                                <Label>Transport</Label>
                                <Controller
                                    control={form.control}
                                    name="mcp_transport"
                                    render={({ field }) => (
                                        <Select value={field.value} onValueChange={field.onChange}>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder="Select transport" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="stdio">stdio</SelectItem>
                                                <SelectItem value="remote">remote</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    )}
                                />
                                <p className="text-xs text-muted-foreground">
                                    Use <code className="font-mono">stdio</code> for a local command or <code className="font-mono">remote</code> for a hosted MCP server.
                                </p>
                            </div>

                            <div className="space-y-2">
                                <Label htmlFor="mcp_timeout_seconds">Timeout seconds</Label>
                                <Input
                                    id="mcp_timeout_seconds"
                                    type="number"
                                    min={1}
                                    step={1}
                                    placeholder="15"
                                    {...form.register("mcp_timeout_seconds")}
                                />
                                {form.formState.errors.mcp_timeout_seconds ? (
                                    <p className="text-xs text-destructive">
                                        {form.formState.errors.mcp_timeout_seconds.message}
                                    </p>
                                ) : (
                                    <p className="text-xs text-muted-foreground">
                                        Optional. Used when testing slow MCP servers.
                                    </p>
                                )}
                            </div>

                            {currentTransport === "stdio" ? (
                                <>
                                    <div className="space-y-2 md:col-span-2">
                                        <Label htmlFor="mcp_command">Command <span className="text-destructive">*</span></Label>
                                        <Input
                                            id="mcp_command"
                                            placeholder="npx"
                                            {...form.register("mcp_command")}
                                        />
                                        {form.formState.errors.mcp_command ? (
                                            <p className="text-xs text-destructive">
                                                {form.formState.errors.mcp_command.message}
                                            </p>
                                        ) : (
                                            <p className="text-xs text-muted-foreground">
                                                The executable used to launch the MCP server locally.
                                            </p>
                                        )}
                                    </div>

                                    <div className="space-y-2 md:col-span-2">
                                        <Label htmlFor="mcp_args_text">Arguments</Label>
                                        <Textarea
                                            id="mcp_args_text"
                                            rows={5}
                                            className="font-mono text-xs"
                                            placeholder={"-y\n@modelcontextprotocol/server-filesystem"}
                                            {...form.register("mcp_args_text")}
                                        />
                                        <p className="text-xs text-muted-foreground">
                                            Optional. Add one argument per line in execution order.
                                        </p>
                                    </div>

                                    <div className="space-y-2 md:col-span-2">
                                        <Label htmlFor="mcp_env_text">Environment JSON</Label>
                                        <Textarea
                                            id="mcp_env_text"
                                            rows={8}
                                            className="font-mono text-xs"
                                            placeholder='{"NODE_ENV":"production"}'
                                            {...form.register("mcp_env_text")}
                                        />
                                        {form.formState.errors.mcp_env_text ? (
                                            <p className="text-xs text-destructive">
                                                {form.formState.errors.mcp_env_text.message}
                                            </p>
                                        ) : (
                                            <p className="text-xs text-muted-foreground">
                                                Optional object map of environment variables for the process.
                                            </p>
                                        )}
                                    </div>
                                </>
                            ) : null}

                            {currentTransport === "remote" ? (
                                <>
                                    <div className="space-y-2 md:col-span-2">
                                        <Label htmlFor="mcp_server_url">Server URL <span className="text-destructive">*</span></Label>
                                        <Input
                                            id="mcp_server_url"
                                            placeholder="https://mcp.example.com"
                                            {...form.register("mcp_server_url")}
                                        />
                                        {form.formState.errors.mcp_server_url ? (
                                            <p className="text-xs text-destructive">
                                                {form.formState.errors.mcp_server_url.message}
                                            </p>
                                        ) : (
                                            <p className="text-xs text-muted-foreground">
                                                The reachable base URL for the remote MCP server.
                                            </p>
                                        )}
                                    </div>

                                    <div className="space-y-2 md:col-span-2">
                                        <Label htmlFor="headersText">Headers JSON</Label>
                                        <Textarea
                                            id="headersText"
                                            rows={8}
                                            className="font-mono text-xs"
                                            placeholder='{"Authorization":"Bearer ..."}'
                                            {...form.register("headersText")}
                                        />
                                        {form.formState.errors.headersText ? (
                                            <p className="text-xs text-destructive">
                                                {form.formState.errors.headersText.message}
                                            </p>
                                        ) : (
                                            <p className="text-xs text-muted-foreground">
                                                Optional object map of headers sent to the MCP server.
                                            </p>
                                        )}
                                    </div>
                                </>
                            ) : null}

                            <div className="space-y-4 rounded-lg border bg-muted/20 p-4 md:col-span-2">
                                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                                    <div className="space-y-1">
                                        <div className="flex items-center gap-2">
                                            <PlugZap className="size-4 text-primary" />
                                            <p className="font-medium text-foreground">Test MCP connection</p>
                                        </div>
                                        <p className="text-xs text-muted-foreground">
                                            Validate the current MCP settings and discover tools before saving. Results reflect the last test run.
                                        </p>
                                    </div>
                                    <Button
                                        type="button"
                                        variant="outline"
                                        onClick={handleTestConnection}
                                        disabled={testMutation.isPending}
                                    >
                                        {testMutation.isPending ? (
                                            <>
                                                <Loader2 className="size-4 animate-spin" />
                                                Testing…
                                            </>
                                        ) : (
                                            "Discover tools"
                                        )}
                                    </Button>
                                </div>

                                {testMutation.isError ? (
                                    <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                                        {getErrorMessage(testMutation.error)}
                                    </div>
                                ) : null}

                                {mcpTestResult ? (
                                    <div className="space-y-4">
                                        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                                            <div className="rounded-md border bg-background p-3">
                                                <p className="text-xs text-muted-foreground">Transport</p>
                                                <p className="font-medium text-foreground">{mcpTestResult.transport || "—"}</p>
                                            </div>
                                            <div className="rounded-md border bg-background p-3">
                                                <p className="text-xs text-muted-foreground">Server</p>
                                                <p className="break-all font-medium text-foreground">
                                                    {mcpTestResult.server_url || "Local stdio process"}
                                                </p>
                                            </div>
                                            <div className="rounded-md border bg-background p-3">
                                                <p className="text-xs text-muted-foreground">Server info</p>
                                                <p className="font-medium text-foreground">
                                                    {mcpTestResult.server_info?.name || "Unknown server"}
                                                </p>
                                                <p className="text-xs text-muted-foreground">
                                                    Version {mcpTestResult.server_info?.version || "—"}
                                                </p>
                                            </div>
                                            <div className="rounded-md border bg-background p-3">
                                                <p className="text-xs text-muted-foreground">Discovered tools</p>
                                                <p className="font-medium text-foreground">{mcpTestResult.count}</p>
                                            </div>
                                        </div>

                                        {mcpTestResult.tools.length ? (
                                            <div className="rounded-lg border bg-background">
                                                <Table>
                                                    <TableHeader>
                                                        <TableRow>
                                                            <TableHead>Tool</TableHead>
                                                            <TableHead>Description</TableHead>
                                                            <TableHead>Metadata</TableHead>
                                                        </TableRow>
                                                    </TableHeader>
                                                    <TableBody>
                                                        {mcpTestResult.tools.map((tool) => (
                                                            <TableRow key={tool.name}>
                                                                <TableCell className="align-top whitespace-normal">
                                                                    <div className="space-y-2">
                                                                        <p className="font-medium text-foreground">{tool.name}</p>
                                                                        <div className="flex flex-wrap gap-2">
                                                                            {hasObjectEntries(tool.input_schema) ? (
                                                                                <Badge variant="outline">input schema</Badge>
                                                                            ) : null}
                                                                            {hasObjectEntries(tool.annotations) ? (
                                                                                <Badge variant="outline">annotations</Badge>
                                                                            ) : null}
                                                                        </div>
                                                                    </div>
                                                                </TableCell>
                                                                <TableCell className="max-w-xl align-top whitespace-normal text-sm text-muted-foreground">
                                                                    {tool.description || "—"}
                                                                </TableCell>
                                                                <TableCell className="max-w-xl align-top whitespace-normal">
                                                                    <div className="space-y-2">
                                                                        {hasObjectEntries(tool.input_schema) ? (
                                                                            <details className="rounded-md border bg-muted/30 p-2">
                                                                                <summary className="cursor-pointer text-xs font-medium text-foreground">
                                                                                    View input schema
                                                                                </summary>
                                                                                <pre className="mt-2 overflow-x-auto text-[11px] text-muted-foreground">
                                                                                    {formatJsonValue(tool.input_schema)}
                                                                                </pre>
                                                                            </details>
                                                                        ) : null}
                                                                        {hasObjectEntries(tool.annotations) ? (
                                                                            <details className="rounded-md border bg-muted/30 p-2">
                                                                                <summary className="cursor-pointer text-xs font-medium text-foreground">
                                                                                    View annotations
                                                                                </summary>
                                                                                <pre className="mt-2 overflow-x-auto text-[11px] text-muted-foreground">
                                                                                    {formatJsonValue(tool.annotations)}
                                                                                </pre>
                                                                            </details>
                                                                        ) : null}
                                                                        {!hasObjectEntries(tool.input_schema) && !hasObjectEntries(tool.annotations) ? (
                                                                            <span className="text-sm text-muted-foreground">—</span>
                                                                        ) : null}
                                                                    </div>
                                                                </TableCell>
                                                            </TableRow>
                                                        ))}
                                                    </TableBody>
                                                </Table>
                                            </div>
                                        ) : (
                                            <div className="rounded-md border bg-background p-3 text-sm text-muted-foreground">
                                                The connection succeeded, but the server did not return any tools.
                                            </div>
                                        )}
                                    </div>
                                ) : (
                                    <div className="rounded-md border border-dashed bg-background/60 p-3 text-sm text-muted-foreground">
                                        Run a connection test to fetch server info and discover its available tools.
                                    </div>
                                )}
                            </div>
                        </div>
                    ) : null}

                    {!currentType ? (
                        <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
                            Select <strong>MCP</strong> or <strong>HTTP</strong> above to reveal the required settings.
                        </div>
                    ) : null}
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Authentication</CardTitle>
                    <CardDescription>
                        Add auth only when you need to rotate or set credentials for the tool.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                    <Label htmlFor="authText">Auth JSON</Label>
                    <Textarea
                        id="authText"
                        rows={8}
                        className="font-mono text-xs"
                        placeholder='{"api_key":"..."}'
                        {...form.register("authText")}
                    />
                    <p className="text-xs text-muted-foreground">
                        Existing auth values are intentionally not returned by the API.
                        {mode === "edit" && hasExistingAuth
                            ? " Leave this blank to keep the current auth configuration."
                            : " Add a new auth object only when needed."}
                    </p>
                    {form.formState.errors.authText ? (
                        <p className="text-xs text-destructive">
                            {form.formState.errors.authText.message}
                        </p>
                    ) : null}
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
