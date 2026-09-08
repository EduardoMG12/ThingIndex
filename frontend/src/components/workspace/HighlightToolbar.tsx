"use client";

import { ChevronLeft, ChevronRight, Link2, X } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";
import { classesDaCor, CORES, rotuloDaCor, type HighlightColor, type Legend } from "@/lib/workspace/highlights";

/**
 * Barra flutuante sobre o paragrafo selecionado: cinco chips de cor, "sem cor"
 * e "Ver conexoes". Vive dentro da pagina, acima do paragrafo (abaixo quando
 * nao cabe), para a acao acontecer em cima do PDF e nao na sidebar.
 */

/** Navegador "conexao i de N": aparece quando as conexoes do trecho ja foram pedidas. */
export type NavegadorDeConexoes = {
    total: number;
    /** 0-based; -1 enquanto nenhuma foi escolhida. */
    indice: number;
    carregando: boolean;
    onIr: (indice: number) => void;
};

export type HighlightToolbarProps = {
    /** Rect do paragrafo em px da pagina. */
    rect: { left: number; top: number; width: number; height: number };
    corAtual: HighlightColor | null;
    legenda: Legend;
    onMarcar: (cor: HighlightColor | null) => void;
    onVerConexoes: () => void;
    podeVerConexoes: boolean;
    navegador?: NavegadorDeConexoes | null;
};

const ALTURA_DA_BARRA = 32;

export function HighlightToolbar({ rect, corAtual, legenda, onMarcar, onVerConexoes, podeVerConexoes, navegador = null }: HighlightToolbarProps) {
    const { t } = useI18n();
    const atual = navegador && navegador.total > 0 ? Math.max(navegador.indice, 0) : -1;
    const podeVoltar = Boolean(navegador && navegador.total > 0 && atual > 0);
    const podeAvancar = Boolean(navegador && navegador.total > 0 && atual < navegador.total - 1);
    const nomeDaCor = (cor: HighlightColor) => t(`workspace.highlight.colors.${cor}`);
    const cabeAcima = rect.top >= ALTURA_DA_BARRA + 4;
    const top = cabeAcima ? rect.top - ALTURA_DA_BARRA - 4 : rect.top + rect.height + 4;
    const parar = (event: React.SyntheticEvent) => event.stopPropagation();

    return (
        <div
            role="toolbar"
            aria-label={t("workspace.highlight.toolbar")}
            data-testid="barra-de-grifo"
            className="pointer-events-auto absolute z-10 flex items-center gap-1 rounded-md border border-border bg-card/95 px-1.5 shadow-md"
            style={{ left: Math.max(0, rect.left), top, height: ALTURA_DA_BARRA }}
            onPointerDown={parar}
            onClick={parar}
            tabIndex={-1}
            onKeyDown={(event) => {
                if (!navegador || navegador.total === 0) {
                    return;
                }
                if (event.key === "ArrowRight" && podeAvancar) {
                    event.preventDefault();
                    navegador.onIr(atual + 1);
                } else if (event.key === "ArrowLeft" && podeVoltar) {
                    event.preventDefault();
                    navegador.onIr(atual - 1);
                }
            }}
        >
            {CORES.map((cor) => {
                const ativa = corAtual === cor;
                const rotulo = rotuloDaCor(legenda, cor, nomeDaCor);
                return (
                    <button
                        key={cor}
                        type="button"
                        onClick={() => onMarcar(ativa ? null : cor)}
                        aria-label={rotulo}
                        title={rotulo}
                        aria-pressed={ativa}
                        data-testid={`chip-${cor}`}
                        className={`h-5 w-5 rounded-full border-2 ${classesDaCor(cor).chip} ${
                            ativa ? "border-foreground ring-2 ring-foreground/40" : "border-transparent hover:border-foreground/50"
                        } focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary`}
                    />
                );
            })}
            <button
                type="button"
                onClick={() => onMarcar(null)}
                disabled={corAtual === null}
                aria-label={t("workspace.highlight.noColor")}
                title={t("workspace.highlight.noColor")}
                data-testid="chip-sem-cor"
                className="ml-1 rounded p-0.5 hover:bg-muted disabled:opacity-40"
            >
                <X className="h-4 w-4" aria-hidden />
            </button>
            <span className="mx-1 h-4 w-px bg-border" aria-hidden />
            {navegador ? (
                <span className="flex items-center gap-0.5 text-xs" data-testid="navegador-de-conexoes">
                    {navegador.carregando ? (
                        <span className="px-1 text-muted-foreground">{t("workspace.navigator.loading")}</span>
                    ) : navegador.total === 0 ? (
                        <span className="px-1 text-muted-foreground">{t("workspace.navigator.none")}</span>
                    ) : (
                        <>
                            <button
                                type="button"
                                onClick={() => navegador.onIr(atual - 1)}
                                disabled={!podeVoltar}
                                aria-label={t("workspace.navigator.prev")}
                                className="rounded p-0.5 hover:bg-muted disabled:opacity-40"
                            >
                                <ChevronLeft className="h-4 w-4" aria-hidden />
                            </button>
                            <button
                                type="button"
                                onClick={() => navegador.onIr(atual)}
                                className="rounded px-1 tabular-nums text-primary hover:bg-primary/10"
                                title={t("workspace.navigator.goTo")}
                                data-testid="conexao-i-de-n"
                            >
                                {t("workspace.navigator.of", { i: atual + 1, total: navegador.total })}
                            </button>
                            <button
                                type="button"
                                onClick={() => navegador.onIr(atual + 1)}
                                disabled={!podeAvancar}
                                aria-label={t("workspace.navigator.next")}
                                className="rounded p-0.5 hover:bg-muted disabled:opacity-40"
                            >
                                <ChevronRight className="h-4 w-4" aria-hidden />
                            </button>
                        </>
                    )}
                </span>
            ) : (
                <button
                    type="button"
                    onClick={onVerConexoes}
                    disabled={!podeVerConexoes}
                    title={podeVerConexoes ? undefined : t("workspace.sidebar.noSectionTooltip")}
                    className="flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium text-primary hover:bg-primary/10 disabled:cursor-not-allowed disabled:opacity-50"
                >
                    <Link2 className="h-3.5 w-3.5" aria-hidden />
                    {t("workspace.sidebar.viewConnections")}
                </button>
            )}
        </div>
    );
}
