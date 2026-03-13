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
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const Features = () => {
    const { t } = useI18n();
    const features = [
        { icon: Brain, title: t("landing.features.items.adaptive.title"), desc: t("landing.features.items.adaptive.desc") },
        { icon: Zap, title: t("landing.features.items.async.title"), desc: t("landing.features.items.async.desc") },
        { icon: Database, title: t("landing.features.items.relational.title"), desc: t("landing.features.items.relational.desc") },
        { icon: Layers, title: t("landing.features.items.taxonomy.title"), desc: t("landing.features.items.taxonomy.desc") },
        { icon: Shield, title: t("landing.features.items.security.title"), desc: t("landing.features.items.security.desc") },
        { icon: GitBranch, title: t("landing.features.items.openSource.title"), desc: t("landing.features.items.openSource.desc") },
    ];

    return (
        <section className="mx-auto max-w-6xl px-6 py-20">
            <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                <h2 className="font-display text-2xl font-bold sm:text-3xl">{t("landing.features.title")}</h2>
                <p className="mx-auto mt-2 max-w-lg text-sm text-muted-foreground">
                    {t("landing.features.description")}
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
