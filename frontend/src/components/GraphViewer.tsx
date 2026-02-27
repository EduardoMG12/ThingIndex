"use client";

import { useRef, useEffect, useState, useCallback, useMemo } from "react";
import dynamic from "next/dynamic";
import { GitBranch, ZoomIn, ZoomOut, Maximize2, Filter } from "lucide-react";
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
    x?: number;
    y?: number;
    val?: number;
};

type GraphLink = {
    source: string;
    target: string;
    label?: string;
};

type Props = {
    nodes: GraphNode[];
    links: GraphLink[];
    totalNodes?: number;
    currentContext?: string;
    onContextChange?: (context: string) => void;
    availableTopics?: string[];
};

const COLORS = [
    "hsl(40, 90%, 55%)",  // 0: Origem (Gold)
    "hsl(200, 70%, 55%)", // 1: Topic (Blue)
    "hsl(280, 60%, 55%)", // 2: Document (Purple)
    "hsl(150, 60%, 50%)", // 3: Artigo (Green)
    "hsl(320, 60%, 55%)", // 4: Paragrafo (Pink)
    "hsl(15, 80%, 55%)",  // 5: Inciso (Orange)
    "hsl(180, 70%, 45%)", // 6: Alinea (Teal)
    "hsl(0, 0%, 60%)",    // 7: Others
];

