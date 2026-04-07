import Link from "next/link";

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
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import type { Agent } from "@/lib/types";
import { cn } from "@/lib/utils";

const formatter = new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
});

function formatDate(value?: string | null) {
    if (!value) {
        return "—";
    }

    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "—" : formatter.format(date);
}

type AgentTableProps = {
    items: Agent[];
    promptFlowNames?: Record<string, string>;
    isLoading?: boolean;
    error?: string | null;
    onDelete?: (agent: Agent) => void;
};

export function AgentTable({
    items,
    promptFlowNames,
    isLoading = false,
    error,
    onDelete,
}: AgentTableProps) {
    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Registered agents</CardTitle>
                    <CardDescription>Loading your agent registry…</CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                    {Array.from({ length: 4 }).map((_, index) => (
                        <div key={index} className="h-11 animate-pulse rounded-lg bg-muted" />
                    ))}
                </CardContent>
            </Card>
        );
    }

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Couldn’t load agents</CardTitle>
                    <CardDescription>{error}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    if (items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>No agents yet</CardTitle>
                    <CardDescription>
                        Create your first chat agent and connect it to a prompt flow.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Link
                        href="/agents/new"
                        className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                    >
                        Create your first agent
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Registered agents</CardTitle>
                <CardDescription>
                    Search, inspect, and manage saved agent definitions in one place.
                </CardDescription>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Type</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Prompt flow</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.map((agent) => {
                            const promptFlowName = promptFlowNames?.[agent.prompt_flow_id];

                            return (
                                <TableRow key={agent.id}>
                                    <TableCell className="whitespace-normal">
                                        <div className="space-y-1">
                                            <p className="font-medium">{agent.name}</p>
                                            <p className="text-xs text-muted-foreground">
                                                {agent.description?.trim() || "No description yet."}
                                            </p>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant="outline" className="uppercase">
                                            {agent.type}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={agent.enabled ? "default" : "secondary"}>
                                            {agent.enabled ? "Enabled" : "Disabled"}
                                        </Badge>
                                    </TableCell>
                                    <TableCell className="whitespace-normal">
                                        <div className="space-y-1">
                                            <p className="font-medium">
                                                {promptFlowName || agent.prompt_flow_id || "—"}
                                            </p>
                                            {promptFlowName ? (
                                                <p className="font-mono text-[11px] text-muted-foreground">
                                                    {agent.prompt_flow_id}
                                                </p>
                                            ) : null}
                                        </div>
                                    </TableCell>
                                    <TableCell>{formatDate(agent.updated_at ?? agent.created_at)}</TableCell>
                                    <TableCell>
                                        <div className="flex justify-end gap-2">
                                            <Link
                                                href={`/agents/${agent.id}/execute`}
                                                className={cn(
                                                    buttonVariants({ variant: "secondary", size: "sm" }),
                                                    "inline-flex",
                                                )}
                                            >
                                                Test
                                            </Link>
                                            <Link
                                                href={`/agents/${agent.id}`}
                                                className={cn(
                                                    buttonVariants({ variant: "outline", size: "sm" }),
                                                    "inline-flex",
                                                )}
                                            >
                                                Manage
                                            </Link>
                                            <Button
                                                type="button"
                                                variant="destructive"
                                                size="sm"
                                                onClick={() => onDelete?.(agent)}
                                            >
                                                Delete
                                            </Button>
                                        </div>
                                    </TableCell>
                                </TableRow>
                            );
                        })}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    );
}
