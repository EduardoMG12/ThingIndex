"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";

import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { fetchIndexingPreferences, saveIndexingPreferences, type IndexingPreferences } from "@/lib/api/indexing-preferences";
import { useI18n } from "@/lib/i18n/provider";

// Cada switch salva na hora; o indexador usa as preferencias do dono do documento.
export default function IndexingScopeCard() {
    const { t } = useI18n();
    const [prefs, setPrefs] = useState<IndexingPreferences | null>(null);
    const [salvando, setSalvando] = useState(false);

    useEffect(() => {
        const controller = new AbortController();
        fetchIndexingPreferences(controller.signal)
            .then(setPrefs)
            .catch((erro: unknown) => {
                if (!controller.signal.aborted) {
                    toast.error(erro instanceof Error ? erro.message : t("settings.indexingScope.error"));
                }
            });
        return () => controller.abort();
    }, [t]);

    const alternar = async (chave: keyof IndexingPreferences, valor: boolean) => {
        if (!prefs) {
            return;
        }
        const anterior = prefs;
        const proximo = { ...prefs, [chave]: valor };
        setPrefs(proximo);
        setSalvando(true);
        try {
            setPrefs(await saveIndexingPreferences(proximo));
            toast.success(t("settings.indexingScope.saved"));
        } catch (erro) {
            setPrefs(anterior);
            toast.error(erro instanceof Error ? erro.message : t("settings.indexingScope.error"));
        } finally {
            setSalvando(false);
        }
    };

    const itens: Array<{ chave: keyof IndexingPreferences; titulo: string; dica: string }> = [
        { chave: "documentSynopsis", titulo: t("settings.indexingScope.document"), dica: t("settings.indexingScope.documentHint") },
        { chave: "pageSummaries", titulo: t("settings.indexingScope.pages"), dica: t("settings.indexingScope.pagesHint") },
        { chave: "sectionSummaries", titulo: t("settings.indexingScope.sections"), dica: t("settings.indexingScope.sectionsHint") },
    ];

    return (
        <Card className="border-border/80 bg-card/70" data-testid="escopo-do-enriquecimento">
            <CardHeader className="space-y-1">
                <p className="text-[10px] font-bold uppercase tracking-[0.26em] text-primary">{t("settings.indexingScope.kicker")}</p>
                <h3 className="text-base font-semibold">{t("settings.indexingScope.title")}</h3>
                <p className="text-sm text-muted-foreground">{t("settings.indexingScope.description")}</p>
            </CardHeader>
            <CardContent className="space-y-3">
                {itens.map((item) => (
                    <div key={item.chave} className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-background/70 px-3 py-2">
                        <div>
                            <p className="text-sm font-medium text-foreground">{item.titulo}</p>
                            <p className="text-xs text-muted-foreground">{item.dica}</p>
                        </div>
                        <Switch
                            checked={prefs?.[item.chave] ?? true}
                            disabled={!prefs || salvando}
                            onCheckedChange={(valor) => void alternar(item.chave, valor)}
                            aria-label={item.titulo}
                            data-testid={`escopo-${item.chave}`}
                        />
                    </div>
                ))}
            </CardContent>
        </Card>
    );
}
