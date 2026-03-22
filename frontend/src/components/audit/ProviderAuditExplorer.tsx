"use client";

import { useEffect, useState } from "react";
import { Clock3, Loader2, RefreshCw, Search, Settings2 } from "lucide-react";

import AuditMetadataList from "@/components/audit/AuditMetadataList";
import {
    AuditAppliedFilters,
    AuditQuickFilterButton,
    AuditScopeButton,
    AuditSummaryCard,
} from "@/components/audit/AuditExplorerShared";
import AuditStatusBadge from "@/components/audit/AuditStatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    fetchProviderAuditEvents,
    type ProviderAuditEvent,
    type ProviderAuditQueryResult,
} from "@/lib/api/audit";
import {
    formatAuditActor,
    formatAuditActorType,
    formatAuditDateTime,
    formatAuditStatus,
    formatProviderAuditAction,
    getAuditBucketCount,
    normalizeAuditFilterValue,
    parseAuditActorId,
    toRFC3339FromDateTimeLocal,
} from "@/lib/audit";

type ProviderAuditExplorerProps = {
    locale: string;
    family: "LLM" | "STORAGE";
    activeProviderName: string;
    activeProviderType: string;
    reloadToken?: number;
};

type Scope = "ALL" | "ACTIVE";

type ProviderFilters = {
    scope: Scope;
    action: string;
    status: string;
    actorType: string;
    actorId: string;
    query: string;
    correlationId: string;
    createdFrom: string;
    createdTo: string;
};

const initialFilters: ProviderFilters = {
    scope: "ALL",
    action: "",
    status: "",
    actorType: "",
    actorId: "",
    query: "",
    correlationId: "",
    createdFrom: "",
    createdTo: "",
};

const emptyResult: ProviderAuditQueryResult = {
    items: [],
    summary: {
        total_events: 0,
        unique_providers: 0,
        unique_requests: 0,
        action_breakdown: [],
        status_breakdown: [],
    },
    page_info: {
        limit: 0,
        returned: 0,
        has_next: false,
    },
};

function ProviderAuditEventCard({
    event,
    locale,
}: {
    event: ProviderAuditEvent;
    locale: string;
}) {
    return (
        <div className="rounded-xl border border-border bg-background/70 p-4">
            <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-semibold">{event.provider_type || "Provider"}</p>
                <span className="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {formatProviderAuditAction(event.action)}
                </span>
                <AuditStatusBadge status={event.status} />
            </div>

            <p className="mt-2 text-sm text-foreground/90">{event.detail || "Sem detalhe registrado."}</p>

            <AuditMetadataList metadata={event.metadata} />

            <div className="mt-3 grid gap-2 text-[11px] text-muted-foreground md:grid-cols-2 xl:grid-cols-3">
                <p>Quando: <span className="font-mono text-foreground">{formatAuditDateTime(event.created_at, locale)}</span></p>
                <p>Origem: <span className="font-mono text-foreground">{event.origin || "-"}</span></p>
                <p>Request: <span className="font-mono text-foreground">{event.correlation_id || "-"}</span></p>
                <p>Actor: <span className="font-mono text-foreground">{formatAuditActor(event.actor_type, event.actor_id)}</span></p>
                <p>Familia: <span className="font-mono text-foreground">{event.family || "-"}</span></p>
                <p>Config: <span className="font-mono text-foreground">{event.provider_config_id ?? "-"}</span></p>
            </div>
        </div>
    );
}

