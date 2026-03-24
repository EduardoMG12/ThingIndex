"use client";

import type { ProviderSupport } from "@/lib/api/providers";
import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";
import { ProviderSupportBadge } from "@/components/providers/ProviderSupportBadge";
import { useI18n } from "@/lib/i18n/provider";
import { formatProviderSupportSetupLabel, formatProviderSupportTransport } from "@/lib/provider-support";

export function ProviderSupportMatrix({
    items,
}: {
    items: ProviderSupport[];
}) {
    const { locale } = useI18n();
    const copy = locale === "pt-BR"
        ? {
            unavailable: "A matriz de suporte dos provedores não está disponível agora.",
            headers: {
                provider: "Provedor",
                support: "Suporte",
                transport: "Transporte",
                setup: "Configuração sugerida",
                notes: "Notas operacionais",
            },
            setupFallback: "Use sua própria implantação",
        }
        : locale === "es"
            ? {
                unavailable: "La matriz de soporte de providers no está disponible ahora.",
                headers: {
                    provider: "Provider",
                    support: "Soporte",
                    transport: "Transporte",
                    setup: "Setup sugerido",
                    notes: "Notas operativas",
                },
                setupFallback: "Usa tu propio despliegue",
            }
            : {
                unavailable: "The provider support matrix is unavailable right now.",
                headers: {
                    provider: "Provider",
                    support: "Support",
                    transport: "Transport",
                    setup: "Suggested setup",
                    notes: "Operational notes",
                },
                setupFallback: "Bring your own deployment",
            };
    const responsiveCellClass = "grid gap-2 rounded-xl border border-border/60 bg-background/70 px-4 py-3 before:block before:text-[10px] before:font-bold before:uppercase before:tracking-[0.18em] before:text-muted-foreground before:content-[attr(data-label)] 2xl:table-cell 2xl:rounded-none 2xl:border-0 2xl:bg-transparent 2xl:px-4 2xl:py-4 2xl:align-top 2xl:before:content-none";

    if (items.length === 0) {
        return (
            <div className="rounded-xl border border-border bg-background/50 p-5 text-sm text-muted-foreground">
                {copy.unavailable}
            </div>
        );
    }

    return (
        <div className="overflow-hidden rounded-xl border border-border bg-background/50">
            <div className="overflow-x-auto custom-scrollbar">
                <table className="w-full text-left text-sm 2xl:min-w-[1180px] 2xl:table-fixed">
                    <colgroup className="hidden 2xl:table-column-group">
                        <col className="2xl:w-[18%]" />
                        <col className="2xl:w-[20%]" />
                        <col className="2xl:w-[15%]" />
                        <col className="2xl:w-[23%]" />
                        <col className="2xl:w-[24%]" />
                    </colgroup>
                    <thead className="hidden border-b border-border bg-muted/20 2xl:table-header-group">
                        <tr>
                            <th className="px-4 py-3 font-semibold text-foreground">{copy.headers.provider}</th>
                            <th className="px-4 py-3 font-semibold text-foreground">{copy.headers.support}</th>
                            <th className="px-4 py-3 font-semibold text-foreground">{copy.headers.transport}</th>
                            <th className="px-4 py-3 font-semibold text-foreground">{copy.headers.setup}</th>
                            <th className="px-4 py-3 font-semibold text-foreground">{copy.headers.notes}</th>
                        </tr>
                    </thead>
                    <tbody className="grid gap-4 p-4 2xl:table-row-group 2xl:divide-y 2xl:divide-border/60 2xl:p-0">
                        {items.map((item) => {
                            const suggestedSetup = item.suggested_setup ?? {};
                            const suggestedSetupEntries = Object.entries(suggestedSetup);

                            return (
                                <tr
                                    key={item.provider_type}
                                    className="grid gap-4 rounded-2xl border border-border/70 bg-card/60 p-4 shadow-[0_20px_40px_-28px_rgba(0,0,0,0.6)] 2xl:table-row 2xl:rounded-none 2xl:border-0 2xl:bg-transparent 2xl:p-0 2xl:shadow-none"
                                >
                                    <td data-label={copy.headers.provider} className={responsiveCellClass}>
                                        <div className="flex items-center gap-3">
                                            <ProviderBrandMark providerType={item.provider_type} size="sm" />
                                            <div>
                                                <p className="font-semibold text-foreground">{item.display_name}</p>
                                                <p className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">{item.provider_type}</p>
                                            </div>
                                        </div>
                                    </td>
                                    <td data-label={copy.headers.support} className={responsiveCellClass}>
                                        <div className="space-y-2">
                                            <ProviderSupportBadge level={item.support_level} className="w-max" />
                                            <p className="max-w-sm text-xs leading-relaxed text-muted-foreground">{item.summary}</p>
                                        </div>
                                    </td>
                                    <td data-label={copy.headers.transport} className={responsiveCellClass}>
                                        {formatProviderSupportTransport(locale, item.transport)}
                                    </td>
                                    <td data-label={copy.headers.setup} className={responsiveCellClass}>
                                        {suggestedSetupEntries.length > 0 ? (
                                            <div className="space-y-2 text-sm text-foreground">
                                                {suggestedSetupEntries.map(([key, value]) => (
                                                    <p key={key} className="leading-relaxed">
                                                        {formatProviderSupportSetupLabel(locale, key)}: <span className="font-medium break-all">{String(value)}</span>
                                                    </p>
                                                ))}
                                            </div>
                                        ) : (
                                            <span className="text-sm text-muted-foreground">
                                                {copy.setupFallback}
                                            </span>
                                        )}
                                    </td>
                                    <td data-label={copy.headers.notes} className={responsiveCellClass}>
                                        <ul className="space-y-2 list-disc pl-4 text-xs leading-relaxed text-muted-foreground marker:text-muted-foreground/60">
                                            {item.notes.map((note) => (
                                                <li key={note}>{note}</li>
                                            ))}
                                        </ul>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        </div>
    );
}
