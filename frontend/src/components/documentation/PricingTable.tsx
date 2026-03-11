"use client";

import { MODELS, EMBEDDINGS } from "./constants";

export function PricingTable() {
    return (
        <div className="space-y-8">
            <div>
                <h4 className="mb-4 text-xs font-bold uppercase tracking-widest text-primary flex items-center gap-2">
                    <span className="h-1 w-1 rounded-full bg-primary" />
                    Modelos de Linguagem (por 1M tokens)
                </h4>
                <div className="overflow-x-auto rounded-lg border border-border bg-background/50">
                    <table className="w-full text-[11px]">
                        <thead>
                            <tr className="border-b border-border bg-muted/20">
                                <th className="py-3 pr-3 pl-4 text-left font-semibold text-foreground">Modelo</th>
                                <th className="py-3 px-3 text-left font-semibold text-muted-foreground">Provider</th>
                                <th className="py-3 px-3 text-right font-semibold text-muted-foreground">Input</th>
                                <th className="py-3 px-3 text-right font-semibold text-muted-foreground">Output</th>
                                <th className="py-3 px-3 text-left font-semibold text-muted-foreground">Contexto</th>
                                <th className="py-3 pl-3 pr-4 text-left font-semibold text-muted-foreground">Observação</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/50">
                            {MODELS.map((m, i) => (
                                <tr key={i} className="hover:bg-muted/10 transition-colors">
                                    <td className="py-2.5 pr-3 pl-4 font-bold text-foreground font-mono">{m.name}</td>
                                    <td className="py-2 px-3 text-muted-foreground/80">{m.provider}</td>
                                    <td className="py-2 px-3 text-right font-mono text-emerald-500 font-semibold">{m.input}</td>
                                    <td className="py-2 px-3 text-right font-mono text-blue-500 font-semibold">{m.output}</td>
                                    <td className="py-2 px-3 font-mono text-muted-foreground/70">{m.context}</td>
                                    <td className="py-2 pl-3 pr-4 text-muted-foreground italic text-[10px]">{m.notes}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>

            <div>
                <h4 className="mb-4 text-xs font-bold uppercase tracking-widest text-primary flex items-center gap-2">
                    <span className="h-1 w-1 rounded-full bg-primary" />
                    Modelos de Embedding
                </h4>
                <div className="overflow-x-auto rounded-lg border border-border bg-background/50">
                    <table className="w-full text-[11px]">
                        <thead>
                            <tr className="border-b border-border bg-muted/20">
                                <th className="py-3 pr-3 pl-4 text-left font-semibold text-foreground">Modelo</th>
                                <th className="py-3 px-3 text-left font-semibold text-muted-foreground">Provider</th>
                                <th className="py-3 px-3 text-right font-semibold text-muted-foreground">Preço</th>
                                <th className="py-3 px-3 text-left font-semibold text-muted-foreground">Dimensões</th>
                                <th className="py-3 pl-3 pr-4 text-left font-semibold text-muted-foreground">Obs</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/50">
                            {EMBEDDINGS.map((e, i) => (
                                <tr key={i} className="hover:bg-muted/10 transition-colors">
                                    <td className="py-2.5 pr-3 pl-4 font-bold text-foreground font-mono">{e.name}</td>
                                    <td className="py-2 px-3 text-muted-foreground/80">{e.provider}</td>
                                    <td className="py-2 px-3 text-right font-mono text-emerald-500 font-semibold">{e.price}</td>
                                    <td className="py-2 px-3 font-mono text-muted-foreground/70">{e.dims}</td>
                                    <td className="py-2 pl-3 pr-4 text-muted-foreground italic text-[10px]">{e.notes}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
}