export default function ProviderAuditExplorer({
    locale,
    family,
    activeProviderName,
    activeProviderType,
    reloadToken = 0,
}: ProviderAuditExplorerProps) {
    const [filters, setFilters] = useState<ProviderFilters>(initialFilters);
    const [appliedFilters, setAppliedFilters] = useState<ProviderFilters>(initialFilters);
    const [result, setResult] = useState<ProviderAuditQueryResult>(emptyResult);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isLoadingMore, setIsLoadingMore] = useState(false);

    function applyImmediateFilters(partial: Partial<ProviderFilters>) {
        const nextFilters = {
            ...filters,
            ...appliedFilters,
            ...partial,
        };
        setFilters(nextFilters);
        setAppliedFilters(nextFilters);
    }

    useEffect(() => {
        let cancelled = false;

        async function load() {
            setIsLoading(true);
            setError(null);

            try {
                const nextResult = await fetchProviderAuditEvents({
                    limit: 10,
                    family,
                    providerType: appliedFilters.scope === "ACTIVE" ? activeProviderType : undefined,
                    query: appliedFilters.query || undefined,
                    action: appliedFilters.action || undefined,
                    status: appliedFilters.status || undefined,
                    actorType: appliedFilters.actorType || undefined,
                    actorId: parseAuditActorId(appliedFilters.actorId),
                    correlationId: appliedFilters.correlationId || undefined,
                    createdFrom: toRFC3339FromDateTimeLocal(appliedFilters.createdFrom),
                    createdTo: toRFC3339FromDateTimeLocal(appliedFilters.createdTo),
                });

                if (!cancelled) {
                    setResult(nextResult);
                }
            } catch (nextError) {
                if (!cancelled) {
                    setError(nextError instanceof Error ? nextError.message : "Nao foi possivel carregar a auditoria de providers.");
                    setResult(emptyResult);
                }
            } finally {
                if (!cancelled) {
                    setIsLoading(false);
                }
            }
        }

        void load();

        return () => {
            cancelled = true;
        };
    }, [activeProviderType, appliedFilters, family, reloadToken]);

    async function handleLoadMore() {
        if (!result.page_info.has_next || !result.page_info.next_cursor) {
            return;
        }

        setIsLoadingMore(true);
        setError(null);

        try {
            const nextPage = await fetchProviderAuditEvents({
                limit: 10,
                cursor: result.page_info.next_cursor,
                family,
                providerType: appliedFilters.scope === "ACTIVE" ? activeProviderType : undefined,
                query: appliedFilters.query || undefined,
                action: appliedFilters.action || undefined,
                status: appliedFilters.status || undefined,
                actorType: appliedFilters.actorType || undefined,
                actorId: parseAuditActorId(appliedFilters.actorId),
                correlationId: appliedFilters.correlationId || undefined,
                createdFrom: toRFC3339FromDateTimeLocal(appliedFilters.createdFrom),
                createdTo: toRFC3339FromDateTimeLocal(appliedFilters.createdTo),
            });

            setResult((current) => ({
                ...nextPage,
                items: [...current.items, ...nextPage.items],
            }));
        } catch (nextError) {
            setError(nextError instanceof Error ? nextError.message : "Nao foi possivel carregar mais eventos de auditoria.");
        } finally {
            setIsLoadingMore(false);
        }
    }

    const failedEvents = getAuditBucketCount(result.summary.status_breakdown, "FAILED");
    const configSaveEvents = getAuditBucketCount(result.summary.action_breakdown, "PROVIDER_CONFIG_SAVE");
    const activationEvents = getAuditBucketCount(result.summary.action_breakdown, "PROVIDER_ACTIVATION_CHANGE");
    const appliedActorId = parseAuditActorId(appliedFilters.actorId);
    const activeFilterItems = [
        {
            key: "family",
            label: "Familia",
            value: family,
        },
        appliedFilters.scope === "ACTIVE"
            ? {
                  key: "scope",
                  label: "Escopo",
                  value: activeProviderName,
              }
            : null,
        appliedFilters.query
            ? {
                  key: "query",
                  label: "Busca",
                  value: appliedFilters.query,
              }
            : null,
        appliedFilters.action
            ? {
                  key: "action",
                  label: "Acao",
                  value: formatProviderAuditAction(appliedFilters.action),
              }
            : null,
        appliedFilters.status
            ? {
                  key: "status",
                  label: "Status",
                  value: formatAuditStatus(appliedFilters.status),
              }
            : null,
        appliedFilters.actorType
            ? {
                  key: "actor-type",
                  label: "Actor",
                  value: formatAuditActorType(appliedFilters.actorType),
              }
            : null,
        typeof appliedActorId === "number"
            ? {
                  key: "actor-id",
                  label: "Actor ID",
                  value: String(appliedActorId),
              }
            : null,
        appliedFilters.correlationId
            ? {
                  key: "correlation",
                  label: "Request",
                  value: appliedFilters.correlationId,
              }
            : null,
        appliedFilters.createdFrom
            ? {
                  key: "created-from",
                  label: "Criado de",
                  value: appliedFilters.createdFrom,
              }
            : null,
        appliedFilters.createdTo
            ? {
                  key: "created-to",
                  label: "Criado ate",
                  value: appliedFilters.createdTo,
              }
            : null,
    ].filter((item): item is { key: string; label: string; value: string } => item !== null);

    return (
        <Card className="border-border/80 bg-card/70">
            <CardHeader className="space-y-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                        <p className="text-[10px] font-bold uppercase tracking-[0.28em] text-primary">Audit Trail</p>
                        <h3 className="text-lg font-semibold">
                            Mudancas sensiveis de {family === "LLM" ? "LLMs" : "storage"}
                        </h3>
                        <p className="text-sm text-muted-foreground">
                            Consulta operacional do que foi salvo ou ativado, com leitura agregada, request id, actor e recorte para o provider em foco.
                        </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        <AuditScopeButton active={filters.scope === "ALL"} onClick={() => setFilters((current) => ({ ...current, scope: "ALL" }))}>
                            Toda a familia
                        </AuditScopeButton>
                        <AuditScopeButton active={filters.scope === "ACTIVE"} onClick={() => setFilters((current) => ({ ...current, scope: "ACTIVE" }))}>
                            Ativo: {activeProviderName}
                        </AuditScopeButton>
                    </div>
                </div>
            </CardHeader>

            <CardContent className="space-y-6">
                <form
                    className="grid gap-4 rounded-2xl border border-border bg-background/55 p-4 lg:grid-cols-2 xl:grid-cols-4"
                    onSubmit={(event) => {
                        event.preventDefault();
                        setAppliedFilters(filters);
                    }}
                >
                    <div className="space-y-2 xl:col-span-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-query`}>Busca operacional</Label>
                        <div className="relative">
                            <Search className="pointer-events-none absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                            <Input
                                id={`${family.toLowerCase()}-provider-audit-query`}
                                value={filters.query}
                                onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
                                className="pl-9"
                                placeholder="Provider, detalhe, origem ou request id"
                            />
                        </div>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-action`}>Acao</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-action`}
                            list={`${family.toLowerCase()}-provider-audit-actions`}
                            value={filters.action}
                            onChange={(event) => setFilters((current) => ({ ...current, action: normalizeAuditFilterValue(event.target.value) }))}
                            placeholder="ALL"
                        />
                        <datalist id={`${family.toLowerCase()}-provider-audit-actions`}>
                            <option value="ALL" />
                            <option value="PROVIDER_CONFIG_SAVE" />
                            <option value="PROVIDER_ACTIVATION_CHANGE" />
                        </datalist>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-status`}>Status</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-status`}
                            list={`${family.toLowerCase()}-provider-audit-statuses`}
                            value={filters.status}
                            onChange={(event) => setFilters((current) => ({ ...current, status: normalizeAuditFilterValue(event.target.value) }))}
                            placeholder="ALL"
                        />
                        <datalist id={`${family.toLowerCase()}-provider-audit-statuses`}>
                            <option value="ALL" />
                            <option value="SUCCESS" />
                            <option value="FAILED" />
                        </datalist>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-actor-type`}>Actor</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-actor-type`}
                            list={`${family.toLowerCase()}-provider-audit-actor-types`}
                            value={filters.actorType}
                            onChange={(event) => setFilters((current) => ({ ...current, actorType: normalizeAuditFilterValue(event.target.value) }))}
                            placeholder="ALL"
                        />
                        <datalist id={`${family.toLowerCase()}-provider-audit-actor-types`}>
                            <option value="ALL" />
                            <option value="USER" />
                            <option value="SYSTEM" />
                            <option value="SERVICE" />
                            <option value="WORKER" />
                        </datalist>
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-actor-id`}>Actor ID</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-actor-id`}
                            type="number"
                            min="1"
                            step="1"
                            value={filters.actorId}
                            onChange={(event) => setFilters((current) => ({ ...current, actorId: event.target.value }))}
                            placeholder="52"
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-correlation`}>Request ID</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-correlation`}
                            value={filters.correlationId}
                            onChange={(event) => setFilters((current) => ({ ...current, correlationId: event.target.value }))}
                            placeholder="req-..."
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-created-from`}>Criado de</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-created-from`}
                            type="datetime-local"
                            value={filters.createdFrom}
                            onChange={(event) => setFilters((current) => ({ ...current, createdFrom: event.target.value }))}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor={`${family.toLowerCase()}-provider-audit-created-to`}>Criado ate</Label>
                        <Input
                            id={`${family.toLowerCase()}-provider-audit-created-to`}
                            type="datetime-local"
                            value={filters.createdTo}
                            onChange={(event) => setFilters((current) => ({ ...current, createdTo: event.target.value }))}
                        />
                    </div>

                    <div className="flex items-end gap-2 xl:col-span-4">
                        <Button type="submit" className="gap-2">
                            <Settings2 className="h-4 w-4" />
                            Aplicar filtros
                        </Button>
                        <Button
                            type="button"
                            variant="outline"
                            className="gap-2"
                            onClick={() => {
                                setFilters(initialFilters);
                                setAppliedFilters(initialFilters);
                            }}
                        >
                            <RefreshCw className="h-4 w-4" />
                            Limpar
                        </Button>
                    </div>
                </form>

                <AuditAppliedFilters items={activeFilterItems} />

                <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                    <AuditSummaryCard
                        label="Eventos"
                        value={String(result.summary.total_events)}
                        detail={`${failedEvents} falhas no recorte atual`}
                    />
                    <AuditSummaryCard
                        label="Providers"
                        value={String(result.summary.unique_providers)}
                        detail={appliedFilters.scope === "ACTIVE" ? `Escopo atual: ${activeProviderName}` : "Cobertura da familia filtrada"}
                    />
                    <AuditSummaryCard
                        label="Requests"
                        value={String(result.summary.unique_requests)}
                        detail="Correlacoes distintas registradas"
                    />
                    <AuditSummaryCard
                        label="Ultimo evento"
                        value={formatAuditDateTime(result.summary.last_event_at, locale)}
                        detail={`${configSaveEvents} saves · ${activationEvents} ativacoes`}
                    />
                </div>

                <div className="rounded-xl border border-border bg-background/45 p-4">
                    <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                        Atalhos operacionais
                    </p>
                    <div className="mt-3 flex flex-wrap gap-2">
                        <AuditQuickFilterButton
                            label="Somente falhas"
                            count={failedEvents}
                            active={appliedFilters.status === "FAILED"}
                            tone="danger"
                            onClick={() =>
                                applyImmediateFilters({
                                    status: appliedFilters.status === "FAILED" ? "" : "FAILED",
                                })
                            }
                        />
                        <AuditQuickFilterButton
                            label="Config saves"
                            count={configSaveEvents}
                            active={appliedFilters.action === "PROVIDER_CONFIG_SAVE"}
                            onClick={() =>
                                applyImmediateFilters({
                                    action: appliedFilters.action === "PROVIDER_CONFIG_SAVE" ? "" : "PROVIDER_CONFIG_SAVE",
                                })
                            }
                        />
                        <AuditQuickFilterButton
                            label="Ativacoes"
                            count={activationEvents}
                            active={appliedFilters.action === "PROVIDER_ACTIVATION_CHANGE"}
                            onClick={() =>
                                applyImmediateFilters({
                                    action: appliedFilters.action === "PROVIDER_ACTIVATION_CHANGE" ? "" : "PROVIDER_ACTIVATION_CHANGE",
                                })
                            }
                        />
                    </div>
                </div>

                {error ? (
                    <div className="rounded-2xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                        {error}
                    </div>
                ) : null}

                {isLoading ? (
                    <div className="flex items-center gap-2 rounded-2xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        Carregando auditoria de providers...
                    </div>
                ) : null}

                {!isLoading && !error && result.items.length === 0 ? (
                    <div className="rounded-2xl border border-dashed border-border bg-background/45 p-5 text-sm text-muted-foreground">
                        Nenhum evento de auditoria encontrado para este recorte ainda.
                    </div>
                ) : null}

                {!isLoading && result.items.length > 0 ? (
                    <div className="space-y-3">
                        {result.items.map((event) => (
                            <ProviderAuditEventCard key={event.id} event={event} locale={locale} />
                        ))}
                    </div>
                ) : null}

                {!isLoading && result.page_info.has_next ? (
                    <div className="flex justify-center">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => {
                                void handleLoadMore();
                            }}
                            disabled={isLoadingMore}
                            className="gap-2"
                        >
                            {isLoadingMore ? <Loader2 className="h-4 w-4 animate-spin" /> : <Clock3 className="h-4 w-4" />}
                            Carregar mais eventos
                        </Button>
                    </div>
                ) : null}
            </CardContent>
        </Card>
    );
}
