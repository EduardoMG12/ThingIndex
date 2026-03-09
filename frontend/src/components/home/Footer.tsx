"use client";

import Link from "next/link";

export const Footer = () => {
    return (
        <footer className="border-t border-border bg-secondary/30">
            <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
                <div className="flex items-center gap-2">
                    <div className="h-5 w-5 bg-primary/20 rounded-sm" />
                    <span className="font-mono text-[10px] text-muted-foreground">
                        thingindex v0.1.0 · open source · TCC 2026
                    </span>
                </div>
                <div className="flex items-center gap-4">
                    <Link
                        href="/docs"
                        className="font-mono text-[10px] text-muted-foreground transition-colors hover:text-foreground"
                    >
                        docs
                    </Link>
                    <a
                        href="https://github.com/thingindex"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-mono text-[10px] text-muted-foreground transition-colors hover:text-foreground"
                    >
                        github
                    </a>
                </div>
            </div>
        </footer>
    );
};
