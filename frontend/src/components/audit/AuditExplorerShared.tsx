"use client";

import type { ReactNode } from "react";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function AuditSummaryCard({
    label,
    value,
    detail,
}: {
    label: string;
    value: string;
    detail: string;
}) {
    return (
        <div className="rounded-xl border border-border bg-background/60 p-4">
            <p className="text-[10px] uppercase tracking-[0.24em] text-muted-foreground">{label}</p>
            <p className="mt-3 text-xl font-semibold">{value}</p>
            <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
        </div>
    );
}

export function AuditScopeButton({
    active,
    children,
    onClick,
}: {
    active: boolean;
    children: ReactNode;
    onClick: () => void;
}) {
    return (
        <button
            type="button"
            onClick={onClick}
            className={cn(
                "rounded-full border px-3 py-1 text-[11px] font-semibold transition-colors",
                active
                    ? "border-primary/20 bg-primary/10 text-primary"
                    : "border-border bg-background text-muted-foreground"
            )}
        >
            {children}
        </button>
    );
}

export function AuditAppliedFilters({
    items,
    title,
}: {
    items: Array<{ key: string; label: string; value: string }>;
    title: string;
}) {
    if (items.length === 0) {
        return null;
    }

    return (
        <div className="rounded-xl border border-border bg-background/45 p-4">
            <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                {title}
            </p>
            <div className="mt-3 flex flex-wrap gap-2">
                {items.map((item) => (
                    <Badge key={item.key} variant="outline" className="gap-1 rounded-full bg-background/80 px-3 py-1">
                        <span className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{item.label}</span>
                        <span className="font-mono text-[11px] text-foreground">{item.value}</span>
                    </Badge>
                ))}
            </div>
        </div>
    );
}

export function AuditQuickFilterButton({
    label,
    count,
    active,
    tone = "default",
    onClick,
}: {
    label: string;
    count: number;
    active: boolean;
    tone?: "default" | "danger";
    onClick: () => void;
}) {
    return (
        <button
            type="button"
            disabled={count <= 0}
            onClick={onClick}
            className={cn(
                "inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-45",
                active
                    ? tone === "danger"
                        ? "border-destructive/30 bg-destructive/10 text-destructive"
                        : "border-primary/20 bg-primary/10 text-primary"
                    : "border-border bg-background/80 text-muted-foreground hover:bg-background",
                !active && tone === "danger" ? "hover:border-destructive/25 hover:text-destructive" : ""
            )}
        >
            <span>{label}</span>
            <span className="rounded-full bg-foreground/10 px-2 py-0.5 text-[10px] font-bold tracking-[0.14em]">
                {count}
            </span>
        </button>
    );
}
