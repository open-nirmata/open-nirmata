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
import type { LLMProvider } from "@/lib/types";
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

type ProviderTableProps = {
    items: LLMProvider[];
    isLoading?: boolean;
    error?: string | null;
    onDelete?: (provider: LLMProvider) => void;
};

export function ProviderTable({
    items,
    isLoading = false,
    error,
    onDelete,
}: ProviderTableProps) {
    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Configured providers</CardTitle>
                    <CardDescription>Loading your provider registry…</CardDescription>
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
                    <CardTitle>Couldn’t load providers</CardTitle>
                    <CardDescription>{error}</CardDescription>
                </CardHeader>
            </Card>
        );
    }

    if (items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>No providers yet</CardTitle>
                    <CardDescription>
                        Add your first hosted or local LLM provider to start routing model calls.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Link
                        href="/providers/new"
                        className={cn(buttonVariants({ variant: "default" }), "inline-flex")}
                    >
                        Add your first provider
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Configured providers</CardTitle>
                <CardDescription>
                    Review provider defaults, auth status, and operational availability from one place.
                </CardDescription>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Provider</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Default model</TableHead>
                            <TableHead>Auth</TableHead>
                            <TableHead>Updated</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {items.map((provider) => (
                            <TableRow key={provider.id}>
                                <TableCell className="whitespace-normal">
                                    <div className="space-y-1">
                                        <p className="font-medium">{provider.name}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {provider.description?.trim() || provider.base_url || "No description yet."}
                                        </p>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <Badge variant="outline" className="uppercase">
                                        {provider.provider}
                                    </Badge>
                                </TableCell>
                                <TableCell>
                                    <Badge variant={provider.enabled ? "default" : "secondary"}>
                                        {provider.enabled ? "Enabled" : "Disabled"}
                                    </Badge>
                                </TableCell>
                                <TableCell>{provider.default_model?.trim() || "—"}</TableCell>
                                <TableCell>
                                    <Badge variant={provider.auth_configured ? "default" : "outline"}>
                                        {provider.auth_configured ? "Configured" : "Missing"}
                                    </Badge>
                                </TableCell>
                                <TableCell>{formatDate(provider.updated_at ?? provider.created_at)}</TableCell>
                                <TableCell>
                                    <div className="flex justify-end gap-2">
                                        <Link
                                            href={`/providers/${provider.id}`}
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
                                            onClick={() => onDelete?.(provider)}
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
