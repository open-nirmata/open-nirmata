"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
    Bot,
    Loader2,
    MessageSquareDashed,
    Play,
    RefreshCcw,
    Sparkles,
    User,
    Workflow,
    Wrench,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { getExecution, listExecutions, streamAgentExecution } from "@/lib/api/executions";
import type {
    Agent,
    ExecutionItem,
    ExecutionMessageItem,
    ExecutionStepItem,
    ExecutionStreamEvent,
    JsonObject,
} from "@/lib/types";
import { cn } from "@/lib/utils";

type AgentChatProps = {
    agent: Agent;
};

type DisplayMessage = ExecutionMessageItem & {
    id: string;
    stageName?: string;
    stageType?: string;
};

type NormalizedToolCall = {
    id: string;
    tool_name: string;
    arguments: JsonObject;
};

const dateFormatter = new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
});

export function AgentChat({ agent }: AgentChatProps) {
    const queryClient = useQueryClient();
    const abortRef = useRef<AbortController | null>(null);

    const [draft, setDraft] = useState("");
    const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
    const [showNewConversation, setShowNewConversation] = useState(false);
    const [isStreaming, setIsStreaming] = useState(false);
    const [streamError, setStreamError] = useState<string | null>(null);
    const [liveTranscript, setLiveTranscript] = useState<DisplayMessage[]>([]);
    const [liveEvents, setLiveEvents] = useState<Array<ExecutionStreamEvent<unknown>>>([]);
    const [optimisticExecution, setOptimisticExecution] = useState<ExecutionItem | null>(null);

    const historyQuery = useQuery({
        queryKey: ["executions", "agent", agent.id],
        queryFn: async () => {
            const response = await listExecutions({ agent_id: agent.id, limit: 20 });
            return response.data;
        },
        staleTime: 5_000,
    });

    const executionQuery = useQuery({
        queryKey: ["execution", selectedExecutionId],
        enabled: Boolean(selectedExecutionId) && !showNewConversation,
        queryFn: async () => {
            const response = await getExecution(selectedExecutionId!);
            if (!response.data) {
                throw new Error("Execution not found.");
            }
            return response.data;
        },
        staleTime: 5_000,
    });

    useEffect(() => {
        if (!showNewConversation && !selectedExecutionId && historyQuery.data?.length) {
            setSelectedExecutionId(historyQuery.data[0].id);
        }
    }, [historyQuery.data, selectedExecutionId, showNewConversation]);

    const selectedExecution = useMemo(() => {
        if (showNewConversation) {
            return null;
        }

        if (optimisticExecution && optimisticExecution.id === selectedExecutionId) {
            return optimisticExecution;
        }

        if (executionQuery.data) {
            return executionQuery.data;
        }

        return historyQuery.data?.find((item) => item.id === selectedExecutionId) ?? null;
    }, [executionQuery.data, historyQuery.data, optimisticExecution, selectedExecutionId, showNewConversation]);

    const conversationMessages = useMemo(
        () => (isStreaming ? liveTranscript : buildConversationMessages(selectedExecution)),
        [isStreaming, liveTranscript, selectedExecution],
    );

    const selectedSummary = showNewConversation ? null : selectedExecution;

    const handleSelectExecution = (executionId: string) => {
        setShowNewConversation(false);
        setSelectedExecutionId(executionId);
        setOptimisticExecution(null);
        setLiveTranscript([]);
        setLiveEvents([]);
        setStreamError(null);
    };

    const handleNewConversation = () => {
        abortRef.current?.abort();
        abortRef.current = null;
        setShowNewConversation(true);
        setSelectedExecutionId(null);
        setOptimisticExecution(null);
        setLiveTranscript([]);
        setLiveEvents([]);
        setStreamError(null);
        setIsStreaming(false);
    };

    const handleSend = async () => {
        const message = draft.trim();
        if (!message || isStreaming || !agent.enabled) {
            return;
        }

        const baseMessages = showNewConversation ? [] : buildConversationMessages(selectedExecution);
        const requestMessages = baseMessages.map(stripDisplayMetadata);

        const optimisticConversation: DisplayMessage[] = [
            ...baseMessages,
            {
                id: `user-${Date.now()}`,
                role: "user",
                content: message,
            },
            {
                id: `assistant-stream-${Date.now()}`,
                role: "assistant",
                content: "",
            },
        ];

        setDraft("");
        setIsStreaming(true);
        setStreamError(null);
        setLiveEvents([]);
        setLiveTranscript(optimisticConversation);
        setOptimisticExecution(null);

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            await streamAgentExecution(
                agent.id,
                {
                    message,
                    messages: requestMessages.length > 0 ? requestMessages : undefined,
                    stream: true,
                    metadata: {
                        source: "ui-agent-tester",
                    },
                },
                {
                    signal: controller.signal,
                    onEvent: (event) => {
                        setLiveEvents((current) => [...current, event]);

                        if (event.event === "llm_token") {
                            const delta = extractDelta(event.data);
                            if (delta) {
                                setLiveTranscript((current) => appendAssistantDelta(current, delta));
                            }
                            return;
                        }

                        if (event.event === "tool_result") {
                            const toolMessage = buildToolResultMessage(event.data, Date.now());
                            if (toolMessage) {
                                setLiveTranscript((current) => [...current, toolMessage]);
                            }
                            return;
                        }

                        if (event.event === "error") {
                            const message = extractErrorMessage(event.data) || "Execution failed.";
                            setStreamError(message);
                            return;
                        }

                        if (event.event === "execution_complete") {
                            const execution = toExecutionItem(event.data);
                            if (execution) {
                                queryClient.setQueryData(["execution", execution.id], execution);
                                setOptimisticExecution(execution);
                                setShowNewConversation(false);
                                setSelectedExecutionId(execution.id);
                            }
                        }
                    },
                },
            );

            await queryClient.invalidateQueries({ queryKey: ["executions", "agent", agent.id] });
            if (selectedExecutionId) {
                await queryClient.invalidateQueries({ queryKey: ["execution", selectedExecutionId] });
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : "Execution failed.";
            setStreamError(message);
            toast.error(message);
        } finally {
            abortRef.current = null;
            setIsStreaming(false);
        }
    };

    const canSend =
        Boolean(draft.trim()) &&
        !isStreaming &&
        agent.enabled &&
        !(selectedExecutionId && executionQuery.isLoading && !showNewConversation);

    return (
        <div className="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
            <Card className="h-fit">
                <CardHeader>
                    <div className="flex items-start justify-between gap-3">
                        <div>
                            <CardTitle>Conversation history</CardTitle>
                            <CardDescription>
                                Reopen saved runs for this agent and continue from any prior execution.
                            </CardDescription>
                        </div>
                        <Badge variant="outline">{historyQuery.data?.length ?? 0}</Badge>
                    </div>
                </CardHeader>
                <CardContent className="space-y-3">
                    <Button type="button" variant="outline" className="w-full" onClick={handleNewConversation}>
                        <MessageSquareDashed className="mr-1 size-4" />
                        New conversation
                    </Button>

                    {historyQuery.isLoading ? (
                        <div className="space-y-2">
                            {Array.from({ length: 4 }).map((_, index) => (
                                <div key={index} className="h-18 animate-pulse rounded-xl bg-muted" />
                            ))}
                        </div>
                    ) : historyQuery.isError ? (
                        <p className="text-sm text-destructive">
                            {historyQuery.error instanceof Error
                                ? historyQuery.error.message
                                : "Couldn’t load execution history."}
                        </p>
                    ) : (historyQuery.data?.length ?? 0) === 0 ? (
                        <p className="text-sm text-muted-foreground">
                            No saved executions yet. Send a message to create the first conversation.
                        </p>
                    ) : (
                        <div className="space-y-2">
                            {historyQuery.data?.map((execution) => {
                                const isSelected = !showNewConversation && selectedExecutionId === execution.id;

                                return (
                                    <button
                                        key={execution.id}
                                        type="button"
                                        onClick={() => handleSelectExecution(execution.id)}
                                        className={cn(
                                            "w-full rounded-xl border px-3 py-3 text-left transition-colors",
                                            isSelected
                                                ? "border-primary bg-primary/5"
                                                : "hover:border-primary/40 hover:bg-muted/40",
                                        )}
                                    >
                                        <div className="flex items-center justify-between gap-2">
                                            <Badge variant={getStatusVariant(execution.status)}>
                                                {execution.status}
                                            </Badge>
                                            <span className="text-[11px] text-muted-foreground">
                                                {formatDateTime(execution.created_at)}
                                            </span>
                                        </div>
                                        <p className="mt-2 line-clamp-2 text-sm font-medium text-foreground">
                                            {getExecutionSnippet(execution)}
                                        </p>
                                        <p className="mt-1 text-xs text-muted-foreground">
                                            {execution.steps?.length ?? 0} step(s)
                                            {execution.total_latency_ms ? ` • ${formatLatency(execution.total_latency_ms)}` : ""}
                                        </p>
                                    </button>
                                );
                            })}
                        </div>
                    )}
                </CardContent>
            </Card>

            <div className="space-y-6">
                <Card>
                    <CardHeader>
                        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                            <div>
                                <CardTitle className="flex items-center gap-2">
                                    <Sparkles className="size-5" />
                                    Test `{agent.name}`
                                </CardTitle>
                                <CardDescription>
                                    Send prompts through the execute API and inspect each stage of the run.
                                </CardDescription>
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                                <Badge variant={agent.enabled ? "default" : "secondary"}>
                                    {agent.enabled ? "Enabled" : "Disabled"}
                                </Badge>
                                {selectedSummary ? (
                                    <Badge variant="outline">Execution {selectedSummary.id.slice(0, 8)}</Badge>
                                ) : (
                                    <Badge variant="outline">Fresh conversation</Badge>
                                )}
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="min-h-90 rounded-2xl border bg-muted/20 p-4">
                            {executionQuery.isLoading && selectedExecutionId && !showNewConversation && !isStreaming ? (
                                <div className="flex min-h-80 items-center justify-center text-sm text-muted-foreground">
                                    <Loader2 className="mr-2 size-4 animate-spin" />
                                    Loading execution transcript…
                                </div>
                            ) : conversationMessages.length === 0 ? (
                                <div className="flex min-h-80 flex-col items-center justify-center gap-3 text-center text-sm text-muted-foreground">
                                    <MessageSquareDashed className="size-10" />
                                    <div>
                                        <p className="font-medium text-foreground">No conversation selected</p>
                                        <p>Start a fresh test run or reopen a previous execution from the left.</p>
                                    </div>
                                </div>
                            ) : (
                                <div className="flex max-h-140 flex-col gap-3 overflow-y-auto pr-1">
                                    {conversationMessages.map((message) => (
                                        <MessageBubble key={message.id} message={message} />
                                    ))}
                                </div>
                            )}
                        </div>

                        {streamError ? (
                            <div className="rounded-xl border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                                {streamError}
                            </div>
                        ) : null}

                        {!agent.enabled ? (
                            <div className="rounded-xl border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm text-amber-700 dark:text-amber-300">
                                This agent is disabled. Enable it before running test conversations.
                            </div>
                        ) : null}

                        <div className="space-y-3 rounded-2xl border p-3">
                            <Textarea
                                value={draft}
                                onChange={(event) => setDraft(event.target.value)}
                                placeholder={
                                    showNewConversation || !selectedSummary
                                        ? "Ask the agent a question…"
                                        : "Continue this conversation…"
                                }
                                disabled={!agent.enabled || isStreaming}
                                className="min-h-24"
                            />
                            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                                <p className="text-xs text-muted-foreground">
                                    {showNewConversation || !selectedSummary
                                        ? "A new execution will be created."
                                        : `Continuing execution ${selectedSummary.id.slice(0, 8)} with the saved conversation context.`}
                                </p>
                                <div className="flex gap-2">
                                    <Button
                                        type="button"
                                        variant="outline"
                                        onClick={() => {
                                            setDraft("");
                                            setStreamError(null);
                                        }}
                                        disabled={!draft && !streamError}
                                    >
                                        <RefreshCcw className="mr-1 size-4" />
                                        Clear
                                    </Button>
                                    <Button type="button" onClick={handleSend} disabled={!canSend}>
                                        {isStreaming ? (
                                            <>
                                                <Loader2 className="mr-1 size-4 animate-spin" />
                                                Running…
                                            </>
                                        ) : (
                                            <>
                                                <Play className="mr-1 size-4" />
                                                Send
                                            </>
                                        )}
                                    </Button>
                                </div>
                            </div>
                        </div>
                    </CardContent>
                </Card>

                <ExecutionInspector
                    execution={selectedExecution}
                    isStreaming={isStreaming}
                    liveEvents={liveEvents}
                />
            </div>
        </div>
    );
}

