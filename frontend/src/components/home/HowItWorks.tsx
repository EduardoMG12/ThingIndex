"use client";

import { motion } from "framer-motion";
import { FileText, Brain, BarChart3 } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const HowItWorks = () => {
    const { t } = useI18n();
    const steps = [
        { step: "01", title: t("landing.howItWorks.steps.upload.title"), desc: t("landing.howItWorks.steps.upload.desc"), icon: FileText },
        { step: "02", title: t("landing.howItWorks.steps.pipeline.title"), desc: t("landing.howItWorks.steps.pipeline.desc"), icon: Brain },
        { step: "03", title: t("landing.howItWorks.steps.query.title"), desc: t("landing.howItWorks.steps.query.desc"), icon: BarChart3 },
    ];

    return (
        <section className="border-t border-border bg-secondary/20">
            <div className="mx-auto max-w-6xl px-6 py-20">
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                    <h2 className="font-display text-2xl font-bold sm:text-3xl">
                        {t("landing.howItWorks.title")}
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
