"use client";

import { useRef, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";
import { classesDaCor, CORES, type HighlightColor, type Legend } from "@/lib/workspace/highlights";

/**
 * Legenda das cores no canto do canvas: um chip e um campo de rotulo por cor.
 * O rotulo e salvo ao sair do campo ou com Enter; o painel recolhe e lembra
 * o estado no localStorage (preferencia de tela).
 */

export type LegendPanelProps = {
    legenda: Legend;
    onSalvar: (legenda: Legend) => void;
    /** Contagem de marcas por cor no documento focado, para mostrar ao lado. */
    contagem?: Record<HighlightColor, number> | null;
};

const CHAVE_RECOLHIDO = "thingindex.workspace.legenda.recolhida";

function lerRecolhido(): boolean {
    try {
        return window.localStorage.getItem(CHAVE_RECOLHIDO) === "1";
    } catch {
        return false;
    }
}

export function LegendPanel({ legenda, onSalvar, contagem = null }: LegendPanelProps) {
    const { t } = useI18n();
    const [recolhido, setRecolhido] = useState<boolean>(() => (typeof window === "undefined" ? false : lerRecolhido()));
    const [rascunho, setRascunho] = useState<Legend>(legenda);
    const [base, setBase] = useState<Legend>(legenda);
    // O que ja foi enviado: evita salvar de novo a mesma legenda quando o pai
    // ainda nao devolveu a prop atualizada (salvamento otimista em voo).
    const ultimaEnviada = useRef<Legend | null>(null);

    // Legenda nova vinda de fora (carregou da API) substitui o rascunho, mas
    // sem efeito: comparacao durante o render, padrao "derived state".
    if (base !== legenda) {
        setBase(legenda);
        setRascunho(legenda);
    }

    const alternar = () => {
        const proximo = !recolhido;
        setRecolhido(proximo);
        try {
            window.localStorage.setItem(CHAVE_RECOLHIDO, proximo ? "1" : "0");
        } catch {
            // preferencia so na sessao
        }
    };

    const confirmar = () => {
        const limpa: Legend = {};
        for (const cor of CORES) {
            const valor = rascunho[cor]?.trim();
            if (valor) {
                limpa[cor] = valor;
            }
        }
        const referencia = ultimaEnviada.current ?? legenda;
        const mudou = CORES.some((cor) => (limpa[cor] ?? "") !== (referencia[cor] ?? ""));
        if (mudou) {
            ultimaEnviada.current = limpa;
            onSalvar(limpa);
        }
    };

    return (
        <section
            className="absolute right-3 top-3 w-56 rounded-md border border-border bg-card/95 shadow-sm"
            aria-label={t("workspace.legend.title")}
            data-testid="legenda"
            onPointerDown={(event) => event.stopPropagation()}
        >
            <button
                type="button"
                onClick={alternar}
                className="flex w-full items-center justify-between px-3 py-1.5 text-xs font-semibold"
                aria-expanded={!recolhido}
            >
                {t("workspace.legend.title")}
                {recolhido ? <ChevronDown className="h-4 w-4" aria-hidden /> : <ChevronUp className="h-4 w-4" aria-hidden />}
            </button>
            {recolhido ? null : (
                <ul className="space-y-1 border-t border-border px-2 py-2">
                    {CORES.map((cor) => (
                        <li key={cor} className="flex items-center gap-2">
                            <span className={`h-4 w-4 shrink-0 rounded-full ${classesDaCor(cor).chip}`} aria-hidden />
                            <input
                                type="text"
                                value={rascunho[cor] ?? ""}
                                placeholder={t(`workspace.highlight.colors.${cor}`)}
                                aria-label={t("workspace.legend.labelFor", { color: t(`workspace.highlight.colors.${cor}`) })}
                                maxLength={60}
                                onChange={(event) => setRascunho({ ...rascunho, [cor]: event.target.value })}
                                onBlur={confirmar}
                                onKeyDown={(event) => {
                                    if (event.key === "Enter") {
                                        event.currentTarget.blur();
                                    }
                                }}
                                className="min-w-0 flex-1 rounded border border-border bg-background px-1.5 py-0.5 text-xs"
                                data-testid={`legenda-${cor}`}
                            />
                            {contagem ? (
                                <span className="w-6 text-right text-[10px] tabular-nums text-muted-foreground" data-testid={`contagem-${cor}`}>
                                    {contagem[cor]}
                                </span>
                            ) : null}
                        </li>
                    ))}
                </ul>
            )}
        </section>
    );
}
