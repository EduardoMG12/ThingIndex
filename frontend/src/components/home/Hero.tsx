"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import { ArrowRight, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const Hero = () => {
    const { t, href } = useI18n();

    return (
        <section className="relative overflow-hidden">
            <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,hsl(var(--primary)/0.08),transparent_60%)]" />
            <div className="mx-auto max-w-6xl px-6 pb-20 pt-24 text-center lg:pt-32">
                <motion.div {...fadeUp} transition={{ duration: 0.6 }}>
                    <span className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                        <FileText className="h-3 w-3 text-primary" />
                        {t("landing.hero.badge")}
                    </span>
                </motion.div>

                <motion.h1
                    {...fadeUp}
                    transition={{ delay: 0.1, duration: 0.6 }}
                    className="mx-auto mt-6 max-w-3xl font-display text-4xl font-bold leading-tight tracking-tight sm:text-5xl lg:text-6xl text-foreground"
                >
                    {t("landing.hero.titleStart")}{" "}
                    <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                        {t("landing.hero.titleHighlight")}
                    </span>
                </motion.h1>

                <motion.p
                    {...fadeUp}
                    transition={{ delay: 0.2, duration: 0.6 }}
                    className="mx-auto mt-5 max-w-xl text-base text-muted-foreground sm:text-lg"
                >
                    {t("landing.hero.description")}
                </motion.p>

                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.3, duration: 0.6 }}
                    className="mt-8 flex items-center justify-center gap-3"
                >
                    <Link href={href("/register")}>
                        <Button size="lg" className="gap-2 text-sm font-semibold rounded-full px-8">
                            {t("landing.hero.primaryCta")}
                            <ArrowRight className="h-4 w-4" />
                        </Button>
                    </Link>
                    <Link href={href("/documentation")}>
                        <Button variant="outline" size="lg" className="text-sm rounded-full px-8">
                            {t("landing.hero.secondaryCta")}
                        </Button>
                    </Link>
                </motion.div>

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
                            {t("landing.hero.terminalTitle")}
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
                            ✓ Documento armazenado com sucesso
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ Indexação estrutural iniciada · resumo e outline em processamento
                        </p>
                        <p className="mt-2">
                            <span className="text-accent">$</span> thingindex context
                            --collection contrato-principal
                        </p>
                        <p className="mt-1 text-foreground/70">
                            ✓ 12 parágrafos · 2 tabelas · 9 metadados agregados
                        </p>
                        <p className="mt-1 text-primary">
                            → Contexto reconstruído para navegação e auditoria
                        </p>
                    </div>
                </motion.div>
            </div>
        </section>
    );
};
