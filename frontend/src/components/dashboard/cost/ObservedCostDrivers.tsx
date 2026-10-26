"use client";

import { Coins } from "lucide-react";

import type { DocumentMonitoringModelAggregate } from "@/lib/api/documents";
import { formatProviderTypeLabel } from "@/lib/provider-runtime";
import { useI18n } from "@/lib/i18n/provider";
import { formatUSD } from "@/lib/format";

export function ObservedCostDrivers({
    models,
}: {
    models: DocumentMonitoringModelAggregate[];
}) {
    const { locale, t } = useI18n();

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-5">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-[0.28em] text-primary">
                <Coins className="h-3.5 w-3.5" />
                {t("dashboard.cost.models.title")}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
                {t("dashboard.cost.models.description")}
            </p>

            {models.length > 0 ? (
                <div className="mt-4 space-y-3">
                    {models.map((model) => (
                        <div key={`${model.provider_type || "unknown"}:${model.model || "unknown"}`} className="rounded-xl border border-border bg-card/60 p-4">
                            <div className="flex items-start justify-between gap-3">
                                <div className="min-w-0">
                                    <p className="truncate text-sm font-semibold text-foreground">
                                        {model.model || t("dashboard.cost.models.modelMissing")}
                                    </p>
                                    <p className="mt-1 text-xs text-muted-foreground">
                                        {formatProviderTypeLabel(model.provider_type)} · {t("dashboard.cost.models.documents", {
                                            count: model.document_count,
                                        })}
                                    </p>
                                </div>
                                <span className="shrink-0 rounded-full border border-border bg-background/80 px-2 py-1 text-[10px] font-mono text-muted-foreground">
                                    {t("dashboard.cost.models.requests", { count: model.request_count })}
                                </span>
                            </div>

                            <p className="mt-3 text-xs text-foreground/85">
                                {t("dashboard.cost.models.tokensAndCost", {
                                    tokens: model.total_tokens.toLocaleString(locale),
                                    cost: formatUSD(model.total_cost_usd, locale),
                                })}
                            </p>
                        </div>
                    ))}
                </div>
            ) : (
                <p className="mt-4 text-xs text-muted-foreground">
                    {t("dashboard.cost.models.empty")}
                </p>
            )}
        </div>
    );
}
