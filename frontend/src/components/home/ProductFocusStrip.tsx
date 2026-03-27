"use client";

import { motion } from "framer-motion";
import { Microscope, ShieldCheck, Target, Users } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const ProductFocusStrip = () => {
    const { t } = useI18n();

    const items = [
        { icon: Target, title: t("landing.focus.items.useCase.title"), desc: t("landing.focus.items.useCase.desc") },
        { icon: Users, title: t("landing.focus.items.audience.title"), desc: t("landing.focus.items.audience.desc") },
        { icon: ShieldCheck, title: t("landing.focus.items.promise.title"), desc: t("landing.focus.items.promise.desc") },
        { icon: Microscope, title: t("landing.focus.items.researchBoundary.title"), desc: t("landing.focus.items.researchBoundary.desc") },
    ];

    return (
        <section className="border-y border-border bg-secondary/30">
            <div className="mx-auto max-w-6xl px-6 py-16">
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                    <h2 className="font-display text-2xl font-bold sm:text-3xl">{t("landing.focus.title")}</h2>
                    <p className="mx-auto mt-3 max-w-3xl text-sm leading-relaxed text-muted-foreground">
                        {t("landing.focus.description")}
                    </p>
                </motion.div>

                <div className="mt-10 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                    {items.map((item, index) => (
                        <motion.div
                            key={item.title}
                            {...fadeUp}
                            transition={{ delay: index * 0.06, duration: 0.5 }}
                            className="rounded-xl border border-border bg-card/95 p-5"
                        >
                            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10">
                                <item.icon className="h-4.5 w-4.5 text-primary" />
                            </div>
                            <h3 className="mt-4 text-sm font-semibold text-foreground">{item.title}</h3>
                            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                        </motion.div>
                    ))}
                </div>
            </div>
        </section>
    );
};
