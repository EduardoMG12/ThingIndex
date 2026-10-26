"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
    Activity,
    AlertTriangle,
    BrainCircuit,
    Clock3,
    Coins,
    Cpu,
    Database,
    Download,
    ExternalLink,
    FileText,
    FolderOpen,
    Gauge,
    Hash,
    Loader2,
    PanelRight,
    RotateCw,
    Table2,
    Tag,
    Trash2,
    X,
} from "lucide-react";
import { toast } from "sonner";
import { formatDocumentStatus } from "@/lib/documents/status";
import { formatNonNegativeUSD } from "@/lib/format";

import AppShell from "@/components/layout/AppShell";
import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";
import DocumentAuditExplorer from "@/components/audit/DocumentAuditExplorer";
import OperationalObservabilityPanel from "@/components/documents/OperationalObservabilityPanel";
import { AggregatedCostCard } from "@/components/documents/AggregatedCostCard";
import { DangerConfirmModal } from "@/components/ui/danger-confirm-modal";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDocuments } from "@/hooks/use-documents";
import { useDocumentPreviewData } from "@/hooks/use-document-preview-data";
import { useProviderRuntimeOverview } from "@/hooks/use-provider-runtime-overview";
import type { CollectionContext } from "@/lib/api/contracts/document-context";
import {
    downloadDocumentIndexExport,
    downloadDocumentOriginal,
    deleteCollectionDocuments,
    deleteDocument,
    fetchDocumentMonitoringSummary,
    retryDocumentIndexing,
    type Document,
    type DocumentIndex,
    type DocumentMonitoringSummary,
    type DocumentProcessingAttempt,
    type IndexMonitoring,
} from "@/lib/api/documents";
import { type DocumentOperationalFilter } from "@/lib/document-observability";
import {
    getDocumentOperationalIssue,
    isOrphanedDocument,
    isPreviewableDocument,
    type DocumentOperationalIssue,
} from "@/lib/document-state";
import { useI18n } from "@/lib/i18n/provider";
import { translate } from "@/lib/i18n/translate";
import { cn } from "@/lib/utils";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };
type TranslateFn = (key: string, variables?: Record<string, string | number | boolean>) => string;

function formatSize(bytes: number) {
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
        return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(value: string | undefined, locale: string) {
    if (!value) {
        return "-";
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "short",
        timeStyle: "short",
    }).format(new Date(value));
}

function formatDuration(durationMs: number) {
    if (!durationMs || durationMs <= 0) {
        return "-";
    }
    if (durationMs < 1000) {
        return `${durationMs} ms`;
    }
    if (durationMs < 60_000) {
        return `${(durationMs / 1000).toFixed(1)} s`;
    }
    return `${(durationMs / 60_000).toFixed(1)} min`;
}

function formatMonitoringTreeStateLabel(treeState: string | undefined, t: TranslateFn) {
    switch (treeState) {
        case "healthy":
            return t("documentsPage.monitoring.treeStateHealthy");
        case "fallback":
            return t("documentsPage.monitoring.treeStateFallback");
        case "degraded":
            return t("documentsPage.monitoring.treeStateDegraded");
        default:
            return t("documentsPage.monitoring.treeStateUnknown");
    }
}

function monitoringTreeStateBadgeClassName(treeState: string | undefined) {
    switch (treeState) {
        case "healthy":
            return "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
        case "fallback":
            return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300";
        case "degraded":
            return "border-destructive/30 bg-destructive/10 text-destructive";
        default:
            return "border-border bg-muted/40 text-muted-foreground";
    }
}

function formatMonitoringTreeConcernLabel(concern: string, t: TranslateFn) {
    switch (concern) {
        case "fallback_builder":
            return t("documentsPage.monitoring.concernFallbackBuilder");
        case "verification_failed":
            return t("documentsPage.monitoring.concernVerificationFailed");
        case "low_coverage":
            return t("documentsPage.monitoring.concernLowCoverage");
        case "low_summary_coverage":
            return t("documentsPage.monitoring.concernLowSummaryCoverage");
        case "low_confidence_nodes":
            return t("documentsPage.monitoring.concernLowConfidenceNodes");
        case "shallow_tree":
            return t("documentsPage.monitoring.concernShallowTree");
        case "oversized_nodes":
            return t("documentsPage.monitoring.concernOversizedNodes");
        case "retrieval_not_ready":
            return t("documentsPage.monitoring.concernRetrievalNotReady");
        default:
            return concern;
    }
}

function formatStageLabel(stage: string | undefined, t: TranslateFn) {
    if (!stage) {
        return t("documentsPage.operational.noStage");
    }

    switch (stage) {
        case "QUEUED":
            return t("dashboard.upload.status.queued");
        case "PROCESSING":
            return t("dashboard.upload.status.processing");
        case "RETRY_SCHEDULED":
            return t("documentsPage.operational.stage.retryScheduled");
        case "MOVED_TO_DLQ":
            return t("documentsPage.operational.stage.movedToDlq");
        case "FAILED_PERMANENT":
            return t("documentsPage.operational.stage.failedPermanent");
        case "FAILED_PARSING_STRUCTURE":
            return t("documentsPage.operational.stage.failedParsingStructure");
        case "FAILED_RESOLVING_STORAGE":
            return t("documentsPage.operational.stage.failedResolvingStorage");
        case "FAILED_PERSISTING_INDEX":
            return t("documentsPage.operational.stage.failedPersistingIndex");
        case "FAILED_READING_SOURCE":
            return t("documentsPage.operational.stage.failedReadingSource");
        case "FAILED_LOADING_DOCUMENT":
            return t("documentsPage.operational.stage.failedLoadingDocument");
        case "POST_PROCESSING":
            return t("documentsPage.operational.stage.postProcessing");
        default:
            break;
    }

    return stage
        .toLowerCase()
        .split("_")
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(" ");
}

function formatFailureKind(kind: string | undefined, t: TranslateFn) {
    switch (kind) {
        case "TRANSIENT":
            return t("documentsPage.operational.failureKind.transient");
        case "PERMANENT":
            return t("documentsPage.operational.failureKind.permanent");
        case "SOURCE_MISSING":
            return t("documentsPage.operational.failureKind.sourceMissing");
        default:
            return t("documentsPage.operational.failureKind.fallback");
    }
}

