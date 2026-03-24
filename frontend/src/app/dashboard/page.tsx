"use client";

import { motion } from "framer-motion";
import Link from "next/link";
import { FileStack, MessageSquare, Info, FolderOpen, Tag, ArrowUpRight } from "lucide-react";
import AppShell from "@/components/layout/AppShell";
import AccountStatus from "@/components/dashboard/AccountStatus";
import UploadTab from "@/components/dashboard/UploadTab";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

export default function DashboardPage() {
    const { t, href } = useI18n();

    return (
        <AppShell>
            <motion.div
                {...fadeUp}
                transition={{ duration: 0.5 }}
                className="mb-8"
            >
                <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                        <span className="h-px w-8 bg-primary" />
                        <span className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">{t("dashboard.eyebrow")}</span>
                    </div>
                    <Button asChild variant="ghost" size="sm" className="h-8 gap-1.5 px-2 text-xs text-muted-foreground whitespace-nowrap hover:text-foreground">
                        <Link href={href("/chat")}>
                            <MessageSquare className="h-3.5 w-3.5" />
                            {t("dashboard.openChat")}
                            <ArrowUpRight className="h-3.5 w-3.5" />
                        </Link>
                    </Button>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    Thing<span className="text-primary italic">Index</span> Dashboard
                </h1>
                <p className="mt-4 max-w-2xl text-base text-muted-foreground leading-relaxed">
                    {t("dashboard.description")}
                </p>
            </motion.div>

            <motion.div {...fadeUp} transition={{ delay: 0.1, duration: 0.4 }}>
                <AccountStatus />
            </motion.div>

            <div className="mt-10 space-y-14">
                <motion.section
                    {...fadeUp}
                    transition={{ delay: 0.2, duration: 0.4 }}
                    className="min-w-0"
                >
                    <div className="mb-6 flex items-center gap-2">
                        <FileStack className="h-4 w-4 text-primary" />
                        <span className="text-xs font-bold uppercase tracking-widest text-primary">{t("dashboard.tabs.pipeline")}</span>
                    </div>
                    <UploadTab />
                </motion.section>

                <motion.section
                    {...fadeUp}
                    transition={{ delay: 0.3, duration: 0.4 }}
                    className="min-w-0"
                >
                    <div className="max-w-3xl">
                        <div className="surface-elevated rounded-xl border border-border p-6 space-y-5">
                            <div className="flex items-center gap-2 text-primary">
                                <Info className="h-4 w-4" />
                                <h3 className="text-sm font-bold tracking-tight">{t("dashboard.guide.title")}</h3>
                            </div>

                            <div className="space-y-4">
                                <div className="flex gap-4">
                                    <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                                        <FolderOpen className="h-3 w-3 text-muted-foreground" />
                                    </div>
                                    <div>
                                        <h4 className="mb-1 text-[11px] font-bold uppercase tracking-wider">{t("dashboard.guide.categoriesTitle")}</h4>
                                        <p className="text-[11px] leading-relaxed text-muted-foreground">{t("dashboard.guide.categoriesDesc")}</p>
                                    </div>
                                </div>

                                <div className="flex gap-4">
                                    <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                                        <Tag className="h-3 w-3 text-muted-foreground" />
                                    </div>
                                    <div>
                                        <h4 className="mb-1 text-[11px] font-bold uppercase tracking-wider">{t("dashboard.guide.collectionsTitle")}</h4>
                                        <p className="text-[11px] leading-relaxed text-muted-foreground">{t("dashboard.guide.collectionsDesc")}</p>
                                    </div>
                                </div>

                                <div className="flex gap-4">
                                    <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-muted">
                                        <MessageSquare className="h-3 w-3 text-muted-foreground" />
                                    </div>
                                    <div>
                                        <h4 className="mb-1 text-[11px] font-bold uppercase tracking-wider">{t("dashboard.guide.contextTitle")}</h4>
                                        <p className="text-[11px] leading-relaxed text-muted-foreground">{t("dashboard.guide.contextDesc")}</p>
                                    </div>
                                </div>
                            </div>

                            <div className="pt-2">
                                <div className="rounded-lg border border-primary/10 bg-primary/5 p-3">
                                    <span className="text-[10px] font-mono font-bold text-primary">{t("dashboard.guide.accuracy")}</span>
                                    <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{t("dashboard.guide.accuracyHint")}</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </motion.section>
            </div>
        </AppShell>
    );
}
