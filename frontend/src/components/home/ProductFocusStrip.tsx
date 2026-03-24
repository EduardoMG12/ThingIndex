"use client";

import { motion } from "framer-motion";
import { Microscope, ShieldCheck, Target, Users } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

const fadeUp = {
    initial: { opacity: 0, y: 24 },
    whileInView: { opacity: 1, y: 0 },
    viewport: { once: true },
};

export const ProductFocusStrip = () => {
    const { t } = useI18n();

    const items = [
        {
            icon: Target,
            eyebrow: t("landing.focus.items.useCase.eyebrow"),
            title: t("landing.focus.items.useCase.title"),
            desc: t("landing.focus.items.useCase.desc"),
            note: t("landing.focus.items.useCase.note"),
            className: "xl:col-span-3",
            glowClassName: "from-primary/12 via-primary/5 to-transparent",
            iconClassName: "border-primary/15 bg-primary/10 text-primary",
            dotClassName: "bg-primary/70",
        },
        {
            icon: Users,
            eyebrow: t("landing.focus.items.audience.eyebrow"),
            title: t("landing.focus.items.audience.title"),
            desc: t("landing.focus.items.audience.desc"),
            note: t("landing.focus.items.audience.note"),
            className: "xl:col-span-3",
            glowClassName: "from-accent/12 via-accent/5 to-transparent",
            iconClassName: "border-accent/20 bg-accent/10 text-accent",
            dotClassName: "bg-accent/70",
        },
        {
            icon: ShieldCheck,
            eyebrow: t("landing.focus.items.promise.eyebrow"),
            title: t("landing.focus.items.promise.title"),
            desc: t("landing.focus.items.promise.desc"),
            note: t("landing.focus.items.promise.note"),
            className: "xl:col-span-4",
            glowClassName: "from-emerald-500/12 via-emerald-500/5 to-transparent",
            iconClassName: "border-emerald-500/15 bg-emerald-500/10 text-emerald-600",
            dotClassName: "bg-emerald-500/70",
        },
        {
            icon: Microscope,
            eyebrow: t("landing.focus.items.researchBoundary.eyebrow"),
            title: t("landing.focus.items.researchBoundary.title"),
            desc: t("landing.focus.items.researchBoundary.desc"),
            note: t("landing.focus.items.researchBoundary.note"),
            className: "xl:col-span-2",
            glowClassName: "from-amber-500/12 via-amber-500/5 to-transparent",
            iconClassName: "border-amber-500/15 bg-amber-500/10 text-amber-600",
            dotClassName: "bg-amber-500/70",
        },
    ];

    return (
        <section className="relative overflow-hidden border-y border-border bg-secondary/25">
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_top,hsl(var(--primary)/0.08),transparent_48%)]" />
            <div className="relative mx-auto max-w-6xl px-6 py-16">
                <motion.div {...fadeUp} transition={{ duration: 0.5 }} className="text-center">
                    <p className="text-[11px] font-bold uppercase tracking-[0.34em] text-primary/80">
                        {t("landing.focus.eyebrow")}
                    </p>
                    <h2 className="font-display text-2xl font-bold sm:text-3xl">{t("landing.focus.title")}</h2>
                    <p className="mx-auto mt-3 max-w-3xl text-sm leading-relaxed text-muted-foreground">
                        {t("landing.focus.description")}
                    </p>
                </motion.div>

                <div className="mt-10 grid gap-4 md:grid-cols-2 xl:grid-cols-6">
                    {items.map((item, index) => (
                        <motion.article
                            key={item.title}
                            {...fadeUp}
                            transition={{ delay: index * 0.06, duration: 0.5 }}
                            className={cn(
                                "group relative overflow-hidden rounded-[28px] border border-border/80 bg-card/95 p-6 shadow-[0_16px_40px_-32px_hsl(var(--foreground)/0.35)] transition-all duration-300 hover:-translate-y-1 hover:border-primary/20",
                                item.className,
                            )}
                        >
                            <div className={cn("absolute inset-0 bg-gradient-to-br opacity-80", item.glowClassName)} />
                            <div className="relative flex h-full flex-col">
                                <div className="flex items-start justify-between gap-4">
                                    <div className="max-w-sm">
                                        <p className="text-[10px] font-bold uppercase tracking-[0.32em] text-primary/80">
                                            {item.eyebrow}
                                        </p>
                                        <h3 className="mt-3 font-display text-xl font-semibold leading-tight text-foreground">
                                            {item.title}
                                        </h3>
                                    </div>
                                    <div
                                        className={cn(
                                            "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border",
                                            item.iconClassName,
                                        )}
                                    >
                                        <item.icon className="h-5 w-5" />
                                    </div>
                                </div>
                                <p className="mt-4 max-w-lg text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                                <div className="mt-6 flex items-center gap-2 border-t border-border/70 pt-4 text-xs font-medium text-muted-foreground">
                                    <span className={cn("h-2 w-2 rounded-full", item.dotClassName)} />
                                    {item.note}
                                </div>
                            </div>
                        </motion.article>
                    ))}
                </div>
            </div>
        </section>
    );
};
