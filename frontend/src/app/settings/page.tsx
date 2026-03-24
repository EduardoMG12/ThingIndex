"use client";

import { useEffect, useMemo, useState, useTransition } from "react";
import { motion } from "framer-motion";
import { Activity, Bot, CheckCircle2, Database, History, KeyRound, Loader2, ServerCog, ShieldCheck } from "lucide-react";
import { toast } from "sonner";

import ProviderModelField from "@/components/settings/ProviderModelField";
import ProviderReadinessPanel from "@/components/settings/ProviderReadinessPanel";
import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";
import { ProviderSupportBadge } from "@/components/providers/ProviderSupportBadge";
import { ProviderSupportPanel } from "@/components/providers/ProviderSupportPanel";
import {
    getInitialFormState,
    getInitialLLMSelectionState,
    getLocalizedProviders,
    PROVIDERS,
    type ProviderDefinition,
} from "@/components/settings/provider-catalog";
import AppShell from "@/components/layout/AppShell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";
import {
    fetchProviders,
    fetchProviderSupportMatrix,
    listProviderModels,
    saveProvider,
    setProviderEnabled,
    testProviderConnection,
    type ProviderConfig,
    type ProviderFamily,
    type ProviderReadinessReport,
    type ProviderSupport,
    type ProviderType
} from "@/lib/api/providers";

type ActivationTarget = "STORAGE" | "INDEXING" | "CHAT";
type LLMSelectionState = Record<ProviderType, { enabledForIndexing: boolean; enabledForChat: boolean }>;

function getActiveProviderTypes(data: ProviderConfig[]) {
    const activeStorage = data.find((provider) => provider.family === "STORAGE" && provider.is_enabled);
    const activeLLM = data.find((provider) => provider.family === "LLM" && provider.enabled_for_indexing)
        ?? data.find((provider) => provider.family === "LLM" && provider.enabled_for_chat);

    return {
        activeStorageType: activeStorage?.provider_type,
        activeLLMType: activeLLM?.provider_type,
    };
}

const FAMILY_ICON: Record<ProviderFamily, typeof Database> = {
    STORAGE: Database,
    LLM: Bot,
};

function hasSavedConfig(configs: ProviderConfig[], family: ProviderFamily, providerType: ProviderType) {
    return configs.some((config) => config.family === family && config.provider_type === providerType);
}

function isStorageEnabled(configs: ProviderConfig[], providerType: ProviderType) {
    return configs.some((config) => config.family === "STORAGE" && config.provider_type === providerType && config.is_enabled);
}

function isLLMEnabledForIndexing(configs: ProviderConfig[], providerType: ProviderType) {
    return configs.some((config) => config.family === "LLM" && config.provider_type === providerType && config.enabled_for_indexing);
}

function isLLMEnabledForChat(configs: ProviderConfig[], providerType: ProviderType) {
    return configs.some((config) => config.family === "LLM" && config.provider_type === providerType && config.enabled_for_chat);
}

function getSavedConfigs(configs: ProviderConfig[], family: ProviderFamily) {
    return configs
        .filter((config) => config.family === family)
        .sort((left, right) => {
            const leftScore = Number(left.is_enabled) + Number(left.enabled_for_indexing) + Number(left.enabled_for_chat);
            const rightScore = Number(right.is_enabled) + Number(right.enabled_for_indexing) + Number(right.enabled_for_chat);
            return rightScore - leftScore;
        });
}

function mergeModelSuggestions(staticSuggestions: string[] = [], discoveredSuggestions: string[] = []) {
    return [...new Set([...discoveredSuggestions, ...staticSuggestions].map((item) => item.trim()).filter(Boolean))];
}

