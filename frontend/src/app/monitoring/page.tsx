"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import Header from "@/components/Header";
import { Switch } from "@/components/ui/switch";
import { Wifi, WifiOff, LayoutDashboard, Network } from "lucide-react";
import { usePipelineSimulation } from "@/hooks/use-pipeline-simulation";
import { useTelemetryStream } from "@/hooks/use-telemetry-stream";
import MetricCards from "@/components/monitoring/MetricCards";
import PipelineFlow from "@/components/monitoring/PipelineFlow";
import EventConsole from "@/components/monitoring/EventConsole";
import ArchitectureView from "@/components/monitoring/ArchitectureView";
import { cn } from "@/lib/utils";

const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } };

type ViewTab = "pipeline" | "architecture";

const STATUS_LABELS: Record<string, { text: string; color: string; }> = {
    connected: { text: "Conectado", color: "text-green-400" },
    connecting: { text: "Conectando…", color: "text-yellow-400" },
    disconnected: { text: "Desconectado", color: "text-muted-foreground" },
};

const TABS: { id: ViewTab; label: string; icon: typeof LayoutDashboard; }[] = [
    { id: "pipeline", label: "Pipeline", icon: LayoutDashboard },
    { id: "architecture", label: "Arquitetura", icon: Network },
];

export default function MonitoringPage() {
    const [isRealTime, setIsRealTime] = useState(false);
    const [activeTab, setActiveTab] = useState<ViewTab>("pipeline");

    // Simulation hook (always ready, only active when toggled)
    const simulation = usePipelineSimulation();

    // Real-time WebSocket hook (connects only when enabled)
    const realTime = useTelemetryStream(isRealTime);

    // Pick the active data source based on toggle
    const logs = isRealTime ? realTime.logs : simulation.logs;
    const metrics = isRealTime ? realTime.metrics : simulation.metrics;
    const activeStages = isRealTime ? realTime.activeStages : simulation.activeStages;

    const statusInfo = STATUS_LABELS[realTime.status] ?? STATUS_LABELS.disconnected;

    return (
        <div className="min-h-screen bg-background">
            <Header />
            <main id="main-content" className="container py-8">
                {/* Title row */}
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="flex flex-wrap items-center justify-between gap-4">
                    <div>
                        <h1 className="text-3xl font-bold">
                            <span className="text-gradient-gold">Monitoramento</span>
                        </h1>
                        <p className="mt-2 text-sm text-muted-foreground">
                            Dashboard de observabilidade do pipeline Crawler → Parser → Neo4j
                        </p>
                    </div>

                    {/* Toggle + Status */}
                    <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-2.5" role="group" aria-label="Alternar modo de monitoramento">
                        <span className={`flex items-center gap-1.5 text-sm font-medium ${!isRealTime ? "text-primary" : "text-muted-foreground"}`}>
                            <WifiOff className="h-3.5 w-3.5" aria-hidden="true" /> Simulação
                        </span>
                        <Switch
                            checked={isRealTime}
                            onCheckedChange={setIsRealTime}
                            aria-label="Alternar entre simulação e tempo real"
                        />
                        <span className={`flex items-center gap-1.5 text-sm font-medium ${isRealTime ? "text-primary" : "text-muted-foreground"}`}>
                            <Wifi className="h-3.5 w-3.5" aria-hidden="true" /> Tempo Real
                        </span>
                        {isRealTime && (
                            <span className={`ml-2 flex items-center gap-1.5 text-xs ${statusInfo.color}`}>
                                <span className={`inline-block h-1.5 w-1.5 rounded-full ${realTime.status === "connected" ? "bg-green-400 animate-pulse" : realTime.status === "connecting" ? "bg-yellow-400 animate-pulse" : "bg-muted-foreground"}`} />
                                {statusInfo.text}
                            </span>
                        )}
                    </div>
                </motion.div>

                {/* Metrics */}
                <div aria-live="polite" aria-atomic="true">
                    <MetricCards metrics={metrics} activeStages={activeStages} />
                </div>

                {/* Tab switcher */}
                <div className="mt-6 flex gap-1 rounded-lg border border-border bg-card p-1 w-fit" role="tablist" aria-label="Visualização do monitoramento">
                    {TABS.map(({ id, label, icon: Icon }) => (
                        <button
                            key={id}
                            role="tab"
                            aria-selected={activeTab === id}
                            aria-controls={`panel-${id}`}
                            onClick={() => setActiveTab(id)}
                            className={cn(
                                "relative flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors",
                                activeTab === id
                                    ? "text-primary"
                                    : "text-muted-foreground hover:text-foreground"
                            )}
                        >
                            {activeTab === id && (
                                <motion.div
                                    layoutId="tab-bg"
                                    className="absolute inset-0 rounded-md bg-secondary"
                                    transition={{ type: "spring", stiffness: 400, damping: 30 }}
                                />
                            )}
                            <span className="relative flex items-center gap-2">
                                <Icon className="h-4 w-4" aria-hidden="true" />
                                {label}
                            </span>
                        </button>
                    ))}
                </div>

                {/* Tab content */}
                <div className="mt-4">
                    {activeTab === "pipeline" && (
                        <div id="panel-pipeline" role="tabpanel" aria-label="Pipeline de dados" className="grid gap-6 lg:grid-cols-[1fr_380px]">
                            <PipelineFlow
                                activeStages={activeStages}
                                simRunning={isRealTime ? realTime.status === "connected" : simulation.simRunning}
                                simDone={isRealTime ? false : simulation.simDone}
                                isRealTime={isRealTime}
                                onStartSimulation={simulation.startSimulation}
                            />
                            <EventConsole logs={logs} isRealTime={isRealTime} />
                        </div>
                    )}

                    {activeTab === "architecture" && (
                        <div id="panel-architecture" role="tabpanel" aria-label="Arquitetura do sistema" className="grid gap-6 lg:grid-cols-[1fr_380px]">
                            <ArchitectureView
                                activeStages={activeStages}
                                simRunning={isRealTime ? realTime.status === "connected" : simulation.simRunning}
                                isRealTime={isRealTime}
                                onStartSimulation={simulation.startSimulation}
                            />
                            <EventConsole logs={logs} isRealTime={isRealTime} />
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
