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
import type { PromptFlow } from "@/lib/types";
import { cn } from "@/lib/utils";

const formatter = new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
});

function formatDate(value?: string | null) {
    if (!value) return "—";
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "—" : formatter.format(date);
}

type PromptFlowTableProps = {
    items: PromptFlow[];
    isLoading?: boolean;
    error?: string | null;
    onDelete?: (flow: PromptFlow) => void;
};

export function PromptFlowTable({
    items,
    isLoading = false,
    error,
    onDelete,
}: PromptFlowTableProps) {
    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Prompt flows</CardTitle>
                    <CardDescription>Loading your prompt flow configurations…</CardDescription>
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
                    <CardTitle>Couldn't load prompt flows</CardTitle>
                    <CardDescription>{error}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    if (items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>No prompt flows yet</CardTitle>
                    <CardDescription>
                        Start by creating your first multi-stage conversational flow.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Link
                        href="/prompt-flows/new"
                        className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                    >
                        Create your first flow
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Prompt flows</CardTitle>
                <CardDescription>
                    Manage multi-stage conversational workflows for your AI agents.
                </CardDescription>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Stages</TableHead>
                            <TableHead>Entry stage</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.map((flow) => (
                            <TableRow key={flow.id}>
                                <TableCell className="whitespace-normal">
                                    <div className="space-y-1">
                                        <p className="font-medium">{flow.name}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {flow.description?.trim() || "No description."}
                                        </p>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={flow.enabled ? "default" : "secondary"}>
                                        {flow.enabled ? "Enabled" : "Disabled"}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    <Badge variant="outline">
                                        {flow.stages?.length ?? 0} stage{(flow.stages?.length ?? 0) !== 1 ? "s" : ""}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    {flow.entry_stage_id ? (
                                        <span className="font-mono text-xs text-muted-foreground">
                                            {flow.entry_stage_id}
                                        </span>
                                    ) : (
                                        <span className="text-xs text-muted-foreground">—</span>
                                    )}
                                </TableCell>
                                <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                                    {formatDate(flow.updated_at)}
                                </TableCell>
                                <TableCell className="text-right">
                                    <div className="flex justify-end gap-2">
                                        <Link
                                            href={`/prompt-flows/${flow.id}`}
                                            className={cn(
                                                buttonVariants({ variant: "outline", size: "sm" }),
                                            )}
                                        >
                                            Edit
                                        </Link>
                                        {onDelete && (
                                            <Button
                                                variant="destructive"
                                                size="sm"
                                                onClick={() => onDelete(flow)}
                                            >
                                                Delete
                                            </Button>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    );
}
