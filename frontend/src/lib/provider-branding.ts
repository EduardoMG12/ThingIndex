import {
    Bot,
    BrainCircuit,
    Database,
    ServerCog,
    ShieldCheck,
    Sparkles,
    type LucideIcon,
} from "lucide-react";

import type { PricingProviderType } from "@/lib/api/pricing";
import type { ProviderType } from "@/lib/api/providers";

export type ProviderBrandKey = ProviderType | PricingProviderType;

export type ProviderBranding = {
    icon: LucideIcon;
    accentClass: string;
    monogram: string;
};

const FALLBACK_BRANDING: ProviderBranding = {
    icon: Bot,
    accentClass: "from-slate-500/20 via-slate-400/10 to-transparent",
    monogram: "AI",
};

const PROVIDER_BRANDING: Record<ProviderType, ProviderBranding> = {
    MINIO: {
        icon: Database,
        accentClass: "from-amber-500/20 via-orange-500/10 to-transparent",
        monogram: "M",
    },
    AWS_S3: {
        icon: ServerCog,
        accentClass: "from-orange-500/20 via-amber-500/10 to-transparent",
        monogram: "S3",
    },
    CLOUDFLARE_R2: {
        icon: ShieldCheck,
        accentClass: "from-sky-500/20 via-blue-500/10 to-transparent",
        monogram: "R2",
    },
    DIGITALOCEAN_SPACES: {
        icon: Database,
        accentClass: "from-cyan-500/20 via-sky-500/10 to-transparent",
        monogram: "DO",
    },
    WASABI: {
        icon: ServerCog,
        accentClass: "from-lime-500/20 via-emerald-500/10 to-transparent",
        monogram: "WS",
    },
    BACKBLAZE_B2: {
        icon: ShieldCheck,
        accentClass: "from-rose-500/20 via-red-500/10 to-transparent",
        monogram: "B2",
    },
    GCP_STORAGE: {
        icon: ShieldCheck,
        accentClass: "from-emerald-500/20 via-teal-500/10 to-transparent",
        monogram: "GC",
    },
    OPENAI: {
        icon: Bot,
        accentClass: "from-emerald-500/20 via-teal-500/10 to-transparent",
        monogram: "OA",
    },
    GOOGLE_GEMINI: {
        icon: Sparkles,
        accentClass: "from-sky-500/20 via-cyan-500/10 to-transparent",
        monogram: "GM",
    },
    DEEP_SEEK: {
        icon: BrainCircuit,
        accentClass: "from-indigo-500/20 via-blue-500/10 to-transparent",
        monogram: "DS",
    },
    GROK: {
        icon: Sparkles,
        accentClass: "from-fuchsia-500/20 via-pink-500/10 to-transparent",
        monogram: "GX",
    },
    CLAUDE: {
        icon: ShieldCheck,
        accentClass: "from-violet-500/20 via-purple-500/10 to-transparent",
        monogram: "CL",
    },
    OLLAMA: {
        icon: ServerCog,
        accentClass: "from-orange-500/20 via-yellow-500/10 to-transparent",
        monogram: "OL",
    },
    OPENROUTER: {
        icon: Bot,
        accentClass: "from-blue-500/20 via-indigo-500/10 to-transparent",
        monogram: "OR",
    },
};

export function getProviderBranding(providerType?: ProviderBrandKey | null): ProviderBranding {
    if (!providerType) {
        return FALLBACK_BRANDING;
    }

    return PROVIDER_BRANDING[providerType as ProviderType] ?? FALLBACK_BRANDING;
}
