"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import Header from "@/components/Header";
import { Switch } from "@/components/ui/switch";
import { Wifi, WifiOff } from "lucide-react";
import { usePipelineSimulation } from "@/hooks/use-pipeline-simulation";
import MetricCards from "@/components/monitoring/MetricCards";
import PipelineFlow from "@/components/monitoring/PipelineFlow";
import EventConsole from "@/components/monitoring/EventConsole";

const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } };

export default function MonitoringPage() {
    const [isRealTime, setIsRealTime] = useState(false);
    const { logs, metrics, activeStages, simRunning, simDone, startSimulation } = usePipelineSimulation();

    return (
        <div className="min-h-screen bg-background">
            <Header />
            <main className="container py-8">
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

                    {/* Toggle */}
                    <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-2.5">
                        <span className={`flex items-center gap-1.5 text-sm font-medium ${!isRealTime ? "text-primary" : "text-muted-foreground"}`}>
                            <WifiOff className="h-3.5 w-3.5" /> Simulação
                        </span>
                        <Switch checked={isRealTime} onCheckedChange={setIsRealTime} />
                        <span className={`flex items-center gap-1.5 text-sm font-medium ${isRealTime ? "text-primary" : "text-muted-foreground"}`}>
                            <Wifi className="h-3.5 w-3.5" /> Tempo Real
                        </span>
                    </div>
                </motion.div>

                {/* Metrics */}
                <MetricCards metrics={metrics} />

                {/* Pipeline + Console */}
                <div className="mt-6 grid gap-6 lg:grid-cols-[1fr_380px]">
                    <PipelineFlow
                        activeStages={activeStages}
                        simRunning={simRunning}
                        simDone={simDone}
                        isRealTime={isRealTime}
                        onStartSimulation={startSimulation}
                    />
                    <EventConsole logs={logs} isRealTime={isRealTime} />
                </div>
            </main>
        </div>
    );
}
