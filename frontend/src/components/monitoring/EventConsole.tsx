"use client";

import { useEffect, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Terminal } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { STAGES, GROUP_COLORS, type LogEntry } from "@/hooks/use-pipeline-simulation";

const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } };

function getLogColor(log: LogEntry): string {
    const stage = STAGES.find((s) => s.id === log.stage);
    if (!stage) return "hsl(var(--foreground))";

    if (log.type === "success") return "hsl(150, 60%, 50%)";
    if (log.type === "warning") return "hsl(40, 90%, 55%)";
    if (log.type === "error") return "hsl(0, 70%, 50%)";

    return GROUP_COLORS[stage.group] ?? "hsl(var(--foreground))";
}

export default function EventConsole({ logs, isRealTime }: { logs: LogEntry[]; isRealTime: boolean; }) {
    const logEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        logEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [logs]);

    return (
        <motion.div {...fadeUp} transition={{ delay: 0.3, duration: 0.5 }} className="surface-card flex flex-col rounded-lg">
            <div className="flex items-center gap-2 border-b border-border px-4 py-3">
                <Terminal className="h-4 w-4 text-primary" />
                <h3 className="text-sm font-semibold">Console de Eventos</h3>
                <span className="ml-auto text-[10px] text-muted-foreground">{logs.length} eventos</span>
            </div>
            <ScrollArea className="h-[480px] flex-1">
                <div className="space-y-0.5 p-3 font-mono text-[11px]">
                    {logs.length === 0 && (
                        <p className="py-8 text-center text-muted-foreground">
                            {isRealTime ? "Aguardando eventos via Webhook…" : 'Clique em "Iniciar Simulação" para começar'}
                        </p>
                    )}
                    <AnimatePresence>
                        {logs.map((log) => (
                            <motion.div
                                key={log.id}
                                initial={{ opacity: 0, x: -12, height: 0 }}
                                animate={{ opacity: 1, x: 0, height: "auto" }}
                                transition={{ duration: 0.15 }}
                                className="flex gap-2 rounded px-2 py-1 hover:bg-secondary/50"
                            >
                                <span className="shrink-0 text-muted-foreground">
                                    {log.ts.toLocaleTimeString("pt-BR")}
                                </span>
                                <span style={{ color: getLogColor(log) }}>
                                    [{STAGES.find((s) => s.id === log.stage)?.label ?? log.stage}]
                                </span>
                                <span className="text-muted-foreground">{log.message}</span>
                            </motion.div>
                        ))}
                    </AnimatePresence>
                    <div ref={logEndRef} />
                </div>
            </ScrollArea>
        </motion.div>
    );
}
