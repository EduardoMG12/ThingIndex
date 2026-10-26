"use client";

import { AlertTriangle, ArrowRight, Clock3, ListChecks, ShieldCheck } from "lucide-react";

import type { Document } from "@/lib/api/documents";
import type {
    DocumentOperationalActionKey,
    DocumentOperationalActionSummary,
    DocumentOperationalFilter,
    DocumentOperationalOverview,
} from "@/lib/document-observability";
import { formatDateTime } from "@/lib/format";
import { useI18n } from "@/lib/i18n/provider";

type OperationalExecutiveOverviewProps = {
    overview: DocumentOperationalOverview;
    processingCount: number;
    queuedCount: number;
    lastCompletedAt?: string;
    isSliceActive: boolean;
    onFocusFilter: (filter: DocumentOperationalFilter) => void;
    onClearSlice: () => void;
    onOpenPreview: (document: Document) => void;
    primaryDocument: Document | null;
};

export default function OperationalExecutiveOverview({
    overview,
    processingCount,
    queuedCount,
    lastCompletedAt,
    isSliceActive,
    onFocusFilter,
    onClearSlice,
    onOpenPreview,
    primaryDocument,
}: OperationalExecutiveOverviewProps) {
    const { locale, t } = useI18n();
    const primaryActionTitle = t(`documentsPage.observability.actionPlan.items.${overview.primaryAction.key}.title`);
    const headline =
        overview.primaryAction.key === "nominal"
            ? t("documentsPage.observability.briefing.nominalHeadline")
            : t("documentsPage.observability.briefing.activeHeadline", {
                count: overview.snapshot.counts.attention,
            });

    return (
        <div className="grid gap-4 xl:grid-cols-[1.08fr_0.92fr]">
            <div className="rounded-2xl border border-border bg-background/70 p-5">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div className="max-w-2xl">
                        <p className="text-[10px] uppercase tracking-[0.28em] text-primary">
                            {t("documentsPage.observability.briefing.title")}
                        </p>
                        <h3 className="mt-3 text-xl font-semibold text-foreground">{headline}</h3>
                        <p className="mt-2 text-sm text-muted-foreground">
                            {t("documentsPage.observability.briefing.description")}
                        </p>
                    </div>
                    <div className={buildHealthTone(overview.snapshot.health)}>
                        {renderHealthIcon(overview.snapshot.health)}
                        <span>{t(`documentsPage.observability.health.${overview.snapshot.health}`)}</span>
                    </div>
                </div>

                <div className="mt-4 flex flex-wrap gap-2 text-[11px] text-muted-foreground">
                    <span className="rounded-full border border-border bg-card/70 px-3 py-1">
                        {t("documentsPage.observability.snapshot.processing", { count: processingCount })}
                    </span>
                    <span className="rounded-full border border-border bg-card/70 px-3 py-1">
                        {t("documentsPage.observability.snapshot.queued", { count: queuedCount })}
                    </span>
                    <span className="rounded-full border border-border bg-card/70 px-3 py-1">
                        {t("documentsPage.observability.snapshot.retrying", { count: overview.snapshot.counts.retrying })}
                    </span>
                    <span className="rounded-full border border-border bg-card/70 px-3 py-1">
                        {lastCompletedAt
                            ? t("documentsPage.observability.snapshot.lastCompleted", {
                                value: formatDateTime(lastCompletedAt, locale),
                            })
                            : t("documentsPage.observability.snapshot.lastCompletedMissing")}
                    </span>
                </div>

                <div className="mt-5 rounded-2xl border border-border bg-card/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">
                        {t("documentsPage.observability.briefing.focusLabel")}
                    </p>
                    <p className="mt-2 text-base font-semibold text-foreground">
                        {primaryActionTitle}
                    </p>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {t(`documentsPage.observability.actionPlan.items.${overview.primaryAction.key}.description`, {
                            count: overview.primaryAction.count,
                        })}
                    </p>
                    {overview.primaryAction.sampleTitles.length > 0 ? (
                        <p className="mt-3 text-xs text-foreground/85">
                            <span className="font-semibold text-foreground">
                                {t("documentsPage.observability.actionPlan.documentsLabel")}:
                            </span>{" "}
                            {overview.primaryAction.sampleTitles.join(" · ")}
                        </p>
                    ) : null}

                    <div className="mt-4 flex flex-wrap gap-2">
                        <button
                            type="button"
                            onClick={() => onFocusFilter(overview.primaryAction.filter)}
                            className="inline-flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary"
                        >
                            <ArrowRight className="h-3.5 w-3.5" />
                            {t("documentsPage.observability.actionPlan.focusTarget", {
                                target: primaryActionTitle,
                            })}
                        </button>
                        {primaryDocument ? (
                            <button
                                type="button"
                                onClick={() => onOpenPreview(primaryDocument)}
                                className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground"
                            >
                                {t("documentsPage.observability.briefing.openPriority")}
                            </button>
                        ) : null}
                        {isSliceActive ? (
                            <button
                                type="button"
                                onClick={onClearSlice}
                                className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-muted-foreground"
                            >
                                {t("documentsPage.observability.sliceClear")}
                            </button>
                        ) : null}
                    </div>
                </div>
            </div>

            <div className="rounded-2xl border border-border bg-background/60 p-5">
                <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                    <ListChecks className="h-3.5 w-3.5 text-primary" />
                    {t("documentsPage.observability.actionPlan.title")}
                </div>
                <p className="mt-2 text-sm text-muted-foreground">
                    {t("documentsPage.observability.actionPlan.description")}
                </p>

                <div className="mt-4 space-y-3">
                    {overview.actions.map((action) => (
                        <ActionCard
                            key={action.key}
                            action={action}
                            onFocusFilter={onFocusFilter}
                        />
                    ))}
                </div>
            </div>
        </div>
    );
}

