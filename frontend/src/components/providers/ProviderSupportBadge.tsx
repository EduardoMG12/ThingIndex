"use client";

import type { ProviderSupport } from "@/lib/api/providers";
import { cn } from "@/lib/utils";
import { getProviderSupportBadgeClass, getProviderSupportLabel } from "@/lib/provider-support";

export function ProviderSupportBadge({
    level,
    className,
}: {
    level?: ProviderSupport["support_level"] | null;
    className?: string;
}) {
    return (
        <span
            className={cn(
                "inline-flex rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.16em]",
                getProviderSupportBadgeClass(level),
                className
            )}
        >
            {getProviderSupportLabel(level)}
        </span>
    );
}
