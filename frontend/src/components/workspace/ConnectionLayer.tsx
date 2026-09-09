"use client";

import { useI18n } from "@/lib/i18n/provider";
import { caminhoDaLinha, espessuraPorScore, type Linha } from "@/lib/workspace/connections";

/**
 * Linhas de conexao em SVG, em coordenadas do mundo (o <svg> vive dentro do
 * conteiner que recebe o transform do viewport, entao pan e zoom vem de
 * graca). Nada aqui e persistido: as linhas existem enquanto a resposta da
 * API de conexoes estiver em memoria.
 */

export type ConnectionLayerProps = {
    linhas: Linha[];
    realcada: string | null;
    onRealcar: (id: string | null) => void;
    onEscolher: (linha: Linha) => void;
};

export function ConnectionLayer({ linhas, realcada, onRealcar, onEscolher }: ConnectionLayerProps) {
    const { t } = useI18n();
    if (linhas.length === 0) {
        return null;
    }
    return (
        <svg
            className="pointer-events-none absolute left-0 top-0 overflow-visible"
            width={1}
            height={1}
            data-testid="camada-de-conexoes"
            aria-label={t("workspace.connections.label")}
            role="img"
        >
            {linhas.map((linha) => {
                const ativa = realcada === null || realcada === linha.id;
                // O rotulo fica na PONTA de destino, nao no meio da curva: no
                // meio ele flutuava no vazio e nao dizia a que linha pertencia.
                // Ao passar o mouse, mostra os termos que justificam a aresta.
                // Aresta "mesma norma" diz isso no rotulo: os dois trechos citam a mesma lei ou resolucao.
                const contagem = linha.tipo === "mesma-norma"
                    ? t("workspace.sidebar.sameNorm")
                    : linha.score === 1 ? t("workspace.sidebar.term") : t("workspace.sidebar.terms", { count: linha.score });
                const rotulo = realcada === linha.id && linha.termos.length > 0 ? linha.termos.join(", ") : contagem;
                const larguraDoRotulo = Math.min(rotulo.length * 6.4 + 12, 320);
                const ancora = { x: linha.para.x + 8, y: linha.para.y - 22 };
                return (
                    <g
                        key={linha.id}
                        className="pointer-events-auto cursor-pointer"
                        data-testid="linha-de-conexao"
                        data-score={linha.score}
                        opacity={ativa ? 1 : 0.25}
                        onMouseEnter={() => onRealcar(linha.id)}
                        onMouseLeave={() => onRealcar(null)}
                        onClick={() => onEscolher(linha)}
                    >
                        <title>
                            {linha.titulo}: {linha.termos.join(", ")}
                        </title>
                        <path
                            d={caminhoDaLinha(linha.de, linha.para)}
                            fill="none"
                            stroke="hsl(var(--primary))"
                            strokeWidth={espessuraPorScore(linha.score)}
                            strokeDasharray={linha.aproximada ? "6 4" : undefined}
                            strokeLinecap="round"
                        />
                        <circle cx={linha.para.x} cy={linha.para.y} r={3} fill="hsl(var(--primary))" />
                        <rect
                            x={ancora.x}
                            y={ancora.y}
                            width={larguraDoRotulo}
                            height={18}
                            rx={4}
                            fill="hsl(var(--card))"
                            stroke="hsl(var(--primary))"
                            strokeWidth={realcada === linha.id ? 1.5 : 1}
                        />
                        <text
                            x={ancora.x + 6}
                            y={ancora.y + 13}
                            fontSize={11}
                            fill="hsl(var(--primary))"
                            data-testid="rotulo-da-linha"
                        >
                            {rotulo}
                        </text>
                    </g>
                );
            })}
        </svg>
    );
}
