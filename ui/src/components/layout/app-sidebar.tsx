"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
    BookOpenText,
    BrainCircuit,
    HeartPulse,
    Sparkles,
    Wrench,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const navItems = [
    {
        href: "/tools",
        label: "Tools",
        icon: Wrench,
        soon: false,
    },
    {
        href: "/knowledgebase",
        label: "Knowledge Base",
        icon: BookOpenText,
        soon: false,
    },
    {
        href: "/providers",
        label: "LLM Providers",
        icon: BrainCircuit,
        soon: false,
    },
    {
        href: "/health",
        label: "Health",
        icon: HeartPulse,
        soon: false,
    },
] as const;

export function AppSidebar() {
    const pathname = usePathname();

    return (
        <aside className="sticky top-0 hidden h-screen w-72 shrink-0 flex-col justify-between rounded-2xl border bg-card p-4 lg:flex">
            <div className="space-y-6">
                <div className="space-y-3">
                    <div className="flex items-center gap-2">
                        <div className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                            <Sparkles className="size-5" />
                        </div>
                        <div>
                            <p className="text-sm font-semibold">Open Nirmata</p>
                            <p className="text-xs text-muted-foreground">Agent Builder Console</p>
                        </div>
                    </div>
                    <div className="rounded-xl border bg-muted/50 p-3 text-xs text-muted-foreground">
                        Tools, knowledge bases, and LLM providers are live now.
                    </div>
                </div>

                <nav className="space-y-1">
                    {navItems.map((item) => {
                        const isActive =
                            pathname === item.href || pathname.startsWith(`${item.href}/`);
                        const Icon = item.icon;

                        return (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    buttonVariants({
                                        variant: isActive ? "default" : "ghost",
                                        size: "lg",
                                    }),
                                    "w-full justify-start gap-2",
                                )}
                            >
                                <Icon className="size-4" />
                                <span>{item.label}</span>
                                {item.soon ? (
                                    <Badge variant="outline" className="ml-auto text-[10px]">
                                        Soon
                                    </Badge>
                                ) : null}
                            </Link>
                        );
                    })}
                </nav>
            </div>

            <div className="rounded-xl border bg-muted/40 p-3 text-xs text-muted-foreground">
                No auth yet for v1. Set `NEXT_PUBLIC_API_BASE_URL` to point at the Go API.
            </div>
        </aside>
    );
}