function formatDispositionLabel(disposition: string | undefined, t: TranslateFn) {
    switch (disposition) {
        case "RETRY_SCHEDULED":
            return t("documentsPage.operational.disposition.retryScheduled");
        case "MOVED_TO_DLQ":
            return t("documentsPage.operational.disposition.movedToDlq");
        case "FAILED_PERMANENT":
            return t("documentsPage.operational.disposition.failedPermanent");
        default:
            return t("documentsPage.operational.disposition.none");
    }
}

function formatRetryWindow(attempt: DocumentProcessingAttempt | undefined, t: TranslateFn) {
    if (!attempt) {
        return null;
    }

    if (attempt.next_retry_attempt && attempt.max_retry_attempts) {
        return t("documentsPage.operational.retryWindow.next", {
            current: attempt.next_retry_attempt,
            total: attempt.max_retry_attempts,
        });
    }

    if (attempt.retry_count && attempt.max_retry_attempts) {
        return t("documentsPage.operational.retryWindow.attempts", {
            current: attempt.retry_count,
            total: attempt.max_retry_attempts,
        });
    }

    return null;
}

function MonitoringView({
    document,
    monitoring,
}: {
    document: Document;
    monitoring: IndexMonitoring | null;
}) {
    const { locale, t } = useI18n();
    const status = monitoring?.status || document.status;
    const stage = monitoring?.stage || document.processing_stage;
    const detail = monitoring?.status_detail || document.status_detail;
    const startedAt = monitoring?.started_at || document.index_started_at;
    const completedAt = monitoring?.completed_at || document.indexed_at;
    const llm = monitoring?.llm ?? null;
    const qualityScore = monitoring?.estimated_quality_score ?? 0;
    const treeState = monitoring?.tree_state;
    const treeStateDetail = monitoring?.tree_state_detail || t("documentsPage.monitoring.treeStateUnavailable");
    const treeConcernCodes = monitoring?.tree_concern_codes ?? [];

    return (
        <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Activity className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.currentStatus")}
                    </div>
                    <p className="mt-3 text-lg font-semibold">{formatDocumentStatus(status, t)}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{formatStageLabel(stage, t)}</p>
                </div>
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Gauge className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.estimatedQuality")}
                    </div>
                    <p className="mt-3 text-lg font-semibold">{qualityScore.toFixed(1)} / 100</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t("documentsPage.monitoring.estimatedQualityHint")}
                    </p>
                </div>
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <AlertTriangle className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.treeState")}
                    </div>
                    <div className="mt-3 flex flex-wrap items-center gap-2">
                        <span
                            className={cn(
                                "inline-flex rounded-full border px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide",
                                monitoringTreeStateBadgeClassName(treeState)
                            )}
                        >
                            {formatMonitoringTreeStateLabel(treeState, t)}
                        </span>
                    </div>
                    <p className="mt-2 text-xs leading-5 text-muted-foreground">{treeStateDetail}</p>
                    {treeConcernCodes.length > 0 ? (
                        <div className="mt-3 flex flex-wrap gap-2">
                            {treeConcernCodes.map((concern) => (
                                <span
                                    key={concern}
                                    className="inline-flex rounded-full border border-border bg-muted/50 px-2.5 py-1 text-[11px] font-medium text-foreground/80"
                                >
                                    {formatMonitoringTreeConcernLabel(concern, t)}
                                </span>
                            ))}
                        </div>
                    ) : null}
                </div>
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Cpu className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.activeModel")}
                    </div>
                    <p className="mt-3 text-sm font-semibold">{llm?.model || t("documentsPage.monitoring.noLlm")}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{llm?.provider_type || monitoring?.enrichment_status || "-"}</p>
                </div>
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Coins className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.estimatedCost")}
                    </div>
                    <p className="mt-3 text-lg font-semibold">
                        {llm?.pricing_available ? formatNonNegativeUSD(llm.total_cost_usd, locale) : t("documentsPage.monitoring.unavailable")}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {llm?.pricing_available
                            ? t("documentsPage.monitoring.estimatedCostHint")
                            : t("documentsPage.monitoring.noKnownPricing")}
                    </p>
                </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Clock3 className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.process")}
                    </div>
                    <div className="mt-3 space-y-2 text-sm">
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.stage")}:</span> <span className="font-medium">{formatStageLabel(stage, t)}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.detail")}:</span> <span className="font-medium">{detail || "-"}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.startedAt")}:</span> <span className="font-mono">{formatDate(startedAt, locale)}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.finishedAt")}:</span> <span className="font-mono">{formatDate(completedAt, locale)}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.duration")}:</span> <span className="font-mono">{formatDuration(monitoring?.duration_ms ?? 0)}</span></p>
                        {document.last_processing_error ? (
                            <p className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                                {document.last_processing_error}
                            </p>
                        ) : null}
                    </div>
                </div>

                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <BrainCircuit className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.monitoring.llmTelemetry")}
                    </div>
                    <div className="mt-3 space-y-2 text-sm">
                        <p><span className="text-muted-foreground">Requests:</span> <span className="font-mono">{llm?.request_count ?? 0}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.inputTokens")}:</span> <span className="font-mono">{llm?.estimated_input_tokens ?? 0}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.outputTokens")}:</span> <span className="font-mono">{llm?.estimated_output_tokens ?? 0}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.totalTokens")}:</span> <span className="font-mono">{llm?.estimated_total_tokens ?? 0}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.inputCost")}:</span> <span className="font-mono">{llm?.pricing_available ? formatNonNegativeUSD(llm.input_cost_usd, locale) : t("documentsPage.summary.costNotReported")}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.outputCost")}:</span> <span className="font-mono">{llm?.pricing_available ? formatNonNegativeUSD(llm.output_cost_usd, locale) : t("documentsPage.summary.costNotReported")}</span></p>
                    </div>
                </div>
            </div>

            <div className="rounded-xl border border-border bg-background/60 p-4">
                <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.monitoring.versionsAndValidation")}</p>
                <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-4 text-sm">
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.pipeline")}:</span> <span className="font-mono">{monitoring?.pipeline_version || "-"}</span></p>
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.promptProfile")}:</span> <span className="font-mono">{monitoring?.prompt_profile_version || "-"}</span></p>
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.chunkStrategy")}:</span> <span className="font-mono">{monitoring?.chunk_strategy_version || "-"}</span></p>
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.treeBuilder")}:</span> <span className="font-mono">{monitoring?.tree_builder || "-"}</span></p>
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.enrichment")}:</span> <span className="font-mono">{monitoring?.enrichment_status || "-"}</span></p>
                    <p><span className="text-muted-foreground">{t("documentsPage.monitoring.treeVerified")}:</span> <span className="font-mono">{monitoring?.tree_verification_passed ? "true" : "false"}</span></p>
                </div>
            </div>
        </div>
    );
}

