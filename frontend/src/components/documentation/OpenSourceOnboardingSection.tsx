import { ExternalLink, Globe2, TerminalSquare, Wrench } from "lucide-react";

import {
    getDocumentationResources,
    getDocumentationTracks,
    getOpenSourceTroubleshootingItems,
} from "@/components/documentation/open-source-onboarding";
import { useI18n } from "@/lib/i18n/provider";

export function OpenSourceOnboardingSection({
    apiBaseUrl,
}: {
    apiBaseUrl?: string;
}) {
    const { locale } = useI18n();
    const resources = getDocumentationResources(apiBaseUrl, locale);
    const tracks = getDocumentationTracks(apiBaseUrl, locale);
    const troubleshootingItems = getOpenSourceTroubleshootingItems(locale);
    const copy = locale === "pt-BR"
        ? {
            intro:
                "O projeto agora documenta explicitamente os dois caminhos válidos para quem chega de fora: rodar o stack completo a partir do GitHub ou consumir uma API já publicada usando o mesmo contrato HTTP e a mesma referência interativa.",
            openLink: "Abrir link",
            step: "Passo",
            notes: "Notas",
            troubleshooting: "Resolução de problemas",
        }
        : locale === "es"
            ? {
                intro:
                    "El proyecto ahora documenta explícitamente los dos caminos válidos para quien llega desde fuera: correr el stack completo desde GitHub o consumir una API ya publicada usando el mismo contrato HTTP y la misma referencia interactiva.",
                openLink: "Abrir enlace",
                step: "Paso",
                notes: "Notas",
                troubleshooting: "Resolución de problemas",
            }
            : {
                intro:
                    "The project now documents the two valid paths for outside consumers explicitly: run the full stack from GitHub or consume an already published API using the same HTTP contract and interactive reference.",
                openLink: "Open link",
                step: "Step",
                notes: "Notes",
                troubleshooting: "Troubleshooting",
            };

    return (
        <div className="space-y-8">
            <div>
                <p>{copy.intro}</p>

                <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {resources.map((resource) => (
                        <article key={resource.title} className="min-w-0 rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{resource.title}</h4>
                            <p className="mt-2 break-words text-sm leading-relaxed text-muted-foreground">{resource.description}</p>
                            {resource.href ? (
                                <a
                                    href={resource.href}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="mt-3 inline-flex items-center gap-2 text-sm font-semibold text-primary hover:underline"
                                >
                                    {copy.openLink}
                                    <ExternalLink className="h-3.5 w-3.5" />
                                </a>
                            ) : null}
                            {resource.pathHint ? (
                                <p className="mt-3 break-all">
                                    <code>{resource.pathHint}</code>
                                </p>
                            ) : null}
                        </article>
                    ))}
                </div>
            </div>

            <div className="grid gap-5 xl:grid-cols-2">
                {tracks.map((track) => (
                    <div key={track.title} className="min-w-0 rounded-xl border border-border bg-background/50 p-5">
                        <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.2em] text-primary">
                            {track.kind === "hosted" ? (
                                <Globe2 className="h-4 w-4" />
                            ) : (
                                <TerminalSquare className="h-4 w-4" />
                            )}
                            {track.badge}
                        </div>
                        <h3 className="mt-3 text-base font-semibold text-foreground">{track.title}</h3>
                        <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{track.description}</p>

                        <div className="mt-4 space-y-4">
                            {track.steps.map((step, index) => (
                                <div key={`${track.title}-${step.command}`} className="min-w-0 rounded-xl border border-border bg-card px-4 py-4">
                                    <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">
                                        {copy.step} {index + 1}
                                    </p>
                                    <h4 className="mt-2 text-sm font-semibold text-foreground">{step.title}</h4>
                                    <p className="mt-2 break-all">
                                        <code>{step.command}</code>
                                    </p>
                                    <p className="mt-2 break-words text-sm text-muted-foreground">{step.detail}</p>
                                </div>
                            ))}
                        </div>

                        <div className="mt-4 rounded-xl border border-border bg-card px-4 py-4">
                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">
                                {copy.notes}
                            </p>
                            <ul className="mt-3 space-y-2 break-words text-sm leading-relaxed text-muted-foreground">
                                {track.notes.map((note) => (
                                    <li key={note}>{note}</li>
                                ))}
                            </ul>
                        </div>
                    </div>
                ))}
            </div>

            <div className="rounded-xl border border-border bg-background/50 p-5">
                <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.2em] text-primary">
                    <Wrench className="h-4 w-4" />
                    {copy.troubleshooting}
                </div>
                <div className="mt-4 grid gap-4 xl:grid-cols-2">
                    {troubleshootingItems.map((item) => (
                        <div key={item.title} className="min-w-0 rounded-xl border border-border bg-card px-4 py-4">
                            <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                            <p className="mt-2 break-words text-sm leading-relaxed text-muted-foreground">{item.detail}</p>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}
