"use client";

import { FolderOpen, MessageSquareText, ShieldCheck } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

export function ChatWorkspaceHero({
    conversationTitle,
    activeCollectionName,
    conversationCount,
}: {
    conversationTitle: string;
    activeCollectionName: string | null;
    conversationCount: number;
}) {
    const { t } = useI18n();

    const cards = [
        {
            icon: FolderOpen,
            label: t("dashboard.chat.collectionLabel"),
            value: activeCollectionName ?? t("dashboard.chat.noCollections"),
            detail: activeCollectionName ? t("chatPage.collectionsDescription") : t("dashboard.chat.emptyNoCollection"),
        },
        {
            icon: MessageSquareText,
            label: t("dashboard.chat.conversationsLabel"),
            value: String(conversationCount),
            detail: conversationTitle,
        },
        {
            icon: ShieldCheck,
            label: t("chatPage.verificationTitle"),
            value: t("dashboard.chat.verifiedSource"),
            detail: t("chatPage.verificationDescription"),
        },
    ];

    return (
        <section className="surface-elevated rounded-[32px] p-6 sm:p-7">
            <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                <div className="max-w-3xl">
                    <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">
                        {t("chatPage.eyebrow")}
                    </p>
                    <h1 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                        {t("chatPage.title")}
                    </h1>
                    <p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted-foreground sm:text-base">
                        {t("chatPage.description")}
                    </p>
                    <p className="mt-4 text-sm font-medium text-foreground/85">
                        {conversationTitle}
                    </p>
                </div>

                <div className="grid gap-3 md:grid-cols-3 lg:min-w-[38rem]">
                    {cards.map((card) => {
                        const Icon = card.icon;

                        return (
                            <div key={card.label} className="rounded-2xl border border-border/80 bg-background/70 p-4">
                                <div className="flex items-center gap-2">
                                    <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/12">
                                        <Icon className="h-4 w-4 text-primary" />
                                    </div>
                                    <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-muted-foreground">
                                        {card.label}
                                    </p>
                                </div>
                                <p className="mt-4 text-sm font-semibold text-foreground">{card.value}</p>
                                <p className="mt-2 text-xs leading-relaxed text-muted-foreground">{card.detail}</p>
                            </div>
                        );
                    })}
                </div>
            </div>
        </section>
    );
}