function MonitoringSummaryPanel({
    summary,
}: {
    summary: DocumentMonitoringSummary;
}) {
    const { locale, t } = useI18n();
    const primaryModel = summary.model_breakdown[0];

    return (
        <div className="mb-6 space-y-4">
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Activity className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.summary.operationalCoverage")}
                    </div>
                    <p className="mt-3 text-lg font-semibold">{summary.indexed_documents} / {summary.total_documents}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t("documentsPage.summary.operationalCoverageDetail", {
                            processing: summary.processing_documents,
                            queued: summary.queued_documents,
                            failed: summary.failed_documents,
                        })}
                    </p>
                </div>
                <AggregatedCostCard summary={summary} />
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <Gauge className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.summary.averageQuality")}
                    </div>
                    <p className="mt-3 text-lg font-semibold">{summary.average_quality_score.toFixed(1)} / 100</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t("documentsPage.summary.averageQualityDetail", {
                            duration: formatDuration(summary.average_duration_ms),
                        })}
                    </p>
                </div>
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                        <BrainCircuit className="h-3.5 w-3.5 text-primary" />
                        {t("documentsPage.summary.primaryModel")}
                    </div>
                    <p className="mt-3 text-sm font-semibold">{primaryModel?.model || t("documentsPage.summary.noConsolidatedLlm")}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                        {t("documentsPage.summary.primaryModelDetail", {
                            provider: primaryModel?.provider_type || "-",
                            count: summary.documents_with_llm,
                        })}
                    </p>
                </div>
            </div>

            <div className="grid gap-3 md:grid-cols-4">
                <div className="rounded-xl border border-sky-500/20 bg-sky-500/10 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-sky-700 dark:text-sky-300">{t("documentsPage.summary.retrying")}</p>
                    <p className="mt-2 text-lg font-semibold text-foreground">{summary.retrying_documents}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{t("documentsPage.summary.retryingHint")}</p>
                </div>
                <div className="rounded-xl border border-destructive/20 bg-destructive/10 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-destructive">DLQ</p>
                    <p className="mt-2 text-lg font-semibold text-foreground">{summary.dlq_documents}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{t("documentsPage.summary.dlqHint")}</p>
                </div>
                <div className="rounded-xl border border-amber-500/20 bg-amber-500/10 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-amber-700 dark:text-amber-300">{t("documentsPage.summary.permanentFailure")}</p>
                    <p className="mt-2 text-lg font-semibold text-foreground">{summary.permanent_failed_documents}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{t("documentsPage.summary.permanentFailureHint")}</p>
                </div>
                <div className="rounded-xl border border-destructive/20 bg-destructive/10 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-destructive">{t("documentsPage.summary.orphaned")}</p>
                    <p className="mt-2 text-lg font-semibold text-foreground">{summary.orphaned_documents}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{t("documentsPage.summary.orphanedHint")}</p>
                </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.summary.consolidatedTelemetry")}</p>
                    <div className="mt-3 grid gap-2 md:grid-cols-2 text-sm">
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.inputTokens")}:</span> <span className="font-mono">{summary.total_input_tokens.toLocaleString(locale)}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.monitoring.outputTokens")}:</span> <span className="font-mono">{summary.total_output_tokens.toLocaleString(locale)}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.summary.pricingAwareDocuments")}:</span> <span className="font-mono">{summary.pricing_aware_documents}</span></p>
                        <p><span className="text-muted-foreground">{t("documentsPage.summary.lastCompletedAt")}:</span> <span className="font-mono">{formatDate(summary.last_completed_at, locale)}</span></p>
                    </div>
                </div>

                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.summary.modelsInUse")}</p>
                    {summary.model_breakdown.length > 0 ? (
                        <div className="mt-3 space-y-2">
                            {summary.model_breakdown.slice(0, 3).map((item) => (
                                <div key={`${item.provider_type || "unknown"}:${item.model || "unknown"}`} className="rounded-lg border border-border bg-card/70 px-3 py-2">
                                    <p className="text-xs font-semibold">{item.model || t("documentsPage.summary.modelNotProvided")}</p>
                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                        {t("documentsPage.summary.modelBreakdownDetail", {
                                            provider: item.provider_type || "-",
                                            documents: item.document_count,
                                            requests: item.request_count,
                                        })}
                                    </p>
                                    <p className="mt-1 font-mono text-[11px] text-foreground/85">
                                        {t("documentsPage.summary.modelBreakdownCost", {
                                            tokens: item.total_tokens.toLocaleString(locale),
                                            cost: formatNonNegativeUSD(item.total_cost_usd, locale),
                                        })}
                                    </p>
                                </div>
                            ))}
                        </div>
                    ) : (
                        <p className="mt-3 text-xs text-muted-foreground">
                            {t("documentsPage.summary.noModelUsage")}
                        </p>
                    )}
                </div>
            </div>
        </div>
    );
}

function buildPreviewURL(document: Document) {
    return `/api/documents/${document.id}/preview`;
}

function buildIssueTone(issue: DocumentOperationalIssue) {
    if (issue.kind === "orphaned" || issue.kind === "dlq") {
        return {
            container: "border-destructive/30 bg-destructive/10 text-destructive",
            detail: "border-destructive/20 bg-background/70 text-foreground/90",
        };
    }

    if (issue.kind === "retrying") {
        return {
            container: "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300",
            detail: "border-sky-500/20 bg-background/70 text-foreground/90",
        };
    }

    return {
        container: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300",
        detail: "border-amber-500/20 bg-background/70 text-foreground/90",
    };
}

