import { AppSidebar } from "@/components/layout/app-sidebar";
import { Separator } from "@/components/ui/separator";
import { API_BASE_URL } from "@/lib/api/client";

export function AppShell({ children }: { children: React.ReactNode }) {
    return (
        <div className="min-h-screen bg-muted/30">
            <div className="mx-auto flex min-h-screen max-w-7xl gap-6 px-4 py-4 lg:px-6 lg:py-6">
                <AppSidebar />

                <div className="flex min-w-0 flex-1 flex-col rounded-2xl border bg-background shadow-sm">
                    <header className="space-y-4 px-5 py-4 lg:px-6">
                        <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
                            <div>
                                <p className="text-sm text-muted-foreground">Open Nirmata</p>
                                <h1 className="text-2xl font-semibold tracking-tight">
                                    AI agent platform admin UI
                                </h1>
                            </div>
                            <div className="rounded-lg border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
                                API base: <span className="font-medium text-foreground">{API_BASE_URL}</span>
                            </div>
                        </div>
                        <p className="max-w-2xl text-sm text-muted-foreground">
                            Manage tools today and keep the structure ready for knowledge bases,
                            providers, and more entities later.
                        </p>
                        <Separator />
                    </header>

                    <main className="flex-1 px-5 pb-5 lg:px-6 lg:pb-6">{children}</main>
                </div>
            </div>
        </div>
    );
}