function MessageBubble({ message }: { message: DisplayMessage }) {
    const isUser = message.role === "user";
    const isAssistant = message.role === "assistant";
    const isTool = message.role === "tool";

    if (isTool) {
        return (
            <div className="flex justify-start">
                <details className="max-w-[85%] rounded-2xl border border-emerald-500/20 bg-emerald-500/5 shadow-sm">
                    <summary className="cursor-pointer list-none px-4 py-3">
                        <div className="mb-2 flex flex-wrap items-center gap-2 text-[11px] uppercase tracking-wide opacity-80">
                            <span className="inline-flex items-center gap-1 font-medium">
                                <Wrench className="size-3" />
                                {message.role}
                            </span>
                            {message.stageName ? <Badge variant="outline">{message.stageName}</Badge> : null}
                            {message.name ? <Badge variant="outline">{message.name}</Badge> : null}
                            <Badge variant="secondary">Click to expand</Badge>
                        </div>
                        <p className="text-sm leading-6 text-muted-foreground">
                            {getToolMessagePreview(message.content)}
                        </p>
                    </summary>
                    <div className="border-t px-4 py-3">
                        <p className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                            JSON response
                        </p>
                        <pre className="overflow-x-auto whitespace-pre-wrap rounded-xl bg-background/70 p-3 text-[11px] leading-6 text-foreground">
                            {formatToolMessageContent(message.content)}
                        </pre>
                    </div>
                </details>
            </div>
        );
    }

    return (
        <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
            <div
                className={cn(
                    "max-w-[85%] rounded-2xl border px-4 py-3 shadow-sm",
                    isUser
                        ? "border-primary/20 bg-primary text-primary-foreground"
                        : isAssistant
                            ? "bg-background"
                            : "bg-background",
                )}
            >
                <div className="mb-2 flex flex-wrap items-center gap-2 text-[11px] uppercase tracking-wide opacity-80">
                    <span className="inline-flex items-center gap-1 font-medium">
                        {isUser ? <User className="size-3" /> : <Bot className="size-3" />}
                        {message.role}
                    </span>
                    {message.stageName ? <Badge variant="outline">{message.stageName}</Badge> : null}
                    {message.name ? <Badge variant="outline">{message.name}</Badge> : null}
                </div>
                {isAssistant ? (
                    <MarkdownContent content={message.content} />
                ) : (
                    <p className="whitespace-pre-wrap wrap-break-word text-sm leading-6">
                        {message.content?.trim() || "(no content)"}
                    </p>
                )}
                {message.tool_calls?.length ? (
                    <div className="mt-3 space-y-2 rounded-xl border border-dashed px-3 py-2 text-xs">
                        {message.tool_calls.map((toolCall) => (
                            <div key={toolCall.id}>
                                <p className="font-medium">Tool call: {toolCall.tool_name}</p>
                                <pre className="mt-1 overflow-x-auto whitespace-pre-wrap text-[11px] text-muted-foreground">
                                    {safeStringify(toolCall.arguments)}
                                </pre>
                            </div>
                        ))}
                    </div>
                ) : null}
            </div>
        </div>
    );
}

