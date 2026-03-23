"use client";

import { ArrowRight, AlertTriangle, Clock3, RotateCw } from "lucide-react";

import type {
    DocumentOperationalAnomalySummary,
    DocumentOperationalFilter,
} from "@/lib/document-observability";
import { useI18n } from "@/lib/i18n/provider";

type OperationalAnomalyBoardProps = {
    anomalies: DocumentOperationalAnomalySummary[];
    onFocusFilter: (filter: DocumentOperationalFilter) => void;
};

export default function OperationalAnomalyBoard({
    anomalies,
    onFocusFilter,
}: OperationalAnomalyBoardProps) {
    const { t } = useI18n();

    return (
        <div className="rounded-2xl border border-border bg-background/60 p-5">
            <div className="flex items-center gap-2 text-[10px] uppercase tracking-widest text-muted-foreground">
                <AlertTriangle className="h-3.5 w-3.5 text-primary" />
                {t("documentsPage.observability.anomalies.title")}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
                {t("documentsPage.observability.anomalies.description")}
            </p>

            {anomalies.length > 0 ? (
                <div className="mt-4 grid gap-3 md:grid-cols-2">
                    {anomalies.map((anomaly) => {
                        const anomalyTitle = t(`documentsPage.observability.anomalies.kinds.${anomaly.kind}`);

                        return (
                            <div
                                key={anomaly.kind}
                                className={`rounded-xl border p-4 ${buildAnomalyTone(anomaly.severity)}`}
                            >
                                <div className="flex items-start justify-between gap-3">
                                    <div className="min-w-0">
                                        <div className="flex items-center gap-2 text-foreground">
                                            {renderAnomalyIcon(anomaly.kind)}
                                            <p className="text-sm font-semibold">{anomalyTitle}</p>
                                        </div>
                                        <p className="mt-2 text-xs text-muted-foreground">
                                            {t("documentsPage.observability.anomalies.nextAction")}:
                                        </p>
                                        <p className="mt-1 text-xs text-foreground/90">
                                            {t(`documentsPage.issues.${anomaly.kind}.hint`)}
                                        </p>
                                    </div>
                                    <span className="shrink-0 rounded-full border border-border bg-background/80 px-2 py-1 text-[10px] font-mono text-muted-foreground">
                                        {anomaly.count}
                                    </span>
                                </div>

                                {anomaly.sampleTitles.length > 0 ? (
                                    <p className="mt-3 text-[11px] text-foreground/85">
                                        <span className="font-semibold text-foreground">
                                            {t("documentsPage.observability.anomalies.documentsLabel")}:
                                        </span>{" "}
                                        {anomaly.sampleTitles.join(" · ")}
                                    </p>
                                ) : null}

                                {typeof anomaly.oldestAgeMinutes === "number" ? (
                                    <p className="mt-2 text-[11px] text-muted-foreground">
                                        {t("documentsPage.observability.anomalies.oldestAge", {
                                            count: anomaly.oldestAgeMinutes,
                                        })}
                                    </p>
                                ) : null}

                                <button
                                    type="button"
                                    onClick={() => onFocusFilter(anomaly.filter)}
                                    className="mt-4 inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground"
                                >
                                    <ArrowRight className="h-3.5 w-3.5" />
                                    {t("documentsPage.observability.actionPlan.focusTarget", {
                                        target: anomalyTitle,
                                    })}
                                </button>
                            </div>
                        );
                    })}
                </div>
            ) : (
                <p className="mt-4 text-xs text-muted-foreground">
                    {t("documentsPage.observability.anomalies.empty")}
                </p>
            )}
        </div>
    );
}

function buildAnomalyTone(severity: "nominal" | "attention" | "critical") {
    switch (severity) {
        case "critical":
            return "border-destructive/20 bg-destructive/10";
        case "attention":
            return "border-amber-500/20 bg-amber-500/10";
        default:
            return "border-border bg-card/70";
    }
}

function renderAnomalyIcon(kind: DocumentOperationalAnomalySummary["kind"]) {
    switch (kind) {
        case "retrying":
            return <RotateCw className="h-4 w-4" />;
        case "stalledProcessing":
        case "stalledQueue":
            return <Clock3 className="h-4 w-4" />;
        default:
            return <AlertTriangle className="h-4 w-4" />;
    }
}