function OperationalIssueBanner({
    issue,
    compact = false,
}: {
    issue: DocumentOperationalIssue;
    compact?: boolean;
}) {
    const { t } = useI18n();
    const tone = buildIssueTone(issue);
    const retryWindow = formatRetryWindow(issue.attempt, t);

    return (
        <div className={`rounded-xl border p-4 ${tone.container}`}>
            <div className="flex items-start gap-3">
                <AlertTriangle className={`mt-0.5 shrink-0 ${compact ? "h-4 w-4" : "h-5 w-5"}`} />
                <div className="min-w-0">
                    <p className={`font-semibold ${compact ? "text-sm" : "text-base"}`}>{issue.title}</p>
                    <p className={`mt-1 leading-relaxed ${compact ? "text-xs" : "text-sm"}`}>{issue.description}</p>
                    {issue.attempt ? (
                        <div className="mt-3 flex flex-wrap gap-2 text-[11px]">
                            <span className={`rounded-full border px-2 py-1 ${tone.detail}`}>
                                {formatFailureKind(issue.attempt.failure_kind, t)}
                            </span>
                            <span className={`rounded-full border px-2 py-1 ${tone.detail}`}>
                                {formatDispositionLabel(issue.attempt.disposition, t)}
                            </span>
                            {retryWindow ? (
                                <span className={`rounded-full border px-2 py-1 ${tone.detail}`}>
                                    {retryWindow}
                                </span>
                            ) : null}
                            {issue.attempt.stage ? (
                                <span className={`rounded-full border px-2 py-1 ${tone.detail}`}>
                                    {formatStageLabel(issue.attempt.stage, t)}
                                </span>
                            ) : null}
                        </div>
                    ) : null}
                    {issue.detail ? (
                        <div className={`mt-3 rounded-lg border px-3 py-2 text-xs ${tone.detail}`}>
                            {issue.detail}
                        </div>
                    ) : null}
                    <p className={`mt-3 ${compact ? "text-[11px]" : "text-xs"} text-current/85`}>
                        {issue.hint}
                    </p>
                </div>
            </div>
        </div>
    );
}

function RecentAttemptsPanel({
    attempts,
}: {
    attempts: DocumentProcessingAttempt[] | undefined;
}) {
    const { locale, t } = useI18n();
    if (!attempts || attempts.length === 0) {
        return null;
    }

    return (
        <div className="rounded-xl border border-border bg-background/60 p-4">
            <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.recentAttempts.title")}</p>
            <div className="mt-3 space-y-3">
                {attempts.slice(0, 4).map((attempt) => (
                    <div key={attempt.id} className="rounded-lg border border-border bg-card/70 p-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <span className="text-xs font-semibold">{formatFailureKind(attempt.failure_kind, t)}</span>
                            <span className="rounded-full border border-border px-2 py-1 text-[10px] font-mono text-muted-foreground">
                                {formatDispositionLabel(attempt.disposition, t)}
                            </span>
                            {attempt.stage ? (
                                <span className="rounded-full border border-border px-2 py-1 text-[10px] font-mono text-muted-foreground">
                                    {formatStageLabel(attempt.stage, t)}
                                </span>
                            ) : null}
                        </div>
                        <p className="mt-2 text-xs text-foreground/90">{attempt.error_message || attempt.detail || "-"}</p>
                        <p className="mt-2 text-[11px] text-muted-foreground">
                            {formatRetryWindow(attempt, t) || t("documentsPage.recentAttempts.noRetryScheduled")} · {formatDate(attempt.created_at, locale)}
                        </p>
                    </div>
                ))}
            </div>
        </div>
    );
}

function FilePreview({ document }: { document: Document }) {
    const { t } = useI18n();
    if (!isPreviewableDocument(document)) {
        return (
            <div className="rounded-xl border border-border bg-background/60 p-5 text-sm text-muted-foreground">
                {t("documentsPage.preview.unsupportedInline")}
            </div>
        );
    }

    return (
        <iframe
            title={t("documentsPage.preview.inlineTitle", { title: document.title })}
            src={buildPreviewURL(document)}
            className="h-[70vh] w-full rounded-xl border border-border bg-white"
            sandbox="allow-same-origin allow-scripts allow-downloads"
        />
    );
}

