"use client";

import { History, Loader2 } from "lucide-react";

import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";
import type { ProviderReadinessReport } from "@/lib/api/providers";
import type { ProviderDefinition } from "@/components/settings/provider-catalog";
import { useI18n } from "@/lib/i18n/provider";
import {
    getProviderReadinessCheckClass,
    getProviderReadinessCheckLabel,
    getProviderReadinessCheckMessage,
    getProviderReadinessStatusClass,
    getProviderReadinessStatusLabel,
    getProviderReadinessSummary,
} from "@/lib/provider-readiness";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

function formatCheckedAt(locale: string, checkedAt: string) {
    const date = new Date(checkedAt);
    if (Number.isNaN(date.getTime())) {
        return checkedAt;
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "medium",
        timeStyle: "short",
    }).format(date);
}

export default function ProviderReadinessPanel({
    provider,
    report,
    discoveredModels,
    isLoadingModels,
    onLoadModels,
}: {
    provider: ProviderDefinition;
    report: ProviderReadinessReport | null;
    discoveredModels: string[];
    isLoadingModels: boolean;
    onLoadModels: (provider: ProviderDefinition) => void;
}) {
    const { locale, t } = useI18n();
    const visibleModels = report?.discovered_models?.length ? report.discovered_models : discoveredModels;

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-4">
            <div className="flex items-start justify-between gap-3">
                <div className="flex items-start gap-3">
                    <ProviderBrandMark providerType={provider.providerType} size="sm" />
                    <div className="space-y-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.readiness.title")}</p>
                            <Badge variant="outline" className={getProviderReadinessStatusClass(report?.status ?? "IDLE")}>
                                {getProviderReadinessStatusLabel(t, report?.status ?? "IDLE")}
                            </Badge>
                        </div>
                        <div>
                            <p className="text-sm font-semibold text-foreground">{provider.name}</p>
                            <p className="mt-1 text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{provider.tagline}</p>
                        </div>
                        <p className="text-sm leading-relaxed text-foreground/85">
                            {getProviderReadinessSummary(t, provider, report)}
                        </p>
                        {report ? (
                            <p className="text-xs text-muted-foreground">
                                {t("settings.readiness.checkedAt", { value: formatCheckedAt(locale, report.checked_at) })}
                            </p>
                        ) : null}
                    </div>
                </div>
                {provider.family === "LLM" ? (
                    <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => onLoadModels(provider)}
                        disabled={isLoadingModels}
                        className="gap-2"
                    >
                        {isLoadingModels ? <Loader2 className="h-4 w-4 animate-spin" /> : <History className="h-4 w-4" />}
                        {t("settings.readiness.loadModels")}
                    </Button>
                ) : null}
            </div>

            {report ? (
                <div className="mt-4 grid gap-3">
                    {report.checks.map((check) => (
                        <div
                            key={`${check.code}-${check.status}`}
                            className={`rounded-xl border px-3 py-3 ${getProviderReadinessCheckClass(check)}`}
                        >
                            <div className="flex items-center justify-between gap-3">
                                <p className="text-sm font-medium text-foreground">
                                    {getProviderReadinessCheckLabel(t, check)}
                                </p>
                                <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                    {check.status}
                                </span>
                            </div>
                            <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                                {getProviderReadinessCheckMessage(t, provider, report, check)}
                            </p>
                        </div>
                    ))}
                </div>
            ) : null}

            {provider.family === "LLM" ? (
                visibleModels.length > 0 ? (
                    <div className="mt-4 space-y-2">
                        <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                            {t("settings.readiness.discoveredModelsTitle")}
                        </p>
                        <div className="flex flex-wrap gap-2">
                            {visibleModels.map((model) => (
                                <span
                                    key={model}
                                    className="rounded-full border border-border bg-background/80 px-3 py-1 text-xs text-foreground/90"
                                >
                                    {model}
                                </span>
                            ))}
                        </div>
                    </div>
                ) : (
                    <p className="mt-4 text-xs text-muted-foreground">
                        {t("settings.readiness.noModels")}
                    </p>
                )
            ) : null}
        </div>
    );
}
