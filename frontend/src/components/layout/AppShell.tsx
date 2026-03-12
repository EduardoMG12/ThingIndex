"use client";

import { LogOut, User, Settings, LayoutDashboard, Database, HelpCircle, BookOpen } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { logout } from "@/components/auth/actions";
import { Logo } from "@/components/ui/Logo";

interface AppShellProps {
    children: React.ReactNode;
}

const AppShell = ({ children }: AppShellProps) => {
    const pathname = usePathname();

    const navItems = [
        { label: "Dashboard", icon: LayoutDashboard, href: "/dashboard" },
        { label: "Documentos", icon: Database, href: "/documents" },
        { label: "Documentação", icon: BookOpen, href: "/documentation" },
        { label: "Configurações", icon: Settings, href: "/settings" },
    ];
    const activeItem = navItems.find((item) => pathname === item.href || pathname.startsWith(`${item.href}/`));

    return (
        <div className="flex h-screen w-full bg-background text-foreground overflow-hidden">
            <aside className="w-64 border-r border-border bg-card flex flex-col shrink-0">
                <div className="p-6 border-b border-border flex items-center gap-3">
                    <Logo className="text-primary" size={32} />
                    <div>
                        <h2 className="text-sm font-bold tracking-tight">thing<span className="text-primary">index</span></h2>
                        <p className="text-[10px] text-muted-foreground font-mono uppercase tracking-widest">v0.1.0</p>
                    </div>
                </div>

                <nav className="flex-1 p-4 space-y-1">
                    {navItems.map((item) => {
                        const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`);
                        const Icon = item.icon;
                        return (
                            <Link
                                key={item.href}
                                href={item.href}
                                className={cn(
                                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all group",
                                    isActive
                                        ? "bg-primary/10 text-primary font-medium"
                                        : "text-muted-foreground hover:bg-muted hover:text-foreground"
                                )}
                            >
                                <Icon className={cn("h-4 w-4", isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground")} />
                                {item.label}
                            </Link>
                        );
                    })}
                </nav>

                <div className="p-4 border-t border-border space-y-2">
                    <div className="flex items-center gap-3 px-3 py-2 rounded-lg bg-secondary/30 border border-border">
                        <div className="h-8 w-8 rounded-full bg-primary/20 flex items-center justify-center">
                            <User className="h-4 w-4 text-primary" />
                        </div>
                        <div className="min-w-0 flex-1">
                            <p className="text-xs font-semibold truncate leading-none">Eduardo MG</p>
                            <p className="text-[10px] text-muted-foreground truncate uppercase mt-1">Plano Explorer</p>
                        </div>
                    </div>
                    <form action={logout}>
                        <button type="submit" className="flex w-full items-center gap-3 px-3 py-2 rounded-lg text-sm text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-all">
                            <LogOut className="h-4 w-4" />
                            Sair
                        </button>
                    </form>
                </div>
            </aside>

            <main className="flex-1 flex flex-col overflow-hidden">
                <header className="h-14 border-b border-border bg-background/50 backdrop-blur-sm flex items-center justify-between px-8 shrink-0">
                    <div className="flex items-center gap-2">
                        <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">Workspace</span>
                        <span className="text-muted-foreground/30">/</span>
                        <span className="text-[10px] font-mono font-medium uppercase tracking-widest">{activeItem?.label || "Workspace"}</span>
                    </div>
                    <div className="flex items-center gap-4">
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground">
                            <HelpCircle className="h-4 w-4" />
                        </Button>
                    </div>
                </header>

                <div className="flex-1 overflow-y-auto custom-scrollbar" id="main-content">
                    <div className="max-w-7xl mx-auto p-8">
                        {children}
                    </div>
                </div>
            </main>
        </div>
    );
};

export default AppShell;
