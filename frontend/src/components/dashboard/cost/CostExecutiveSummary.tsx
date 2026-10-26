"use client";

import Link from "next/link";
import { ArrowRight, ShieldCheck, TriangleAlert, Wrench } from "lucide-react";

import type { CostVisibilityHealth, CostVisibilityOverview } from "@/lib/cost-visibility";
import { useI18n } from "@/lib/i18n/provider";
import { formatUSD } from "@/lib/format";

type CostExecutiveSummaryProps = {
    overview: CostVisibilityOverview;
};

export function CostExecutiveSummary({ overview }: CostExecutiveSummaryProps) {
    const { t, href, locale } = useI18n();
    const isSetupState = overview.health === "setup";
    const metrics = isSetupState
        ? [
            {
                title: t("dashboard.cost.metrics.sources"),
                value: `${overview.syncedSourcesCount}/${overview.enabledSourcesCount}`,
                detail: t("dashboard.cost.metrics.sourcesDetail", {
                    synced: overview.syncedSourcesCount,
                    enabled: overview.enabledSourcesCount,
                    prices: overview.currentPriceCount,
                }),
            },
            {
                title: t("dashboard.cost.metrics.activeRuntime"),
                value: overview.activeModel || overview.activeProviderLabel || t("dashboard.cost.metrics.activeRuntimeMissing"),
                detail: overview.activeProviderLabel
                    ? t("dashboard.cost.metrics.activeRuntimePartial", {
                        provider: overview.activeProviderLabel,
                    })
                    : t("dashboard.cost.metrics.activeRuntimeNotConfigured"),
            },
        ]
        : [
            {
                title: t("dashboard.cost.metrics.observedCost"),
                value: formatUSD(overview.totalEstimatedCostUsd, locale),
                detail: t("dashboard.cost.metrics.observedCostDetail", {
                    tokens: overview.totalTokens.toLocaleString(locale),
                    requests: overview.totalRequests,
                }),
            },
            {
                title: t("dashboard.cost.metrics.coverage"),
                value: `${overview.pricingAwareDocuments}/${overview.documentsWithLlm}`,
                detail: overview.documentsWithLlm > 0
                    ? t("dashboard.cost.metrics.coverageDetail", {
                        priced: overview.pricingAwareDocuments,
                        total: overview.documentsWithLlm,
                    })
                    : t("dashboard.cost.metrics.coverageEmpty"),
            },
            {
                title: t("dashboard.cost.metrics.sources"),
                value: `${overview.syncedSourcesCount}/${overview.enabledSourcesCount}`,
                detail: t("dashboard.cost.metrics.sourcesDetail", {
                    synced: overview.syncedSourcesCount,
                    enabled: overview.enabledSourcesCount,
                    prices: overview.currentPriceCount,
                }),
            },
            {
                title: t("dashboard.cost.metrics.activeRuntime"),
                value: overview.activeModel || overview.activeProviderLabel || t("dashboard.cost.metrics.activeRuntimeMissing"),
                detail: overview.activeProviderLabel
                    ? overview.activeRuntimeCovered
                        ? t("dashboard.cost.metrics.activeRuntimeReady", {
                            provider: overview.activeProviderLabel,
                        })
                        : t("dashboard.cost.metrics.activeRuntimePartial", {
                            provider: overview.activeProviderLabel,
                        })
                    : t("dashboard.cost.metrics.activeRuntimeNotConfigured"),
            },
        ];

    return (
        <div className="rounded-2xl border border-border bg-background/70 p-5">
            <div className="flex flex-col gap-4">
                <div className="flex flex-col gap-3">
                    <div>
                        <p className="text-[10px] uppercase tracking-[0.28em] text-primary">
                            {t("dashboard.cost.executiveTitle")}
                        </p>
                        <h3 className="mt-3 text-lg font-semibold text-foreground">
                            {t(`dashboard.cost.health.title.${overview.health}`)}
                        </h3>
                        <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                            {t(`dashboard.cost.health.description.${overview.health}`)}
                        </p>
                    </div>
                    <div className={`${buildHealthTone(overview.health)} self-start`}>
                        {renderHealthIcon(overview.health)}
                        <span>{t(`dashboard.cost.health.badge.${overview.health}`)}</span>
                    </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                    {metrics.map((metric) => (
                        <MetricCard
                            key={metric.title}
                            title={metric.title}
                            value={metric.value}
                            detail={metric.detail}
                        />
                    ))}
                </div>

                <div className="flex flex-wrap gap-2">
                    <Link
                        href={href("/settings")}
                        className="inline-flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary"
                    >
                        <ArrowRight className="h-3.5 w-3.5" />
                        {t("dashboard.cost.actions.settings")}
                    </Link>
                    {!isSetupState ? (
                        <Link
                            href={href("/documents")}
                            className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground"
                        >
                            {t("dashboard.cost.actions.documents")}
                        </Link>
                    ) : null}
                </div>
            </div>
        </div>
    );
}

function MetricCard({
    title,
    value,
    detail,
}: {
    title: string;
    value: string;
    detail: string;
}) {
    return (
        <div className="rounded-xl border border-border bg-card/60 p-4">
            <p className="text-[10px] uppercase tracking-[0.24em] text-muted-foreground">{title}</p>
            <p className="mt-2 text-base font-semibold leading-tight text-foreground">{value}</p>
            <p className="mt-2 text-[11px] leading-relaxed text-muted-foreground">{detail}</p>
        </div>
    );
}

function buildHealthTone(health: CostVisibilityHealth) {
    switch (health) {
        case "trusted":
            return "inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300";
        case "partial":
            return "inline-flex items-center gap-2 rounded-full border border-amber-500/30 bg-amber-500/10 px-3 py-1.5 text-xs font-semibold text-amber-700 dark:text-amber-300";
        default:
            return "inline-flex items-center gap-2 rounded-full border border-border bg-background px-3 py-1.5 text-xs font-semibold text-muted-foreground";
    }
}

function renderHealthIcon(health: CostVisibilityHealth) {
    switch (health) {
        case "trusted":
            return <ShieldCheck className="h-4 w-4" />;
        case "partial":
            return <TriangleAlert className="h-4 w-4" />;
        default:
            return <Wrench className="h-4 w-4" />;
    }
}
