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
import type { Knowledgebase } from "@/lib/types";
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

type KnowledgebaseTableProps = {
    items: Knowledgebase[];
    isLoading?: boolean;
    error?: string | null;
    onDelete?: (knowledgebase: Knowledgebase) => void;
};

export function KnowledgebaseTable({
    items,
    isLoading = false,
    error,
    onDelete,
}: KnowledgebaseTableProps) {
    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Knowledge bases</CardTitle>
                    <CardDescription>Loading your knowledge base registry…</CardDescription>
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
                    <CardTitle>Couldn’t load knowledge bases</CardTitle>
                    <CardDescription>{error}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    if (items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>No knowledge bases yet</CardTitle>
                    <CardDescription>
                        Create your first retrieval source to manage indexes, namespaces, and embeddings.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Link
                        href="/knowledgebase/new"
                        className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                    >
                        Create your first knowledge base
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Knowledge bases</CardTitle>
                <CardDescription>
                    Review retrieval provider settings, index targets, and auth status from one place.
                </CardDescription>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Provider</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Index target</TableHead>
                            <TableHead>Auth</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.map((knowledgebase) => (
                            <TableRow key={knowledgebase.id}>
                                <TableCell className="whitespace-normal">
                                    <div className="space-y-1">
                                        <p className="font-medium">{knowledgebase.name}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {knowledgebase.description?.trim() ||
                                                knowledgebase.embedding_model?.trim() ||
                                                "No description yet."}
                                        </p>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant="outline" className="uppercase">
                                        {knowledgebase.provider}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={knowledgebase.enabled ? "default" : "secondary"}>
                                        {knowledgebase.enabled ? "Enabled" : "Disabled"}
                                    </Badge>
                                </TableCell>
                                <TableCell className="whitespace-normal">
                                    <div className="space-y-1 text-sm">
                                        <p>{knowledgebase.index_name?.trim() || "—"}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {knowledgebase.namespace?.trim() ||
                                                knowledgebase.base_url?.trim() ||
                                                "No index or namespace configured."}
                                        </p>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={knowledgebase.auth_configured ? "default" : "outline"}>
                                        {knowledgebase.auth_configured ? "Configured" : "Optional"}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    {formatDate(knowledgebase.updated_at ?? knowledgebase.created_at)}
                                </TableCell>
                                <TableCell>
                                    <div className="flex justify-end gap-2">
                                        <Link
                                            href={`/knowledgebase/${knowledgebase.id}`}
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
                                            onClick={() => onDelete?.(knowledgebase)}
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
