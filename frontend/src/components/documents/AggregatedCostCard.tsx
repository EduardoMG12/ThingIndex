"use client";

import { Coins } from "lucide-react";

import { describeAggregatedCost } from "@/lib/aggregated-cost";
import type { DocumentMonitoringSummary } from "@/lib/api/documents";
import { useI18n } from "@/lib/i18n/provider";

type Props = {
    summary: Pick<
        DocumentMonitoringSummary,
        "documents_with_llm" | "pricing_aware_documents" | "total_estimated_cost_usd" | "total_tokens" | "total_llm_requests"
    >;
};

function formatUSD(value: number, locale: string) {
    return new Intl.NumberFormat(locale, {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: 6,
        maximumFractionDigits: 6,
    }).format(value > 0 ? value : 0);
}

// Cartao "Custo agregado" de /documents. Consumo sem preco conhecido aparece
// como "Nao informado", nunca como US$ 0 (ver describeAggregatedCost). Os
// tokens do resumo sao sempre estimados pelo backend (contagem por texto no
// observer da indexacao), por isso o rotulo "estimados" e fixo.
export function AggregatedCostCard({ summary }: Props) {
    const { locale, t } = useI18n();
    const custo = describeAggregatedCost(summary);

    return (
        <div className="rounded-xl border border-border bg-background/60 p-4" data-testid="custo-agregado">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                <Coins className="h-3.5 w-3.5 text-primary" />
                {t("documentsPage.summary.aggregatedCost")}
            </div>
            {custo.kind === "unknown" ? (
                <p className="mt-3 text-lg font-semibold" title={t("documentsPage.summary.costNotReportedHint")}>
                    {t("documentsPage.summary.costNotReported")}
                </p>
            ) : (
                <p className="mt-3 text-lg font-semibold">{formatUSD(custo.costUsd, locale)}</p>
            )}
            {custo.kind === "partial" ? (
                <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                    {t("documentsPage.summary.costPartial", {
                        priced: custo.pricedDocuments,
                        total: custo.documentsWithLlm,
                    })}
                </p>
            ) : null}
            <p className="mt-1 text-xs text-muted-foreground">
                {t("documentsPage.summary.aggregatedCostDetail", {
                    tokens: summary.total_tokens.toLocaleString(locale),
                    requests: summary.total_llm_requests,
                })}
            </p>
        </div>
    );
}
