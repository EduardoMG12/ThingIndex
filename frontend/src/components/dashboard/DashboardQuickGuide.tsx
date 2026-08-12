"use client";

import { FolderOpen, Info, MessageSquare, Tag } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

type DashboardQuickGuideProps = {
    className?: string;
};

export default function DashboardQuickGuide({ className }: DashboardQuickGuideProps) {
    const { t } = useI18n();

    return (
        <div className={cn("surface-elevated rounded-xl border border-border p-6 space-y-5", className)}>
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
    );
}
