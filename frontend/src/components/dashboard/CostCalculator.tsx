"use client";

import { HelpCircle, Calculator } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

const CostCalculator = () => {
    const { t } = useI18n();
    const items = [
        {
            label: t("dashboard.cost.providerState"),
            value: t("dashboard.cost.providerStateValue"),
        },
        {
            label: t("dashboard.cost.storageState"),
            value: t("dashboard.cost.storageStateValue"),
        },
        {
            label: t("dashboard.cost.estimateState"),
            value: t("dashboard.cost.estimateStateValue"),
        },
    ];

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
                    {items.map((item) => (
                        <div key={item.label} className="flex justify-between gap-4 text-[11px]">
                            <span className="text-muted-foreground">{item.label}</span>
                            <span className="text-right font-mono text-foreground">{item.value}</span>
                        </div>
                    ))}
                </div>

                <div className="h-px bg-border" />

                <div className="bg-primary/5 rounded-lg p-3 border border-primary/10">
                    <p className="text-xs font-medium text-foreground">{t("dashboard.cost.summaryTitle")}</p>
                    <p className="mt-1 text-[10px] leading-relaxed text-muted-foreground">{t("dashboard.cost.hint")}</p>
                </div>
            </div>
        </div>
    );
};

export default CostCalculator;