function FamilySection({
    family,
    providers,
    configs,
    activeProviderType,
    onSelect,
    forms,
    llmSelections,
    onFieldChange,
    onLLMSelectionChange,
    onSave,
    onLoadModels,
    onTestConnection,
    onToggle,
    isPending,
    isLoadingModels,
    isTestingProvider,
    discoveredModels,
    readinessReports,
    providerSupportByType,
}: {
    family: ProviderFamily;
    providers: ProviderDefinition[];
    configs: ProviderConfig[];
    activeProviderType: ProviderType;
    onSelect: (providerType: ProviderType) => void;
    forms: Record<ProviderType, Record<string, string>>;
    llmSelections: LLMSelectionState;
    onFieldChange: (providerType: ProviderType, key: string, value: string) => void;
    onLLMSelectionChange: (providerType: ProviderType, key: "enabledForIndexing" | "enabledForChat", value: boolean) => void;
    onSave: (provider: ProviderDefinition) => void;
    onLoadModels: (provider: ProviderDefinition) => void;
    onTestConnection: (provider: ProviderDefinition) => void;
    onToggle: (provider: ProviderDefinition, target: ActivationTarget, enabled: boolean) => void;
    isPending: boolean;
    isLoadingModels: boolean;
    isTestingProvider: boolean;
    discoveredModels: Partial<Record<ProviderType, string[]>>;
    readinessReports: Partial<Record<ProviderType, ProviderReadinessReport | null>>;
    providerSupportByType: Partial<Record<ProviderType, ProviderSupport>>;
}) {
    const { locale, t } = useI18n();
    const familyCopy = {
        title: t(`settings.family.${family}.title`),
        description: t(`settings.family.${family}.description`),
    };
    const FamilyIcon = FAMILY_ICON[family];
    const activeProvider = providers.find((provider) => provider.providerType === activeProviderType) ?? providers[0];
    const activeProviderSupport = providerSupportByType[activeProvider.providerType] ?? null;
    const savedConfigs = getSavedConfigs(configs, family);
    const discoveredModelsForActiveProvider = discoveredModels[activeProvider.providerType] ?? [];
    const readinessReport = readinessReports[activeProvider.providerType] ?? null;
    const supportDisclosureTitle = locale === "pt-BR"
        ? "Perfil avançado do runtime"
        : locale === "es"
            ? "Perfil avanzado del runtime"
            : "Advanced runtime profile";
    const supportDisclosureDescription = locale === "pt-BR"
        ? "Abra este bloco para revisar compatibilidade, transporte e configuração sugerida do provider."
        : locale === "es"
            ? "Abre este bloque para revisar compatibilidad, transporte y configuración sugerida del provider."
            : "Open this block to review compatibility, transport, and the provider suggested setup.";

    return (
        <section className="space-y-6">
            <div className="flex items-start justify-between gap-4">
                <div>
                    <div className="mb-2 flex items-center gap-2">
                        <FamilyIcon className="h-4 w-4 text-primary" />
                        <span className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">{familyCopy.title}</span>
                    </div>
                    <h2 className="text-2xl font-semibold tracking-tight">{familyCopy.title}</h2>
                    <p className="mt-2 max-w-2xl text-sm text-muted-foreground">{familyCopy.description}</p>
                </div>
                <div className="rounded-full border border-border bg-background/70 px-3 py-1 text-[10px] font-mono uppercase tracking-[0.24em] text-muted-foreground">
                    {family === "LLM" ? t("settings.llmTargetsTitle") : t("common.oneActive")}
                </div>
            </div>

            <div className="grid gap-4 xl:grid-cols-[0.95fr_1.05fr]">
                <Card className="flex h-[680px] flex-col border-border/80 bg-card/75">
                    <CardHeader className="space-y-3">
                        <p className="text-[10px] font-bold uppercase tracking-[0.26em] text-primary">{t("settings.steps.select")}</p>
                        <h3 className="text-lg font-semibold">{family === "LLM" ? t("settings.selectTitleLLM") : t("settings.selectTitleStorage")}</h3>
                        <p className="text-sm text-muted-foreground">
                            {t("settings.selectDescription")}
                        </p>
                    </CardHeader>
                    <CardContent className="grid flex-1 auto-rows-max gap-3 overflow-y-auto pr-2 custom-scrollbar">
                        {providers.map((provider) => {
                            const support = providerSupportByType[provider.providerType] ?? null;
                            const saved = hasSavedConfig(configs, family, provider.providerType);
                            const enabled = family === "STORAGE"
                                ? isStorageEnabled(configs, provider.providerType)
                                : isLLMEnabledForIndexing(configs, provider.providerType) || isLLMEnabledForChat(configs, provider.providerType);
                            const selected = activeProviderType === provider.providerType;
                            const llmIndexingActive = family === "LLM" && isLLMEnabledForIndexing(configs, provider.providerType);
                            const llmChatActive = family === "LLM" && isLLMEnabledForChat(configs, provider.providerType);

                            return (
                                <motion.button
                                    key={provider.providerType}
                                    type="button"
                                    whileHover={{ y: -2 }}
                                    whileTap={{ scale: 0.995 }}
                                    onClick={() => onSelect(provider.providerType)}
                                    className={cn(
                                        "relative overflow-hidden rounded-2xl border p-4 text-left transition-colors",
                                        selected ? "border-primary/50 bg-primary/10" : "border-border bg-background/55 hover:border-primary/20"
                                    )}
                                >
                                    <div className={cn("absolute inset-0 bg-gradient-to-br opacity-100", provider.accentClass)} />
                                    <div className="relative flex items-start gap-3">
                                        <ProviderBrandMark providerType={provider.providerType} size="sm" />
                                        <div className="min-w-0">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <h4 className="text-sm font-semibold">{provider.name}</h4>
                                                {saved ? (
                                                    <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-emerald-400">
                                                        <CheckCircle2 className="h-3 w-3" />
                                                        {t("common.saved")}
                                                    </span>
                                                ) : null}
                                                {support ? (
                                                    <ProviderSupportBadge level={support.support_level} />
                                                ) : null}
                                                {enabled ? (
                                                    <span className="rounded-full border border-primary/20 bg-primary/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-primary">
                                                        {t("common.active")}
                                                    </span>
                                                ) : null}
                                                {llmIndexingActive ? (
                                                    <span className="rounded-full border border-sky-500/20 bg-sky-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-sky-300">
                                                        {t("settings.llmIndexingLabel")}
                                                    </span>
                                                ) : null}
                                                {llmChatActive ? (
                                                    <span className="rounded-full border border-fuchsia-500/20 bg-fuchsia-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-fuchsia-300">
                                                        {t("settings.llmChatLabel")}
                                                    </span>
                                                ) : null}
                                            </div>
                                            <p className="mt-1 text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">{provider.tagline}</p>
                                            <p className="mt-2 text-sm leading-relaxed text-foreground/80">{provider.description}</p>
                                        </div>
                                    </div>
                                </motion.button>
                            );
                        })}
                    </CardContent>
                </Card>

                <Card glow className="flex h-[680px] flex-col border-primary/15 bg-card/85">
                    <CardHeader className="space-y-3">
                        <div className="flex items-center gap-3">
                            <ProviderBrandMark providerType={activeProvider.providerType} size="md" />
                            <div>
                                <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">{t("settings.steps.fill")}</p>
                                <h3 className="mt-1 text-lg font-semibold">{activeProvider.name}</h3>
                            </div>
                        </div>
                        <p className="text-sm leading-relaxed text-muted-foreground">{activeProvider.description}</p>
                    </CardHeader>
                    <CardContent className="flex min-h-0 flex-1 flex-col gap-4">
                        <div className="flex-1 space-y-4 overflow-y-auto pr-2 custom-scrollbar">
                            <details className="group rounded-2xl border border-border bg-background/60 p-4">
                                <summary className="flex cursor-pointer list-none items-start justify-between gap-4">
                                    <div>
                                        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                                            {supportDisclosureTitle}
                                        </p>
                                        <p className="mt-2 text-sm leading-relaxed text-foreground/85">
                                            {supportDisclosureDescription}
                                        </p>
                                    </div>
                                    <span className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground transition group-open:text-primary">
                                        {locale === "pt-BR" ? "Abrir" : locale === "es" ? "Abrir" : "Open"}
                                    </span>
                                </summary>
                                <div className="mt-4">
                                    <ProviderSupportPanel support={activeProviderSupport} />
                                </div>
                            </details>

                            {activeProvider.fields.map((field) => (
                                field.kind === "model" ? (
                                    <ProviderModelField
                                        key={field.key}
                                        providerType={activeProvider.providerType}
                                        label={field.label}
                                        optional={field.optional}
                                        placeholder={field.placeholder}
                                        value={forms[activeProvider.providerType]?.[field.key] ?? ""}
                                        suggestions={mergeModelSuggestions(field.suggestions, discoveredModelsForActiveProvider)}
                                        onChange={(value) => onFieldChange(activeProvider.providerType, field.key, value)}
                                    />
                                ) : (
                                    <div key={field.key} className="space-y-2">
                                        <Label htmlFor={`${activeProvider.providerType}-${field.key}`}>
                                            {field.label}
                                            {field.optional ? (
                                                <span className="ml-2 text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{t("common.optional")}</span>
                                            ) : null}
                                        </Label>
                                        {field.type === "textarea" ? (
                                            <Textarea
                                                id={`${activeProvider.providerType}-${field.key}`}
                                                value={forms[activeProvider.providerType]?.[field.key] ?? ""}
                                                onChange={(event) => onFieldChange(activeProvider.providerType, field.key, event.target.value)}
                                                placeholder={field.placeholder}
                                                rows={6}
                                            />
                                        ) : (
                                            <Input
                                                id={`${activeProvider.providerType}-${field.key}`}
                                                type={field.type || "text"}
                                                value={forms[activeProvider.providerType]?.[field.key] ?? ""}
                                                onChange={(event) => onFieldChange(activeProvider.providerType, field.key, event.target.value)}
                                                placeholder={field.placeholder}
                                                autoComplete="off"
                                            />
                                        )}
                                    </div>
                                )
                            ))}

                            {family === "LLM" ? (
                                <div className="rounded-2xl border border-border bg-background/60 p-4">
                                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.llmTargetsTitle")}</p>
                                    <p className="mt-2 text-sm leading-relaxed text-foreground/85">{t("settings.llmTargetsDescription")}</p>
                                    <div className="mt-4 space-y-3">
                                        <div className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-background/70 px-3 py-2">
                                            <div>
                                                <p className="text-sm font-medium text-foreground">{t("settings.llmIndexingLabel")}</p>
                                                <p className="text-xs text-muted-foreground">{t("settings.llmActiveForIndexing")}</p>
                                            </div>
                                            <Switch
                                                checked={llmSelections[activeProvider.providerType]?.enabledForIndexing ?? false}
                                                disabled={isPending}
                                                onCheckedChange={(checked) => onLLMSelectionChange(activeProvider.providerType, "enabledForIndexing", checked)}
                                                aria-label={t("settings.llmSelectForIndexing")}
                                            />
                                        </div>
                                        <div className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-background/70 px-3 py-2">
                                            <div>
                                                <p className="text-sm font-medium text-foreground">{t("settings.llmChatLabel")}</p>
                                                <p className="text-xs text-muted-foreground">{t("settings.llmActiveForChat")}</p>
                                            </div>
                                            <Switch
                                                checked={llmSelections[activeProvider.providerType]?.enabledForChat ?? false}
                                                disabled={isPending}
                                                onCheckedChange={(checked) => onLLMSelectionChange(activeProvider.providerType, "enabledForChat", checked)}
                                                aria-label={t("settings.llmSelectForChat")}
                                            />
                                        </div>
                                    </div>
                                </div>
                            ) : null}

                            <ProviderReadinessPanel
                                provider={activeProvider}
                                report={readinessReport}
                                discoveredModels={discoveredModelsForActiveProvider}
                                isLoadingModels={isPending || isLoadingModels}
                                onLoadModels={onLoadModels}
                            />

                            <div className="rounded-2xl border border-border bg-background/60 p-4">
                                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.behaviorTitle")}</p>
                                <p className="mt-2 text-sm leading-relaxed text-foreground/85">
                                    {family === "LLM" ? t("settings.behaviorDescriptionLLM") : t("settings.behaviorDescription")}
                                </p>
                            </div>
                        </div>

                        <div className={cn("grid shrink-0 gap-3", family === "LLM" ? "md:grid-cols-3" : "md:grid-cols-2")}>
                            {family === "LLM" ? (
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={() => onLoadModels(activeProvider)}
                                    disabled={isPending || isLoadingModels}
                                    className="gap-2"
                                >
                                    {isLoadingModels ? <Loader2 className="h-4 w-4 animate-spin" /> : <History className="h-4 w-4" />}
                                    {t("settings.readiness.loadModels")}
                                </Button>
                            ) : null}
                            <Button
                                type="button"
                                variant="outline"
                                onClick={() => onTestConnection(activeProvider)}
                                disabled={isPending || isTestingProvider}
                                className="gap-2"
                            >
                                {isTestingProvider ? <Loader2 className="h-4 w-4 animate-spin" /> : <Activity className="h-4 w-4" />}
                                {t("settings.testConnection")}
                            </Button>
                            <Button
                                type="button"
                                onClick={() => onSave(activeProvider)}
                                disabled={isPending}
                                className="gap-2"
                            >
                                {isPending && !isTestingProvider ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />}
                                {t("common.saveAndEnable", { name: activeProvider.name })}
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            </div>

            <Card className="border-border/80 bg-card/70">
                <CardHeader className="space-y-3">
                    <p className="text-[10px] font-bold uppercase tracking-[0.26em] text-primary">{t("settings.steps.saved")}</p>
                    <h3 className="text-lg font-semibold">{family === "LLM" ? t("settings.savedTitleLLM") : t("settings.savedTitleStorage")}</h3>
                    <p className="text-sm text-muted-foreground">
                        {t("settings.savedDescription")}
                    </p>
                </CardHeader>
                <CardContent className="grid auto-rows-max gap-3 md:max-h-[320px] md:overflow-y-auto md:pr-2 custom-scrollbar">
                    {savedConfigs.length > 0 ? (
                        savedConfigs.map((config) => {
                            const provider = providers.find((item) => item.providerType === config.provider_type);
                            if (!provider) {
                                return null;
                            }

                            const support = providerSupportByType[provider.providerType] ?? null;

                            return (
                                <div
                                    key={config.id}
                                    className="flex flex-col gap-4 rounded-2xl border border-border bg-background/60 p-4 md:flex-row md:items-center md:justify-between"
                                >
                                    <div className="flex items-start gap-3">
                                        <ProviderBrandMark providerType={provider.providerType} size="sm" />
                                        <div>
                                            <div className="flex flex-wrap items-center gap-2">
                                                <p className="text-sm font-semibold">{provider.name}</p>
                                                {support ? (
                                                    <ProviderSupportBadge level={support.support_level} />
                                                ) : null}
                                                {family === "STORAGE" && config.is_enabled ? (
                                                    <span className="rounded-full border border-primary/20 bg-primary/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-primary">
                                                        {t("common.active")}
                                                    </span>
                                                ) : family === "STORAGE" ? (
                                                    <span className="rounded-full border border-border bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                                                        {t("common.inactive")}
                                                    </span>
                                                ) : null}
                                                {family === "LLM" && config.enabled_for_indexing ? (
                                                    <span className="rounded-full border border-sky-500/20 bg-sky-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-sky-300">
                                                        {t("settings.llmIndexingLabel")}
                                                    </span>
                                                ) : null}
                                                {family === "LLM" && config.enabled_for_chat ? (
                                                    <span className="rounded-full border border-fuchsia-500/20 bg-fuchsia-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-fuchsia-300">
                                                        {t("settings.llmChatLabel")}
                                                    </span>
                                                ) : null}
                                            </div>
                                            <p className="mt-1 text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
                                                {provider.tagline}
                                            </p>
                                            {family === "LLM" && config.public_metadata.model ? (
                                                <p className="mt-2 text-xs text-foreground/80">
                                                    {t("settings.savedModelLabel")}: <span className="font-medium text-foreground">{config.public_metadata.model}</span>
                                                </p>
                                            ) : null}
                                            <button
                                                type="button"
                                                onClick={() => onSelect(provider.providerType)}
                                                className="mt-2 text-xs font-medium text-primary hover:underline"
                                            >
                                                {t("common.editConfig")}
                                            </button>
                                        </div>
                                    </div>

                                    {family === "STORAGE" ? (
                                        <div className="flex items-center gap-3 self-end md:self-auto">
                                            <span className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">
                                                {config.is_enabled ? t("common.enabled") : t("common.disabled")}
                                            </span>
                                            <Switch
                                                checked={config.is_enabled}
                                                disabled={isPending}
                                                onCheckedChange={(checked) => onToggle(provider, "STORAGE", checked)}
                                                aria-label={
                                                    locale === "pt-BR"
                                                        ? `Alternar ativação de ${provider.name}`
                                                        : locale === "es"
                                                            ? `Cambiar activación de ${provider.name}`
                                                            : `Toggle ${provider.name} activation`
                                                }
                                            />
                                        </div>
                                    ) : (
                                        <div className="grid min-w-[260px] gap-3 self-end md:self-auto">
                                            <div className="flex items-center justify-between gap-3">
                                                <span className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">
                                                    {config.enabled_for_indexing ? t("settings.llmActiveForIndexing") : t("settings.llmSelectForIndexing")}
                                                </span>
                                                <Button
                                                    type="button"
                                                    size="sm"
                                                    variant={config.enabled_for_indexing ? "primary" : "outline"}
                                                    disabled={isPending || config.enabled_for_indexing}
                                                    onClick={() => onToggle(provider, "INDEXING", true)}
                                                    className="min-w-[138px]"
                                                >
                                                    {config.enabled_for_indexing ? t("settings.llmActiveForIndexing") : t("settings.llmSelectForIndexing")}
                                                </Button>
                                            </div>
                                            <div className="flex items-center justify-between gap-3">
                                                <span className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">
                                                    {config.enabled_for_chat ? t("settings.llmActiveForChat") : t("settings.llmSelectForChat")}
                                                </span>
                                                <Button
                                                    type="button"
                                                    size="sm"
                                                    variant={config.enabled_for_chat ? "primary" : "outline"}
                                                    disabled={isPending || config.enabled_for_chat}
                                                    onClick={() => onToggle(provider, "CHAT", true)}
                                                    className="min-w-[138px]"
                                                >
                                                    {config.enabled_for_chat ? t("settings.llmActiveForChat") : t("settings.llmSelectForChat")}
                                                </Button>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            );
                        })
                    ) : (
                        <div className="rounded-2xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                            {family === "LLM" ? t("settings.emptyLLM") : t("settings.emptyStorage")}
                        </div>
                    )}
                </CardContent>
            </Card>
        </section>
    );
}

export default function SettingsPage() {
    const { locale, t } = useI18n();
    const [providers, setProviders] = useState<ProviderConfig[]>([]);
    const [discoveredModels, setDiscoveredModels] = useState<Partial<Record<ProviderType, string[]>>>({});
    const [readinessReports, setReadinessReports] = useState<Partial<Record<ProviderType, ProviderReadinessReport | null>>>({});
    const [testingProviderType, setTestingProviderType] = useState<ProviderType | null>(null);
    const [loadingModelsProviderType, setLoadingModelsProviderType] = useState<ProviderType | null>(null);
    const [forms, setForms] = useState<Record<ProviderType, Record<string, string>>>(getInitialFormState);
    const [llmSelections, setLLMSelections] = useState<LLMSelectionState>(getInitialLLMSelectionState);
    const [activeStorageType, setActiveStorageType] = useState<ProviderType>("MINIO");
    const [activeLLMType, setActiveLLMType] = useState<ProviderType>("OPENAI");
    const [visibleFamily, setVisibleFamily] = useState<ProviderFamily>("STORAGE");
    const [providerSupportByType, setProviderSupportByType] = useState<Partial<Record<ProviderType, ProviderSupport>>>({});
    const [isLoading, setIsLoading] = useState(true);
    const [loadWarning, setLoadWarning] = useState<string | null>(null);
    const [isPending, startTransition] = useTransition();
    const localizedProviders = useMemo(() => getLocalizedProviders(locale), [locale]);
    const storageProviders = useMemo(
        () => localizedProviders.filter((provider) => provider.family === "STORAGE"),
        [localizedProviders]
    );
    const llmProviders = useMemo(
        () => localizedProviders.filter((provider) => provider.family === "LLM"),
        [localizedProviders]
    );

    const activeStorageConfig = providers.find((provider) => provider.family === "STORAGE" && provider.is_enabled) ?? null;
    const activeIndexingConfig = providers.find((provider) => provider.family === "LLM" && provider.enabled_for_indexing) ?? null;
    const activeChatConfig = providers.find((provider) => provider.family === "LLM" && provider.enabled_for_chat) ?? null;
    const savedProviderCount = providers.length;
    const savedStorageCount = providers.filter((provider) => provider.family === "STORAGE").length;
    const savedLLMCount = providers.filter((provider) => provider.family === "LLM").length;

    function findProviderLabel(providerType?: ProviderType | null) {
        return localizedProviders.find((provider) => provider.providerType === providerType)?.name ?? providerType ?? null;
    }

    function toProviderSupportMap(items: ProviderSupport[]) {
        return items.reduce<Partial<Record<ProviderType, ProviderSupport>>>((acc, item) => {
            acc[item.provider_type] = item;
            return acc;
        }, {});
    }

    async function loadAllProviderSupport(signal?: AbortSignal) {
        const [storageSupport, llmSupport] = await Promise.all([
            fetchProviderSupportMatrix({ family: "STORAGE", signal }).catch(() => []),
            fetchProviderSupportMatrix({ family: "LLM", signal }).catch(() => []),
        ]);

        return [...storageSupport, ...llmSupport];
    }

    function applyLoadedProviders(data: ProviderConfig[]) {
        setLoadWarning(null);
        setProviders(data);
        syncLLMSelections(data);

        const { activeStorageType, activeLLMType } = getActiveProviderTypes(data);

        if (activeStorageType) {
            setActiveStorageType(activeStorageType);
        }
        if (activeLLMType) {
            setActiveLLMType(activeLLMType);
        }
    }

    useEffect(() => {
        const controller = new AbortController();

        async function loadProviders() {
            try {
                setIsLoading(true);
                setLoadWarning(null);
                const [providersResult, supportData] = await Promise.allSettled([
                    fetchProviders(controller.signal),
                    loadAllProviderSupport(controller.signal),
                ]);

                if (supportData.status === "fulfilled") {
                    setProviderSupportByType(toProviderSupportMap(supportData.value));
                } else {
                    setProviderSupportByType({});
                }

                if (providersResult.status === "fulfilled") {
                    setLoadWarning(null);
                    setProviders(providersResult.value);
                    syncLLMSelections(providersResult.value);

                    const { activeStorageType, activeLLMType } = getActiveProviderTypes(providersResult.value);
                    if (activeStorageType) {
                        setActiveStorageType(activeStorageType);
                    }
                    if (activeLLMType) {
                        setActiveLLMType(activeLLMType);
                    }
                } else {
                    setProviders([]);
                    syncLLMSelections([]);
                    setLoadWarning(t("settings.partialLoadWarning"));
                }
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }
                setLoadWarning(err instanceof Error ? err.message : t("settings.errors.load"));
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadProviders();
        return () => controller.abort();
    }, [t]);

    function reloadProviders() {
        startTransition(async () => {
            try {
                const [data, supportData] = await Promise.all([
                    fetchProviders(),
                    loadAllProviderSupport(),
                ]);
                applyLoadedProviders(data);
                setProviderSupportByType(toProviderSupportMap(supportData));
                setLoadWarning(null);
            } catch (err) {
                setLoadWarning(t("settings.partialLoadWarning"));
                toast.error(err instanceof Error ? err.message : t("settings.errors.refresh"));
            }
        });
    }

    function handleFieldChange(providerType: ProviderType, key: string, value: string) {
        setForms((current) => ({
            ...current,
            [providerType]: {
                ...current[providerType],
                [key]: value,
            },
        }));
        setReadinessReports((current) => ({
            ...current,
            [providerType]: null,
        }));
    }

    function handleSelectLLMProvider(providerType: ProviderType) {
        setActiveLLMType(providerType);
        const support = providerSupportByType[providerType] ?? null;

        setForms((current) => {
            const nextProviderForm = {
                ...current[providerType],
            };

            let changed = false;

            if (
                support?.suggested_setup.base_url &&
                !nextProviderForm.base_url?.trim() &&
                (providerType === "OLLAMA" || support.capabilities.requires_explicit_base_url)
            ) {
                nextProviderForm.base_url = support.suggested_setup.base_url;
                changed = true;
            }

            if (
                support?.suggested_setup.model &&
                !nextProviderForm.model?.trim() &&
                support.capabilities.requires_explicit_model
            ) {
                nextProviderForm.model = support.suggested_setup.model;
                changed = true;
            }

            if (!changed) {
                return current;
            }

            return {
                ...current,
                [providerType]: nextProviderForm,
            };
        });
    }

    function syncLLMSelections(data: ProviderConfig[]) {
        setLLMSelections((current) => {
            const next = { ...current };
            for (const provider of PROVIDERS) {
                if (provider.family !== "LLM") {
                    continue;
                }
                const config = data.find((item) => item.family === "LLM" && item.provider_type === provider.providerType);
                next[provider.providerType] = {
                    enabledForIndexing: config?.enabled_for_indexing ?? false,
                    enabledForChat: config?.enabled_for_chat ?? false,
                };
            }
            return next;
        });
    }

    function handleLLMSelectionChange(providerType: ProviderType, key: "enabledForIndexing" | "enabledForChat", value: boolean) {
        setLLMSelections((current) => ({
            ...current,
            [providerType]: {
                ...current[providerType],
                [key]: value,
            },
        }));
    }

    function buildCredentials(providerType: ProviderType) {
        return Object.entries(forms[providerType]).reduce<Record<string, string>>((acc, [key, value]) => {
            const trimmed = value.trim();
            if (trimmed) {
                acc[key] = trimmed;
            }
            return acc;
        }, {});
    }

    function handleSave(provider: ProviderDefinition) {
        if (provider.family === "LLM") {
            const selection = llmSelections[provider.providerType];
            if (!selection?.enabledForIndexing && !selection?.enabledForChat) {
                toast.error(t("settings.llmSelectionRequired"));
                return;
            }
        }

        const credentials = buildCredentials(provider.providerType);

        startTransition(async () => {
            try {
                await saveProvider({
                    family: provider.family,
                    providerType: provider.providerType,
                    credentials,
                    isEnabled: provider.family === "STORAGE" ? true : undefined,
                    enabledForIndexing: provider.family === "LLM" ? llmSelections[provider.providerType]?.enabledForIndexing : undefined,
                    enabledForChat: provider.family === "LLM" ? llmSelections[provider.providerType]?.enabledForChat : undefined,
                });
                toast.success(t("settings.providerSavedEnabled", { name: provider.name }));
                const latest = await fetchProviders();
                applyLoadedProviders(latest);
            } catch (err) {
                toast.error(err instanceof Error ? err.message : t("settings.errors.saveProvider"));
            }
        });
    }

    function handleTestConnection(provider: ProviderDefinition) {
        const credentials = buildCredentials(provider.providerType);

        setTestingProviderType(provider.providerType);

        startTransition(async () => {
            try {
                const report = await testProviderConnection({
                    family: provider.family,
                    providerType: provider.providerType,
                    credentials,
                });
                setReadinessReports((current) => ({
                    ...current,
                    [provider.providerType]: report,
                }));
                if (report?.discovered_models?.length) {
                    setDiscoveredModels((current) => ({
                        ...current,
                        [provider.providerType]: report.discovered_models,
                    }));
                }
                toast.success(
                    report?.status === "ATTENTION"
                        ? t("settings.readiness.testedWithAttention", { name: provider.name })
                        : t("settings.providerConnectionValidated", { name: provider.name })
                );
            } catch (err) {
                toast.error(err instanceof Error ? err.message : t("settings.errors.testProvider"));
            } finally {
                setTestingProviderType(null);
            }
        });
    }

    function handleLoadModels(provider: ProviderDefinition) {
        setLoadingModelsProviderType(provider.providerType);

        startTransition(async () => {
            try {
                const models = await listProviderModels({
                    family: provider.family,
                    providerType: provider.providerType,
                    credentials: buildCredentials(provider.providerType),
                });

                setDiscoveredModels((current) => ({
                    ...current,
                    [provider.providerType]: models,
                }));

                toast.success(
                    models.length > 0
                        ? t("settings.readiness.modelsLoaded", { count: models.length, name: provider.name })
                        : t("settings.readiness.noModelsReturned", { name: provider.name })
                );
            } catch (err) {
                toast.error(err instanceof Error ? err.message : t("settings.errors.loadModels"));
            } finally {
                setLoadingModelsProviderType(null);
            }
        });
    }

    function handleToggle(provider: ProviderDefinition, target: ActivationTarget, enabled: boolean) {
        startTransition(async () => {
            try {
                await setProviderEnabled({
                    family: provider.family,
                    providerType: provider.providerType,
                    activationTarget: target,
                    enabled,
                });
                if (provider.family === "LLM" && target === "INDEXING") {
                    toast.success(t("settings.providerSelectedForIndexing", { name: provider.name }));
                } else if (provider.family === "LLM" && target === "CHAT") {
                    toast.success(t("settings.providerSelectedForChat", { name: provider.name }));
                } else {
                    toast.success(enabled
                        ? t("settings.providerEnabled", { name: provider.name })
                        : t("settings.providerDisabled", { name: provider.name }));
                }
                const latest = await fetchProviders();
                applyLoadedProviders(latest);
            } catch (err) {
                toast.error(err instanceof Error ? err.message : t("settings.errors.updateProvider"));
            }
        });
    }

    return (
        <AppShell>
            <div className="space-y-8">
                <motion.section
                    initial={{ opacity: 0, y: 16 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.35 }}
                    className="relative overflow-hidden rounded-[28px] border border-primary/15 bg-card/80 p-8 shadow-card"
                >
                    <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(245,166,35,0.16),transparent_26%),radial-gradient(circle_at_bottom_right,rgba(82,168,255,0.1),transparent_28%)]" />
                    <div className="relative flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
                        <div>
                            <p className="text-[10px] font-bold uppercase tracking-[0.3em] text-primary">{t("settings.heroEyebrow")}</p>
                            <h1 className="mt-3 text-3xl font-semibold tracking-tight">{t("settings.heroTitle")}</h1>
                            <p className="mt-3 max-w-3xl text-sm leading-relaxed text-muted-foreground">
                                {t("settings.heroDescription")}
                            </p>
                        </div>

                        <Button type="button" variant="outline" onClick={reloadProviders} disabled={isPending} className="gap-2 self-start">
                            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <ServerCog className="h-4 w-4" />}
                            {t("common.refresh")}
                        </Button>
                    </div>

                    <div className="relative mt-6 flex max-w-2xl items-start gap-3 rounded-2xl border border-emerald-500/20 bg-emerald-500/10 p-4">
                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-emerald-500/20 bg-background/70">
                            <ShieldCheck className="h-5 w-5 text-emerald-400" />
                        </div>
                        <div>
                            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-emerald-300">
                                {t("settings.securityTitle")}
                            </p>
                            <p className="mt-1 text-sm leading-relaxed text-foreground/85">
                                {t("settings.securityDescription")}
                            </p>
                        </div>
                    </div>

                    <div className="relative mt-6 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                        <div className="rounded-2xl border border-border/80 bg-background/60 p-4">
                            <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.family.STORAGE.title")}</p>
                            <p className="mt-2 text-lg font-semibold text-foreground">
                                {findProviderLabel(activeStorageConfig?.provider_type) ?? t("common.inactive")}
                            </p>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {activeStorageConfig ? t("common.active") : t("common.inactive")}
                            </p>
                        </div>
                        <div className="rounded-2xl border border-border/80 bg-background/60 p-4">
                            <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.llmIndexingLabel")}</p>
                            <p className="mt-2 text-lg font-semibold text-foreground">
                                {findProviderLabel(activeIndexingConfig?.provider_type) ?? t("common.inactive")}
                            </p>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {activeIndexingConfig ? t("common.active") : t("common.inactive")}
                            </p>
                        </div>
                        <div className="rounded-2xl border border-border/80 bg-background/60 p-4">
                            <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.llmChatLabel")}</p>
                            <p className="mt-2 text-lg font-semibold text-foreground">
                                {findProviderLabel(activeChatConfig?.provider_type) ?? t("common.inactive")}
                            </p>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {activeChatConfig ? t("common.active") : t("common.inactive")}
                            </p>
                        </div>
                        <div className="rounded-2xl border border-border/80 bg-background/60 p-4">
                            <p className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("common.saved")}</p>
                            <p className="mt-2 text-lg font-semibold text-foreground">{savedProviderCount}</p>
                            <p className="mt-1 text-xs text-muted-foreground">
                                {savedStorageCount} {t("settings.family.STORAGE.title")} · {savedLLMCount} {t("settings.family.LLM.title")}
                            </p>
                        </div>
                    </div>
                </motion.section>

                {isLoading ? (
                    <div className="flex items-center gap-2 rounded-2xl border border-border bg-card/70 p-5 text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        {t("settings.loading")}
                    </div>
                ) : null}

                {!isLoading && loadWarning ? (
                    <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-5 text-sm text-amber-200">
                        {loadWarning}
                    </div>
                ) : null}

                {!isLoading ? (
                    <Tabs value={visibleFamily} onValueChange={(value) => setVisibleFamily(value as ProviderFamily)} className="w-full">
                        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                            <div className="max-w-2xl">
                                <p className="text-sm text-muted-foreground">
                                    {locale === "pt-BR"
                                        ? "Revise uma família por vez para reduzir ruído visual e concentrar credenciais, diagnóstico e ativações no mesmo fluxo."
                                        : locale === "es"
                                            ? "Revisa una familia por vez para reducir ruido visual y concentrar credenciales, diagnóstico y activaciones en el mismo flujo."
                                            : "Review one family at a time to reduce visual noise and keep credentials, readiness, and activation in the same flow."}
                                </p>
                            </div>
                            <TabsList className="h-auto flex-wrap justify-start gap-2 bg-transparent p-0">
                                <TabsTrigger
                                    value="STORAGE"
                                    className="rounded-full border border-border bg-background/70 px-4 py-2 text-xs uppercase tracking-[0.18em] data-[state=active]:border-primary/40 data-[state=active]:bg-primary/12 data-[state=active]:text-primary"
                                >
                                    {t("settings.family.STORAGE.title")}
                                </TabsTrigger>
                                <TabsTrigger
                                    value="LLM"
                                    className="rounded-full border border-border bg-background/70 px-4 py-2 text-xs uppercase tracking-[0.18em] data-[state=active]:border-primary/40 data-[state=active]:bg-primary/12 data-[state=active]:text-primary"
                                >
                                    {t("settings.family.LLM.title")}
                                </TabsTrigger>
                            </TabsList>
                        </div>

                        <TabsContent value="STORAGE" className="mt-6">
                            <FamilySection
                                family="STORAGE"
                                providers={storageProviders}
                                configs={providers}
                                activeProviderType={activeStorageType}
                                onSelect={setActiveStorageType}
                                forms={forms}
                                llmSelections={llmSelections}
                                onFieldChange={handleFieldChange}
                                onLLMSelectionChange={handleLLMSelectionChange}
                                onSave={handleSave}
                                onLoadModels={handleLoadModels}
                                onTestConnection={handleTestConnection}
                                onToggle={handleToggle}
                                isPending={isPending}
                                isLoadingModels={loadingModelsProviderType === activeStorageType}
                                isTestingProvider={testingProviderType === activeStorageType}
                                discoveredModels={discoveredModels}
                                readinessReports={readinessReports}
                                providerSupportByType={providerSupportByType}
                            />
                        </TabsContent>

                        <TabsContent value="LLM" className="mt-6">
                            <FamilySection
                                family="LLM"
                                providers={llmProviders}
                                configs={providers}
                                activeProviderType={activeLLMType}
                                onSelect={handleSelectLLMProvider}
                                forms={forms}
                                llmSelections={llmSelections}
                                onFieldChange={handleFieldChange}
                                onLLMSelectionChange={handleLLMSelectionChange}
                                onSave={handleSave}
                                onLoadModels={handleLoadModels}
                                onTestConnection={handleTestConnection}
                                onToggle={handleToggle}
                                isPending={isPending}
                                isLoadingModels={loadingModelsProviderType === activeLLMType}
                                isTestingProvider={testingProviderType === activeLLMType}
                                discoveredModels={discoveredModels}
                                readinessReports={readinessReports}
                                providerSupportByType={providerSupportByType}
                            />
                        </TabsContent>
                    </Tabs>
                ) : null}
            </div>
        </AppShell>
    );
}
