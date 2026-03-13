"use client";

import { motion } from "framer-motion";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const Metrics = () => {
    const { t } = useI18n();
    const metrics = [
        { value: "3x", label: t("landing.metrics.precision") },
        { value: "<200ms", label: t("landing.metrics.latency") },
        { value: "10k+", label: t("landing.metrics.throughput") },
        { value: "99.9%", label: t("landing.metrics.uptime") },
    ];

    return (
        <section className="border-y border-border bg-secondary/30">
            <div className="mx-auto grid max-w-6xl grid-cols-2 gap-6 px-6 py-12 lg:grid-cols-4">
                {metrics.map((m, i) => (
                    <motion.div
                        key={m.label}
                        {...fadeUp}
                        transition={{ delay: i * 0.05, duration: 0.5 }}
                        className="text-center"
                    >
                        <p className="font-display text-3xl font-bold text-primary">
                            {m.value}
                        </p>
                        <p className="mt-1 text-xs text-muted-foreground">{m.label}</p>
                    </motion.div>
                ))}
            </div>
        </section>
    );
};
