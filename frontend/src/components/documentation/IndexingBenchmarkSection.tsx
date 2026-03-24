"use client";

import {
    localizeBenchmarkScope,
    localizeIndexingBenchmark,
} from "@/components/documentation/documentation-resource-localization";
import { DocumentationMetricCard } from "@/components/documentation/DocumentationMetricCard";
import { DocumentationStatusMessage } from "@/components/documentation/DocumentationStatusMessage";
import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";
import {
    fetchIndexingBenchmark,
    type IndexingBenchmarkCase,
    type IndexingBenchmarkReport,
} from "@/lib/api/indexing-benchmark";
import { useI18n } from "@/lib/i18n/provider";

function formatBenchmarkPercentage(value: number) {
    return `${value.toFixed(1)}%`;
}

function formatExpectedBenchmarkMatch(item: IndexingBenchmarkCase) {
    if (item.scope === "COLLECTION") {
        return item.expected_document_title;
    }

    const parts = [item.expected_node_title];
    if (item.expected_page_number !== null) {
        parts.push(`p.${item.expected_page_number}`);
    }
    return parts.filter(Boolean).join(" · ");
}

function formatActualBenchmarkMatch(item: IndexingBenchmarkCase, emptyLabel: string) {
    if (item.scope === "COLLECTION") {
        return item.actual_document_title || emptyLabel;
    }

    const parts = [item.actual_node_title];
    if (item.actual_page_number !== null) {
        parts.push(`p.${item.actual_page_number}`);
    }
    return parts.filter(Boolean).join(" · ") || emptyLabel;
}

