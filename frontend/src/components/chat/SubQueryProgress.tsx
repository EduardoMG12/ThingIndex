"use client";

import { CheckCircle2, Circle, Loader2, Search } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";

export type SubQuery = {
    id: string;
    question: string;
    status: "pending" | "running" | "done";
};

interface SubQueryProgressProps {
    queries: SubQuery[];
}

const SubQueryProgress = ({ queries }: SubQueryProgressProps) => {
    const { t } = useI18n();

    return (
        <div className="space-y-3 rounded-2xl border border-border/60 bg-background/75 p-4">
            <div className="mb-1 flex items-center gap-2">
                <Search className="h-4 w-4 text-muted-foreground" />
                <h4 className="text-xs font-medium text-muted-foreground">{t("dashboard.chat.queryDecomposition")}</h4>
            </div>

            <div className="space-y-3">
                {queries.map((q) => (
                    <div key={q.id} className="flex items-start gap-3 rounded-2xl bg-muted/30 px-3 py-3">
                        <div className="mt-0.5">
                            {q.status === "done" ? (
                                <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                            ) : q.status === "running" ? (
                                <Loader2 className="h-4 w-4 animate-spin text-primary" />
                            ) : (
                                <Circle className="h-4 w-4 text-muted-foreground/30" />
                            )}
                        </div>
                        <div className="min-w-0 flex-1">
                            <p className={cn(
                                "text-sm leading-snug transition-colors",
                                q.status === "done" ? "text-muted-foreground" : "text-foreground font-medium"
                            )}>
                                {q.question}
                            </p>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
};

export default SubQueryProgress;
