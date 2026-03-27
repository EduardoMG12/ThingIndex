"use client";

import { Languages } from "lucide-react";
import { usePathname, useRouter } from "next/navigation";
import { startTransition } from "react";

import {
    getLocaleLabel,
    replacePathLocale,
    SUPPORTED_LOCALES,
    type AppLocale,
} from "@/lib/i18n/config";
import { useI18n } from "@/lib/i18n/provider";
// hidden from the UI, as Spanish translation is not ready yet and we don't want to show a broken experience.
const VISIBLE_LOCALES: AppLocale[] = SUPPORTED_LOCALES.filter((locale) => locale !== "es");

export function LanguageSwitcher() {
    const router = useRouter();
    const pathname = usePathname();
    const { locale } = useI18n();

    function handleChange(nextLocale: AppLocale) {
        startTransition(() => {
            router.replace(replacePathLocale(pathname, nextLocale));
            router.refresh();
        });
    }

    return (
        <div className="flex items-center gap-1 rounded-full border border-border/70 bg-background/55 px-1.5 py-1 backdrop-blur-sm">
            <div className="flex h-7 w-7 items-center justify-center rounded-full text-muted-foreground/70">
                <Languages className="h-3.5 w-3.5" />
            </div>
            <div className="flex items-center rounded-full bg-muted/40 p-0.5">
                {VISIBLE_LOCALES.map((option) => {
                    const active = option === locale;

                    return (
                        <button
                            key={option}
                            type="button"
                            onClick={() => handleChange(option)}
                            className={`rounded-full px-2.5 py-1 text-[10px] font-semibold tracking-[0.16em] transition-all ${active
                                    ? "bg-foreground text-background"
                                    : "text-muted-foreground/80 hover:text-foreground"
                                }`}
                            aria-pressed={active}
                            aria-label={getLocaleLabel(option)}
                        >
                            {option === "pt-BR" ? "PT" : option.toUpperCase()}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
