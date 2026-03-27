"use client";

import { AlertTriangle, BadgeInfo, DatabaseZap, SearchCheck } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

export function CostTrustBoundary({
    error,
}: {
    error: string | null;
}) {
    const { t } = useI18n();
    const items = [
        {
            icon: DatabaseZap,
            title: t("dashboard.cost.trust.originTitle"),
            detail: t("dashboard.cost.trust.originDetail"),
        },
        {
            icon: SearchCheck,
            title: t("dashboard.cost.trust.coverageTitle"),
            detail: t("dashboard.cost.trust.coverageDetail"),
        },
        {
            icon: BadgeInfo,
            title: t("dashboard.cost.trust.limitTitle"),
            detail: t("dashboard.cost.trust.limitDetail"),
        },
    ];

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-5">
            <p className="text-[10px] uppercase tracking-[0.28em] text-primary">
                {t("dashboard.cost.trust.title")}
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
                {t("dashboard.cost.trust.description")}
            </p>

            {error ? (
                <div className="mt-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-700 dark:text-amber-300">
                    <div className="flex items-start gap-2">
                        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                        <div>
                            <p className="font-semibold">{t("dashboard.cost.partialData.title")}</p>
                            <p className="mt-1 text-xs">{error}</p>
                        </div>
                    </div>
                </div>
            ) : null}

            <div className="mt-4 space-y-3">
                {items.map((item) => {
                    const Icon = item.icon;

                    return (
                        <div key={item.title} className="rounded-xl border border-border bg-card/60 p-4">
                            <div className="flex items-start gap-3">
                                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-primary/15 bg-primary/10 text-primary">
                                    <Icon className="h-4 w-4" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-sm font-semibold text-foreground">{item.title}</p>
                                    <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{item.detail}</p>
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
