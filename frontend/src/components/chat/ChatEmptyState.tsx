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
            href: href("/workspace"),
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
        <div className="flex min-h-[22rem] flex-col items-center justify-center px-6 py-10 text-center">
            <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-muted/70">
                <Sparkles className="h-5 w-5 text-muted-foreground" />
            </div>
            <h2 className="text-2xl font-semibold tracking-tight text-foreground">
                {t("dashboard.chat.emptyTitle")}
            </h2>
            <p className="mt-3 max-w-xl text-sm leading-relaxed text-muted-foreground">
                {hasCollections && hasChatRuntime ? t("dashboard.chat.emptyDescription") : cta?.description}
            </p>
            {activeCollectionName ? (
                <p className="mt-4 text-sm text-foreground/85">
                    {`${t("dashboard.chat.activeCollection")} ${activeCollectionName}`}
                </p>
            ) : null}

            {cta ? (
                <Link
                    href={cta.href}
                    className="mt-6 inline-flex items-center gap-2 rounded-full border border-border/70 bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:border-foreground/20 hover:bg-muted/40"
                >
                    <CtaIcon className="h-4 w-4" />
                    {cta.label}
                </Link>
            ) : null}
        </div>
    );
}
