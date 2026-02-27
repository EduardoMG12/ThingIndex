"use client";

import { motion } from "framer-motion";
import type { ActiveStages } from "@/hooks/use-pipeline-simulation";
import {
    ARCH_NODES,
    ARCH_EDGES,
    SVG_W,
    SVG_H,
    CHILD_BOX_H,
    edgeEndpoints,
    type ArchNode,
    type ArchEdge,
} from "./architecture-data";

/* ── helpers ── */
const nodeMap = Object.fromEntries(ARCH_NODES.map((n) => [n.id, n]));

function isNodeActive(node: ArchNode, stages: ActiveStages) {
    return node.stages?.some((s) => stages.has(s)) ?? false;
}

function isChildActive(stage: string | undefined, stages: ActiveStages) {
    return stage ? stages.has(stage) : false;
}

function isEdgeActive(edge: ArchEdge, stages: ActiveStages) {
    const from = nodeMap[edge.from];
    const to = nodeMap[edge.to];
    return (from && isNodeActive(from, stages)) || (to && isNodeActive(to, stages));
}

/* ── sub-components ── */

function SvgArrowDefs({ edges, stages }: { edges: ArchEdge[]; stages: ActiveStages; }) {
    return (
        <defs>
            <marker id="arrow-muted" viewBox="0 0 10 8" refX="9" refY="4" markerWidth="8" markerHeight="6" orient="auto-start-reverse">
                <path d="M0 0 L10 4 L0 8z" fill="hsl(var(--muted-foreground))" opacity="0.4" />
            </marker>
            {edges.map((e, i) => (
                <marker key={i} id={`arrow-${i}`} viewBox="0 0 10 8" refX="9" refY="4" markerWidth="8" markerHeight="6" orient="auto-start-reverse">
                    <path d="M0 0 L10 4 L0 8z" fill={e.color} />
                </marker>
            ))}
        </defs>
    );
}

function ArchEdgeEl({ edge, index, stages }: { edge: ArchEdge; index: number; stages: ActiveStages; }) {
    const from = nodeMap[edge.from];
    const to = nodeMap[edge.to];
    if (!from || !to) return null;

    let { x1, y1, x2, y2 } = edgeEndpoints(from, to);
    x1 += edge.offsetX || 0;
    x2 += edge.offsetX || 0;
    y1 += edge.offsetY || 0;
    y2 += edge.offsetY || 0;

    const active = isEdgeActive(edge, stages);
    const mx = (x1 + x2) / 2;
    const my = (y1 + y2) / 2;

    return (
        <g>
            <line
                x1={x1} y1={y1} x2={x2} y2={y2}
                stroke={active ? edge.color : "hsl(var(--border))"}
                strokeWidth={active ? 2 : 1}
                strokeDasharray={active ? undefined : "4 4"}
                opacity={active ? 1 : 0.4}
                markerEnd={active ? `url(#arrow-${index})` : "url(#arrow-muted)"}
            />
            {active && (
                <circle r="4" fill={edge.color}>
                    <animateMotion dur="1s" repeatCount="indefinite" path={`M${x1},${y1} L${x2},${y2}`} />
                </circle>
            )}
            {edge.label && (
                <text
                    x={mx + (edge.textOffsetX || 0)}
                    y={my - 6 + (edge.textOffsetY || 0)}
                    textAnchor="middle"
                    fill={active ? edge.color : "hsl(var(--muted-foreground))"}
                    fontSize="9"
                    fontWeight={active ? 600 : 400}
                    opacity={active ? 1 : 0.6}
                >
                    {edge.label}
                </text>
            )}
        </g>
    );
}

