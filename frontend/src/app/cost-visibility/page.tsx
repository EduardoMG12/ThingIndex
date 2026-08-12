"use client";

import { motion } from "framer-motion";

import CostCalculator from "@/components/dashboard/CostCalculator";
import AppShell from "@/components/layout/AppShell";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

export default function CostVisibilityPage() {
    const { t } = useI18n();

    return (
        <AppShell>
            <motion.div
                {...fadeUp}
                transition={{ duration: 0.5 }}
                className="mb-8"
            >
                <div className="mb-3 flex items-center gap-2">
                    <span className="h-px w-8 bg-primary" />
                    <span className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">{t("futureFeaturePage.eyebrow")}</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    {t("dashboard.cost.title")}
                </h1>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-muted-foreground">
                    {t("futureFeaturePage.description")}
                </p>
            </motion.div>

            <motion.div {...fadeUp} transition={{ delay: 0.1, duration: 0.4 }}>
                <CostCalculator />
            </motion.div>
        </AppShell>
    );
}
