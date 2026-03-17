"use client";

import { FileText, ExternalLink } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

export type Source = {
    id: string;
    docName: string;
    page: number;
    snippet: string;
};

interface SourceReferenceProps {
    source: Source;
}

const SourceReference = ({ source }: SourceReferenceProps) => {
    const { t } = useI18n();

    return (
        <div className="group relative flex cursor-pointer items-center gap-2 rounded-2xl border border-border bg-background/80 px-3 py-2 pr-4 transition-colors hover:border-primary/30 hover:bg-muted/80">
            <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/12">
                <FileText className="h-3.5 w-3.5 text-primary" />
            </div>
            <span className="max-w-[160px] truncate text-[11px] font-medium text-foreground">
                {source.docName}
            </span>
            <span className="border-l border-border pl-2 text-[10px] text-muted-foreground">
                {t("dashboard.chat.page")} {source.page}
            </span>

            <div className="absolute bottom-full left-1/2 z-50 mb-3 hidden w-72 -translate-x-1/2 rounded-2xl border border-border bg-card p-4 shadow-xl group-hover:block">
                <p className="text-[11px] italic leading-relaxed text-muted-foreground">
                    &quot;...{source.snippet}...&quot;
                </p>
                <div className="mt-2 flex items-center justify-between border-t border-border pt-2">
                    <span className="text-[10px] font-mono text-primary">{t("dashboard.chat.verifiedSource")}</span>
                    <ExternalLink className="h-3 w-3 text-muted-foreground" />
                </div>
                <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-2 bg-card border-r border-b border-border rotate-45" />
            </div>
        </div>
    );
};

export default SourceReference;
