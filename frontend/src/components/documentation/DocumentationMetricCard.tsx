import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface DocumentationMetricCardProps {
    label: string;
    value: ReactNode;
    detail: ReactNode;
    valueClassName?: string;
    detailClassName?: string;
}

export function DocumentationMetricCard({
    label,
    value,
    detail,
    valueClassName,
    detailClassName,
}: DocumentationMetricCardProps) {
    return (
        <div className="rounded-xl border border-border bg-background/50 p-4">
            <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-muted-foreground">{label}</p>
            <p className={cn("mt-2 text-2xl font-semibold text-foreground", valueClassName)}>{value}</p>
            <p className={cn("mt-1 text-sm text-muted-foreground", detailClassName)}>{detail}</p>
        </div>
    );
}
