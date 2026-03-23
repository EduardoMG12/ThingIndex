"use client";

import type { ProviderSupport } from "@/lib/api/providers";
import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";
import { ProviderSupportBadge } from "@/components/providers/ProviderSupportBadge";
import { formatProviderSupportSetupLabel, formatProviderSupportTransport } from "@/lib/provider-support";

export function ProviderSupportMatrix({
    items,
}: {
    items: ProviderSupport[];
}) {
    if (items.length === 0) {
        return (
            <div className="rounded-xl border border-border bg-background/50 p-5 text-sm text-muted-foreground">
                The provider support matrix is unavailable right now.
            </div>
        );
    }

    return (
        <div className="overflow-x-auto rounded-xl border border-border bg-background/50">
            <table className="w-full min-w-[900px] text-left text-sm">
                <thead className="border-b border-border bg-muted/20">
                    <tr>
                        <th className="px-4 py-3 font-semibold text-foreground">Provider</th>
                        <th className="px-4 py-3 font-semibold text-foreground">Support</th>
                        <th className="px-4 py-3 font-semibold text-foreground">Transport</th>
                        <th className="px-4 py-3 font-semibold text-foreground">Suggested setup</th>
                        <th className="px-4 py-3 font-semibold text-foreground">Operational notes</th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-border/60">
                    {items.map((item) => {
                        const suggestedSetup = item.suggested_setup ?? {};

                        return (
                            <tr key={item.provider_type} className="align-top">
                                <td className="px-4 py-4">
                                    <div className="flex items-center gap-3">
                                        <ProviderBrandMark providerType={item.provider_type} size="sm" />
                                        <div>
                                            <p className="font-semibold text-foreground">{item.display_name}</p>
                                            <p className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">{item.provider_type}</p>
                                        </div>
                                    </div>
                                </td>
                                <td className="px-4 py-4">
                                    <div className="space-y-2">
                                        <ProviderSupportBadge level={item.support_level} />
                                        <p className="max-w-xs text-xs leading-relaxed text-muted-foreground">{item.summary}</p>
                                    </div>
                                </td>
                                <td className="px-4 py-4 text-sm text-foreground">
                                    {formatProviderSupportTransport(item.transport)}
                                </td>
                                <td className="px-4 py-4 text-sm text-foreground">
                                    {Object.keys(suggestedSetup).length > 0 ? (
                                        <div className="space-y-1">
                                            {Object.entries(suggestedSetup).map(([key, value]) => (
                                                <p key={key}>
                                                    {formatProviderSupportSetupLabel(key)}: <span className="font-medium">{value}</span>
                                                </p>
                                            ))}
                                        </div>
                                    ) : (
                                        <span className="text-muted-foreground">Bring your own deployment</span>
                                    )}
                                </td>
                                <td className="px-4 py-4">
                                    <ul className="space-y-2 text-xs leading-relaxed text-muted-foreground">
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
    );
}