function ArchNodeEl({ node, stages }: { node: ArchNode; stages: ActiveStages; }) {
    const active = isNodeActive(node, stages);

    return (
        <g>
            {/* Box */}
            <rect
                x={node.x} y={node.y}
                width={node.w} height={node.h} rx={8}
                fill={active ? `${node.color}15` : "hsl(var(--card))"}
                stroke={active ? node.color : "hsl(var(--border))"}
                strokeWidth={active ? 2 : 1}
            />
            {active && (
                <rect
                    x={node.x} y={node.y}
                    width={node.w} height={node.h} rx={8}
                    fill="none" stroke={node.color} strokeWidth={2} opacity={0.6}
                >
                    <animate attributeName="opacity" values="0.6;0.15;0.6" dur="1s" repeatCount="indefinite" />
                </rect>
            )}

            {/* Title */}
            <text
                x={node.x + 10}
                y={node.y + (node.children ? 18 : node.h / 2 + 4)}
                fill={active ? node.color : "hsl(var(--foreground))"}
                fontSize="11" fontWeight="700"
            >
                {node.label}
                {active && (
                    <tspan>
                        <animate attributeName="opacity" values="1;0.4;1" dur="0.6s" repeatCount="indefinite" />
                        {" ●"}
                    </tspan>
                )}
            </text>

            {/* Children (internal stages) */}
            {node.children?.map((child, ci) => {
                const cActive = isChildActive(child.stage, stages);
                const cy = node.y + 32 + ci * (CHILD_BOX_H + 2);
                return (
                    <g key={ci}>
                        <rect
                            x={node.x + 8} y={cy}
                            width={node.w - 16} height={CHILD_BOX_H - 4} rx={4}
                            fill={cActive ? `${node.color}25` : "hsl(var(--muted))"}
                            stroke={cActive ? node.color : "transparent"}
                            strokeWidth={1}
                        />
                        <text
                            x={node.x + 16}
                            y={cy + (CHILD_BOX_H - 4) / 2 + 4}
                            fill={cActive ? node.color : "hsl(var(--muted-foreground))"}
                            fontSize="9.5"
                            fontWeight={cActive ? 600 : 400}
                        >
                            {child.label}
                        </text>
                    </g>
                );
            })}
        </g>
    );
}

/* ── main component ── */

type Props = {
    activeStages: ActiveStages;
    simRunning: boolean;
    isRealTime: boolean;
    onStartSimulation: () => void;
};

export default function ArchitectureView({ activeStages, simRunning, isRealTime, onStartSimulation }: Props) {
    return (
        <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
            className="surface-card overflow-hidden rounded-lg"
            role="region"
            aria-label="Diagrama de arquitetura do sistema"
        >
            <div className="flex items-center justify-between border-b border-border px-5 py-3">
                <div>
                    <h2 className="text-sm font-semibold">Arquitetura do Sistema</h2>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                        Visão completa dos microsserviços, filas e storages
                    </p>
                </div>
                {!isRealTime && (
                    <button
                        onClick={onStartSimulation}
                        disabled={simRunning}
                        className="rounded-md bg-primary px-4 py-1.5 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
                    >
                        {simRunning ? "Executando…" : "Iniciar Simulação"}
                    </button>
                )}
            </div>

            <div className="overflow-x-auto p-4">
                <svg
                    viewBox={`0 0 ${SVG_W} ${SVG_H}`}
                    className="mx-auto w-full max-w-[740px]"
                    style={{ minWidth: 500 }}
                    role="img"
                    aria-label="Diagrama SVG de arquitetura com microsserviços conectados por filas RabbitMQ"
                >
                    <SvgArrowDefs edges={ARCH_EDGES} stages={activeStages} />

                    {/* Edges first (behind nodes) */}
                    {ARCH_EDGES.map((edge, i) => (
                        <ArchEdgeEl key={i} edge={edge} index={i} stages={activeStages} />
                    ))}

                    {/* Nodes on top */}
                    {ARCH_NODES.map((node) => (
                        <ArchNodeEl key={node.id} node={node} stages={activeStages} />
                    ))}
                </svg>
            </div>
        </motion.div>
    );
}
