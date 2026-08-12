"use client";

import { motion } from "framer-motion";
import { FileText, Brain, BarChart3 } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const HowItWorks = () => {
    const { t } = useI18n();
    const steps = [
        {
            eyebrow: t("landing.howItWorks.steps.upload.eyebrow"),
            title: t("landing.howItWorks.steps.upload.title"),
            desc: t("landing.howItWorks.steps.upload.desc"),
            icon: FileText,
            glowClassName: "from-primary/12 via-primary/5 to-transparent",
            iconClassName: "border-primary/15 bg-primary/10 text-primary",
        },
        {
            eyebrow: t("landing.howItWorks.steps.pipeline.eyebrow"),
            title: t("landing.howItWorks.steps.pipeline.title"),
            desc: t("landing.howItWorks.steps.pipeline.desc"),
            icon: Brain,
            glowClassName: "from-accent/12 via-accent/5 to-transparent",
            iconClassName: "border-accent/20 bg-accent/10 text-accent",
        },
        {
            eyebrow: t("landing.howItWorks.steps.query.eyebrow"),
            title: t("landing.howItWorks.steps.query.title"),
            desc: t("landing.howItWorks.steps.query.desc"),
            icon: BarChart3,
            glowClassName: "from-emerald-500/12 via-emerald-500/5 to-transparent",
            iconClassName: "border-emerald-500/15 bg-emerald-500/10 text-emerald-600",
        },
    ];

    return (
        <section className="border-t border-border bg-secondary/20">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                    <h2 className="font-display text-2xl font-bold sm:text-3xl">
                        {t("landing.howItWorks.title")}
                    </h2>
                    <p className="mx-auto mt-3 max-w-2xl text-sm leading-relaxed text-muted-foreground">
                        {t("landing.howItWorks.description")}
                    </p>
                </motion.div>

                <div className="mt-12 grid gap-4 lg:grid-cols-3">
                    {steps.map((s, i) => (
                        <motion.article
                            key={s.title}
                            {...fadeUp}
                            transition={{ delay: i * 0.1, duration: 0.5 }}
                            className="group relative overflow-hidden rounded-[28px] border border-border/80 bg-card/95 p-6 shadow-[0_16px_40px_-32px_hsl(var(--foreground)/0.35)] transition-all duration-300 hover:-translate-y-1 hover:border-primary/20"
                        >
                            <div className={cn("absolute inset-0 bg-gradient-to-br opacity-80", s.glowClassName)} />
                            <div className="relative flex h-full flex-col">
                                <div className="flex items-start justify-between gap-4">
                                    <div>
                                        <p className="text-[10px] font-bold uppercase tracking-[0.32em] text-primary/80">
                                            {s.eyebrow}
                                        </p>
                                        <h3 className="mt-3 font-display text-xl font-semibold leading-tight text-foreground">
                                            {s.title}
                                        </h3>
                                    </div>
                                    <div
                                        className={cn(
                                            "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border",
                                            s.iconClassName,
                                        )}
                                    >
                                        <s.icon className="h-5 w-5" />
                                    </div>
                                </div>
                                <p className="mt-5 text-sm leading-relaxed text-muted-foreground">
                                    {s.desc}
                                </p>
                            </div>
                        </motion.article>
                    ))}
                </div>
            </div>
        </section>
    );
};
