"use client";

import { motion } from "framer-motion";
import { FileText, Brain, BarChart3 } from "lucide-react";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

const steps = [
    {
        step: "01",
        title: "Upload & Scan",
        desc: "Envie PDFs, defina categoria e collection. ClamAV valida segurança automaticamente.",
        icon: FileText,
    },
    {
        step: "02",
        title: "Pipeline de 3 Prompts",
        desc: "Cada chunk passa por resumo contextual, estruturação semântica e geração de embedding vetorial.",
        icon: Brain,
    },
    {
        step: "03",
        title: "Query & Resposta",
        desc: "Pergunte em linguagem natural. O sistema busca vetorialmente e gera respostas com fontes.",
        icon: BarChart3,
    },
];

export const HowItWorks = () => {
    return (
        <section className="border-t border-border bg-secondary/20">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                    <h2 className="font-display text-2xl font-bold sm:text-3xl">
                        Como funciona
                    </h2>
                </motion.div>

                <div className="mt-12 grid gap-8 lg:grid-cols-3">
                    {steps.map((s, i) => (
                        <motion.div
                            key={s.step}
                            {...fadeUp}
                            transition={{ delay: i * 0.1, duration: 0.5 }}
                            className="relative rounded-xl border border-border bg-card p-6"
                        >
                            <span className="font-display text-3xl font-bold text-primary/20">
                                {s.step}
                            </span>
                            <div className="mt-3 flex items-center gap-2">
                                <s.icon className="h-4 w-4 text-primary" />
                                <h3 className="font-display text-sm font-semibold">
                                    {s.title}
                                </h3>
                            </div>
                            <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                                {s.desc}
                            </p>
                        </motion.div>
                    ))}
                </div>
            </div>
        </section>
    );
};