export default function GraphViewer({
    nodes,
    links,
    totalNodes = 0,
    currentContext = "all",
    onContextChange,
    availableTopics = []
}: Props) {
    const fgRef = useRef<any>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const [dimensions, setDimensions] = useState({ width: 600, height: 400 });

    // 🔄 RE-MOUNT STRATEGY: 
    // We force a complete component recreation on navigation or meaningful data changes.
    // This solves the 'clumped nodes' and 'frozen simulation' bugs by discarding 
    // the old engine state entirely.
    const graphKey = useMemo(() =>
        `graph-${nodes.length}-${links.length}-${currentContext}`,
        [nodes.length, links.length, currentContext]
    );

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
        nodes: nodes.map((n) => {
            // Define absolute values for our manual radius calculation
            let val = 3;
            if (n.group === 0) val = 30;
            if (n.group === 1) val = 18;
            if (n.group === 2) val = 10;
            return { ...n, name: n.label, val };
        }),
        links: links.map((l) => ({ ...l })),
    }), [nodes, links]);

    // ⚡ PHYSICS INITIALIZATION: 
    // This runs once per remount. We use imperative calls to D3 for maximum stability.
    useEffect(() => {
        const fg = fgRef.current;
        if (fg) {
            // 1. Setup Forces: Stronger repulsion to prevent 'too close' nodes
            fg.d3Force('charge').strength(-4500);
            fg.d3Force('link').distance(400);

            // Add Collision force to prevent any node from overlapping
            // Radius matches the manual scale of 1.8 (radius ~ 25px max)
            fg.d3Force('collide', (window as any).d3?.forceCollide(25));

            // 2. High energy start (Warm-up phase)
            // We use the underlying d3 object if available for deep control
            const d3Sim = fg.d3AlphaTarget ? fg : null; // Guard against SSR/Ref timing

            if (fg.d3AlphaTarget) fg.d3AlphaTarget(0.4);
            fg.d3ReheatSimulation();

            // 3. Gradual settle and Fit-to-view (Tight zoom)
            const settleTimer = setTimeout(() => {
                if (fgRef.current) {
                    if (fgRef.current.d3AlphaTarget) fgRef.current.d3AlphaTarget(0);
                    // Minimal padding (20) for maximum zoom depth
                    fgRef.current.zoomToFit(1000, 20);
                }
            }, 1200);

            // 4. Emergency initial fit
            const initialFit = setTimeout(() => {
                if (fgRef.current) fgRef.current.zoomToFit(200, 100);
            }, 250);

            return () => {
                clearTimeout(settleTimer);
                clearTimeout(initialFit);
            };
        }
    }, [graphKey]);

    return (
        <div className="surface-card rounded-lg p-5 flex flex-col h-full">
            <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <GitBranch className="h-4 w-4 text-primary" aria-hidden="true" />
                    <h3 className="text-sm font-semibold">Banco de Grafos</h3>

                    {availableTopics.length > 0 && onContextChange && (
                        <div className="ml-4 flex items-center gap-2 border-l border-border pl-4">
                            <Filter className="h-3.5 w-3.5 text-muted-foreground" />
                            <select
                                className="h-7 cursor-pointer rounded-md border border-input bg-background px-2 py-0 text-xs font-medium focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                                value={currentContext}
                                onChange={(e) => onContextChange(e.target.value)}
                                aria-label="Filtrar contexto do grafo"
                            >
                                <option value="all">Visão Geral Completa</option>
                                {availableTopics.map(topic => (
                                    <option key={topic} value={topic}>{topic}</option>
                                ))}
                            </select>
                        </div>
                    )}
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
                        key={graphKey}
                        ref={fgRef}
                        graphData={graphData}
                        width={dimensions.width}
                        height={dimensions.height}
                        backgroundColor="transparent"
                        d3VelocityDecay={0.4}
                        nodeRelSize={1}
                        nodeColor={(node: any) => COLORS[node.group % COLORS.length]}
                        nodeLabel={(node: any) => node.label}
                        linkColor={() => "rgba(255, 255, 255, 0.1)"}
                        linkWidth={0.8}
                        nodeCanvasObjectMode={() => "replace"}
                        nodeCanvasObject={(node: any, ctx, globalScale) => {
                            const label = node.label;
                            const isCore = node.group === 0 || node.group === 1;

                            // 📐 ABSOLUTE SIZING: Scale adjusted to 1.8 equivalent
                            const val = node.val || 1;
                            const radius = Math.sqrt(val) * 3.0;
                            const color = COLORS[node.group % COLORS.length];

                            // 1. Draw Node Circle
                            ctx.beginPath();
                            ctx.arc(node.x, node.y, radius, 0, 2 * Math.PI, false);
                            ctx.fillStyle = color;
                            ctx.fill();

                            // 2. Glow effect for important nodes
                            if (isCore || node.group === 2) {
                                ctx.save();
                                ctx.beginPath();
                                ctx.arc(node.x, node.y, radius * 1.6, 0, 2 * Math.PI, false);
                                ctx.shadowBlur = isCore ? 15 : 8;
                                ctx.shadowColor = color;
                                ctx.fillStyle = "rgba(0, 0, 0, 0)";
                                ctx.fill();
                                ctx.restore();
                            }

                            // 3. Render Labels
                            if (globalScale >= 3 || (isCore && globalScale >= 0.2)) {
                                const fontSize = isCore ? (13 / globalScale) : (10 / globalScale);
                                ctx.font = `${isCore ? '600' : '400'} ${fontSize}px "Inter", system-ui, sans-serif`;
                                ctx.textAlign = "center";
                                ctx.textBaseline = "top";

                                // High contrast halo
                                ctx.strokeStyle = "rgba(0,0,0,1)";
                                ctx.lineWidth = 3 / globalScale;
                                ctx.lineJoin = "round";

                                const yOffset = radius + (4 / globalScale);

                                ctx.strokeText(label, node.x, node.y + yOffset);
                                ctx.fillStyle = isCore ? color : "rgba(255,255,255,0.98)";
                                ctx.fillText(label, node.x, node.y + yOffset);
                            }
                        }}
                    />
                )}
            </div>

            <div className="mt-3 flex items-center justify-between">
                <span className="text-[11px] text-muted-foreground">
                    Exibindo {nodes.length} de {totalNodes || nodes.length} nós estruturados · {links.length} conexões
                </span>
                <div className="flex flex-wrap gap-x-3 gap-y-1 justify-end max-w-[80%]">
                    {[
                        { name: "Brasil", group: 0 },
                        { name: "Tópico", group: 1 },
                        { name: "Doc", group: 2 },
                        { name: "Art.", group: 3 },
                        { name: "§", group: 4 },
                        { name: "Inc.", group: 5 },
                        { name: "Al.", group: 6 }
                    ].map((item) => (
                        <span key={item.name} className="flex items-center gap-1.5 text-[10px] text-muted-foreground whitespace-nowrap">
                            <span
                                className="inline-block h-1.5 w-1.5 rounded-full"
                                style={{ backgroundColor: COLORS[item.group] }}
                            />
                            {item.name}
                        </span>
                    ))}
                </div>
            </div>
        </div>
    );
}
