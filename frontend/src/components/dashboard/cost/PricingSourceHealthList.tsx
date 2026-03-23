"use client";

import { AlertTriangle, CheckCircle2, Clock3, PowerOff, Settings2 } from "lucide-react";

import type { CostVisibilitySourceStatus, PricingSourceHealth } from "@/lib/cost-visibility";
import { useI18n } from "@/lib/i18n/provider";

export function PricingSourceHealthList({
    sources,
}: {
    sources: CostVisibilitySourceStatus[];
}) {
    const { locale, t } = useI18n();

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-5">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.28em] text-primary">
                <Settings2 className="h-3.5 w-3.5" />
                {t("dashboard.cost.sources.title")}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
                {t("dashboard.cost.sources.description")}
            </p>

            {sources.length > 0 ? (
                <div className="mt-4 space-y-3">
                    {sources.slice(0, 4).map((source) => (
                        <div key={source.id} className={`rounded-xl border p-4 ${buildSourceTone(source.health)}`}>
                            <div className="flex items-start justify-between gap-3">
                                <div className="min-w-0">
                                    <div className="flex items-center gap-2">
                                        {renderSourceIcon(source.health)}
                                        <p className="truncate text-sm font-semibold text-foreground">{source.name}</p>
                                    </div>
                                    <p className="mt-1 text-xs text-muted-foreground">
                                        {source.providerLabel}
                                        {source.isActiveRuntime ? ` · ${t("dashboard.cost.sources.activeRuntime")}` : ""}
                                    </p>
                                </div>
                                <span className="shrink-0 rounded-full border border-border bg-background/80 px-2 py-1 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
                                    {t(`dashboard.cost.sources.status.${source.health}`)}
                                </span>
                            </div>

                            <div className="mt-3 grid gap-2 text-[11px] text-muted-foreground">
                                <p>{t("dashboard.cost.sources.priceRows", { count: source.lastPriceCount })}</p>
                                <p>
                                    {source.lastSyncedAt
                                        ? t("dashboard.cost.sources.lastSync", {
                                            value: formatDateTime(source.lastSyncedAt, locale),
                                        })
                                        : t("dashboard.cost.sources.lastSyncMissing")}
                                </p>
                                {source.lastSyncError ? (
                                    <p className="text-destructive">
                                        {t("dashboard.cost.sources.lastError", {
                                            value: source.lastSyncError,
                                        })}
                                    </p>
                                ) : null}
                            </div>
                        </div>
                    ))}
                </div>
            ) : (
                <p className="mt-4 text-xs text-muted-foreground">
                    {t("dashboard.cost.sources.empty")}
                </p>
            )}
        </div>
    );
}

function buildSourceTone(health: PricingSourceHealth) {
    switch (health) {
        case "synced":
            return "border-emerald-500/20 bg-emerald-500/10";
        case "failed":
            return "border-destructive/20 bg-destructive/10";
        case "pending":
            return "border-amber-500/20 bg-amber-500/10";
        default:
            return "border-border bg-card/70";
    }
}

function renderSourceIcon(health: PricingSourceHealth) {
    switch (health) {
        case "synced":
            return <CheckCircle2 className="h-4 w-4 text-emerald-600 dark:text-emerald-300" />;
        case "failed":
            return <AlertTriangle className="h-4 w-4 text-destructive" />;
        case "pending":
            return <Clock3 className="h-4 w-4 text-amber-700 dark:text-amber-300" />;
        default:
            return <PowerOff className="h-4 w-4 text-muted-foreground" />;
    }
}

function formatDateTime(value: string, locale: string) {
    return new Intl.DateTimeFormat(locale, {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}