function IndexStructureView({
    document,
    context,
    documentIndex,
}: {
    document: Document;
    context: CollectionContext | null;
    documentIndex: DocumentIndex | null;
}) {
    const { t } = useI18n();
    const paragraphNodes =
        documentIndex?.paragraph_nodes ??
        context?.paragraphNodes.filter((node) => node.documentId === document.id) ??
        [];
    const tableSections =
        documentIndex?.table_sections ??
        context?.tableSections.filter((section) => section.documentId === document.id) ??
        [];
    const metadataPairs =
        documentIndex?.metadata_pairs ??
        context?.metadataPairs.filter((pair) => pair.documentId === document.id) ??
        [];
    const outlineNodes =
        documentIndex?.outline_nodes ??
        context?.outlineNodes.filter((node) => node.documentId === document.id) ??
        [];
    const chunks = documentIndex?.chunks ?? [];
    const documentSummary =
        documentIndex?.document_summary ||
        metadataPairs.find((pair) => pair.key === "document_summary")?.value ||
        document.document_summary ||
        "";
    const monitoring = documentIndex?.monitoring ?? null;

    return (
        <div className="space-y-4">
            <MonitoringView document={document} monitoring={monitoring} />

            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.index.blocks")}</p>
                    <p className="mt-2 text-lg font-semibold">{paragraphNodes.length}</p>
                </div>
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Chunks</p>
                    <p className="mt-2 text-lg font-semibold">{chunks.length}</p>
                </div>
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.index.tables")}</p>
                    <p className="mt-2 text-lg font-semibold">{tableSections.length}</p>
                </div>
                <div className="rounded-lg border border-border bg-background/60 p-3">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.index.metadata")}</p>
                    <p className="mt-2 text-lg font-semibold">{metadataPairs.length}</p>
                </div>
            </div>

            <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{t("documentsPage.index.summaryTitle")}</p>
                    <p className="mt-3 text-sm leading-relaxed text-foreground/90">
                        {documentSummary || t("documentsPage.index.summaryEmpty")}
                    </p>
                </div>

                <div className="rounded-xl border border-border bg-background/60 p-4">
                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">Outline</p>
                    {outlineNodes.length > 0 ? (
                        <div className="mt-3 space-y-2">
                            {outlineNodes.map((node) => (
                                <div key={String(node.id)} className="rounded-lg border border-border bg-card/70 px-3 py-2">
                                    <p className="text-[10px] font-mono uppercase tracking-wider text-muted-foreground">
                                        {t("documentsPage.index.outlineItem", {
                                            level: node.level,
                                            section: node.sequenceOrder,
                                        })}
                                    </p>
                                    <p className="mt-1 text-sm font-medium">{node.title}</p>
                                    {node.summary ? (
                                        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{node.summary}</p>
                                    ) : null}
                                </div>
                            ))}
                        </div>
                    ) : (
                        <p className="mt-3 text-xs text-muted-foreground">
                            {t("documentsPage.index.outlineEmpty")}
                        </p>
                    )}
                </div>
            </div>

            {chunks.length > 0 && (
                <div className="space-y-2">
                    <h3 className="text-sm font-semibold">{t("documentsPage.index.generatedChunks")}</h3>
                    <div className="space-y-2">
                        {chunks.map((chunk) => (
                            <div key={String(chunk.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground">
                                    <span className="rounded bg-muted px-2 py-1 font-mono">{chunk.chunk_type}</span>
                                    <span className="rounded bg-muted px-2 py-1 font-mono">seq:{chunk.sequence_order}</span>
                                    <span className="rounded bg-muted px-2 py-1 font-mono">tokens:{chunk.token_estimate}</span>
                                    {typeof chunk.source_paragraph_start === "number" &&
                                    typeof chunk.source_paragraph_end === "number" ? (
                                        <span className="rounded bg-muted px-2 py-1 font-mono">
                                            p:{chunk.source_paragraph_start}-{chunk.source_paragraph_end}
                                        </span>
                                    ) : null}
                                </div>
                                {chunk.title ? <p className="mt-2 text-xs font-semibold">{chunk.title}</p> : null}
                                <p className="mt-2 text-xs leading-relaxed text-foreground/90">{chunk.content}</p>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            <div className="space-y-2">
                <h3 className="text-sm font-semibold">{t("documentsPage.index.previewTitle")}</h3>
                {paragraphNodes.length > 0 ? (
                    <div className="space-y-2">
                        {paragraphNodes.map((node) => (
                            <div key={String(node.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                <div className="flex flex-wrap gap-2 text-[10px] text-muted-foreground">
                                    <span className="rounded bg-muted px-2 py-1 font-mono">chunk:{node.sequenceOrder}</span>
                                    {node.pageNumber ? <span className="rounded bg-muted px-2 py-1 font-mono">page:{node.pageNumber}</span> : null}
                                    <span className="rounded bg-muted px-2 py-1 font-mono">chars:{node.content.length}</span>
                                </div>
                                <p className="mt-2 text-xs leading-relaxed text-foreground/90">{node.content}</p>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-sm text-muted-foreground">
                        {t("documentsPage.index.previewEmpty")}
                    </p>
                )}
            </div>

            {(tableSections.length > 0 || metadataPairs.length > 0) && (
                <div className="grid gap-4 lg:grid-cols-2">
                    <div className="space-y-2">
                        <h4 className="flex items-center gap-2 text-sm font-semibold">
                            <Table2 className="h-4 w-4 text-primary" />
                            {t("documentsPage.index.structuredTables")}
                        </h4>
                        {tableSections.length > 0 ? (
                            tableSections.map((section) => (
                                <div key={String(section.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                    <p className="text-xs font-semibold">{section.title || t("documentsPage.index.tableFallback", { order: section.sequenceOrder })}</p>
                                    <p className="mt-1 text-[11px] text-muted-foreground">
                                        {t("documentsPage.index.tableDetail", {
                                            columns: section.headers.length,
                                            rows: section.rows.length,
                                        })}
                                    </p>
                                </div>
                            ))
                        ) : (
                            <p className="text-xs text-muted-foreground">{t("documentsPage.index.noTables")}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <h4 className="text-sm font-semibold">{t("documentsPage.index.extractedMetadata")}</h4>
                        {metadataPairs.length > 0 ? (
                            metadataPairs.map((pair) => (
                                <div key={String(pair.id)} className="rounded-lg border border-border bg-background/60 p-3">
                                    <p className="text-[10px] uppercase tracking-widest text-muted-foreground">{pair.key}</p>
                                    <p className="mt-1 text-xs text-foreground/90">{pair.value}</p>
                                </div>
                            ))
                        ) : (
                            <p className="text-xs text-muted-foreground">{t("documentsPage.index.noMetadata")}</p>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

function DocumentPreviewPanel({
    document,
    uploadHref,
    onRetry,
    onDownload,
    onDownloadIndexExport,
    onDelete,
    isRetrying,
    isDownloading,
    isExportingIndex,
    isDeleting,
    onClose,
}: {
    document: Document | null;
    uploadHref: string;
    onRetry: (document: Document) => void;
    onDownload: (document: Document) => void;
    onDownloadIndexExport: (document: Document) => void;
    onDelete: (document: Document) => void;
    isRetrying: boolean;
    isDownloading: boolean;
    isExportingIndex: boolean;
    isDeleting: boolean;
    onClose: () => void;
}) {
    const { locale, t } = useI18n();
    const { context, documentIndex, isLoading, error } = useDocumentPreviewData(document, locale);

    if (!document) {
        return null;
    }

    const issue = getDocumentOperationalIssue(document, locale);
    const canRenderPreview = isPreviewableDocument(document) && issue?.kind !== "orphaned";

    return (
        <div className="fixed inset-0 z-50 flex">
            <button
                type="button"
                onClick={onClose}
                className="flex-1 bg-black/55 backdrop-blur-sm"
                aria-label={t("documentsPage.preview.closePanel")}
            />
            <aside className="relative h-full w-full max-w-[980px] shrink-0 border-l border-border bg-card shadow-2xl">
                <div className="flex h-full flex-col">
                    <div className="flex items-start justify-between gap-4 border-b border-border px-6 py-5">
                        <div className="min-w-0">
                            <div className="mb-2 flex items-center gap-2">
                                <PanelRight className="h-4 w-4 text-primary" />
                                <span className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">{t("documentsPage.preview.panelEyebrow")}</span>
                            </div>
                            <h2 className="truncate text-lg font-semibold">{document.title}</h2>
                            <p className="mt-2 break-all font-mono text-[11px] text-muted-foreground">
                                {document.file_path || document.file_url}
                            </p>
                        </div>
                        <button
                            type="button"
                            onClick={onClose}
                            className="rounded-lg border border-border p-2 text-muted-foreground transition-colors hover:text-foreground"
                        >
                            <X className="h-4 w-4" />
                        </button>
                    </div>

                    <div className="border-b border-border px-6 py-3">
                        <div className="flex flex-wrap gap-2">
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <Hash className="h-3 w-3" />
                                {formatSize(document.size)}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <FileText className="h-3 w-3" />
                                {document.mime_type}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <FolderOpen className="h-3 w-3" />
                                {document.collection || t("documentsPage.collectionFallback")}
                            </span>
                            <span className="flex items-center gap-1 rounded border border-border bg-muted px-2 py-1 font-mono text-[10px] text-muted-foreground">
                                <Tag className="h-3 w-3" />
                                {document.category?.name || t("documentsPage.categoryFallback")}
                            </span>
                        </div>
                    </div>

                    <div className="flex-1 overflow-y-auto p-6">
                        <Tabs defaultValue="document" className="w-full">
                            <TabsList className="mb-6 h-auto justify-start gap-3 bg-transparent p-0">
                                <TabsTrigger
                                    value="document"
                                    className="border border-transparent px-4 py-2 text-xs font-bold uppercase tracking-widest data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary"
                                >
                                    {t("documentsPage.preview.documentTab")}
                                </TabsTrigger>
                                <TabsTrigger
                                    value="index"
                                    className="border border-transparent px-4 py-2 text-xs font-bold uppercase tracking-widest data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary"
                                >
                                    {t("documentsPage.preview.indexTab")}
                                </TabsTrigger>
                            </TabsList>

                            <TabsContent value="document" className="mt-0 space-y-4">
                                <div className="flex items-center justify-between">
                                    <div>
                                        <h3 className="text-sm font-semibold">{t("documentsPage.preview.originalDocument")}</h3>
                                        <p className="mt-1 text-xs text-muted-foreground">
                                            {t("documentsPage.preview.originalDocumentHint")}
                                        </p>
                                    </div>
                                    {canRenderPreview ? (
                                        <a
                                            href={buildPreviewURL(document)}
                                            target="_blank"
                                            rel="noreferrer"
                                            className="inline-flex items-center gap-1 text-[11px] font-medium text-primary"
                                        >
                                            {t("documentsPage.preview.openInNewTab")}
                                            <ExternalLink className="h-3.5 w-3.5" />
                                        </a>
                                    ) : null}
                                </div>
                                {issue ? <OperationalIssueBanner issue={issue} /> : null}
                                <RecentAttemptsPanel attempts={document.processing_attempts} />

                                <div className="flex flex-wrap gap-2">
                                    <button
                                        type="button"
                                        onClick={() => onDownload(document)}
                                        disabled={isDownloading}
                                        className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground transition hover:bg-muted disabled:cursor-not-allowed disabled:opacity-60"
                                    >
                                        {isDownloading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                                        {t("documentsPage.actions.downloadOriginal")}
                                    </button>

                                    {(document.status === "FAILED" || isOrphanedDocument(document)) ? (
                                        <button
                                            type="button"
                                            onClick={() => onRetry(document)}
                                            disabled={isRetrying}
                                            className="inline-flex items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs font-semibold text-amber-700 transition hover:bg-amber-500/15 disabled:cursor-not-allowed disabled:opacity-60 dark:text-amber-300"
                                        >
                                            {isRetrying ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCw className="h-4 w-4" />}
                                            {isOrphanedDocument(document)
                                                ? t("documentsPage.actions.retryAfterRestore")
                                                : t("documentsPage.actions.retryIndexing")}
                                        </button>
                                    ) : null}

                                    {isOrphanedDocument(document) ? (
                                        <Link
                                            href={uploadHref}
                                            className="inline-flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/10 px-3 py-2 text-xs font-semibold text-primary transition hover:bg-primary/15"
                                        >
                                            {t("documentsPage.actions.openDashboardForReupload")}
                                            <ExternalLink className="h-4 w-4" />
                                        </Link>
                                    ) : null}

                                    {isOrphanedDocument(document) ? (
                                        <button
                                            type="button"
                                            onClick={() => onDelete(document)}
                                            disabled={isDeleting}
                                            className="inline-flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs font-semibold text-destructive transition hover:bg-destructive/15 disabled:cursor-not-allowed disabled:opacity-60"
                                        >
                                            {isDeleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                                            {t("documentsPage.actions.deleteRecord")}
                                        </button>
                                    ) : null}
                                </div>

                                {canRenderPreview ? <FilePreview document={document} /> : null}

                                {!canRenderPreview && !issue && !isPreviewableDocument(document) ? (
                                    <div className="rounded-xl border border-border bg-background/60 p-5 text-sm text-muted-foreground">
                                        {t("documentsPage.preview.unsupportedInlineDownloadHint")}
                                    </div>
                                ) : null}

                                {!isPreviewableDocument(document) && (
                                    <p className="text-xs text-muted-foreground">
                                        {t("documentsPage.preview.noNativeBrowserRendering")}
                                    </p>
                                )}
                            </TabsContent>

                            <TabsContent value="index" className="mt-0 space-y-4">
                                <div className="flex flex-wrap items-start justify-between gap-3 rounded-xl border border-border bg-background/60 p-4">
                                    <div className="space-y-1">
                                        <h3 className="text-sm font-semibold">{t("documentsPage.index.exportTitle")}</h3>
                                        <p className="max-w-xl text-xs text-muted-foreground">
                                            {t("documentsPage.index.exportHint")}
                                        </p>
                                    </div>
                                    <button
                                        type="button"
                                        onClick={() => onDownloadIndexExport(document)}
                                        disabled={isExportingIndex}
                                        className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground transition hover:bg-muted disabled:cursor-not-allowed disabled:opacity-60"
                                    >
                                        {isExportingIndex ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                                        {t("documentsPage.actions.downloadIndexExport")}
                                    </button>
                                </div>

                                {isLoading && (
                                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                        {t("documentsPage.index.loading")}
                                    </div>
                                )}

                                {!isLoading && error && (
                                    <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive">
                                        {error}
                                    </div>
                                )}

                                {!isLoading && !error && (
                                    <IndexStructureView document={document} context={context} documentIndex={documentIndex} />
                                )}
                            </TabsContent>
                        </Tabs>
                    </div>
                </div>
            </aside>
        </div>
    );
}

type PendingDeletion =
    | {
        kind: "document";
        document: Document;
      }
    | {
        kind: "collection";
        collectionId: number;
        collectionName: string;
        documentCount: number;
      };

type DocumentsWorkspaceTab = "operations" | "audit";

export default function DocumentsPage() {
    const { locale, t, href } = useI18n();
    const { documents, isLoading, error, reloadDocuments, setDocuments } = useDocuments();
    const {
        overview: providerRuntimeOverview,
        isLoading: isProviderRuntimeLoading,
        error: providerRuntimeError,
    } = useProviderRuntimeOverview();
    const [monitoringSummary, setMonitoringSummary] = useState<DocumentMonitoringSummary | null>(null);
    const [monitoringError, setMonitoringError] = useState<string | null>(null);
    const [documentAuditReloadToken, setDocumentAuditReloadToken] = useState(0);
    const [selectedDocument, setSelectedDocument] = useState<Document | null>(null);
    const [downloadingDocumentId, setDownloadingDocumentId] = useState<number | null>(null);
    const [exportingIndexDocumentId, setExportingIndexDocumentId] = useState<number | null>(null);
    const [retryingDocumentId, setRetryingDocumentId] = useState<number | null>(null);
    const [pendingDeletion, setPendingDeletion] = useState<PendingDeletion | null>(null);
    const [deleteConfirmationValue, setDeleteConfirmationValue] = useState("");
    const [deletingTargetKey, setDeletingTargetKey] = useState<string | null>(null);
    const [operationalFilter, setOperationalFilter] = useState<DocumentOperationalFilter>("all");
    const [operationalSearch, setOperationalSearch] = useState("");
    const [workspaceTab, setWorkspaceTab] = useState<DocumentsWorkspaceTab>("operations");
    const workspaceDescription = {
        operations: t("documentsPage.workspace.descriptions.operations"),
        audit: t("documentsPage.workspace.descriptions.audit"),
    } satisfies Record<DocumentsWorkspaceTab, string>;
    const uploadHref = href("/workspace");

    useEffect(() => {
        const controller = new AbortController();

        async function loadMonitoringSummary(signal?: AbortSignal) {
            try {
                setMonitoringError(null);
                const data = await fetchDocumentMonitoringSummary(signal);
                setMonitoringSummary(data);
            } catch (err) {
                if (signal?.aborted) {
                    return;
                }

                setMonitoringError(err instanceof Error ? err.message : translate(locale, "documentsPage.errors.loadMonitoringSummary"));
            }
        }

        void loadMonitoringSummary(controller.signal);

        return () => controller.abort();
    }, [locale]);

    useEffect(() => {
        if (!selectedDocument) {
            return;
        }

        const latestDocument = documents.find((item) => item.id === selectedDocument.id);
        if (!latestDocument) {
            setSelectedDocument(null);
            return;
        }

        if (latestDocument !== selectedDocument) {
            setSelectedDocument(latestDocument);
        }
    }, [documents, selectedDocument]);

    const handleRetryDocument = async (document: Document) => {
        setRetryingDocumentId(document.id);

        try {
            const updatedDocument = await retryDocumentIndexing(document.id);

            setDocuments((current) => current.map((item) => (item.id === updatedDocument.id ? updatedDocument : item)));

            if (selectedDocument?.id === updatedDocument.id) {
                setSelectedDocument(updatedDocument);
            }

            await refreshAfterMutation();
            toast.success(
                isOrphanedDocument(document)
                    ? t("documentsPage.toast.retriedAfterRestore")
                    : t("documentsPage.toast.retried")
            );
        } catch (err) {
            toast.error(err instanceof Error ? err.message : t("documentsPage.errors.retry"));
        } finally {
            setRetryingDocumentId(null);
        }
    };

    const handleDownloadDocument = async (document: Document) => {
        setDownloadingDocumentId(document.id);

        try {
            await downloadDocumentOriginal(document);
            toast.success(t("documentsPage.toast.downloadStarted"));
        } catch (err) {
            const message = err instanceof Error ? err.message : t("documentsPage.errors.download");
            toast.error(message);

            if (message.toLowerCase().includes(t("documentsPage.errors.notFoundToken").toLowerCase())) {
                void reloadDocuments().catch(() => {
                    // The primary failure was already surfaced to the user.
                });
            }
        } finally {
            setDownloadingDocumentId(null);
        }
    };

    const handleDownloadDocumentIndexExport = async (document: Document) => {
        setExportingIndexDocumentId(document.id);

        try {
            await downloadDocumentIndexExport(document);
            toast.success(t("documentsPage.toast.indexExportStarted"));
        } catch (err) {
            toast.error(err instanceof Error ? err.message : t("documentsPage.errors.indexExport"));
        } finally {
            setExportingIndexDocumentId(null);
        }
    };

    const closeDeletionModal = () => {
        if (deletingTargetKey) {
            return;
        }

        setPendingDeletion(null);
        setDeleteConfirmationValue("");
    };

    const openDocumentDeletion = (document: Document) => {
        setPendingDeletion({ kind: "document", document });
        setDeleteConfirmationValue("");
    };

    const refreshAfterMutation = async () => {
        await Promise.all([
            reloadDocuments(),
            fetchDocumentMonitoringSummary().then(setMonitoringSummary).catch((err) => {
                setMonitoringError(err instanceof Error ? err.message : t("documentsPage.errors.loadMonitoringSummary"));
            }),
        ]);
        setDocumentAuditReloadToken((current) => current + 1);
    };

    const confirmDeletion = async () => {
        if (!pendingDeletion) {
            return;
        }

        const targetKey = pendingDeletion.kind === "document"
            ? `document:${pendingDeletion.document.id}`
            : `collection:${pendingDeletion.collectionId}`;

        setDeletingTargetKey(targetKey);

        try {
            if (pendingDeletion.kind === "document") {
                await deleteDocument(pendingDeletion.document.id);
                if (selectedDocument?.id === pendingDeletion.document.id) {
                    setSelectedDocument(null);
                }
                setDocuments((current) => current.filter((item) => item.id !== pendingDeletion.document.id));
                toast.success(t("documentsPage.toast.documentDeleted"));
            } else {
                await deleteCollectionDocuments(pendingDeletion.collectionId);
                setDocuments((current) => current.filter((item) => item.collection_id !== pendingDeletion.collectionId));
                if (selectedDocument?.collection_id === pendingDeletion.collectionId) {
                    setSelectedDocument(null);
                }
                toast.success(t("documentsPage.toast.collectionDeleted"));
            }

            await refreshAfterMutation();
            setPendingDeletion(null);
            setDeleteConfirmationValue("");
        } catch (err) {
            toast.error(err instanceof Error ? err.message : t("documentsPage.errors.delete"));
        } finally {
            setDeletingTargetKey(null);
        }
    };

    return (
        <AppShell>
            <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="mb-8">
                <div className="mb-3 flex items-center gap-2">
                    <span className="h-px w-8 bg-primary" />
                    <span className="font-bold uppercase tracking-[0.3em] text-[10px] text-primary">{t("documentsPage.hero.eyebrow")}</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">{t("documentsPage.hero.title")}</h1>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-muted-foreground">
                    {t("documentsPage.hero.description")}
                </p>
            </motion.div>

            <motion.div
                {...fadeUp}
                transition={{ delay: 0.1, duration: 0.4 }}
                className="mb-6"
            >
                <ProviderRuntimeCallout
                    role="indexing"
                    assignment={providerRuntimeOverview.indexing}
                    isLoading={isProviderRuntimeLoading}
                    error={providerRuntimeError}
                />
            </motion.div>

            <motion.div
                {...fadeUp}
                transition={{ delay: 0.14, duration: 0.4 }}
                className="surface-elevated rounded-xl border border-border"
            >
                <div className="flex items-center justify-between border-b border-border px-6 py-4">
                    <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                            <Database className="h-5 w-5" />
                        </div>
                        <div>
                            <h2 className="text-sm font-bold tracking-tight">{t("shell.nav.documents")}</h2>
                            <p className="text-xs text-muted-foreground">
                                {t("documentsPage.library.count", { count: documents.length })}
                            </p>
                        </div>
                    </div>
                </div>

                <div className="p-6">
                    <Tabs value={workspaceTab} onValueChange={(value) => setWorkspaceTab(value as DocumentsWorkspaceTab)} className="w-full">
                        <div className="mb-6 flex flex-col gap-4 border-b border-border pb-6 lg:flex-row lg:items-end lg:justify-between">
                            <div className="max-w-2xl">
                                <TabsList className="h-auto flex-wrap justify-start gap-2 bg-transparent p-0">
                                    <TabsTrigger
                                        value="operations"
                                        className="rounded-full border border-border bg-background px-4 py-2 text-xs font-bold uppercase tracking-[0.24em] text-muted-foreground data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none"
                                    >
                                        {t("documentsPage.workspace.tabs.operations")}
                                    </TabsTrigger>
                                    <TabsTrigger
                                        value="audit"
                                        className="rounded-full border border-border bg-background px-4 py-2 text-xs font-bold uppercase tracking-[0.24em] text-muted-foreground data-[state=active]:border-primary/20 data-[state=active]:bg-primary/10 data-[state=active]:text-primary data-[state=active]:shadow-none"
                                    >
                                        {t("documentsPage.workspace.tabs.audit")}
                                    </TabsTrigger>
                                </TabsList>
                                <p className="mt-3 text-sm text-muted-foreground">
                                    {workspaceDescription[workspaceTab]}
                                </p>
                            </div>

                        </div>

                        {/* O callout de runtime ja fica no topo da pagina. */}
                        <TabsContent value="operations" className="mt-0 space-y-6">
                            {monitoringError && (
                                <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                                    {monitoringError}
                                </div>
                            )}

                            {isLoading && (
                                <div className="flex min-h-64 items-center justify-center gap-3 text-sm text-muted-foreground">
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                    {t("documentsPage.loading")}
                                </div>
                            )}

                            {!isLoading && error && (
                                <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                                    {error}
                                </div>
                            )}

                            {!isLoading && !error && documents.length === 0 && (
                                <div className="flex min-h-48 flex-col items-center justify-center rounded-xl border border-border bg-background/40 px-6 text-center">
                                    <FileText className="mb-4 h-10 w-10 text-muted-foreground" />
                                    <h3 className="text-sm font-semibold">{t("documentsPage.empty.title")}</h3>
                                    <p className="mt-2 max-w-md text-xs leading-relaxed text-muted-foreground">
                                        {t("documentsPage.empty.description")}
                                    </p>
                                </div>
                            )}

                            {!isLoading && !error && monitoringSummary && (
                                <MonitoringSummaryPanel summary={monitoringSummary} />
                            )}

                            {!isLoading && !error && documents.length > 0 && (
                                <OperationalObservabilityPanel
                                    documents={documents}
                                    summary={monitoringSummary}
                                    activeFilter={operationalFilter}
                                    searchTerm={operationalSearch}
                                    onFilterChange={setOperationalFilter}
                                    onSearchTermChange={setOperationalSearch}
                                    onOpenPreview={setSelectedDocument}
                                    onRetry={handleRetryDocument}
                                    retryingDocumentId={retryingDocumentId}
                                />
                            )}
                        </TabsContent>

                        <TabsContent value="audit" className="mt-0">
                            <DocumentAuditExplorer
                                locale={locale}
                                reloadToken={documentAuditReloadToken}
                                selectedDocument={selectedDocument}
                            />
                        </TabsContent>
                    </Tabs>
                </div>
            </motion.div>

            <DocumentPreviewPanel
                document={selectedDocument}
                uploadHref={uploadHref}
                onRetry={handleRetryDocument}
                onDownload={handleDownloadDocument}
                onDownloadIndexExport={handleDownloadDocumentIndexExport}
                onDelete={openDocumentDeletion}
                isRetrying={retryingDocumentId === selectedDocument?.id}
                isDownloading={downloadingDocumentId === selectedDocument?.id}
                isExportingIndex={exportingIndexDocumentId === selectedDocument?.id}
                isDeleting={deletingTargetKey === `document:${selectedDocument?.id ?? ""}`}
                onClose={() => setSelectedDocument(null)}
            />
            <DangerConfirmModal
                isOpen={pendingDeletion !== null}
                eyebrow={t("documentsPage.deleteModal.eyebrow")}
                title={pendingDeletion?.kind === "document"
                    ? t("documentsPage.deleteModal.documentTitle")
                    : t("documentsPage.deleteModal.collectionTitle")}
                description={pendingDeletion?.kind === "document"
                    ? t("documentsPage.deleteModal.documentDescription")
                    : t("documentsPage.deleteModal.collectionDescription", { count: pendingDeletion?.documentCount ?? 0 })}
                confirmLabel={t("documentsPage.deleteModal.confirmLabel")}
                cancelLabel={t("common.dangerConfirm.cancelLabel")}
                closeLabel={t("common.dangerConfirm.closeLabel")}
                expectedValue={pendingDeletion?.kind === "document" ? pendingDeletion.document.title : pendingDeletion?.collectionName}
                confirmationValue={deleteConfirmationValue}
                confirmationPrompt={pendingDeletion
                    ? t("common.dangerConfirm.typeToConfirm", {
                        value: pendingDeletion.kind === "document" ? pendingDeletion.document.title : pendingDeletion.collectionName,
                    })
                    : undefined}
                inputLabel={t("common.dangerConfirm.inputLabel")}
                onConfirmationChange={setDeleteConfirmationValue}
                onCancel={closeDeletionModal}
                onConfirm={() => void confirmDeletion()}
                isPending={deletingTargetKey !== null}
            />
        </AppShell>
    );
}
