"use client";

import { motion } from "framer-motion";
import {
    Zap,
    Brain,
    Database,
    Shield,
    GitBranch,
    Layers,
} from "lucide-react";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

const features = [
    {
        icon: Brain,
        title: "3-Prompt Adaptive Chain",
        desc: "Cada chunk passa por resumo, estruturação e embedding — não é um RAG genérico.",
    },
    {
        icon: Zap,
        title: "Pipeline Assíncrono",
        desc: "RabbitMQ + workers paralelos processam milhares de páginas sem travar.",
    },
    {
        icon: Database,
        title: "Busca Vetorial",
        desc: "Qdrant com embeddings de alta dimensão para retrieval preciso e rápido.",
    },
    {
        icon: Layers,
        title: "Categorias & Collections",
        desc: "Organize documentos com metadados ricos para filtragem inteligente.",
    },
    {
        icon: Shield,
        title: "Segurança First",
        desc: "ClamAV escaneia cada upload. Seus dados nunca saem da sua infra.",
    },
    {
        icon: GitBranch,
        title: "Open Source",
        desc: "MIT License. Rode na sua VM com suas keys ou use nossa API gerenciada.",
    },
];

export const Features = () => {
    return (
        <section className="mx-auto max-w-6xl px-6 py-20">
            <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                <h2 className="font-display text-2xl font-bold sm:text-3xl">
                    Por que <span className="text-primary">ThingIndex</span>?
                </h2>
                <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground">
                    Não é só mais um wrapper de LLM. É uma pipeline completa de
                    document intelligence.
                </p>
            </motion.div>

            <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {features.map((f, i) => (
                    <motion.div
                        key={f.title}
                        {...fadeUp}
                        transition={{ delay: i * 0.06, duration: 0.5 }}
                        className="group rounded-xl border border-border bg-card p-6 transition-colors hover:border-primary/30 hover:bg-card/80"
                    >
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
                            <f.icon className="h-4.5 w-4.5 text-primary" />
                        </div>
                        <h3 className="mt-4 font-display text-sm font-semibold">
                            {f.title}
                        </h3>
                        <p className="mt-1.5 text-xs leading-relaxed text-muted-foreground">
                            {f.desc}
                        </p>
                    </motion.div>
                ))}
            </div>
        </section>
    );
};
