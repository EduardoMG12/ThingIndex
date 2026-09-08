"use client";

import { Link2, X } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";
import { classesDaCor, CORES, rotuloDaCor, type HighlightColor, type Legend } from "@/lib/workspace/highlights";

/**
 * Barra flutuante sobre o paragrafo selecionado: cinco chips de cor, "sem cor"
 * e "Ver conexoes". Vive dentro da pagina, acima do paragrafo (abaixo quando
 * nao cabe), para a acao acontecer em cima do PDF e nao na sidebar.
 */

export type HighlightToolbarProps = {
    /** Rect do paragrafo em px da pagina. */
    rect: { left: number; top: number; width: number; height: number };
    corAtual: HighlightColor | null;
    legenda: Legend;
    onMarcar: (cor: HighlightColor | null) => void;
    onVerConexoes: () => void;
    podeVerConexoes: boolean;
};

const ALTURA_DA_BARRA = 32;

export function HighlightToolbar({ rect, corAtual, legenda, onMarcar, onVerConexoes, podeVerConexoes }: HighlightToolbarProps) {
    const { t } = useI18n();
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
        </div>
    );
}