function ActionCard({
    action,
    onFocusFilter,
}: {
    action: DocumentOperationalActionSummary;
    onFocusFilter: (filter: DocumentOperationalFilter) => void;
}) {
    const { t } = useI18n();
    const actionTitle = t(`documentsPage.observability.actionPlan.items.${action.key}.title`);

    return (
        <div className={`rounded-xl border p-4 ${buildActionTone(action.key)}`}>
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <p className="text-sm font-semibold text-foreground">
                        {actionTitle}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t(`documentsPage.observability.actionPlan.items.${action.key}.description`, {
                            count: action.count,
                        })}
                    </p>
                </div>
                <span className="shrink-0 rounded-full border border-border bg-background/80 px-2 py-1 text-[10px] font-mono text-muted-foreground">
                    {action.count}
                </span>
            </div>

            {action.sampleTitles.length > 0 ? (
                <p className="mt-3 text-[11px] text-foreground/85">
                    <span className="font-semibold text-foreground">
                        {t("documentsPage.observability.actionPlan.documentsLabel")}:
                    </span>{" "}
                    {action.sampleTitles.join(" · ")}
                </p>
            ) : null}

            <button
                type="button"
                onClick={() => onFocusFilter(action.filter)}
                className="mt-4 inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground"
            >
                <ArrowRight className="h-3.5 w-3.5" />
                {t("documentsPage.observability.actionPlan.focusTarget", {
                    target: actionTitle,
                })}
            </button>
        </div>
    );
}

function buildActionTone(key: DocumentOperationalActionKey) {
    switch (key) {
        case "manual":
            return "border-destructive/20 bg-destructive/10";
        case "stalled":
            return "border-sky-500/20 bg-sky-500/10";
        case "recentFailures":
            return "border-amber-500/20 bg-amber-500/10";
        case "retrying":
            return "border-emerald-500/20 bg-emerald-500/10";
        default:
            return "border-border bg-card/70";
    }
}

function buildHealthTone(health: "nominal" | "attention" | "critical") {
    switch (health) {
        case "critical":
            return "inline-flex items-center gap-2 rounded-full border border-destructive/30 bg-destructive/10 px-4 py-2 text-sm font-semibold text-destructive";
        case "attention":
            return "inline-flex items-center gap-2 rounded-full border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm font-semibold text-amber-700 dark:text-amber-300";
        default:
            return "inline-flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-4 py-2 text-sm font-semibold text-emerald-700 dark:text-emerald-300";
    }
}

function renderHealthIcon(health: "nominal" | "attention" | "critical") {
    switch (health) {
        case "critical":
            return <AlertTriangle className="h-4 w-4" />;
        case "attention":
            return <Clock3 className="h-4 w-4" />;
        default:
            return <ShieldCheck className="h-4 w-4" />;
    }
}
