"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { fetchNodeConnections } from "@/lib/api/collections";
import type { CollectionContext, OutlineNode } from "@/lib/api/contracts/document-context";
import type { NodeConnection } from "@/lib/api/contracts/node-connections";

/**
 * Painel navegavel dos documentos de uma colecao.
 *
 * # Por que nao React Flow
 *
 * A spec 001 previa React Flow. Ao implementar ficou claro que a biblioteca
 * resolveria um problema que este painel nao tem: nao ha arestas persistentes
 * para rotear, nem grafo para fazer layout automatico. O que existe sao cartoes
 * arrastaveis e uma sidebar -- cerca de 150 linhas de CSS transform contra uma
 * dependencia de ~50 kB gzipped que traria seu proprio modelo de estado.
 *
 * YAGNI: se o painel evoluir para arestas desenhadas entre cartoes, a decisao
 * deve ser revisitada. Hoje seria peso sem contrapartida.
 *
 * # O que este componente NAO faz
 *
 * Nao chama LLM. Titulos e resumos exibidos na sidebar ja estao no indice,
 * gravados na indexacao deterministica. A tela e leitura de dado existente, e
 * essa e a razao de ela ser instantanea.
 */

type WorkspaceCanvasProps = {
    context: CollectionContext;
};

type Posicao = { x: number; y: number };

const LARGURA_CARTAO = 260;
const ESPACO_ENTRE_CARTOES = 40;
const CARTOES_POR_LINHA = 3;

function chaveDeArmazenamento(collectionId: number | string): string {
    return `thingindex.workspace.positions.${collectionId}`;
}

/**
 * Posicao inicial em grade. Deterministica de proposito: a mesma colecao abre
 * sempre com o mesmo arranjo, o que torna a tela reproduzivel numa demonstracao.
 */
function posicaoInicial(indice: number): Posicao {
    const coluna = indice % CARTOES_POR_LINHA;
    const linha = Math.floor(indice / CARTOES_POR_LINHA);
    return {
        x: coluna * (LARGURA_CARTAO + ESPACO_ENTRE_CARTOES),
        y: linha * 190,
    };
}

function lerPosicoesSalvas(collectionId: number | string): Record<string, Posicao> {
    if (typeof window === "undefined") {
        return {};
    }
    try {
        const bruto = window.localStorage.getItem(chaveDeArmazenamento(collectionId));
        if (!bruto) {
            return {};
        }
        const analisado = JSON.parse(bruto) as unknown;
        return typeof analisado === "object" && analisado !== null
            ? (analisado as Record<string, Posicao>)
            : {};
    } catch {
        // localStorage corrompido nao pode impedir a tela de abrir; o custo de
        // ignorar e reposicionar os cartoes, e nao perder o acesso.
        return {};
    }
}