function MarkdownContent({
    content,
    className,
}: {
    content?: string;
    className?: string;
}) {
    const value = content?.trim();

    if (!value) {
        return <p className={cn("text-sm leading-6", className)}>(no content)</p>;
    }

    return (
        <div
            className={cn(
                "max-w-none text-sm leading-6 wrap-break-word [&_p]:mb-3 [&_p:last-child]:mb-0 [&_ul]:my-3 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:my-3 [&_ol]:list-decimal [&_ol]:pl-5 [&_li]:mb-1 [&_blockquote]:my-3 [&_blockquote]:border-l-2 [&_blockquote]:pl-3 [&_blockquote]:italic [&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded-xl [&_pre]:bg-muted/60 [&_pre]:p-3 [&_pre]:text-[12px] [&_code]:rounded [&_code]:bg-muted/60 [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.95em] [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_table]:my-3 [&_table]:w-full [&_th]:border [&_th]:px-2 [&_th]:py-1 [&_td]:border [&_td]:px-2 [&_td]:py-1",
                className,
            )}
        >
            <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                components={{
                    a: ({ ...props }) => (
                        <a
                            {...props}
                            target="_blank"
                            rel="noreferrer"
                            className="font-medium underline underline-offset-4"
                        />
                    ),
                }}
            >
                {value}
            </ReactMarkdown>
        </div>
    );
}

