"use client";

import type { PricingProviderType } from "@/lib/api/pricing";
import type { ProviderType } from "@/lib/api/providers";
import { getProviderBranding } from "@/lib/provider-branding";
import { cn } from "@/lib/utils";

type ProviderBrandMarkSize = "sm" | "md" | "lg";

const SIZE_CLASSES: Record<
    ProviderBrandMarkSize,
    {
        frame: string;
        icon: string;
        monogram: string;
        inset: string;
    }
> = {
    sm: {
        frame: "h-10 w-10 rounded-2xl",
        icon: "h-4.5 w-4.5",
        monogram: "px-1.5 py-0.5 text-[8px]",
        inset: "inset-[1px] rounded-[15px]",
    },
    md: {
        frame: "h-11 w-11 rounded-[20px]",
        icon: "h-5 w-5",
        monogram: "px-1.5 py-0.5 text-[8px]",
        inset: "inset-[1px] rounded-[19px]",
    },
    lg: {
        frame: "h-12 w-12 rounded-[22px]",
        icon: "h-5.5 w-5.5",
        monogram: "px-2 py-0.5 text-[9px]",
        inset: "inset-[1px] rounded-[21px]",
    },
};

export function ProviderBrandMark({
    providerType,
    size = "md",
    className,
    showMonogram = true,
}: {
    providerType?: ProviderType | PricingProviderType | null;
    size?: ProviderBrandMarkSize;
    className?: string;
    showMonogram?: boolean;
}) {
    const { icon: Icon, accentClass, monogram } = getProviderBranding(providerType);
    const sizeClass = SIZE_CLASSES[size];
    const testIdProviderType = providerType ?? "unknown";

    return (
        <div
            aria-hidden="true"
            data-testid={`provider-brand-${testIdProviderType}`}
            className={cn(
                "relative isolate overflow-hidden border border-white/10 bg-slate-950/90 shadow-[0_18px_36px_-24px_rgba(15,23,42,0.95)]",
                sizeClass.frame,
                className
            )}
        >
            <div className={cn("absolute inset-0 bg-gradient-to-br opacity-95", accentClass)} />
            <div
                className={cn(
                    "absolute bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.24),transparent_45%),linear-gradient(180deg,rgba(15,23,42,0.78),rgba(15,23,42,0.92))]",
                    sizeClass.inset
                )}
            />
            <div className="relative flex h-full items-center justify-center">
                <Icon className={cn("text-white/90 drop-shadow-[0_4px_12px_rgba(15,23,42,0.45)]", sizeClass.icon)} />
            </div>
            {showMonogram ? (
                <span
                    className={cn(
                        "absolute bottom-1.5 right-1.5 rounded-full border border-white/15 bg-slate-950/75 font-mono font-semibold uppercase tracking-[0.18em] text-white/85 backdrop-blur",
                        sizeClass.monogram
                    )}
                >
                    {monogram}
                </span>
            ) : null}
        </div>
    );
}
