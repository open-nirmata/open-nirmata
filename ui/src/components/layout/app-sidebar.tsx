"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
    BookOpenText,
    BrainCircuit,
    ChevronLeft,
    ChevronRight,
    GitFork,
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
        href: "/prompt-flows",
        label: "Prompt Flows",
        icon: GitFork,
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
    const [isCollapsed, setIsCollapsed] = useState(false);

    // Load collapsed state from localStorage on mount
    useEffect(() => {
        const saved = localStorage.getItem("sidebar-collapsed");
        if (saved !== null) {
            setIsCollapsed(saved === "true");
        }
    }, []);

    // Save collapsed state to localStorage
    const toggleCollapsed = () => {
        const newState = !isCollapsed;
        setIsCollapsed(newState);
        localStorage.setItem("sidebar-collapsed", String(newState));
    };

    return (
        <aside
            className={cn(
                "sticky top-0 hidden h-screen shrink-0 flex-col justify-between rounded-2xl border bg-card p-4 transition-all duration-300 ease-in-out lg:flex",
                isCollapsed ? "w-20" : "w-72",
            )}
        >
            <div className="space-y-6">
                <div className="space-y-3">
                    <div className="flex items-center gap-2">
                        <div className="flex size-10 items-center justify-center rounded-xl">
                            <img src="/open-nirmata.png" alt="Open Nirmata" className="size-5" />
                        </div>
                        {!isCollapsed && (
                            <div className="min-w-0 flex-1">
                                <p className="text-sm font-semibold truncate">Open Nirmata</p>
                                <p className="text-xs text-muted-foreground truncate">Agent Builder Console</p>
                            </div>
                        )}
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
                                title={isCollapsed ? item.label : undefined}
                                className={cn(
                                    buttonVariants({
                                        variant: isActive ? "default" : "ghost",
                                        size: "lg",
                                    }),
                                    "w-full gap-2 transition-all",
                                    isCollapsed ? "justify-center px-2" : "justify-start",
                                )}
                            >
                                <Icon className="size-4 shrink-0" />
                                {!isCollapsed && (
                                    <>
                                        <span className="truncate">{item.label}</span>
                                        {item.soon ? (
                                            <Badge variant="outline" className="ml-auto text-[10px]">
                                                Soon
                                            </Badge>
                                        ) : null}
                                    </>
                                )}
                            </Link>
                        );
                    })}
                </nav>
            </div>

            <div className="space-y-3">
                <button
                    onClick={toggleCollapsed}
                    className={cn(
                        "flex w-full items-center gap-2 rounded-lg border bg-muted/40 p-2 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                        isCollapsed && "justify-center",
                    )}
                    title={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
                >
                    {isCollapsed ? (
                        <ChevronRight className="size-4" />
                    ) : (
                        <>
                            <ChevronLeft className="size-4" />
                            <span className="flex-1 text-left">Collapse</span>
                        </>
                    )}
                </button>
            </div>
        </aside>
    );
}