export function WorkspaceCanvas({ context }: WorkspaceCanvasProps) {
    const collectionIdBruto = context.collection?.id ?? 0;
    // A API aceita id numerico; o contrato do frontend admite string porque
    // algumas respostas antigas devolviam assim.
    const collectionId =
        typeof collectionIdBruto === "number" ? collectionIdBruto : Number(collectionIdBruto) || 0;
    const documentos = useMemo(() => context.documents ?? [], [context.documents]);
    const nosDoSumario = useMemo(() => context.outlineNodes ?? [], [context.outlineNodes]);

    const [posicoes, setPosicoes] = useState<Record<string, Posicao>>({});
    const [documentoFocado, setDocumentoFocado] = useState<number | null>(null);
    const [noSelecionado, setNoSelecionado] = useState<string | null>(null);
    const [conexoes, setConexoes] = useState<NodeConnection[] | null>(null);
    const [carregandoConexoes, setCarregandoConexoes] = useState(false);
    const [erroConexoes, setErroConexoes] = useState<string | null>(null);

    const arrastando = useRef<{ documentId: number; offsetX: number; offsetY: number } | null>(null);

    useEffect(() => {
        setPosicoes(lerPosicoesSalvas(collectionId));
    }, [collectionId]);

    const posicaoDe = useCallback(
        (documentId: number, indice: number): Posicao =>
            posicoes[String(documentId)] ?? posicaoInicial(indice),
        [posicoes]
    );

    const aoIniciarArrasto = useCallback(
        (event: React.PointerEvent<HTMLButtonElement>, documentId: number, indice: number) => {
            const atual = posicaoDe(documentId, indice);
            arrastando.current = {
                documentId,
                offsetX: event.clientX - atual.x,
                offsetY: event.clientY - atual.y,
            };
        },
        [posicaoDe]
    );

    useEffect(() => {
        function aoMover(event: PointerEvent) {
            const estado = arrastando.current;
            if (!estado) {
                return;
            }
            setPosicoes((anterior) => ({
                ...anterior,
                [String(estado.documentId)]: {
                    x: event.clientX - estado.offsetX,
                    y: event.clientY - estado.offsetY,
                },
            }));
        }

        function aoSoltar() {
            if (!arrastando.current) {
                return;
            }
            arrastando.current = null;
            setPosicoes((atual) => {
                try {
                    window.localStorage.setItem(
                        chaveDeArmazenamento(collectionId),
                        JSON.stringify(atual)
                    );
                } catch {
                    // Quota estourada ou modo privativo: o arrasto continua
                    // valendo na sessao, so nao sobrevive ao recarregamento.
                }
                return atual;
            });
        }

        window.addEventListener("pointermove", aoMover);
        window.addEventListener("pointerup", aoSoltar);
        return () => {
            window.removeEventListener("pointermove", aoMover);
            window.removeEventListener("pointerup", aoSoltar);
        };
    }, [collectionId]);

    const nosDoDocumentoFocado = useMemo<OutlineNode[]>(() => {
        if (documentoFocado === null) {
            return [];
        }
        return nosDoSumario.filter((no) => no.documentId === documentoFocado);
    }, [documentoFocado, nosDoSumario]);

    const pedirConexoes = useCallback(
        async (nodeId: string) => {
            setNoSelecionado(nodeId);
            setCarregandoConexoes(true);
            setErroConexoes(null);
            setConexoes(null);

            try {
                const resultado = await fetchNodeConnections(collectionId, nodeId);
                setConexoes(resultado.connections);
            } catch (erro) {
                setErroConexoes(
                    erro instanceof Error ? erro.message : "Falha ao buscar conexoes."
                );
            } finally {
                setCarregandoConexoes(false);
            }
        },
        [collectionId]
    );

    const documentoAtual = documentos.find((documento) => documento.id === documentoFocado);

    return (
        <div className="flex h-full w-full gap-4">
            <div
                className="relative flex-1 overflow-auto rounded-lg border border-border bg-muted/20"
                data-testid="workspace-canvas"
                style={{ minHeight: 420 }}
            >
                {documentos.map((documento, indice) => {
                    const posicao = posicaoDe(documento.id, indice);
                    const focado = documento.id === documentoFocado;

                    return (
                        <div
                            key={documento.id}
                            className="absolute"
                            style={{
                                transform: `translate(${posicao.x}px, ${posicao.y}px)`,
                                width: LARGURA_CARTAO,
                            }}
                        >
                            <button
                                type="button"
                                onPointerDown={(event) =>
                                    aoIniciarArrasto(event, documento.id, indice)
                                }
                                onClick={() => {
                                    setDocumentoFocado(documento.id);
                                    setConexoes(null);
                                    setNoSelecionado(null);
                                }}
                                className={`w-full cursor-grab rounded-lg border p-4 text-left transition ${
                                    focado
                                        ? "border-primary bg-card shadow-lg"
                                        : "border-border bg-card/80 hover:border-primary/50"
                                }`}
                            >
                                <span className="block text-sm font-medium">
                                    {documento.title}
                                </span>
                                <span className="mt-1 block text-xs text-muted-foreground">
                                    {
                                        nosDoSumario.filter(
                                            (no) => no.documentId === documento.id
                                        ).length
                                    }{" "}
                                    trechos indexados
                                </span>
                            </button>
                        </div>
                    );
                })}
            </div>

            <aside
                className="w-80 shrink-0 overflow-y-auto rounded-lg border border-border bg-card p-4"
                aria-label="Detalhes do documento"
            >
                {documentoFocado === null ? (
                    <p className="text-sm text-muted-foreground">
                        Selecione um documento no painel para ver sua estrutura.
                    </p>
                ) : (
                    <div className="space-y-4">
                        <h2 className="text-sm font-semibold">{documentoAtual?.title}</h2>

                        {nosDoDocumentoFocado.length === 0 ? (
                            <p className="text-sm text-muted-foreground">
                                Documento sem estrutura indexada.
                            </p>
                        ) : (
                            <ul className="space-y-3">
                                {nosDoDocumentoFocado.map((no) => (
                                    <li
                                        key={no.nodeId}
                                        className={`rounded-md border p-3 ${
                                            no.nodeId === noSelecionado
                                                ? "border-primary"
                                                : "border-border"
                                        }`}
                                    >
                                        <p className="text-sm font-medium">{no.title}</p>
                                        {no.summary ? (
                                            <p className="mt-1 text-xs text-muted-foreground">
                                                {no.summary}
                                            </p>
                                        ) : null}
                                        <button
                                            type="button"
                                            onClick={() => pedirConexoes(no.nodeId)}
                                            className="mt-2 text-xs font-medium text-primary underline-offset-2 hover:underline"
                                        >
                                            Ver conexões
                                        </button>
                                    </li>
                                ))}
                            </ul>
                        )}

                        {carregandoConexoes ? (
                            <p className="text-xs text-muted-foreground">Buscando conexões…</p>
                        ) : null}

                        {erroConexoes ? (
                            <p className="text-xs text-destructive">{erroConexoes}</p>
                        ) : null}

                        {conexoes !== null && !carregandoConexoes ? (
                            conexoes.length === 0 ? (
                                // Ausencia DECLARADA, nunca area em branco. Uma
                                // regiao vazia seria lida como falha de
                                // carregamento, e o sistema perderia credito no
                                // unico ponto em que sabe dizer "nao encontrei".
                                <p className="text-xs text-muted-foreground">
                                    Nenhuma conexão acima do limiar para este trecho.
                                </p>
                            ) : (
                                <ul className="space-y-2 border-t border-border pt-3">
                                    {conexoes.map((conexao) => (
                                        <li
                                            key={`${conexao.documentId}-${conexao.nodeId}`}
                                            className="rounded-md bg-muted/40 p-2"
                                        >
                                            <p className="text-xs font-medium">{conexao.title}</p>
                                            <p className="mt-1 text-[11px] text-muted-foreground">
                                                Termos em comum:{" "}
                                                {conexao.matchedTerms.join(", ")}
                                            </p>
                                        </li>
                                    ))}
                                </ul>
                            )
                        ) : null}
                    </div>
                )}
            </aside>
        </div>
    );
}
