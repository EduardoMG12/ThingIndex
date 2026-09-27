"use client";

import { ChevronDown, FileText, Crosshair } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

export type Source = {
    id: string;
    docName: string;
    page: number;
    snippet: string;
    documentId?: number;
    nodeId?: string;
    paragraphId?: number;
    lines?: number[];
    matchedTerms?: string[];
    box?: { xMin: number; yMin: number; xMax: number; yMax: number; pageWidth: number; pageHeight: number };
    lineBoxes?: number[][];
    indexedAt?: string;
    continuesOnPage?: number;
};

interface SourceReferenceProps {
    source: Source;
    /** true quando o documento foi reindexado depois desta resposta. */
    reindexado?: boolean;
    /** Posicao na lista (1..n), a mesma ordem em que o modelo recebeu as fontes. */
    numero?: number;
    /** Fonte que esta focada no canvas. */
    ativa?: boolean;
    /** Quando existe (e a fonte tem documentId), mostra "Ver no documento". */
    onVerNoDocumento?: (source: Source) => void;
}

const SourceReference = ({ source, numero, ativa = false, reindexado = false, onVerNoDocumento }: SourceReferenceProps) => {
    const { t } = useI18n();
    const podeIr = Boolean(onVerNoDocumento && source.documentId);

    return (
        <details
            className={`group rounded-2xl border bg-background/70 transition-colors open:bg-muted/40 ${ativa ? "border-primary ring-1 ring-primary/40" : "border-border/60"}`}
            data-testid="fonte"
            data-ativa={ativa ? "true" : undefined}
        >
            <summary className="flex cursor-pointer list-none items-center gap-3 px-3 py-2.5 marker:hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35">
                {numero ? (
                    <span className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] font-semibold tabular-nums ${ativa ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}>
                        {numero}
                    </span>
                ) : (
                    <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
                )}
                <div className="min-w-0 flex-1">
                    <p className="truncate text-[11px] font-semibold text-foreground" title={source.docName}>
                        {source.docName}
                    </p>
                    <p className="flex flex-wrap items-center gap-x-1 text-[10px] text-muted-foreground">
                        {/* Um texto so e sem quebra: "Pag. 4-5" nao pode virar tres linhas. */}
                        <span className="whitespace-nowrap tabular-nums">
                            {`${t("dashboard.chat.page")} ${source.page}${source.continuesOnPage ? `–${source.continuesOnPage}` : ""}`}
                        </span>
                        {reindexado ? (
                            <span className="rounded bg-amber-500/15 px-1 py-0.5 text-[9px] font-medium text-amber-600 dark:text-amber-400" title={t("dashboard.chat.reindexedTitle")} data-testid="fonte-reindexada">
                                {t("dashboard.chat.reindexed")}
                            </span>
                        ) : null}
                    </p>
                </div>

                {podeIr ? (
                    <button
                        type="button"
                        data-testid="ver-no-documento"
                        title={t("workspace.chat.viewInDocument")}
                        aria-label={t("workspace.chat.viewInDocument")}
                        className="shrink-0 rounded-md border border-border/60 p-1 text-primary hover:bg-primary/10"
                        onClick={(event) => {
                            // Dentro do <summary>: sem isto o clique abre/fecha o card.
                            event.preventDefault();
                            event.stopPropagation();
                            onVerNoDocumento?.(source);
                        }}
                    >
                        <Crosshair className="h-3.5 w-3.5" aria-hidden />
                    </button>
                ) : null}
                <ChevronDown
                    aria-hidden="true"
                    className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180"
                />
            </summary>

            <div className="space-y-2 border-t border-border/60 px-4 py-3">
                <p className="text-sm leading-relaxed text-muted-foreground">&quot;...{source.snippet}...&quot;</p>
                {source.matchedTerms && source.matchedTerms.length > 0 ? (
                    <ul className="flex flex-wrap gap-1" aria-label={t("dashboard.chat.matchedTerms")} data-testid="termos-da-fonte">
                        {source.matchedTerms.map((termo) => (
                            <li key={termo} className="rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[10px] text-primary">
                                {termo}
                            </li>
                        ))}
                    </ul>
                ) : null}
                <p className="text-[10px] text-muted-foreground">
                    {t("dashboard.chat.sourceOrigin")}
                </p>
            </div>
        </details>
    );
};

export default SourceReference;
