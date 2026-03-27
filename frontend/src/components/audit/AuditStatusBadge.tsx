"use client";

import { cn } from "@/lib/utils";
import type { AppLocale } from "@/lib/i18n/config";
import { formatAuditStatus } from "@/lib/audit";

export default function AuditStatusBadge({
    status,
    locale,
}: {
    status?: string;
    locale: AppLocale;
}) {
    return (
        <span
            className={cn(
                "rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider",
                status === "SUCCESS"
                    ? "border-emerald-500/20 bg-emerald-500/10 text-emerald-400"
                    : "border-destructive/20 bg-destructive/10 text-destructive"
            )}
        >
            {formatAuditStatus(status, locale)}
        </span>
    );
}
