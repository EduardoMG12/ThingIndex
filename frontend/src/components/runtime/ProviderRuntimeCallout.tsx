"use client";

import Link from "next/link";
import { AlertTriangle, Bot, BrainCircuit, Settings2 } from "lucide-react";

import type { ProviderRuntimeAssignment } from "@/lib/api/providers";
import { useI18n } from "@/lib/i18n/provider";
import { getProviderRuntimeModel, formatProviderTypeLabel } from "@/lib/provider-runtime";
import { cn } from "@/lib/utils";

type ProviderRuntimeRole = "chat" | "indexing";

function getRoleCopyKey(role: ProviderRuntimeRole) {
    return role === "chat" ? "providerRuntime.chat" : "providerRuntime.indexing";
}

export default function ProviderRuntimeCallout({
    role,
    assignment,
    isLoading,
    error,
    className,
}: {
    role: ProviderRuntimeRole;
    assignment: ProviderRuntimeAssignment | null;
    isLoading: boolean;
    error: string | null;
    className?: string;
}) {
    const { t, href, locale } = useI18n();
    const roleCopyKey = getRoleCopyKey(role);
    const model = getProviderRuntimeModel(assignment);
    const updatedAt = assignment?.updated_at
        ? new Intl.DateTimeFormat(locale, {
            dateStyle: "short",
            timeStyle: "short",
        }).format(new Date(assignment.updated_at))
        : "-";
    const Icon = role === "chat" ? Bot : BrainCircuit;

    return (
        <section
            aria-label={t(`${roleCopyKey}.title`)}
            className={cn(
                "rounded-2xl border bg-card/70 p-5 shadow-sm",
                assignment
                    ? "border-primary/20"
                    : "border-amber-500/25 bg-amber-500/5",
                className
            )}
        >
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="flex items-start gap-3">
                    <div className={cn(
                        "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border bg-background/80",
                        assignment ? "border-primary/20 text-primary" : "border-amber-500/20 text-amber-500"
                    )}>
                        {assignment ? <Icon className="h-5 w-5" /> : <AlertTriangle className="h-5 w-5" />}
                    </div>
                    <div className="space-y-2">
                        <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">
                            {t("providerRuntime.eyebrow")}
                        </p>
                        <div>
                            <h2 className="text-lg font-semibold">{t(`${roleCopyKey}.title`)}</h2>
                            <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
                                {t(`${roleCopyKey}.description`)}
                            </p>
                        </div>
                    </div>
                </div>

                <Link
                    href={href("/settings")}
                    className="inline-flex items-center gap-2 rounded-full border border-border bg-background/70 px-3 py-2 text-xs font-semibold text-foreground transition-colors hover:border-primary/30 hover:text-primary"
                >
                    <Settings2 className="h-3.5 w-3.5" />
                    {t("providerRuntime.settingsCta")}
                </Link>
            </div>

            {error ? (
                <div className="mt-4 rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                    {error}
                </div>
            ) : null}

            {!error && isLoading ? (
                <div className="mt-4 grid gap-3 md:grid-cols-3">
                    {Array.from({ length: 3 }).map((_, index) => (
                        <div key={index} className="rounded-xl border border-border bg-background/60 p-4">
                            <div className="h-2.5 w-24 animate-pulse rounded-full bg-muted" />
                            <div className="mt-4 h-3 w-32 animate-pulse rounded-full bg-muted/80" />
                        </div>
                    ))}
                </div>
            ) : null}

            {!error && !isLoading && assignment ? (
                <div className="mt-4 grid gap-3 md:grid-cols-3">
                    <div className="rounded-xl border border-border bg-background/60 p-4">
                        <p className="text-[10px] uppercase tracking-[0.22em] text-muted-foreground">
                            {t("providerRuntime.providerLabel")}
                        </p>
                        <p className="mt-3 text-sm font-semibold text-foreground">
                            {formatProviderTypeLabel(assignment.provider_type)}
                        </p>
                    </div>
                    <div className="rounded-xl border border-border bg-background/60 p-4">
                        <p className="text-[10px] uppercase tracking-[0.22em] text-muted-foreground">
                            {t("providerRuntime.modelLabel")}
                        </p>
                        <p className="mt-3 text-sm font-semibold text-foreground">
                            {model ?? t("providerRuntime.notConfiguredLabel")}
                        </p>
                    </div>
                    <div className="rounded-xl border border-border bg-background/60 p-4">
                        <p className="text-[10px] uppercase tracking-[0.22em] text-muted-foreground">
                            {t("providerRuntime.updatedLabel")}
                        </p>
                        <p className="mt-3 text-sm font-semibold text-foreground">
                            {updatedAt}
                        </p>
                    </div>
                </div>
            ) : null}

            {!error && !isLoading && !assignment ? (
                <div className="mt-4 rounded-xl border border-amber-500/20 bg-background/60 px-4 py-3 text-sm text-foreground/90">
                    {t(`${roleCopyKey}.missing`)}
                </div>
            ) : null}
        </section>
    );
}
