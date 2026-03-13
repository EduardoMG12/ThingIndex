"use client";

import { HelpCircle, Calculator } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

interface CostCalculatorProps {
    config: {
        aiProvider: string;
        contextStore: string;
        fileCount: number;
        estimatedTokens: number;
    };
}

const CostCalculator = ({ config }: CostCalculatorProps) => {
    const { t } = useI18n();
    const estimatedCost = (config.estimatedTokens / 1000) * 0.002;

    return (
        <div className="surface-elevated rounded-xl border border-border overflow-hidden">
            <div className="p-4 border-b border-border bg-muted/30 flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Calculator className="h-4 w-4 text-primary" />
                    <h3 className="text-[10px] font-bold uppercase tracking-widest text-foreground">{t("dashboard.cost.title")}</h3>
                </div>
                <HelpCircle className="h-3.5 w-3.5 text-muted-foreground cursor-help" />
            </div>

            <div className="p-4 space-y-4">
                <div className="space-y-2">
                    <div className="flex justify-between text-[11px]">
                        <span className="text-muted-foreground">{t("dashboard.cost.llmProvider")}</span>
                        <span className="font-mono text-foreground">{config.aiProvider}</span>
                    </div>
                    <div className="flex justify-between text-[11px]">
                        <span className="text-muted-foreground">{t("dashboard.cost.contextStore")}</span>
                        <span className="font-mono text-foreground">{config.contextStore}</span>
                    </div>
                    <div className="flex justify-between text-[11px]">
                        <span className="text-muted-foreground">{t("dashboard.cost.indexedFiles")}</span>
                        <span className="font-mono text-foreground">{config.fileCount}</span>
                    </div>
                    <div className="flex justify-between text-[11px]">
                        <span className="text-muted-foreground">{t("dashboard.cost.estimatedTokens")}</span>
                        <span className="font-mono text-foreground">~{config.estimatedTokens.toLocaleString()}</span>
                    </div>
                </div>

                <div className="h-px bg-border" />

                <div className="bg-primary/5 rounded-lg p-3 border border-primary/10">
                    <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-foreground">{t("dashboard.cost.estimatedCost")}</span>
                        <span className="text-sm font-bold text-primary">${estimatedCost.toFixed(4)}</span>
                    </div>
                    <p className="mt-1 text-[9px] text-muted-foreground italic">
                        {t("dashboard.cost.hint")}
                    </p>
                </div>
            </div>
        </div>
    );
};

export default CostCalculator;
