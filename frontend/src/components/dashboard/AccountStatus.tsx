"use client";

import { ShieldCheck, Zap, Clock } from "lucide-react";
import { useI18n } from "@/lib/i18n/provider";

const AccountStatus = () => {
    const { t } = useI18n();

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="surface-elevated rounded-xl p-4 border border-border flex items-center gap-4">
                <div className="h-10 w-10 rounded-full bg-emerald-500/10 flex items-center justify-center shrink-0">
                    <ShieldCheck className="h-5 w-5 text-emerald-500" />
                </div>
                <div className="min-w-0">
                    <p className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{t("dashboard.account.apiStatus")}</p>
                    <p className="text-sm font-bold text-foreground truncate">{t("dashboard.account.connected")}</p>
                </div>
            </div>

            <div className="surface-elevated rounded-xl p-4 border border-border flex items-center gap-4">
                <div className="h-10 w-10 rounded-full bg-blue-500/10 flex items-center justify-center shrink-0">
                    <Zap className="h-5 w-5 text-blue-500" />
                </div>
                <div className="min-w-0">
                    <p className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{t("dashboard.account.monthlyUsage")}</p>
                    <p className="text-sm font-bold text-foreground truncate">1.2k / 5k tokens</p>
                </div>
            </div>

            <div className="surface-elevated rounded-xl p-4 border border-border flex items-center gap-4">
                <div className="h-10 w-10 rounded-full bg-amber-500/10 flex items-center justify-center shrink-0">
                    <Clock className="h-5 w-5 text-amber-500" />
                </div>
                <div className="min-w-0">
                    <p className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground">{t("dashboard.account.session")}</p>
                    <p className="text-sm font-bold text-foreground truncate">{t("dashboard.account.expiresIn")}</p>
                </div>
            </div>
        </div>
    );
};

export default AccountStatus;
