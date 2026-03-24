"use client";

import { motion } from "framer-motion";
import Link from "next/link";
import { FileStack, MessageSquare, ArrowUpRight } from "lucide-react";
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
            </div>
        </AppShell>
    );
}
