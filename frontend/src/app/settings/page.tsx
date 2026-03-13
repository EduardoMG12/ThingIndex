"use client";

import { useEffect, useState, useTransition } from "react";
import { motion } from "framer-motion";
import { Bot, CheckCircle2, Database, KeyRound, Loader2, ServerCog, ShieldCheck, Sparkles } from "lucide-react";
import { toast } from "sonner";

import AppShell from "@/components/layout/AppShell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useI18n } from "@/lib/i18n/provider";
import { cn } from "@/lib/utils";
import {
    fetchProviders,
    saveProvider,
    setProviderEnabled,
    type ProviderConfig,
    type ProviderFamily,
    type ProviderType,
} from "@/lib/api/providers";

type ProviderField = {
    key: string;
    label: string;
    placeholder: string;
    type?: "text" | "password";
    optional?: boolean;
};

type ProviderDefinition = {
    family: ProviderFamily;
    providerType: ProviderType;
    name: string;
    tagline: string;
    description: string;
    icon: typeof Database;
    accentClass: string;
    fields: ProviderField[];
};

const PROVIDERS: ProviderDefinition[] = [
    {
        family: "STORAGE",
        providerType: "MINIO",
        name: "MinIO / S3-compatible",
        tagline: "Object storage for document binaries",
        description: "Configure endpoint, access key, secret key and bucket for uploads, previews and indexing.",
        icon: Database,
        accentClass: "from-amber-500/20 via-orange-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "minio.example.com:9000" },
            { key: "access_key", label: "Access key", placeholder: "Access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "use_ssl", label: "Use SSL", placeholder: "true or false", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "AWS_S3",
        name: "AWS S3",
        tagline: "Managed Amazon S3 buckets",
        description: "Use an S3 bucket with region-based routing. You can also override the endpoint for custom-compatible gateways.",
        icon: ServerCog,
        accentClass: "from-orange-500/20 via-amber-500/10 to-transparent",
        fields: [
            { key: "access_key", label: "Access key", placeholder: "AWS access key" },
            { key: "secret_key", label: "Secret key", placeholder: "AWS secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "my-documents-bucket" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "CLOUDFLARE_R2",
        name: "Cloudflare R2",
        tagline: "R2 object storage",
        description: "Connect your Cloudflare R2 account by account ID, access key, secret key and bucket.",
        icon: ShieldCheck,
        accentClass: "from-sky-500/20 via-blue-500/10 to-transparent",
        fields: [
            { key: "account_id", label: "Account ID", placeholder: "Cloudflare account ID" },
            { key: "access_key", label: "Access key", placeholder: "R2 access key" },
            { key: "secret_key", label: "Secret key", placeholder: "R2 secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
            { key: "endpoint", label: "Custom endpoint", placeholder: "Optional override", optional: true },
        ],
    },
    {
        family: "STORAGE",
        providerType: "DIGITALOCEAN_SPACES",
        name: "DigitalOcean Spaces",
        tagline: "Managed S3-compatible object storage",
        description: "Configure your Spaces endpoint, region, keys and bucket to use DigitalOcean as the active document store.",
        icon: Database,
        accentClass: "from-cyan-500/20 via-sky-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "nyc3.digitaloceanspaces.com" },
            { key: "region", label: "Region", placeholder: "nyc3" },
            { key: "access_key", label: "Access key", placeholder: "Spaces access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Spaces secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "STORAGE",
        providerType: "WASABI",
        name: "Wasabi",
        tagline: "Low-cost S3-compatible storage",
        description: "Connect a Wasabi bucket with the regional endpoint, keys and bucket name.",
        icon: ServerCog,
        accentClass: "from-lime-500/20 via-emerald-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-east-1.wasabisys.com" },
            { key: "region", label: "Region", placeholder: "us-east-1" },
            { key: "access_key", label: "Access key", placeholder: "Wasabi access key" },
            { key: "secret_key", label: "Secret key", placeholder: "Wasabi secret key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "STORAGE",
        providerType: "BACKBLAZE_B2",
        name: "Backblaze B2",
        tagline: "B2 S3-compatible buckets",
        description: "Use Backblaze B2 through its S3-compatible endpoint with region, keys and bucket name.",
        icon: ShieldCheck,
        accentClass: "from-rose-500/20 via-red-500/10 to-transparent",
        fields: [
            { key: "endpoint", label: "Endpoint", placeholder: "s3.us-west-004.backblazeb2.com" },
            { key: "region", label: "Region", placeholder: "us-west-004" },
            { key: "access_key", label: "Access key", placeholder: "B2 key ID" },
            { key: "secret_key", label: "Secret key", placeholder: "B2 application key", type: "password" },
            { key: "bucket_name", label: "Bucket name", placeholder: "thingindex-docs" },
        ],
    },
    {
        family: "LLM",
        providerType: "OPENAI",
        name: "OpenAI",
        tagline: "Default OpenAI-compatible setup",
        description: "Ideal when you want the standard OpenAI API flow with optional custom model/base URL.",
        icon: Bot,
        accentClass: "from-emerald-500/20 via-teal-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "sk-...", type: "password" },
            { key: "model", label: "Model", placeholder: "gpt-4o-mini", optional: true },
            { key: "base_url", label: "Base URL", placeholder: "https://api.openai.com/v1", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "GOOGLE_GEMINI",
        name: "Google Gemini",
        tagline: "Gemini through an OpenAI-compatible bridge",
        description: "Use when your Gemini endpoint is exposed behind an OpenAI-compatible API surface.",
        icon: Sparkles,
        accentClass: "from-sky-500/20 via-cyan-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Gemini API key", type: "password" },
            { key: "model", label: "Model", placeholder: "gemini-1.5-pro", optional: true },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "DEEP_SEEK",
        name: "DeepSeek",
        tagline: "Reasoning-focused LLM provider",
        description: "Configure your DeepSeek API key with optional model and base URL overrides.",
        icon: ServerCog,
        accentClass: "from-indigo-500/20 via-blue-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "DeepSeek API key", type: "password" },
            { key: "model", label: "Model", placeholder: "deepseek-chat", optional: true },
            { key: "base_url", label: "Base URL", placeholder: "https://api.deepseek.com", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "GROK",
        name: "Grok",
        tagline: "xAI Grok models",
        description: "Connect Grok through the xAI API or a compatible proxy.",
        icon: Sparkles,
        accentClass: "from-fuchsia-500/20 via-pink-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "xAI API key", type: "password" },
            { key: "model", label: "Model", placeholder: "grok-2-latest", optional: true },
            { key: "base_url", label: "Base URL", placeholder: "https://api.x.ai/v1", optional: true },
        ],
    },
    {
        family: "LLM",
        providerType: "CLAUDE",
        name: "Claude",
        tagline: "Anthropic or compatible bridge",
        description: "Use an OpenAI-compatible bridge for Claude-compatible deployments.",
        icon: ShieldCheck,
        accentClass: "from-violet-500/20 via-purple-500/10 to-transparent",
        fields: [
            { key: "api_key", label: "API key", placeholder: "Anthropic API key", type: "password" },
            { key: "model", label: "Model", placeholder: "claude-3-5-sonnet", optional: true },
            { key: "base_url", label: "Base URL", placeholder: "OpenAI-compatible endpoint", optional: true },
        ],
    },
];

const FAMILY_COPY: Record<ProviderFamily, { title: string; description: string; icon: typeof Database }> = {
    STORAGE: {
        title: "Storage Provider",
        description: "Exactly one storage backend can stay enabled at a time for uploads, previews and indexing.",
        icon: Database,
    },
    LLM: {
        title: "LLM Provider",
        description: "Exactly one LLM can stay enabled at a time for enrichment and reasoning workloads.",
        icon: Bot,
    },
};

function getProvidersByFamily(family: ProviderFamily) {
    return PROVIDERS.filter((provider) => provider.family === family);
}

function getInitialFormState() {
    return PROVIDERS.reduce<Record<ProviderType, Record<string, string>>>((state, provider) => {
        state[provider.providerType] = provider.fields.reduce<Record<string, string>>((fieldState, field) => {
            fieldState[field.key] = "";
            return fieldState;
        }, {});
        return state;
    }, {} as Record<ProviderType, Record<string, string>>);
}

function hasSavedConfig(configs: ProviderConfig[], family: ProviderFamily, providerType: ProviderType) {
    return configs.some((config) => config.family === family && config.provider_type === providerType);
}

function isEnabled(configs: ProviderConfig[], family: ProviderFamily, providerType: ProviderType) {
    return configs.some((config) => config.family === family && config.provider_type === providerType && config.is_enabled);
}

function getSavedConfigs(configs: ProviderConfig[], family: ProviderFamily) {
    return configs
        .filter((config) => config.family === family)
        .sort((left, right) => Number(right.is_enabled) - Number(left.is_enabled));
}

function FamilySection({
    family,
    providers,
    configs,
    activeProviderType,
    onSelect,
    forms,
    onFieldChange,
    onSave,
    onToggle,
    isPending,
}: {
    family: ProviderFamily;
    providers: ProviderDefinition[];
    configs: ProviderConfig[];
    activeProviderType: ProviderType;
    onSelect: (providerType: ProviderType) => void;
    forms: Record<ProviderType, Record<string, string>>;
    onFieldChange: (providerType: ProviderType, key: string, value: string) => void;
    onSave: (provider: ProviderDefinition) => void;
    onToggle: (provider: ProviderDefinition, enabled: boolean) => void;
    isPending: boolean;
}) {
    const { t } = useI18n();
    const familyCopy = {
        ...FAMILY_COPY[family],
        title: t(`settings.family.${family}.title`),
        description: t(`settings.family.${family}.description`),
    };
    const FamilyIcon = familyCopy.icon;
    const activeProvider = providers.find((provider) => provider.providerType === activeProviderType) ?? providers[0];
    const ActiveProviderIcon = activeProvider.icon;
    const savedConfigs = getSavedConfigs(configs, family);

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
                    {t("common.oneActive")}
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
                            const ProviderIcon = provider.icon;
                            const saved = hasSavedConfig(configs, family, provider.providerType);
                            const enabled = isEnabled(configs, family, provider.providerType);
                            const selected = activeProviderType === provider.providerType;

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
                                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-primary/20 bg-background/80">
                                            <ProviderIcon className="h-4.5 w-4.5 text-primary" />
                                        </div>
                                        <div className="min-w-0">
                                            <div className="flex flex-wrap items-center gap-2">
                                                <h4 className="text-sm font-semibold">{provider.name}</h4>
                                                {saved ? (
                                                    <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-emerald-400">
                                                        <CheckCircle2 className="h-3 w-3" />
                                                        {t("common.saved")}
                                                    </span>
                                                ) : null}
                                                {enabled ? (
                                                    <span className="rounded-full border border-primary/20 bg-primary/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-primary">
                                                        {t("common.active")}
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
                            <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-primary/20 bg-background/70">
                                <ActiveProviderIcon className="h-5 w-5 text-primary" />
                            </div>
                            <div>
                                <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">{t("settings.steps.fill")}</p>
                                <h3 className="mt-1 text-lg font-semibold">{activeProvider.name}</h3>
                            </div>
                        </div>
                        <p className="text-sm leading-relaxed text-muted-foreground">{activeProvider.description}</p>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        {activeProvider.fields.map((field) => (
                            <div key={field.key} className="space-y-2">
                                <Label htmlFor={`${activeProvider.providerType}-${field.key}`}>
                                    {field.label}
                                    {field.optional ? (
                                        <span className="ml-2 text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{t("common.optional")}</span>
                                    ) : null}
                                </Label>
                                <Input
                                    id={`${activeProvider.providerType}-${field.key}`}
                                    type={field.type || "text"}
                                    value={forms[activeProvider.providerType]?.[field.key] ?? ""}
                                    onChange={(event) => onFieldChange(activeProvider.providerType, field.key, event.target.value)}
                                    placeholder={field.placeholder}
                                    autoComplete="off"
                                />
                            </div>
                        ))}

                        <div className="rounded-2xl border border-border bg-background/60 p-4">
                            <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">{t("settings.behaviorTitle")}</p>
                            <p className="mt-2 text-sm leading-relaxed text-foreground/85">
                                {t("settings.behaviorDescription")}
                            </p>
                        </div>

                        <Button
                            type="button"
                            onClick={() => onSave(activeProvider)}
                            disabled={isPending}
                            className="w-full gap-2"
                        >
                            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />}
                            {t("common.saveAndEnable", { name: activeProvider.name })}
                        </Button>
                    </CardContent>
                </Card>
            </div>

            <Card className="flex h-[460px] flex-col border-border/80 bg-card/70">
                <CardHeader className="space-y-3">
                    <p className="text-[10px] font-bold uppercase tracking-[0.26em] text-primary">{t("settings.steps.saved")}</p>
                    <h3 className="text-lg font-semibold">{family === "LLM" ? t("settings.savedTitleLLM") : t("settings.savedTitleStorage")}</h3>
                    <p className="text-sm text-muted-foreground">
                        {t("settings.savedDescription")}
                    </p>
                </CardHeader>
                <CardContent className="grid flex-1 auto-rows-max gap-3 overflow-y-auto pr-2 custom-scrollbar">
                    {savedConfigs.length > 0 ? (
                        savedConfigs.map((config) => {
                            const provider = providers.find((item) => item.providerType === config.provider_type);
                            if (!provider) {
                                return null;
                            }

                            const ProviderIcon = provider.icon;

                            return (
                                <div
                                    key={config.id}
                                    className="flex flex-col gap-4 rounded-2xl border border-border bg-background/60 p-4 md:flex-row md:items-center md:justify-between"
                                >
                                    <div className="flex items-start gap-3">
                                        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-primary/20 bg-background/80">
                                            <ProviderIcon className="h-4.5 w-4.5 text-primary" />
                                        </div>
                                        <div>
                                            <div className="flex flex-wrap items-center gap-2">
                                                <p className="text-sm font-semibold">{provider.name}</p>
                                                {config.is_enabled ? (
                                                    <span className="rounded-full border border-primary/20 bg-primary/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-primary">
                                                        {t("common.active")}
                                                    </span>
                                                ) : (
                                                    <span className="rounded-full border border-border bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                                                        {t("common.inactive")}
                                                    </span>
                                                )}
                                            </div>
                                            <p className="mt-1 text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
                                                {provider.tagline}
                                            </p>
                                            <button
                                                type="button"
                                                onClick={() => onSelect(provider.providerType)}
                                                className="mt-2 text-xs font-medium text-primary hover:underline"
                                            >
                                                {t("common.editConfig")}
                                            </button>
                                        </div>
                                    </div>

                                    <div className="flex items-center gap-3 self-end md:self-auto">
                                        <span className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">
                                            {config.is_enabled ? t("common.enabled") : t("common.disabled")}
                                        </span>
                                        <Switch
                                            checked={config.is_enabled}
                                            disabled={isPending}
                                            onCheckedChange={(checked) => onToggle(provider, checked)}
                                            aria-label={`Toggle ${provider.name}`}
                                        />
                                    </div>
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
    const { t } = useI18n();
    const [providers, setProviders] = useState<ProviderConfig[]>([]);
    const [forms, setForms] = useState<Record<ProviderType, Record<string, string>>>(getInitialFormState);
    const [activeStorageType, setActiveStorageType] = useState<ProviderType>("MINIO");
    const [activeLLMType, setActiveLLMType] = useState<ProviderType>("OPENAI");
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isPending, startTransition] = useTransition();

    useEffect(() => {
        const controller = new AbortController();

        async function loadProviders() {
            try {
                setIsLoading(true);
                setError(null);
                const data = await fetchProviders(controller.signal);
                setProviders(data);

                const activeStorage = data.find((provider) => provider.family === "STORAGE" && provider.is_enabled);
                const activeLLM = data.find((provider) => provider.family === "LLM" && provider.is_enabled);

                if (activeStorage) {
                    setActiveStorageType(activeStorage.provider_type);
                }
                if (activeLLM) {
                    setActiveLLMType(activeLLM.provider_type);
                }
            } catch (err) {
                if (controller.signal.aborted) {
                    return;
                }
                setError(err instanceof Error ? err.message : "Nao foi possivel carregar as configuracoes.");
            } finally {
                if (!controller.signal.aborted) {
                    setIsLoading(false);
                }
            }
        }

        void loadProviders();
        return () => controller.abort();
    }, []);

    function reloadProviders() {
        startTransition(async () => {
            try {
                const data = await fetchProviders();
                setProviders(data);
            } catch (err) {
                toast.error(err instanceof Error ? err.message : "Could not refresh providers.");
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
    }

    function handleSave(provider: ProviderDefinition) {
        const credentials = Object.entries(forms[provider.providerType]).reduce<Record<string, string>>((acc, [key, value]) => {
            const trimmed = value.trim();
            if (trimmed) {
                acc[key] = trimmed;
            }
            return acc;
        }, {});

        startTransition(async () => {
            try {
                await saveProvider({
                    family: provider.family,
                    providerType: provider.providerType,
                    credentials,
                });
                toast.success(`${provider.name} saved and enabled.`);
                const latest = await fetchProviders();
                setProviders(latest);
            } catch (err) {
                toast.error(err instanceof Error ? err.message : "Nao foi possivel salvar o provider.");
            }
        });
    }

    function handleToggle(provider: ProviderDefinition, enabled: boolean) {
        startTransition(async () => {
            try {
                await setProviderEnabled({
                    family: provider.family,
                    providerType: provider.providerType,
                    isEnabled: enabled,
                });
                toast.success(enabled ? `${provider.name} enabled.` : `${provider.name} disabled.`);
                const latest = await fetchProviders();
                setProviders(latest);
            } catch (err) {
                toast.error(err instanceof Error ? err.message : "Nao foi possivel atualizar o provider.");
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
                </motion.section>

                {isLoading ? (
                    <div className="flex items-center gap-2 rounded-2xl border border-border bg-card/70 p-5 text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        {t("settings.loading")}
                    </div>
                ) : null}

                {!isLoading && error ? (
                    <div className="rounded-2xl border border-destructive/30 bg-destructive/10 p-5 text-sm text-destructive">
                        {error}
                    </div>
                ) : null}

                {!isLoading && !error ? (
                    <div className="space-y-10">
                        <FamilySection
                            family="STORAGE"
                            providers={getProvidersByFamily("STORAGE")}
                            configs={providers}
                            activeProviderType={activeStorageType}
                            onSelect={setActiveStorageType}
                            forms={forms}
                            onFieldChange={handleFieldChange}
                            onSave={handleSave}
                            onToggle={handleToggle}
                            isPending={isPending}
                        />

                        <FamilySection
                            family="LLM"
                            providers={getProvidersByFamily("LLM")}
                            configs={providers}
                            activeProviderType={activeLLMType}
                            onSelect={setActiveLLMType}
                            forms={forms}
                            onFieldChange={handleFieldChange}
                            onSave={handleSave}
                            onToggle={handleToggle}
                            isPending={isPending}
                        />
                    </div>
                ) : null}
            </div>
        </AppShell>
    );
}
