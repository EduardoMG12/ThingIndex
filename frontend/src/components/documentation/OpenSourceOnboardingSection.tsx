import { ExternalLink, Globe2, TerminalSquare, Wrench } from "lucide-react";

import {
    OPEN_SOURCE_TROUBLESHOOTING_ITEMS,
    getDocumentationResources,
    getDocumentationTracks,
} from "@/components/documentation/open-source-onboarding";

export function OpenSourceOnboardingSection({
    apiBaseUrl,
}: {
    apiBaseUrl?: string;
}) {
    const resources = getDocumentationResources(apiBaseUrl);
    const tracks = getDocumentationTracks(apiBaseUrl);

    return (
        <div className="space-y-8">
            <div>
                <p>
                    O projeto agora documenta explicitamente os dois caminhos válidos para quem chega
                    de fora: rodar o stack completo a partir do GitHub ou consumir uma API já
                    publicada usando o mesmo contrato HTTP e a mesma referência interativa.
                </p>

                <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                    {resources.map((resource) => (
                        <article key={resource.title} className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{resource.title}</h4>
                            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{resource.description}</p>
                            {resource.href ? (
                                <a
                                    href={resource.href}
                                    target="_blank"
                                    rel="noreferrer"
                                    className="mt-3 inline-flex items-center gap-2 text-sm font-semibold text-primary hover:underline"
                                >
                                    Open link
                                    <ExternalLink className="h-3.5 w-3.5" />
                                </a>
                            ) : null}
                            {resource.pathHint ? (
                                <p className="mt-3">
                                    <code>{resource.pathHint}</code>
                                </p>
                            ) : null}
                        </article>
                    ))}
                </div>
            </div>

            <div className="grid gap-5 xl:grid-cols-2">
                {tracks.map((track) => (
                    <div key={track.title} className="rounded-xl border border-border bg-background/50 p-5">
                        <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-[0.2em] text-primary">
                            {track.title === "Hosted API consumption" ? (
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
                                <div key={`${track.title}-${step.command}`} className="rounded-xl border border-border bg-card px-4 py-4">
                                    <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">
                                        Step {index + 1}
                                    </p>
                                    <h4 className="mt-2 text-sm font-semibold text-foreground">{step.title}</h4>
                                    <p className="mt-2">
                                        <code>{step.command}</code>
                                    </p>
                                    <p className="mt-2 text-sm text-muted-foreground">{step.detail}</p>
                                </div>
                            ))}
                        </div>

                        <div className="mt-4 rounded-xl border border-border bg-card px-4 py-4">
                            <p className="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground">
                                Notes
                            </p>
                            <ul className="mt-3 space-y-2 text-sm leading-relaxed text-muted-foreground">
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
                    Troubleshooting
                </div>
                <div className="mt-4 grid gap-4 xl:grid-cols-2">
                    {OPEN_SOURCE_TROUBLESHOOTING_ITEMS.map((item) => (
                        <div key={item.title} className="rounded-xl border border-border bg-card px-4 py-4">
                            <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{item.detail}</p>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}
