"use client";

import { useEffect, useState } from "react";

import {
    fetchIndexingBenchmark,
    type IndexingBenchmarkCase,
    type IndexingBenchmarkReport,
} from "@/lib/api/indexing-benchmark";

function formatPercent(value: number) {
    return `${value.toFixed(1)}%`;
}

function caseExpectation(item: IndexingBenchmarkCase) {
    if (item.scope === "COLLECTION") {
        return item.expected_document_title;
    }

    const parts = [item.expected_node_title];
    if (item.expected_page_number !== null) {
        parts.push(`p.${item.expected_page_number}`);
    }
    return parts.filter(Boolean).join(" · ");
}

function caseActual(item: IndexingBenchmarkCase) {
    if (item.scope === "COLLECTION") {
        return item.actual_document_title || "No ranked document";
    }

    const parts = [item.actual_node_title];
    if (item.actual_page_number !== null) {
        parts.push(`p.${item.actual_page_number}`);
    }
    return parts.filter(Boolean).join(" · ") || "No ranked result";
}

export function IndexingBenchmarkSection() {
    const [report, setReport] = useState<IndexingBenchmarkReport | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const controller = new AbortController();

        async function load() {
            try {
                setIsLoading(true);
                setError(null);
                setReport(await fetchIndexingBenchmark(controller.signal));
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }
                setError(err instanceof Error ? err.message : "Could not load the public indexing benchmark.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void load();
        return () => controller.abort();
    }, []);

    return (
        <div className="space-y-5">
            <p>
                Esta seção consome o endpoint público <code>/api/v1/meta/indexing-benchmark</code> para separar duas coisas que
                antes ficavam misturadas: o score heurístico operacional e a medição objetiva da baseline atual em um corpus
                controlado com ground truth explícito.
            </p>

            {isLoading ? (
                <div className="rounded-xl border border-border bg-background/50 p-5 text-sm text-muted-foreground">
                    Carregando benchmark público de indexação...
                </div>
            ) : error ? (
                <div className="rounded-xl border border-destructive/30 bg-destructive/10 p-5 text-sm text-destructive">
                    {error}
                </div>
            ) : report ? (
                <>
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                        {[
                            {
                                label: "Exact match",
                                value: formatPercent(report.metrics.exact_match_accuracy),
                                detail: `${report.metrics.document_queries + report.metrics.collection_queries} casos controlados`,
                            },
                            {
                                label: "Node top-1",
                                value: formatPercent(report.metrics.tree_node_top1_accuracy),
                                detail: `Recall@3 ${formatPercent(report.metrics.tree_node_recall_at_3)}`,
                            },
                            {
                                label: "Collection top-1",
                                value: formatPercent(report.metrics.collection_document_top1_accuracy),
                                detail: `Recall@3 ${formatPercent(report.metrics.collection_document_recall_at_3)}`,
                            },
                            {
                                label: "Avg heuristic",
                                value: report.metrics.average_heuristic_score.toFixed(1),
                                detail: `Range ${report.metrics.minimum_heuristic_score.toFixed(1)}-${report.metrics.maximum_heuristic_score.toFixed(1)}`,
                            },
                        ].map((item) => (
                            <div key={item.label} className="rounded-xl border border-border bg-background/50 p-4">
                                <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">{item.label}</p>
                                <p className="mt-2 text-2xl font-semibold text-foreground">{item.value}</p>
                                <p className="mt-1 text-sm text-muted-foreground">{item.detail}</p>
                            </div>
                        ))}
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Como o benchmark roda</h4>
                            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{report.summary}</p>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {report.methodology.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Como interpretar</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {report.pass_criteria.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                            <p className="mt-4 rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm leading-relaxed text-foreground/90">
                                {report.heuristic_quality_note}
                            </p>
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <div className="flex flex-wrap items-center gap-3">
                            <h4 className="text-sm font-semibold text-foreground">Corpus versionado</h4>
                            <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                {report.dataset_version}
                            </span>
                            <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                reviewed {report.last_reviewed_on}
                            </span>
                        </div>
                        <p className="mt-3 font-mono text-xs text-primary">{report.corpus_source_path}</p>
                        <div className="mt-4 grid gap-4 lg:grid-cols-3">
                            {report.corpus.map((item) => (
                                <div key={item.id} className="rounded-xl border border-border/70 bg-background/70 p-4">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <p className="text-sm font-semibold text-foreground">{item.title}</p>
                                        <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                            {item.mime_type}
                                        </span>
                                    </div>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.focus}</p>
                                    <div className="mt-3 space-y-1 text-xs text-muted-foreground">
                                        <p>Queries: <span className="font-mono text-foreground">{item.query_count}</span></p>
                                        <p>Outline nodes: <span className="font-mono text-foreground">{item.outline_node_count}</span></p>
                                        <p>Tables: <span className="font-mono text-foreground">{item.table_count}</span></p>
                                        <p>Heuristic score: <span className="font-mono text-foreground">{item.heuristic_quality_score.toFixed(1)}</span></p>
                                        <p>Structure: <span className="font-mono text-foreground">{item.structure_passed ? "pass" : "fail"}</span></p>
                                        <p>Retrieval ready: <span className="font-mono text-foreground">{item.retrieval_ready ? "true" : "false"}</span></p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <h4 className="text-sm font-semibold text-foreground">Casos avaliados</h4>
                        <div className="mt-4 space-y-3">
                            {report.cases.map((item) => (
                                <div key={item.id} className="rounded-xl border border-border/70 bg-background/70 p-4">
                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                        <div>
                                            <p className="text-sm font-semibold text-foreground">{item.query}</p>
                                            <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{item.scope}</p>
                                        </div>
                                        <span
                                            className={`rounded-full px-3 py-1 text-[10px] font-bold uppercase tracking-[0.18em] ${
                                                item.passed
                                                    ? "border border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                                                    : "border border-destructive/20 bg-destructive/10 text-destructive"
                                            }`}
                                        >
                                            {item.passed ? "pass" : "fail"}
                                        </span>
                                    </div>
                                    {item.document_title ? (
                                        <p className="mt-2 text-sm text-muted-foreground">
                                            Documento: <span className="font-medium text-foreground">{item.document_title}</span>
                                        </p>
                                    ) : null}
                                    <div className="mt-3 grid gap-3 md:grid-cols-2">
                                        <div>
                                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Expected</p>
                                            <p className="mt-1 text-sm text-foreground">{caseExpectation(item)}</p>
                                        </div>
                                        <div>
                                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">Actual</p>
                                            <p className="mt-1 text-sm text-foreground">{caseActual(item)}</p>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <h4 className="text-sm font-semibold text-foreground">Limites explícitos</h4>
                        <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                            {report.limitations.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                    </div>
                </>
            ) : null}
        </div>
    );
}
