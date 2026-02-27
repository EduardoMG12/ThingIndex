"use client";

import { useRef, useEffect, useState, useCallback, useMemo } from "react";
import dynamic from "next/dynamic";
import { GitBranch, ZoomIn, ZoomOut, Maximize2 } from "lucide-react";
import { Button } from "@/components/ui/Button";

// Dynamic import for client-side only rendering of ForceGraph2D
const ForceGraph2D = dynamic(() => import("react-force-graph-2d"), {
    ssr: false,
    loading: () => <div className="h-[350px] flex items-center justify-center">Carregando visualização...</div>
});

type GraphNode = {
    id: string;
    label: string;
    group: number;
};

type GraphLink = {
    source: string;
    target: string;
    label?: string;
};

type Props = {
    nodes: GraphNode[];
    links: GraphLink[];
};

const COLORS = [
    "hsl(40, 90%, 55%)",
    "hsl(200, 70%, 55%)",
    "hsl(150, 60%, 50%)",
    "hsl(320, 60%, 55%)",
    "hsl(15, 80%, 55%)",
];

export default function GraphViewer({ nodes, links }: Props) {
    const fgRef = useRef<any>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 600, height: 400 });

    const updateDimensions = useCallback(() => {
        if (containerRef.current) {
            const rect = containerRef.current.getBoundingClientRect();
            setDimensions({ width: rect.width, height: Math.max(rect.height, 350) });
        }
    }, []);

    useEffect(() => {
        updateDimensions();
        window.addEventListener("resize", updateDimensions);
        return () => window.removeEventListener("resize", updateDimensions);
    }, [updateDimensions]);

    const graphData = useMemo(() => ({
        nodes: nodes.map((n) => ({ ...n, name: n.label })),
        links: links.map((l) => ({ ...l })),
    }), [nodes, links]);

    return (
        <div className="surface-card rounded-lg p-5 flex flex-col h-full">
            <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <GitBranch className="h-4 w-4 text-primary" aria-hidden="true" />
                    <h3 className="text-sm font-semibold">Banco de Grafos</h3>
                </div>
                <div className="flex gap-1">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => fgRef.current?.zoomToFit(400)}
                        aria-label="Ajustar zoom ao conteúdo"
                    >
                        <Maximize2 className="h-4 w-4" aria-hidden="true" />
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => {
                            const z = fgRef.current?.zoom();
                            fgRef.current?.zoom(z * 1.3, 300);
                        }}
                        aria-label="Aumentar zoom"
                    >
                        <ZoomIn className="h-4 w-4" aria-hidden="true" />
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => {
                            const z = fgRef.current?.zoom();
                            fgRef.current?.zoom(z * 0.7, 300);
                        }}
                        aria-label="Diminuir zoom"
                    >
                        <ZoomOut className="h-4 w-4" aria-hidden="true" />
                    </Button>
                </div>
            </div>

            <div
                ref={containerRef}
                className="graph-container relative flex-1 min-h-[350px] overflow-hidden rounded-lg border border-border bg-[#0a0c10]"
                role="img"
                aria-label={`Grafo interativo com ${nodes.length} nós e ${links.length} conexões`}
            >
                {nodes.length === 0 ? (
                    <div className="flex h-full items-center justify-center">
                        <p className="text-sm text-muted-foreground">
                            Nenhum grafo disponível. Envie um prompt para gerar.
                        </p>
                    </div>
                ) : (
                    <ForceGraph2D
                        ref={fgRef}
                        graphData={graphData}
                        width={dimensions.width}
                        height={dimensions.height}
                        backgroundColor="transparent"
                        nodeRelSize={6}
                        nodeColor={(node: any) => COLORS[node.group % COLORS.length]}
                        nodeLabel={(node: any) => node.label}
                        linkColor={() => "hsl(220, 15%, 25%)"}
                        linkWidth={1.5}
                        nodeCanvasObjectMode={() => "after"}
                        nodeCanvasObject={(node: any, ctx, globalScale) => {
                            const label = node.label;
                            const fontSize = 11 / globalScale;
                            ctx.font = `${fontSize}px Space Grotesk`;
                            ctx.textAlign = "center";
                            ctx.textBaseline = "middle";
                            ctx.fillStyle = "hsl(45, 20%, 90%)";
                            ctx.fillText(label, node.x, node.y + 10);
                        }}
                    />
                )}
            </div>

            <div className="mt-3 flex items-center justify-between">
                <span className="text-[11px] text-muted-foreground">
                    {nodes.length} nós · {links.length} conexões
                </span>
                <div className="flex gap-3">
                    {["Documento", "Seção", "Artigo", "Conceito"].map((label, i) => (
                        <span key={label} className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                            <span
                                className="inline-block h-2 w-2 rounded-full"
                                style={{ backgroundColor: COLORS[i] }}
                            />
                            {label}
                        </span>
                    ))}
                </div>
            </div>
        </div>
    );
}
