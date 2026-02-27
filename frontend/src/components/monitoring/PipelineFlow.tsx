"use client";

import { motion } from "framer-motion";
import { Badge } from "@/components/ui/badge";
import { ArrowDown, Wifi, WifiOff } from "lucide-react";
import {
    STAGES,
    GROUPS,
    GROUP_COLORS,
    type ActiveStages,
    type PipelineStage,
} from "@/hooks/use-pipeline-simulation";

const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } };

type Props = {
    activeStages: ActiveStages;
    simRunning: boolean;
    simDone: boolean;
    isRealTime: boolean;
    onStartSimulation: () => void;
};

function ActiveDot({ color, size = 10 }: { color: string; size?: number; }) {
    return (
        <motion.span
            className="absolute -right-1 -top-1 rounded-full"
            style={{ width: size, height: size, backgroundColor: color }}
            animate={{
                scale: [1, 1.5, 1],
                opacity: [1, 0.6, 1],
            }}
            transition={{ duration: 0.6, repeat: Infinity, ease: "easeInOut" }}
        />
    );
}

function GroupActivityIndicator({ group, activeStages }: { group: string; activeStages: ActiveStages; }) {
    const stages = STAGES.filter((s) => s.group === group);
    const isActive = stages.some((s) => activeStages.has(s.id));
    if (!isActive) return null;

    return (
        <motion.span
            className="ml-2 inline-block h-2 w-2 rounded-full"
            style={{ backgroundColor: GROUP_COLORS[group] }}
            animate={{
                scale: [1, 1.6, 1],
                opacity: [0.7, 1, 0.7],
            }}
            transition={{ duration: 0.5, repeat: Infinity, ease: "easeInOut" }}
        />
    );
}

export default function PipelineFlow({ activeStages, simRunning, simDone, isRealTime, onStartSimulation }: Props) {
    return (
        <motion.div {...fadeUp} transition={{ delay: 0.2, duration: 0.5 }} className="surface-card rounded-lg p-6">
            <div className="mb-5 flex items-center justify-between">
                <h2 className="text-sm font-semibold">Pipeline de Dados</h2>
                {!isRealTime && (
                    <button
                        onClick={onStartSimulation}
                        disabled={simRunning}
                        className="rounded-md bg-primary px-4 py-1.5 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
                    >
                        {simRunning ? "Executando…" : "Iniciar Simulação"}
                    </button>
                )}
                {isRealTime && (
                    <Badge variant="outline" className="border-primary/40 text-primary">
                        <span className="mr-1.5 inline-block h-2 w-2 animate-pulse rounded-full bg-primary" />
                        Aguardando Webhook
                    </Badge>
                )}
            </div>

            <div className="space-y-4">
                {GROUPS.map((group) => {
                    const stages = STAGES.filter((s) => s.group === group);
                    const groupColor = GROUP_COLORS[group];
                    return (
                        <div key={group} className="rounded-lg border border-border bg-background/50 p-4">
                            <p className="mb-4 flex items-center text-xs font-semibold uppercase tracking-wider" style={{ color: groupColor }}>
                                {group}
                                <GroupActivityIndicator group={group} activeStages={activeStages} />
                            </p>
                            <div className="flex flex-wrap items-center gap-3">
                                {stages.map((stage, idx) => {
                                    const isActive = activeStages.has(stage.id);
                                    return (
                                        <div key={stage.id} className="flex items-center gap-3">
                                            <motion.div
                                                className="relative rounded-md border px-4 py-2 text-xs font-medium transition-colors"
                                                style={{
                                                    borderColor: isActive ? groupColor : "hsl(var(--border))",
                                                    background: isActive ? `${groupColor}20` : "transparent",
                                                    color: isActive ? groupColor : "hsl(var(--muted-foreground))",
                                                }}
                                                animate={
                                                    isActive
                                                        ? {
                                                            boxShadow: [
                                                                `0 0 0px ${groupColor}00`,
                                                                `0 0 24px ${groupColor}50`,
                                                                `0 0 0px ${groupColor}00`,
                                                            ],
                                                        }
                                                        : { boxShadow: "0 0 0px transparent" }
                                                }
                                                transition={isActive ? { duration: 0.8, repeat: Infinity } : { duration: 0.3 }}
                                            >
                                                {isActive && <ActiveDot color={groupColor} />}
                                                {stage.label}
                                            </motion.div>
                                            {idx < stages.length - 1 && (
                                                <ArrowDown className="h-4 w-4 rotate-[-90deg] text-muted-foreground/50" />
                                            )}
                                        </div>
                                    );
                                })}
                            </div>
                        </div>
                    );
                })}

                {/* Main connector at the bottom indicating flow between workers */}
                <div className="flex items-center gap-2 py-2 pl-4">
                    <ArrowDown className="h-4 w-4 text-muted-foreground/50" />
                    <span className="text-[10px] text-muted-foreground">fluxo entre workers</span>
                </div>
            </div>

            {simDone && (
                <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="mt-4 rounded-md border border-[hsl(150,60%,50%)]/30 bg-[hsl(150,60%,50%)]/5 p-3 text-xs text-[hsl(150,60%,50%)]"
                >
                    ✓ Pipeline concluído com sucesso — 243 nós criados
                </motion.div>
            )}
        </motion.div>
    );
}
