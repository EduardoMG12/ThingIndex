"use client";

import Link from "next/link";
import { FolderPlus, Sparkles, Settings2 } from "lucide-react";

import { useI18n } from "@/lib/i18n/provider";

export function ChatEmptyState({
    hasCollections,
    hasChatRuntime,
    activeCollectionName,
}: {
    hasCollections: boolean;
    hasChatRuntime: boolean;
    activeCollectionName: string | null;
}) {
    const { t, href } = useI18n();

    const cta = !hasCollections
        ? {
            href: href("/documents"),
            label: t("dashboard.chat.documentsCta"),
            icon: FolderPlus,
            description: t("dashboard.chat.emptyNoCollection"),
        }
        : !hasChatRuntime
            ? {
                href: href("/settings"),
                label: t("dashboard.chat.settingsCta"),
                icon: Settings2,
                description: t("providerRuntime.chat.missing"),
            }
            : null;

    const CtaIcon = cta?.icon ?? Sparkles;

    return (
        <div className="flex min-h-[28rem] flex-col items-center justify-center rounded-[32px] border border-dashed border-border bg-card/45 px-8 py-10 text-center">
            <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-[22px] bg-primary/12">
                <Sparkles className="h-8 w-8 text-primary" />
            </div>
            <h2 className="text-2xl font-semibold tracking-tight text-foreground">
                {t("dashboard.chat.emptyTitle")}
            </h2>
            <p className="mt-3 max-w-2xl text-base leading-relaxed text-muted-foreground">
                {hasCollections && hasChatRuntime ? t("dashboard.chat.emptyDescription") : cta?.description}
            </p>
            {activeCollectionName ? (
                <p className="mt-4 text-sm font-medium text-foreground/85">
                    {`${t("dashboard.chat.activeCollection")} ${activeCollectionName}`}
                </p>
            ) : null}

            {cta ? (
                <Link
                    href={cta.href}
                    className="mt-6 inline-flex items-center gap-2 rounded-full border border-border bg-background/80 px-4 py-2.5 text-sm font-semibold text-foreground transition-colors hover:border-primary/30 hover:text-primary"
                >
                    <CtaIcon className="h-4 w-4" />
                    {cta.label}
                </Link>
            ) : null}
        </div>
    );
}
