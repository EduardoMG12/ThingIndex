"use client";

import { FEATURES, type DocumentationFeature } from "./constants";

export function ComparisonTable() {
    return (
        <div className="overflow-x-auto rounded-lg border border-border bg-background/50">
            <table className="w-full text-xs">
                <thead>
                    <tr className="border-b border-border bg-muted/20">
                        <th className="py-3 pr-4 pl-4 text-left font-semibold text-foreground">Característica</th>
                        <th className="py-3 px-4 text-left font-semibold text-muted-foreground">Baseline comum</th>
                        <th className="py-3 pl-4 pr-4 text-left font-semibold text-primary">ThingIndex</th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-border/50">
                    {FEATURES.map((f: DocumentationFeature, i: number) => (
                        <tr key={i} className="hover:bg-muted/10 transition-colors">
                            <td className="py-2.5 pr-4 pl-4 font-medium text-foreground">{f.feature}</td>
                            <td className="py-2.5 px-4 text-muted-foreground">{f.rag}</td>
                            <td className="py-2.5 pl-4 pr-4 text-foreground font-medium">{f.thingindex}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