function ExecutionInspector({
    execution,
    isStreaming,
    liveEvents,
}: {
    execution: ExecutionItem | null;
    isStreaming: boolean;
    liveEvents: Array<ExecutionStreamEvent<unknown>>;
}) {
    return (
        <Card>
            <CardHeader>
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                        <CardTitle className="flex items-center gap-2">
                            <Workflow className="size-5" />
                            Execution metadata
                        </CardTitle>
                        <CardDescription>
                            Inspect stage inputs, model output, tool activity, and routing transitions.
                        </CardDescription>
                    </div>
                    {execution ? (
                        <div className="flex flex-wrap items-center gap-2">
                            <Badge variant={getStatusVariant(execution.status)}>{execution.status}</Badge>
                            <Badge variant="outline">{execution.steps?.length ?? 0} step(s)</Badge>
                            {execution.total_latency_ms ? (
                                <Badge variant="outline">{formatLatency(execution.total_latency_ms)}</Badge>
                            ) : null}
                        </div>
                    ) : isStreaming ? (
                        <Badge variant="secondary">Streaming live events…</Badge>
                    ) : null}
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                {execution ? (
                    <>
                        <div className="grid gap-3 md:grid-cols-4">
                            <SummaryStat label="Created" value={formatDateTime(execution.created_at)} />
                            <SummaryStat label="Updated" value={formatDateTime(execution.updated_at)} />
                            <SummaryStat label="Completed" value={formatDateTime(execution.completed_at)} />
                            <SummaryStat label="Final output" value={execution.final_output?.trim() ? "Available" : "—"} />
                        </div>

                        {execution.metadata && Object.keys(execution.metadata).length > 0 ? (
                            <details className="rounded-xl border bg-muted/20">
                                <summary className="cursor-pointer px-4 py-3 text-sm font-medium">Execution metadata</summary>
                                <pre className="overflow-x-auto border-t px-4 py-3 text-xs text-muted-foreground">
                                    {safeStringify(execution.metadata)}
                                </pre>
                            </details>
                        ) : null}

                        {Object.keys(execution.input ?? {}).length > 0 ? (
                            <details className="rounded-xl border bg-muted/20">
                                <summary className="cursor-pointer px-4 py-3 text-sm font-medium">Raw execution input</summary>
                                <pre className="overflow-x-auto border-t px-4 py-3 text-xs text-muted-foreground">
                                    {safeStringify(execution.input)}
                                </pre>
                            </details>
                        ) : null}

                        {(execution.steps ?? []).length === 0 ? (
                            <p className="text-sm text-muted-foreground">This execution has no recorded steps yet.</p>
                        ) : (
                            <div className="space-y-3">
                                {(execution.steps ?? []).map((step, index) => (
                                    <StepCard key={step.id || `${step.stage_id}-${index}`} step={step} index={index} />
                                ))}
                            </div>
                        )}
                    </>
                ) : isStreaming ? (
                    <div className="space-y-2">
                        {liveEvents.length === 0 ? (
                            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                <Loader2 className="size-4 animate-spin" />
                                Waiting for stage events…
                            </div>
                        ) : (
                            liveEvents.filter(event => event.event != "llm_token").map((event, index) => (
                                <div key={`${event.event}-${index}`} className="rounded-xl border px-3 py-2 text-sm">
                                    <div className="flex items-center justify-between gap-2">
                                        <span className="font-medium">{event.event}</span>
                                        <Badge variant="outline">live</Badge>
                                    </div>
                                    <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-muted-foreground">
                                        {safeStringify(event.data)}
                                    </pre>
                                </div>
                            ))
                        )}
                    </div>
                ) : (
                    <p className="text-sm text-muted-foreground">
                        Select an execution to inspect its routing, model calls, and tool activity.
                    </p>
                )}
            </CardContent>
        </Card>
    );
}

