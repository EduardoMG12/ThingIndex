"use client";

import { Calculator } from "lucide-react";

import { CostExecutiveSummary } from "@/components/dashboard/cost/CostExecutiveSummary";
import { CostTrustBoundary } from "@/components/dashboard/cost/CostTrustBoundary";
import { ObservedCostDrivers } from "@/components/dashboard/cost/ObservedCostDrivers";
import { PricingSourceHealthList } from "@/components/dashboard/cost/PricingSourceHealthList";
import { useCostVisibility } from "@/hooks/use-cost-visibility";
import { useProviderRuntimeOverview } from "@/hooks/use-provider-runtime-overview";
import { createCostVisibilityOverview } from "@/lib/cost-visibility";
import { useI18n } from "@/lib/i18n/provider";

const CostCalculator = () => {
    const { t } = useI18n();
    const { summary, pricingSources, prices, isLoading, error } = useCostVisibility();
    const {
        overview: providerRuntimeOverview,
        isLoading: isRuntimeLoading,
        error: runtimeError,
    } = useProviderRuntimeOverview();

    const overview = createCostVisibilityOverview({
        summary,
        pricingSources,
        prices,
        runtimeOverview: providerRuntimeOverview,
    });

    const isPanelLoading = isLoading || isRuntimeLoading;
    const panelError = error || runtimeError;

    return (
        <section className="surface-elevated overflow-hidden rounded-2xl border border-border">
            <div className="border-b border-border bg-muted/25 p-5">
                <div className="flex items-center gap-2 text-primary">
                    <Calculator className="h-4 w-4" />
                    <p className="text-[10px] font-bold uppercase tracking-[0.3em]">{t("dashboard.cost.title")}</p>
                </div>
                <h2 className="mt-3 text-lg font-semibold text-foreground">{t("dashboard.cost.heading")}</h2>
                <p className="mt-2 text-sm text-muted-foreground">
                    {t("dashboard.cost.description")}
                </p>
            </div>

            {isPanelLoading ? (
                <CostCalculatorSkeleton />
            ) : (
                <div className="space-y-5 p-5">
                    <CostExecutiveSummary overview={overview} />
                    <div className="grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_minmax(320px,0.95fr)]">
                        <CostTrustBoundary error={panelError} />
                        <PricingSourceHealthList sources={overview.sourceStatuses} />
                    </div>
                    <ObservedCostDrivers models={overview.topModels} />
                </div>
            )}
        </section>
    );
};

export default CostCalculator;

function CostCalculatorSkeleton() {
    return (
        <div className="space-y-5 p-5">
            <div className="rounded-2xl border border-border bg-background/60 p-5">
                <div className="h-3 w-32 animate-pulse rounded-full bg-muted" />
                <div className="mt-4 h-6 w-3/4 animate-pulse rounded-full bg-muted/80" />
                <div className="mt-3 h-4 w-full animate-pulse rounded-full bg-muted/70" />
                <div className="mt-6 grid gap-3 sm:grid-cols-2">
                    {Array.from({ length: 4 }).map((_, index) => (
                        <div key={index} className="rounded-xl border border-border bg-card/60 p-4">
                            <div className="h-2.5 w-24 animate-pulse rounded-full bg-muted" />
                            <div className="mt-4 h-5 w-24 animate-pulse rounded-full bg-muted/80" />
                            <div className="mt-3 h-3 w-full animate-pulse rounded-full bg-muted/70" />
                        </div>
                    ))}
                </div>
            </div>

            <div className="grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_minmax(320px,0.95fr)]">
                {Array.from({ length: 2 }).map((_, index) => (
                    <div key={index} className="rounded-2xl border border-border bg-background/60 p-5">
                        <div className="h-3 w-32 animate-pulse rounded-full bg-muted" />
                        <div className="mt-4 space-y-3">
                            {Array.from({ length: 3 }).map((__, itemIndex) => (
                                <div key={itemIndex} className="rounded-xl border border-border bg-card/60 p-4">
                                    <div className="h-3 w-40 animate-pulse rounded-full bg-muted/80" />
                                    <div className="mt-3 h-3 w-full animate-pulse rounded-full bg-muted/70" />
                                </div>
                            ))}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}
