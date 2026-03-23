"use client";

import { DocumentationMetricCard } from "@/components/documentation/DocumentationMetricCard";
import { DocumentationStatusMessage } from "@/components/documentation/DocumentationStatusMessage";
import { useDocumentationResource } from "@/components/documentation/useDocumentationResource";
import { fetchRuntimeProfile, type RuntimeProfile } from "@/lib/api/runtime-profile";

export function ProductFocusSection() {
    const {
        resource: profile,
        isLoading,
        error,
    } = useDocumentationResource<RuntimeProfile | null>(
        fetchRuntimeProfile,
        "Could not load the runtime profile."
    );

    return (
        <div className="space-y-5">
            <p>
                Esta seção consome o endpoint público <code>/api/v1/meta/runtime-profile</code> para deixar explícito qual é o
                caso de uso principal do ThingIndex hoje, para quem ele foi desenhado e quais limites seguem marcados como
                pesquisa ou extensão futura.
            </p>

            {isLoading ? (
                <DocumentationStatusMessage>
                    Carregando perfil público do runtime...
                </DocumentationStatusMessage>
            ) : error ? (
                <DocumentationStatusMessage tone="error">{error}</DocumentationStatusMessage>
            ) : profile ? (
                <>
                    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                        {[
                            { label: "Produto", value: profile.product_name, detail: profile.tagline },
                            { label: "Caso de uso principal", value: "Collections auditáveis", detail: profile.primary_use_case },
                            { label: "Problema central", value: "Contexto e proveniência", detail: profile.core_problem },
                            { label: "Última revisão", value: profile.last_reviewed_on, detail: "Snapshot público do runtime atual" },
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
                            <h4 className="text-sm font-semibold text-foreground">Perfis prioritários</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {profile.target_users.map((item) => (
                                    <li key={item.name}>
                                        <strong className="text-foreground">{item.name}:</strong> {item.description}
                                    </li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Escopo assumido pelo runtime principal</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {profile.current_scope.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Diferenciais assumidos</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {profile.differentiators.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                            <p className="mt-4 text-sm leading-relaxed text-foreground/85">{profile.value_proposition}</p>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Não objetivos explícitos</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {profile.non_goals.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>
                    </div>

                    <div className="grid gap-4 lg:grid-cols-2">
                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Trilhas acadêmicas relacionadas</h4>
                            <ul className="mt-4 space-y-3 text-sm leading-relaxed text-muted-foreground">
                                {profile.research_tracks.map((item) => (
                                    <li key={item}>{item}</li>
                                ))}
                            </ul>
                        </div>

                        <div className="rounded-xl border border-border bg-background/50 p-5">
                            <h4 className="text-sm font-semibold text-foreground">Artefatos versionados</h4>
                            <div className="mt-4 space-y-3">
                                {profile.artifact_refs.map((artifact) => (
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
