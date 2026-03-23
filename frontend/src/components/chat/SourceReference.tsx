"use client";

import { ChevronDown, FileText } from "lucide-react";

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
        <details className="group rounded-2xl border border-border bg-background/80 transition-colors open:border-primary/30 open:bg-muted/65">
            <summary className="flex cursor-pointer list-none items-center gap-3 px-3 py-2.5 marker:hidden focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35">
                <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-primary/12">
                    <FileText className="h-4 w-4 text-primary" />
                </div>

                <div className="min-w-0 flex-1">
                    <p className="truncate text-[11px] font-semibold text-foreground">{source.docName}</p>
                    <p className="text-[10px] text-muted-foreground">
                        {t("dashboard.chat.page")} {source.page}
                    </p>
                </div>

                <ChevronDown
                    aria-hidden="true"
                    className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180"
                />
            </summary>

            <div className="space-y-3 border-t border-border/70 px-4 py-3">
                <p className="text-sm leading-relaxed text-muted-foreground">&quot;...{source.snippet}...&quot;</p>
                <p className="text-[10px] font-mono uppercase tracking-[0.24em] text-primary">
                    {t("dashboard.chat.verifiedSource")}
                </p>
            </div>
        </details>
    );
};

export default SourceReference;
