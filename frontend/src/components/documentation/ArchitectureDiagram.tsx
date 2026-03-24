"use client";

import { Fragment } from "react";
import { ArrowDown, ArrowRight } from "lucide-react";

import type { DocumentationLayer } from "@/components/documentation/page-content";
import { useI18n } from "@/lib/i18n/provider";

export function ArchitectureDiagram({
    items,
}: {
    items: DocumentationLayer[];
}) {
    const { locale } = useI18n();
    const eyebrow = locale === "pt-BR"
        ? "Fluxo principal do runtime"
        : locale === "es"
            ? "Flujo principal del runtime"
            : "Main runtime flow";

    return (
        <div className="mt-6 rounded-2xl border border-border bg-background/50 p-5">
            <p className="text-[10px] font-bold uppercase tracking-[0.22em] text-primary">{eyebrow}</p>

            <div className="mt-4 flex min-w-0 flex-col gap-3 lg:flex-row lg:items-stretch">
                {items.map((item, index) => (
                    <Fragment key={item.label}>
                        <div className="min-w-0 flex-1 rounded-2xl border border-border bg-card/70 p-4 shadow-[0_20px_40px_-30px_rgba(0,0,0,0.7)]">
                            <div className="flex items-center gap-3">
                                <span className="rounded-full border border-primary/20 bg-primary/10 px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-primary">
                                    {String(index + 1).padStart(2, "0")}
                                </span>
                                <h4 className="min-w-0 text-sm font-semibold text-foreground">{item.label}</h4>
                            </div>

                            <p className="mt-3 break-words text-sm leading-relaxed text-muted-foreground">
                                {item.description}
                            </p>
                        </div>

                        {index < items.length - 1 ? (
                            <div className="flex justify-center text-primary/70 lg:items-center">
                                <ArrowDown className="h-5 w-5 lg:hidden" />
                                <ArrowRight className="hidden h-5 w-5 lg:block" />
                            </div>
                        ) : null}
                    </Fragment>
                ))}
            </div>
        </div>
    );
}
