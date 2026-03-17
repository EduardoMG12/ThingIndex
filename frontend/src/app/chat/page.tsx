"use client";

import { motion } from "framer-motion";
import { FolderOpen, MessageSquare, Sparkles } from "lucide-react";

import ChatTab from "@/components/dashboard/ChatTab";
import AppShell from "@/components/layout/AppShell";
import { useI18n } from "@/lib/i18n/provider";

const fadeUp = { initial: { opacity: 0, y: 16 }, animate: { opacity: 1, y: 0 } };

export default function ChatPage() {
    const { t } = useI18n();

    return (
        <AppShell contentClassName="max-w-none px-4 py-8 lg:px-6 xl:px-8">
            <motion.div
                {...fadeUp}
                transition={{ duration: 0.5 }}
                className="mb-8"
            >
                <div className="mb-3 flex items-center gap-2">
                    <span className="h-px w-8 bg-primary" />
                    <span className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">{t("chatPage.eyebrow")}</span>
                </div>
                <h1 className="font-display text-4xl font-bold tracking-tight text-foreground">
                    {t("chatPage.title")}
                </h1>
                <p className="mt-4 max-w-3xl text-base leading-relaxed text-muted-foreground">
                    {t("chatPage.description")}
                </p>
            </motion.div>

            <div className="space-y-6">
                <motion.div
                    {...fadeUp}
                    transition={{ delay: 0.1, duration: 0.4 }}
                    className="min-w-0"
                >
                    <ChatTab className="h-[calc(100vh-16rem)] min-h-[720px]" />
                </motion.div>

                <motion.aside
                    {...fadeUp}
                    transition={{ delay: 0.2, duration: 0.4 }}
                    className="grid gap-4 lg:grid-cols-3"
                >
                    <div className="surface-elevated rounded-2xl border border-border p-5">
                        <div className="flex items-center gap-2 text-primary">
                            <MessageSquare className="h-4 w-4" />
                            <h2 className="text-sm font-semibold tracking-tight">{t("chatPage.panelTitle")}</h2>
                        </div>
                        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                            {t("chatPage.panelDescription")}
                        </p>
                    </div>

                    <div className="surface-elevated rounded-2xl border border-border p-5">
                        <div className="flex items-center gap-2 text-primary">
                            <FolderOpen className="h-4 w-4" />
                            <h2 className="text-sm font-semibold tracking-tight">{t("chatPage.collectionsTitle")}</h2>
                        </div>
                        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                            {t("chatPage.collectionsDescription")}
                        </p>
                    </div>

                    <div className="surface-elevated rounded-2xl border border-border p-5">
                        <div className="flex items-center gap-2 text-primary">
                            <Sparkles className="h-4 w-4" />
                            <h2 className="text-sm font-semibold tracking-tight">{t("chatPage.verificationTitle")}</h2>
                        </div>
                        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                            {t("chatPage.verificationDescription")}
                        </p>
                    </div>
                </motion.aside>
            </div>
        </AppShell>
    );
}
