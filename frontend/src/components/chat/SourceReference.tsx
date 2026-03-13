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
        <div className="group relative flex items-center gap-2 rounded-md border border-border bg-muted/50 px-2 py-1 pr-3 transition-colors hover:bg-muted hover:border-primary/30 cursor-pointer">
            <FileText className="h-3 w-3 text-primary" />
            <span className="text-[10px] font-medium text-foreground truncate max-w-[120px]">
                {source.docName}
            </span>
            <span className="text-[9px] text-muted-foreground border-l border-border pl-2">
                {t("dashboard.chat.page")} {source.page}
            </span>

            <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block w-64 p-3 bg-card border border-border rounded-lg shadow-xl z-50">
                <p className="text-[10px] text-muted-foreground leading-relaxed italic">
                    &quot;...{source.snippet}...&quot;
                </p>
                <div className="mt-2 flex items-center justify-between border-t border-border pt-2">
                    <span className="text-[9px] font-mono text-primary">{t("dashboard.chat.verifiedSource")}</span>
                    <ExternalLink className="h-2.5 w-2.5 text-muted-foreground" />
                </div>
                <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-2 h-2 bg-card border-r border-b border-border rotate-45" />
            </div>
        </div>
    );
};

export default SourceReference;
