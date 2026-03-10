"use client";

import Link from "next/link";
import { LogIn, UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";

export const Navbar = () => {
    return (
        <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-xl">
            <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
                <Link href="/" className="flex items-center gap-2.5">
                    <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10">
                        <div className="h-4 w-4 bg-primary rounded-sm" />
                    </div>
                    <span className="font-display text-base font-bold tracking-tight">
                        thing<span className="text-primary">index</span>
                    </span>
                    <span className="rounded-full border border-border bg-secondary px-2 py-0.5 font-mono text-[9px] uppercase tracking-wider text-muted-foreground">
                        beta
                    </span>
                </Link>
                <div className="flex items-center gap-3">
                    <Link
                        href="/docs"
                        className="text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
                    >
                        Documentação
                    </Link>
                    <Link href="/login">
                        <Button variant="ghost" size="sm" className="text-xs h-8">
                            Login
                        </Button>
                    </Link>
                    <Link href="/register">
                        <Button size="sm" className="text-xs h-8">
                            Começar grátis
                        </Button>
                    </Link>
                </div>
            </div>
        </header>
    );
};
