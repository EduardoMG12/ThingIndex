"use client";

import { motion } from "framer-motion";
import { Card, CardContent } from "@/components/ui/Card";
import { Activity, Database, HardDrive, Layers } from "lucide-react";
import type { Metrics } from "@/hooks/use-pipeline-simulation";

const fadeUp = { initial: { opacity: 0, y: 20 }, animate: { opacity: 1, y: 0 } };

function AnimatedCounter({ value }: { value: number; }) {
    return (
        <motion.span
            key={value}
            initial={{ scale: 1.4, color: "hsl(40, 90%, 55%)" }}
            animate={{ scale: 1, color: "hsl(45, 20%, 90%)" }}
            transition={{ duration: 0.4, ease: "easeOut" }}
            className="text-2xl font-bold inline-block"
        >
            {value.toLocaleString("pt-BR")}
        </motion.span>
    );
}

export default function MetricCards({ metrics }: { metrics: Metrics; }) {
    const cards = [
        { label: "Mensagens na Fila", value: metrics.queueMessages, icon: Activity, color: "text-primary" },
        { label: "Arquivos no MinIO", value: metrics.minioFiles, icon: HardDrive, color: "text-[hsl(200,70%,55%)]" },
        { label: "Nodes no Neo4j", value: metrics.neo4jNodes, icon: Database, color: "text-[hsl(150,60%,50%)]" },
        { label: "Depth Atual", value: metrics.currentDepth, icon: Layers, color: "text-[hsl(320,60%,55%)]" },
    ];

    return (
        <motion.div {...fadeUp} transition={{ delay: 0.1, duration: 0.5 }} className="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
            {cards.map((m) => (
                <Card key={m.label} className="border-border bg-card overflow-hidden">
                    <CardContent className="flex items-center gap-4 p-5">
                        <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary ${m.color}`}>
                            <m.icon className="h-5 w-5" />
                        </div>
                        <div>
                            <AnimatedCounter value={m.value} />
                            <p className="text-xs text-muted-foreground">{m.label}</p>
                        </div>
                    </CardContent>
                </Card>
            ))}
        </motion.div>
    );
}
