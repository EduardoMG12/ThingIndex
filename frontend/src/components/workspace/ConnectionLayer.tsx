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
                const meio = { x: (linha.de.x + linha.para.x) / 2, y: (linha.de.y + linha.para.y) / 2 };
                const rotulo = linha.score === 1 ? t("workspace.sidebar.term") : t("workspace.sidebar.terms", { count: linha.score });
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
                            x={meio.x - rotulo.length * 3.4 - 4}
                            y={meio.y - 9}
                            width={rotulo.length * 6.8 + 8}
                            height={18}
                            rx={4}
                            fill="hsl(var(--card))"
                            stroke="hsl(var(--primary))"
                            strokeWidth={1}
                        />
                        <text
                            x={meio.x}
                            y={meio.y + 4}
                            textAnchor="middle"
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