export function IndexingBenchmarkSection() {
    const { locale } = useI18n();
    const copy = locale === "pt-BR"
        ? {
            fallbackError: "Nao foi possivel carregar o benchmark publico de indexacao.",
            intro: "Esta seção consome o endpoint público ",
            introSuffix:
                " para separar duas coisas que antes ficavam misturadas: a pontuação heurística operacional e a medição objetiva da linha de base atual em um corpus controlado com verdade de referência explícita.",
            loading: "Carregando benchmark público de indexação...",
            metrics: {
                exactMatch: "Correspondência exata",
                nodeTop1: "Nó top-1",
                collectionTop1: "Documento top-1 por coleção",
                avgHeuristic: "Média heurística",
                controlledCases: "casos controlados",
                recall: "Recall@3",
                range: "Faixa",
            },
            sections: {
                howItRuns: "Como o benchmark roda",
                interpretation: "Como interpretar",
                versionedCorpus: "Corpus versionado",
                reviewed: "revisado",
                queries: "Consultas",
                outlineNodes: "Nós estruturais",
                tables: "Tabelas",
                heuristicScore: "Pontuação heurística",
                structure: "Estrutura",
                retrievalReady: "Pronto para recuperação",
                assessedCases: "Casos avaliados",
                document: "Documento",
                expected: "Esperado",
                actual: "Obtido",
                explicitLimits: "Limites explícitos",
            },
            status: {
                pass: "aprovado",
                fail: "falhou",
                yes: "sim",
                no: "não",
            },
            noRanked: "Nenhum resultado ranqueado",
        }
        : locale === "es"
            ? {
                fallbackError: "No se pudo cargar el benchmark público de indexación.",
                intro: "Esta sección consume el endpoint público ",
                introSuffix:
                    " para separar dos cosas que antes aparecían mezcladas: el score heurístico operativo y la medición objetiva de la baseline actual en un corpus controlado con ground truth explícito.",
                loading: "Cargando benchmark público de indexación...",
                metrics: {
                    exactMatch: "Exact match",
                    nodeTop1: "Node top-1",
                    collectionTop1: "Collection top-1",
                    avgHeuristic: "Promedio heurístico",
                    controlledCases: "casos controlados",
                    recall: "Recall@3",
                    range: "Rango",
                },
                sections: {
                    howItRuns: "Cómo corre el benchmark",
                    interpretation: "Cómo interpretarlo",
                    versionedCorpus: "Corpus versionado",
                    reviewed: "revisado",
                    queries: "Queries",
                    outlineNodes: "Nodos de outline",
                    tables: "Tablas",
                    heuristicScore: "Score heurístico",
                    structure: "Estructura",
                    retrievalReady: "Listo para recuperación",
                    assessedCases: "Casos evaluados",
                    document: "Documento",
                    expected: "Esperado",
                    actual: "Obtenido",
                    explicitLimits: "Límites explícitos",
                },
                status: {
                    pass: "pass",
                    fail: "fail",
                    yes: "sí",
                    no: "no",
                },
                noRanked: "Sin resultado rankeado",
            }
            : {
                fallbackError: "Could not load the public indexing benchmark.",
                intro: "This section consumes the public endpoint ",
                introSuffix:
                    " to separate two things that used to be mixed together: the operational heuristic score and the objective measurement of the current baseline on a controlled corpus with explicit ground truth.",
                loading: "Loading public indexing benchmark...",
                metrics: {
                    exactMatch: "Exact match",
                    nodeTop1: "Node top-1",
                    collectionTop1: "Collection top-1",
                    avgHeuristic: "Avg heuristic",
                    controlledCases: "controlled cases",
                    recall: "Recall@3",
                    range: "Range",
                },
                sections: {
                    howItRuns: "How the benchmark runs",
                    interpretation: "How to interpret it",
                    versionedCorpus: "Versioned corpus",
                    reviewed: "reviewed",
                    queries: "Queries",
                    outlineNodes: "Outline nodes",
                    tables: "Tables",
                    heuristicScore: "Heuristic score",
                    structure: "Structure",
                    retrievalReady: "Retrieval ready",
                    assessedCases: "Assessed cases",
                    document: "Document",
                    expected: "Expected",
                    actual: "Actual",
                    explicitLimits: "Explicit limits",
                },
                status: {
                    pass: "pass",
                    fail: "fail",
                    yes: "yes",
                    no: "no",
                },
                noRanked: "No ranked result",
            };
    const {
        resource: report,
        isLoading,
        error,
    } = useDocumentationResource<IndexingBenchmarkReport | null>(
        fetchIndexingBenchmark,
        copy.fallbackError
    );
    const localizedReport = report ? localizeIndexingBenchmark(report, locale) : null;

    return (
        <div className="space-y-5">
            <p>
                {copy.intro}<code>/api/v1/meta/indexing-benchmark</code>{copy.introSuffix}
            </p>

            {isLoading ? (
                <DocumentationStatusMessage>
                    {copy.loading}
                </DocumentationStatusMessage>
            ) : error ? (
                <DocumentationStatusMessage tone="error">{error}</DocumentationStatusMessage>
            ) : localizedReport ? (
                <>
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                        {[
                            {
                                label: copy.metrics.exactMatch,
                                value: formatBenchmarkPercentage(localizedReport.metrics.exact_match_accuracy),
                                detail: `${localizedReport.metrics.document_queries + localizedReport.metrics.collection_queries} ${copy.metrics.controlledCases}`,
                            },
                            {
                                label: copy.metrics.nodeTop1,
                                value: formatBenchmarkPercentage(localizedReport.metrics.tree_node_top1_accuracy),
                                detail: `${copy.metrics.recall} ${formatBenchmarkPercentage(localizedReport.metrics.tree_node_recall_at_3)}`,
                            },
                            {
                                label: copy.metrics.collectionTop1,
                                value: formatBenchmarkPercentage(localizedReport.metrics.collection_document_top1_accuracy),
                                detail: `${copy.metrics.recall} ${formatBenchmarkPercentage(localizedReport.metrics.collection_document_recall_at_3)}`,
                            },
                            {
                                label: copy.metrics.avgHeuristic,
                                value: localizedReport.metrics.average_heuristic_score.toFixed(1),
                                detail: `${copy.metrics.range} ${localizedReport.metrics.minimum_heuristic_score.toFixed(1)}-${localizedReport.metrics.maximum_heuristic_score.toFixed(1)}`,
                            },
                        ].map((item) => (
                            <DocumentationMetricCard
                                key={item.label}
                                label={item.label}
                                value={item.value}
                                detail={item.detail}
                            />
                        ))}
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.sections.howItRuns}</h4>
                            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{localizedReport.summary}</p>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedReport.methodology.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.sections.interpretation}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedReport.pass_criteria.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                            <p className="mt-4 rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 text-sm leading-relaxed text-foreground/90">
                                {localizedReport.heuristic_quality_note}
                            </p>
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <div className="flex flex-wrap items-center gap-3">
                            <h4 className="text-sm font-semibold text-foreground">{copy.sections.versionedCorpus}</h4>
                            <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                {localizedReport.dataset_version}
                            </span>
                            <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                {copy.sections.reviewed} {localizedReport.last_reviewed_on}
                            </span>
                        </div>
                        <p className="mt-3 font-mono text-xs text-primary">{localizedReport.corpus_source_path}</p>
                        <div className="mt-4 grid gap-4 lg:grid-cols-3">
                            {localizedReport.corpus.map((item) => (
                                <div key={item.id} className="rounded-xl border border-border/70 bg-background/70 p-4">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <p className="text-sm font-semibold text-foreground">{item.title}</p>
                                        <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                            {item.mime_type}
                                        </span>
                                    </div>
                                    <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.focus}</p>
                                    <div className="mt-3 space-y-1 text-xs text-muted-foreground">
                                        <p>{copy.sections.queries}: <span className="font-mono text-foreground">{item.query_count}</span></p>
                                        <p>{copy.sections.outlineNodes}: <span className="font-mono text-foreground">{item.outline_node_count}</span></p>
                                        <p>{copy.sections.tables}: <span className="font-mono text-foreground">{item.table_count}</span></p>
                                        <p>{copy.sections.heuristicScore}: <span className="font-mono text-foreground">{item.heuristic_quality_score.toFixed(1)}</span></p>
                                        <p>{copy.sections.structure}: <span className="font-mono text-foreground">{item.structure_passed ? copy.status.pass : copy.status.fail}</span></p>
                                        <p>{copy.sections.retrievalReady}: <span className="font-mono text-foreground">{item.retrieval_ready ? copy.status.yes : copy.status.no}</span></p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <h4 className="text-sm font-semibold text-foreground">{copy.sections.assessedCases}</h4>
                        <div className="mt-4 space-y-3">
                            {localizedReport.cases.map((item) => (
                                <div key={item.id} className="rounded-xl border border-border/70 bg-background/70 p-4">
                                    <div className="flex flex-wrap items-center justify-between gap-2">
                                        <div>
                                            <p className="text-sm font-semibold text-foreground">{item.query}</p>
                                            <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                                                {localizeBenchmarkScope(item.scope, locale)}
                                            </p>
                                        </div>
                                        <span
                                            className={`rounded-full px-3 py-1 text-[10px] font-bold uppercase tracking-[0.18em] ${
                                                item.passed
                                                    ? "border border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                                                    : "border border-destructive/20 bg-destructive/10 text-destructive"
                                            }`}
                                        >
                                            {item.passed ? copy.status.pass : copy.status.fail}
                                        </span>
                                    </div>
                                    {item.document_title ? (
                                        <p className="mt-2 text-sm text-muted-foreground">
                                            {copy.sections.document}: <span className="font-medium text-foreground">{item.document_title}</span>
                                        </p>
                                    ) : null}
                                    <div className="mt-3 grid gap-3 md:grid-cols-2">
                                        <div>
                                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">{copy.sections.expected}</p>
                                            <p className="mt-1 text-sm text-foreground">{formatExpectedBenchmarkMatch(item)}</p>
                                        </div>
                                        <div>
                                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">{copy.sections.actual}</p>
                                            <p className="mt-1 text-sm text-foreground">{formatActualBenchmarkMatch(item, copy.noRanked)}</p>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="rounded-xl border border-border bg-background/50 p-5">
                        <h4 className="text-sm font-semibold text-foreground">{copy.sections.explicitLimits}</h4>
                        <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                            {localizedReport.limitations.map((item) => (
                                <li key={item}>{item}</li>
                            ))}
                        </ul>
                    </div>
                </>
            ) : null}
        </div>
    );
}
