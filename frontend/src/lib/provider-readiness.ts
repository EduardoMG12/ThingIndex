import type { ProviderReadinessCheck, ProviderReadinessReport } from "@/lib/api/providers";
import type { ProviderDefinition } from "@/components/settings/provider-catalog";

type Translate = (key: string, variables?: Record<string, string | number>) => string;

export function getProviderReadinessSummary(t: Translate, provider: ProviderDefinition, report: ProviderReadinessReport | null) {
    if (!report) {
        return t("settings.readiness.idle");
    }

    const discoveryCheck = report.checks.find((check) => check.code === "MODEL_DISCOVERY");
    if (report.status === "READY" && discoveryCheck?.status === "PASSED") {
        return t("settings.readiness.summaryReadyWithDiscovery", { name: provider.name, count: report.discovered_models.length });
    }
    if (report.status === "READY") {
        return t("settings.readiness.summaryReady", { name: provider.name });
    }
    if (report.status === "FAILED") {
        return t("settings.readiness.summaryFailed", { name: provider.name });
    }

    return t("settings.readiness.summaryAttention", { name: provider.name });
}

export function getProviderReadinessStatusLabel(t: Translate, status: ProviderReadinessReport["status"] | "IDLE") {
    switch (status) {
        case "READY":
            return t("settings.readiness.status.ready");
        case "ATTENTION":
            return t("settings.readiness.status.attention");
        case "FAILED":
            return t("settings.readiness.status.failed");
        default:
            return t("settings.readiness.status.idle");
    }
}

export function getProviderReadinessStatusClass(status: ProviderReadinessReport["status"] | "IDLE") {
    switch (status) {
        case "READY":
            return "border-emerald-500/30 bg-emerald-500/10 text-emerald-200";
        case "ATTENTION":
            return "border-amber-500/30 bg-amber-500/10 text-amber-200";
        case "FAILED":
            return "border-destructive/30 bg-destructive/10 text-destructive";
        default:
            return "border-border bg-background/70 text-muted-foreground";
    }
}

export function getProviderReadinessCheckLabel(t: Translate, check: ProviderReadinessCheck) {
    switch (check.code) {
        case "CREDENTIALS":
            return t("settings.readiness.checks.credentials.label");
        case "CONNECTION":
            return t("settings.readiness.checks.connection.label");
        case "MODEL_DISCOVERY":
            return t("settings.readiness.checks.modelDiscovery.label");
        default:
            return check.code;
    }
}

export function getProviderReadinessCheckMessage(t: Translate, provider: ProviderDefinition, report: ProviderReadinessReport, check: ProviderReadinessCheck) {
    switch (check.code) {
        case "CREDENTIALS":
            return t("settings.readiness.checks.credentials.passed", { name: provider.name });
        case "CONNECTION":
            return provider.family === "LLM"
                ? t("settings.readiness.checks.connection.llmPassed", { name: provider.name })
                : t("settings.readiness.checks.connection.storagePassed", { name: provider.name });
        case "MODEL_DISCOVERY":
            switch (check.status) {
                case "PASSED":
                    return t("settings.readiness.checks.modelDiscovery.passed", { count: report.discovered_models.length || 0 });
                case "WARNING":
                    return t("settings.readiness.checks.modelDiscovery.warning");
                case "UNSUPPORTED":
                    return t("settings.readiness.checks.modelDiscovery.unsupported");
                default:
                    return t("settings.readiness.checks.modelDiscovery.warning");
            }
        default:
            return "";
    }
}

export function getProviderReadinessCheckClass(check: ProviderReadinessCheck) {
    switch (check.status) {
        case "PASSED":
            return "border-emerald-500/20 bg-emerald-500/10";
        case "WARNING":
            return "border-amber-500/20 bg-amber-500/10";
        case "FAILED":
            return "border-destructive/20 bg-destructive/10";
        case "UNSUPPORTED":
            return "border-slate-500/20 bg-slate-500/10";
        default:
            return "border-border/80 bg-background/70";
    }
}