function StepCard({ step, index }: { step: ExecutionStepItem; index: number }) {
    return (
        <details className="rounded-xl border bg-muted/10" open={index === 0}>
            <summary className="flex cursor-pointer list-none flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                    <p className="font-medium text-foreground">{step.stage_name || step.stage_id || `Step ${index + 1}`}</p>
                    <p className="text-xs text-muted-foreground">
                        {step.stage_type} • {formatDateTime(step.started_at)}
                    </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={getStatusVariant(step.status)}>{step.status}</Badge>
                    {step.next_stage_id ? <Badge variant="outline">→ {step.next_stage_id}</Badge> : null}
                </div>
            </summary>
            <div className="space-y-4 border-t px-4 py-4">
                {(step.input_messages ?? []).length > 0 ? (
                    <div className="space-y-2">
                        <p className="text-sm font-medium">LLM / stage input</p>
                        <MessageList messages={step.input_messages ?? []} />
                    </div>
                ) : null}

                {step.output_message ? (
                    <div className="space-y-2">
                        <p className="text-sm font-medium">Stage output</p>
                        <MessageList messages={[step.output_message]} />
                    </div>
                ) : null}

                {step.transition_reason ? (
                    <div className="rounded-xl border border-dashed px-3 py-2 text-sm">
                        <p className="font-medium">Stage transition</p>
                        <p className="mt-1 text-muted-foreground">{step.transition_reason}</p>
                    </div>
                ) : null}

                {(step.retrieved_context ?? []).length > 0 ? (
                    <div className="space-y-2">
                        <p className="text-sm font-medium">Retrieved context</p>
                        <ul className="space-y-2 text-sm text-muted-foreground">
                            {(step.retrieved_context ?? []).map((item, itemIndex) => (
                                <li key={itemIndex} className="rounded-lg border px-3 py-2">
                                    {item}
                                </li>
                            ))}
                        </ul>
                    </div>
                ) : null}

                {(step.llm_calls ?? []).length > 0 ? (
                    <div className="space-y-2">
                        <p className="text-sm font-medium">LLM call metadata</p>
                        <div className="grid gap-3 md:grid-cols-2">
                            {(step.llm_calls ?? []).map((call, callIndex) => (
                                <div key={`${call.provider}-${call.model}-${callIndex}`} className="rounded-xl border px-3 py-3 text-sm">
                                    <p className="font-medium text-foreground">
                                        {call.provider} / {call.model}
                                    </p>
                                    <div className="mt-2 space-y-1 text-muted-foreground">
                                        <p>Prompt tokens: {call.prompt_tokens ?? 0}</p>
                                        <p>Completion tokens: {call.completion_tokens ?? 0}</p>
                                        <p>Total tokens: {call.total_tokens ?? 0}</p>
                                        <p>Latency: {formatLatency(call.latency_ms)}</p>
                                        <p>Finish reason: {call.finish_reason || "—"}</p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ) : null}

                {(step.tool_calls ?? []).length > 0 ? (
                    <div className="space-y-2">
                        <p className="text-sm font-medium">Tool activity</p>
                        <div className="space-y-3">
                            {(step.tool_calls ?? []).map((toolCall) => (
                                <div key={toolCall.id} className="rounded-xl border px-3 py-3 text-sm">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <Badge variant="outline">{toolCall.tool_name}</Badge>
                                        {toolCall.tool_type ? (
                                            <Badge variant="secondary">{toolCall.tool_type}</Badge>
                                        ) : null}
                                        {toolCall.latency_ms ? (
                                            <Badge variant="outline">{formatLatency(toolCall.latency_ms)}</Badge>
                                        ) : null}
                                    </div>
                                    <div className="mt-3 grid gap-3 lg:grid-cols-2">
                                        <div>
                                            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                                Arguments
                                            </p>
                                            <pre className="mt-1 overflow-x-auto rounded-lg bg-muted/40 p-2 text-[11px] text-muted-foreground">
                                                {safeStringify(toolCall.arguments)}
                                            </pre>
                                        </div>
                                        <div>
                                            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                                Result
                                            </p>
                                            <pre className="mt-1 overflow-x-auto rounded-lg bg-muted/40 p-2 text-[11px] text-muted-foreground">
                                                {toolCall.error ? `Error: ${toolCall.error}` : toolCall.result || "(no result)"}
                                            </pre>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ) : null}

                {step.error ? (
                    <div className="rounded-xl border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                        {step.error}
                    </div>
                ) : null}

                {step.metadata && Object.keys(step.metadata).length > 0 ? (
                    <details className="rounded-xl border bg-muted/20">
                        <summary className="cursor-pointer px-3 py-2 text-sm font-medium">Raw step metadata</summary>
                        <pre className="overflow-x-auto border-t px-3 py-2 text-[11px] text-muted-foreground">
                            {safeStringify(step.metadata)}
                        </pre>
                    </details>
                ) : null}
            </div>
        </details>
    );
}

function SummaryStat({ label, value }: { label: string; value: string }) {
    return (
        <div className="rounded-xl border bg-muted/20 px-3 py-3">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
            <p className="mt-1 text-sm font-medium text-foreground">{value}</p>
        </div>
    );
}

function MessageList({ messages }: { messages: ExecutionMessageItem[] }) {
    return (
        <div className="space-y-2">
            {messages.map((message, index) => (
                <div key={`${message.role}-${index}`} className="rounded-lg border px-3 py-2 text-sm">
                    <div className="mb-1 flex items-center gap-2">
                        <Badge variant="outline">{message.role}</Badge>
                        {message.name ? <Badge variant="secondary">{message.name}</Badge> : null}
                    </div>
                    {message.role === "assistant" ? (
                        <MarkdownContent content={message.content} className="text-muted-foreground" />
                    ) : (
                        <p className="whitespace-pre-wrap wrap-break-word text-muted-foreground">
                            {message.content?.trim() || "(no content)"}
                        </p>
                    )}
                </div>
            ))}
        </div>
    );
}

function buildConversationMessages(execution: ExecutionItem | null): DisplayMessage[] {
    if (!execution) {
        return [];
    }

    const messages: DisplayMessage[] = [];
    const initialMessages = getInitialMessages(execution.input);

    initialMessages.forEach((message, index) => {
        messages.push({
            ...message,
            id: `input-${index}`,
        });
    });

    (execution.steps ?? []).forEach((step, stepIndex) => {
        if (step.output_message) {
            messages.push({
                ...step.output_message,
                id: `${step.id || step.stage_id || stepIndex}-output`,
                stageName: step.stage_name,
                stageType: step.stage_type,
            });
        }

        (step.tool_calls ?? []).forEach((toolCall, toolIndex) => {
            if (!toolCall.result && !toolCall.error) {
                return;
            }

            messages.push({
                id: `${step.id || step.stage_id || stepIndex}-tool-${toolIndex}`,
                role: "tool",
                name: toolCall.tool_name,
                tool_call_id: toolCall.id,
                content: toolCall.error ? `Error: ${toolCall.error}` : toolCall.result,
                stageName: step.stage_name,
                stageType: step.stage_type,
            });
        });
    });

    if (!messages.some((message) => message.role === "assistant") && execution.final_output?.trim()) {
        messages.push({
            id: `${execution.id}-final-output`,
            role: "assistant",
            content: execution.final_output,
            stageName: "Final output",
            stageType: "chat",
        });
    }

    return messages;
}

function getInitialMessages(input: JsonObject): ExecutionMessageItem[] {
    const rawMessages = input["messages"];
    if (Array.isArray(rawMessages)) {
        return rawMessages
            .map((item) => normalizeMessage(item))
            .filter((item): item is ExecutionMessageItem => item !== null);
    }

    const rawMessage = input["message"];
    if (typeof rawMessage === "string" && rawMessage.trim()) {
        return [
            {
                role: "user",
                content: rawMessage,
            },
        ];
    }

    return [];
}

function normalizeMessage(value: unknown): ExecutionMessageItem | null {
    if (!isJsonObject(value) || typeof value.role !== "string") {
        return null;
    }

    return {
        role: value.role,
        content: typeof value.content === "string" ? value.content : "",
        tool_calls: Array.isArray(value.tool_calls)
            ? value.tool_calls
                .map((toolCall) => normalizeToolCall(toolCall))
                .filter((toolCall): toolCall is NormalizedToolCall => toolCall !== null)
            : undefined,
        tool_call_id: typeof value.tool_call_id === "string" ? value.tool_call_id : undefined,
        name: typeof value.name === "string" ? value.name : undefined,
    };
}

function normalizeToolCall(value: unknown): NormalizedToolCall | null {
    if (!isJsonObject(value) || typeof value.id !== "string" || typeof value.tool_name !== "string") {
        return null;
    }

    return {
        id: value.id,
        tool_name: value.tool_name,
        arguments: isJsonObject(value.arguments) ? value.arguments : {},
    };
}

function stripDisplayMetadata(message: DisplayMessage): ExecutionMessageItem {
    return {
        role: message.role,
        content: message.content,
        tool_calls: message.tool_calls,
        tool_call_id: message.tool_call_id,
        name: message.name,
    };
}

function appendAssistantDelta(messages: DisplayMessage[], delta: string): DisplayMessage[] {
    if (!delta) {
        return messages;
    }

    const next = [...messages];
    const lastIndex = next.length - 1;
    const lastMessage = next[lastIndex];

    if (lastMessage?.role === "assistant") {
        next[lastIndex] = {
            ...lastMessage,
            content: `${lastMessage.content ?? ""}${delta}`,
        };
        return next;
    }

    next.push({
        id: `assistant-${next.length}`,
        role: "assistant",
        content: delta,
    });

    return next;
}

function buildToolResultMessage(data: unknown, index: number): DisplayMessage | null {
    if (!isJsonObject(data)) {
        return null;
    }

    const toolName = typeof data.tool_name === "string" ? data.tool_name : "tool";
    const result = typeof data.result === "string" ? data.result : "";
    const error = typeof data.error === "string" ? data.error : "";

    if (!result && !error) {
        return null;
    }

    return {
        id: `tool-result-${index}`,
        role: "tool",
        name: toolName,
        content: error ? `Error: ${error}` : result,
    };
}

function extractDelta(data: unknown): string {
    if (typeof data === "string") {
        return data;
    }

    if (!isJsonObject(data)) {
        return "";
    }

    if (typeof data.delta === "string") {
        return data.delta;
    }

    if (typeof data.token === "string") {
        return data.token;
    }

    return "";
}

function extractErrorMessage(data: unknown): string {
    if (typeof data === "string") {
        return data;
    }

    if (isJsonObject(data) && typeof data.error === "string") {
        return data.error;
    }

    return "";
}

function toExecutionItem(data: unknown): ExecutionItem | null {
    if (!isJsonObject(data) || typeof data.id !== "string") {
        return null;
    }

    return data as unknown as ExecutionItem;
}

function getExecutionSnippet(execution: ExecutionItem): string {
    const finalOutput = execution.final_output?.trim();
    if (finalOutput) {
        return finalOutput;
    }

    const initialMessages = getInitialMessages(execution.input);
    const lastMessage = initialMessages[initialMessages.length - 1];
    return lastMessage?.content?.trim() || "Execution recorded";
}

function formatDateTime(value?: string | null): string {
    if (!value) {
        return "—";
    }

    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}

function formatLatency(value?: number): string {
    if (!value || Number.isNaN(value)) {
        return "—";
    }

    if (value < 1_000) {
        return `${value} ms`;
    }

    return `${(value / 1_000).toFixed(2)} s`;
}

function getStatusVariant(status?: string) {
    switch ((status || "").toLowerCase()) {
        case "completed":
            return "default" as const;
        case "failed":
            return "destructive" as const;
        case "running":
            return "secondary" as const;
        default:
            return "outline" as const;
    }
}

function isJsonObject(value: unknown): value is JsonObject {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function getToolMessagePreview(value?: string): string {
    const trimmed = value?.trim();

    if (!trimmed) {
        return "No tool response.";
    }

    try {
        const parsed = JSON.parse(trimmed) as unknown;

        if (Array.isArray(parsed)) {
            return `JSON array (${parsed.length} item${parsed.length === 1 ? "" : "s"})`;
        }

        if (parsed && typeof parsed === "object") {
            const keys = Object.keys(parsed as Record<string, unknown>);
            return keys.length ? `JSON object: ${keys.slice(0, 4).join(", ")}` : "JSON object";
        }
    } catch {
        // Fall back to plain text preview below.
    }

    return trimmed.length > 140 ? `${trimmed.slice(0, 137)}…` : trimmed;
}

function formatToolMessageContent(value?: string): string {
    const trimmed = value?.trim();

    if (!trimmed) {
        return safeStringify({ result: null });
    }

    try {
        return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
        if (trimmed.startsWith("Error:")) {
            return safeStringify({ error: trimmed.slice("Error:".length).trim() });
        }

        return safeStringify({ result: trimmed });
    }
}

function safeStringify(value: unknown): string {
    try {
        return JSON.stringify(value, null, 2);
    } catch {
        return String(value);
    }
}
