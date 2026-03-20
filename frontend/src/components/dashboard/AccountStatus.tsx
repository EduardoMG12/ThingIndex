"use client";

import { ShieldCheck, Zap, Clock } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

const AccountStatus = () => {
    const { t } = useI18n();
    const items = [
        {
            icon: ShieldCheck,
            iconClassName: "bg-emerald-500/10 text-emerald-500",
            label: t("dashboard.account.accessMode"),
            value: t("dashboard.account.accessModeValue"),
            hint: t("dashboard.account.accessModeHint"),
        },
        {
            icon: Zap,
            iconClassName: "bg-blue-500/10 text-blue-500",
            label: t("dashboard.account.usageView"),
            value: t("dashboard.account.usageViewValue"),
            hint: t("dashboard.account.usageViewHint"),
        },
        {
            icon: Clock,
            iconClassName: "bg-amber-500/10 text-amber-500",
            label: t("dashboard.account.sessionPolicy"),
            value: t("dashboard.account.sessionPolicyValue"),
            hint: t("dashboard.account.sessionPolicyHint"),
        },
    ];

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {items.map((item) => {
                const Icon = item.icon;

                return (
                    <div key={item.label} className="surface-elevated rounded-xl border border-border p-4">
                        <div className="flex items-start gap-4">
                            <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${item.iconClassName}`}>
                                <Icon className="h-5 w-5" />
                            </div>
                            <div className="min-w-0">
                                <p className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{item.label}</p>
                                <p className="mt-1 text-sm font-bold text-foreground">{item.value}</p>
                                <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{item.hint}</p>
                            </div>
                        </div>
                    </div>
                );
            })}
        </div>
    );
};

export default AccountStatus;
