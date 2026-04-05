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
import type { Tool } from "@/lib/types";
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

type ToolTableProps = {
    items: Tool[];
    isLoading?: boolean;
    error?: string | null;
    onDelete?: (tool: Tool) => void;
};

export function ToolTable({
    items,
    isLoading = false,
    error,
    onDelete,
}: ToolTableProps) {
    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Registered tools</CardTitle>
                    <CardDescription>Loading your tool inventory…</CardDescription>
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
                    <CardTitle>Couldn’t load tools</CardTitle>
                    <CardDescription>{error}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    if (items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>No tools yet</CardTitle>
                    <CardDescription>
                        Start by creating your first MCP or HTTP tool.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Link
                        href="/tools/new"
                        className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                    >
                        Create your first tool
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Registered tools</CardTitle>
                <CardDescription>
                    Search, inspect, and manage tool configurations from one place.
                </CardDescription>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Type</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Tags</TableHead>
                            <TableHead>Auth</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.map((tool) => (
                            <TableRow key={tool.id}>
                                <TableCell className="whitespace-normal">
                                    <div className="space-y-1">
                                        <p className="font-medium">{tool.name}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {tool.description?.trim() || "No description yet."}
                                        </p>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant="outline" className="uppercase">
                                        {tool.type}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={tool.enabled ? "default" : "secondary"}>
                                        {tool.enabled ? "Enabled" : "Disabled"}
                                    </Badge>
                                </TableCell>
                                <TableCell className="whitespace-normal">
                                    <div className="flex flex-wrap gap-1">
                                        {tool.tags?.length ? (
                                            tool.tags.map((tag) => (
                                                <Badge key={tag} variant="secondary">
                                                    {tag}
                                                </Badge>
                                            ))
                                        ) : (
                                            <span className="text-xs text-muted-foreground">—</span>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={tool.auth_configured ? "default" : "outline"}>
                                        {tool.auth_configured ? "Configured" : "None"}
                                    </Badge>
                                </TableCell>
                                <TableCell>{formatDate(tool.updated_at ?? tool.created_at)}</TableCell>
                                <TableCell>
                                    <div className="flex justify-end gap-2">
                                        <Link
                                            href={`/tools/${tool.id}`}
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
                                            onClick={() => onDelete?.(tool)}
                                        >
                                            Delete
                                        </Button>
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
