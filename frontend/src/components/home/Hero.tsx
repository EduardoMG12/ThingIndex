"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight, FileText } from "lucide-react";
import { Button } from "@/components/ui/Button";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const Hero = () => {
    return (
        <section className="relative overflow-hidden">
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,hsl(var(--primary)/0.08),transparent_60%)]" />
            <div className="mx-auto max-w-6xl px-6 pb-20 pt-24 text-center lg:pt-32">
                <motion.div {...fadeUp} transition={{ duration: 0.6 }}>
                    <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                        <FileText className="h-3 w-3 text-primary" />
                        Document Intelligence Platform
                    </span>
                </motion.div>

                <motion.h1
                    {...fadeUp}
                    transition={{ delay: 0.1, duration: 0.6 }}
                    className="mx-auto mt-6 max-w-3xl font-display text-4xl font-bold leading-tight tracking-tight sm:text-5xl lg:text-6xl text-foreground"
                >
                    Transforme documentos em{" "}
                    <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                        conhecimento estruturado
                    </span>
                </motion.h1>

                <motion.p
                    {...fadeUp}
                    transition={{ delay: 0.2, duration: 0.6 }}
                    className="mx-auto mt-5 max-w-xl text-base text-muted-foreground sm:text-lg"
                >
                    Pipeline RAG adaptativo com cadeia de 3 prompts. Indexe, estruture e
                    consulte qualquer documento com precisão superior.
                </motion.p>

                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.3, duration: 0.6 }}
                    className="mt-8 flex items-center justify-center gap-3"
                >
                    <Link href="/register">
                        <Button size="lg" className="gap-2 text-sm font-semibold rounded-full px-8">
                            Começar agora
                            <ArrowRight className="h-4 w-4" />
                        </Button>
                    </Link>
                    <Link href="/docs">
                        <Button variant="outline" size="lg" className="text-sm rounded-full px-8">
                            Ver documentação
                        </Button>
                    </Link>
                </motion.div>

                {/* Terminal preview */}
                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.4, duration: 0.7 }}
                    className="mx-auto mt-14 max-w-2xl overflow-hidden rounded-xl border border-border bg-card shadow-lg"
                >
                    <div className="flex items-center gap-1.5 border-b border-border bg-secondary/50 px-4 py-2.5">
                        <div className="h-2.5 w-2.5 rounded-full bg-destructive/60" />
                        <div className="h-2.5 w-2.5 rounded-full bg-primary/60" />
                        <div className="h-2.5 w-2.5 rounded-full bg-accent/60" />
                        <span className="ml-2 font-mono text-[10px] text-muted-foreground">
                            thingindex pipeline
                        </span>
                    </div>
                    <div className="p-5 text-left font-mono text-xs leading-relaxed text-muted-foreground">
                        <p>
                            <span className="text-accent">$</span> thingindex upload
                            --file contrato.pdf --category juridico
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ Upload concluído · 42 páginas detectadas
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ ClamAV scan passed · nenhuma ameaça
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ Enfileirado no RabbitMQ · 42 chunks criados
                        </p>
                        <p className="mt-2">
                            <span className="text-accent">$</span> thingindex query
                            "cláusulas de rescisão"
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ 3 chunks relevantes encontrados · score 0.94
                        </p>
                        <p className="mt-1 text-primary">
                            → Resposta gerada em 180ms com 3 fontes
                        </p>
                    </div>
                </motion.div>
            </div>
        </section>
    );
};
