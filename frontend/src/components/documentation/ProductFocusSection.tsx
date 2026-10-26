"use client";

import { localizeRuntimeProfile } from "@/components/documentation/documentation-resource-localization";
import { DocumentationMetricCard } from "@/components/documentation/DocumentationMetricCard";
import { DocumentationStatusMessage } from "@/components/documentation/DocumentationStatusMessage";
import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";
import { fetchRuntimeProfile, type RuntimeProfile } from "@/lib/api/runtime-profile";
import { useI18n } from "@/lib/i18n/provider";

function formatDocumentationDate(locale: string, value: string) {
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }

    return new Intl.DateTimeFormat(locale, {
        dateStyle: "medium",
    }).format(parsed);
}

export function ProductFocusSection() {
    const { locale } = useI18n();
    const copy = locale === "pt-BR"
        ? {
            fallbackError: "Não foi possível carregar o perfil público do sistema.",
            intro: "Esta seção consome o endpoint público ",
            introSuffix:
                " para deixar explícito qual é o caso de uso principal do ThingIndex hoje, para quem ele foi desenhado e quais limites seguem marcados como pesquisa ou extensão futura.",
            loading: "Carregando perfil público do sistema...",
            metrics: {
                product: "Produto",
                useCase: "Caso de uso principal",
                coreProblem: "Problema central",
                reviewedAt: "Última revisão",
                collections: "Coleções auditáveis",
                provenance: "Contexto e proveniência",
                reviewedDetail: "Panorama público do sistema atual",
            },
            priorityProfiles: "Perfis prioritários",
            currentScope: "Escopo assumido pelo sistema principal",
            differentiators: "Diferenciais assumidos",
            nonGoals: "Não objetivos explícitos",
            researchTracks: "Trilhas acadêmicas relacionadas",
            artifacts: "Artefatos versionados",
        }
        : locale === "es"
            ? {
                fallbackError: "No se pudo cargar el perfil público del runtime.",
                intro: "Esta sección consume el endpoint público ",
                introSuffix:
                    " para dejar explícito cuál es el caso de uso principal de ThingIndex hoy, para quién fue diseñado y qué límites siguen marcados como investigación o extensión futura.",
                loading: "Cargando perfil público del runtime...",
                metrics: {
                    product: "Producto",
                    useCase: "Caso de uso principal",
                    coreProblem: "Problema central",
                    reviewedAt: "Última revisión",
                    collections: "Collections auditables",
                    provenance: "Contexto y procedencia",
                    reviewedDetail: "Snapshot público del runtime actual",
                },
                priorityProfiles: "Perfiles prioritarios",
                currentScope: "Alcance asumido por el runtime principal",
                differentiators: "Diferenciales asumidos",
                nonGoals: "No objetivos explícitos",
                researchTracks: "Líneas académicas relacionadas",
                artifacts: "Artefactos versionados",
            }
            : {
                fallbackError: "Could not load the public runtime profile.",
                intro: "This section consumes the public endpoint ",
                introSuffix:
                    " to make the current ThingIndex use case explicit, show who it was designed for, and clarify which boundaries remain marked as research or future extension.",
                loading: "Loading public runtime profile...",
                metrics: {
                    product: "Product",
                    useCase: "Primary use case",
                    coreProblem: "Core problem",
                    reviewedAt: "Last review",
                    collections: "Auditable collections",
                    provenance: "Context and provenance",
                    reviewedDetail: "Public snapshot of the current runtime",
                },
                priorityProfiles: "Priority profiles",
                currentScope: "Scope assumed by the main runtime",
                differentiators: "Declared differentiators",
                nonGoals: "Explicit non-goals",
                researchTracks: "Related academic tracks",
                artifacts: "Versioned artifacts",
            };
    const {
        resource: profile,
        isLoading,
        error,
    } = useDocumentationResource<RuntimeProfile | null>(
        fetchRuntimeProfile,
        copy.fallbackError
    );
    const localizedProfile = profile ? localizeRuntimeProfile(profile, locale) : null;

    return (
        <div className="space-y-5">
            <p>
                {copy.intro}<code>/api/v1/meta/runtime-profile</code>{copy.introSuffix}
            </p>

            {isLoading ? (
                <DocumentationStatusMessage>
                    {copy.loading}
                </DocumentationStatusMessage>
            ) : error ? (
                <DocumentationStatusMessage tone="error">{error}</DocumentationStatusMessage>
            ) : localizedProfile ? (
                <>
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                        {[
                            { label: copy.metrics.product, value: localizedProfile.product_name, detail: localizedProfile.tagline },
                            { label: copy.metrics.useCase, value: copy.metrics.collections, detail: localizedProfile.primary_use_case },
                            { label: copy.metrics.coreProblem, value: copy.metrics.provenance, detail: localizedProfile.core_problem },
                            {
                                label: copy.metrics.reviewedAt,
                                value: formatDocumentationDate(locale, localizedProfile.last_reviewed_on),
                                detail: copy.metrics.reviewedDetail,
                            },
                        ].map((item) => (
                            <DocumentationMetricCard
                                key={item.label}
                                label={item.label}
                                value={item.value}
                                detail={item.detail}
                                valueClassName="text-sm font-semibold leading-relaxed"
                                detailClassName="mt-2 leading-relaxed"
                            />
                        ))}
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.priorityProfiles}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedProfile.target_users.map((item) => (
                                    <li key={item.name}>
                                        <strong className="text-foreground">{item.name}:</strong> {item.description}
                                    </li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.currentScope}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedProfile.current_scope.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.differentiators}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedProfile.differentiators.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                            <p className="mt-4 text-sm leading-relaxed text-foreground/85">{localizedProfile.value_proposition}</p>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.nonGoals}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedProfile.non_goals.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.researchTracks}</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {localizedProfile.research_tracks.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">{copy.artifacts}</h4>
                            <div className="mt-4 space-y-3">
                                {localizedProfile.artifact_refs.map((artifact) => (
                                    <div key={artifact.path} className="rounded-xl border border-border/70 bg-background/70 px-4 py-3">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <p className="text-sm font-semibold text-foreground">{artifact.label}</p>
                                            <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-bold uppercase tracking-[0.18em] text-muted-foreground">
                                                {artifact.kind}
                                            </span>
                                        </div>
                                        <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{artifact.summary}</p>
                                        <p className="mt-2 font-mono text-xs text-primary">{artifact.path}</p>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                </>
            ) : null}
        </div>
    );
}
