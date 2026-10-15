"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import {
    aplicarZoom,
    centralizarEm as centralizarEmGeometria,
    deslocar,
    enquadrar as enquadrarGeometria,
    telaParaMundo,
    type Ponto,
    type Rect,
    type Tamanho,
    type Viewport,
} from "@/lib/workspace/geometry";

// Roda = pan, Ctrl/Cmd+roda e pinch = zoom no cursor (como Figma).
// Listener de wheel na mao: o onWheel do React e passivo e nao deixa
// chamar preventDefault.

const VIEWPORT_INICIAL: Viewport = { x: 24, y: 24, escala: 1 };
const PASSO_DOS_BOTOES = 1.25;

type EstadoDeArrasto = { ultimo: Ponto };
type EstadoDePinch = { distancia: number };

export function useViewport(inicial?: Viewport) {
    const containerRef = useRef<HTMLDivElement | null>(null);
    const [viewport, setViewport] = useState<Viewport>(inicial ?? VIEWPORT_INICIAL);
    const [tamanho, setTamanho] = useState<Tamanho>({ largura: 0, altura: 0 });
    const [arrastandoFundo, setArrastandoFundo] = useState(false);

    const arrasto = useRef<EstadoDeArrasto | null>(null);
    const ponteiros = useRef<Map<number, Ponto>>(new Map());
    const pinch = useRef<EstadoDePinch | null>(null);

    // ResizeObserver ja dispara ao montar, sem setState sincrono no efeito.
    useEffect(() => {
        const el = containerRef.current;
        if (!el) {
            return;
        }
        const observador = new ResizeObserver((entradas) => {
            const caixa = entradas[0]?.contentRect;
            if (caixa) {
                setTamanho({ largura: caixa.width, altura: caixa.height });
            }
        });
        observador.observe(el);
        return () => observador.disconnect();
    }, []);

    const pontoRelativo = useCallback((clientX: number, clientY: number): Ponto => {
        const caixa = containerRef.current?.getBoundingClientRect();
        return { x: clientX - (caixa?.left ?? 0), y: clientY - (caixa?.top ?? 0) };
    }, []);

    useEffect(() => {
        const el = containerRef.current;
        if (!el) {
            return;
        }
        function aoRolar(event: WheelEvent) {
            event.preventDefault();
            const ponto = pontoRelativo(event.clientX, event.clientY);
            if (event.ctrlKey || event.metaKey) {
                // deltaY negativo = afastar os dedos / rolar para cima = aproximar.
                const fator = Math.exp(-event.deltaY * 0.01);
                setViewport((atual) => aplicarZoom(atual, fator, ponto));
            } else {
                setViewport((atual) => deslocar(atual, { x: -event.deltaX, y: -event.deltaY }));
            }
        }
        el.addEventListener("wheel", aoRolar, { passive: false });
        return () => el.removeEventListener("wheel", aoRolar);
    }, [pontoRelativo]);

    const aoPressionarFundo = useCallback(
        (event: React.PointerEvent<HTMLDivElement>) => {
            // So o fundo inicia pan; cartoes e grifos param a propagacao.
            if (event.target !== event.currentTarget) {
                return;
            }
            ponteiros.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
            if (ponteiros.current.size === 2) {
                const [a, b] = [...ponteiros.current.values()];
                pinch.current = { distancia: Math.hypot(a.x - b.x, a.y - b.y) };
                arrasto.current = null;
                return;
            }
            arrasto.current = { ultimo: { x: event.clientX, y: event.clientY } };
            setArrastandoFundo(true);
            event.currentTarget.setPointerCapture(event.pointerId);
        },
        []
    );

    const aoMoverPonteiro = useCallback(
        (event: React.PointerEvent<HTMLDivElement>) => {
            if (!ponteiros.current.has(event.pointerId)) {
                return;
            }
            ponteiros.current.set(event.pointerId, { x: event.clientX, y: event.clientY });

            if (pinch.current && ponteiros.current.size === 2) {
                const [a, b] = [...ponteiros.current.values()];
                const distancia = Math.hypot(a.x - b.x, a.y - b.y);
                const fator = distancia / pinch.current.distancia;
                pinch.current = { distancia };
                const meio = pontoRelativo((a.x + b.x) / 2, (a.y + b.y) / 2);
                setViewport((atual) => aplicarZoom(atual, fator, meio));
                return;
            }

            const estado = arrasto.current;
            if (!estado) {
                return;
            }
            const delta = { x: event.clientX - estado.ultimo.x, y: event.clientY - estado.ultimo.y };
            estado.ultimo = { x: event.clientX, y: event.clientY };
            setViewport((atual) => deslocar(atual, delta));
        },
        [pontoRelativo]
    );

    const aoSoltarPonteiro = useCallback((event: React.PointerEvent<HTMLDivElement>) => {
        ponteiros.current.delete(event.pointerId);
        if (ponteiros.current.size < 2) {
            pinch.current = null;
        }
        if (ponteiros.current.size === 0) {
            arrasto.current = null;
            setArrastandoFundo(false);
        }
    }, []);

    const zoomPorBotao = useCallback(
        (fator: number) => {
            setViewport((atual) =>
                aplicarZoom(atual, fator, { x: tamanho.largura / 2, y: tamanho.altura / 2 })
            );
        },
        [tamanho]
    );

    const enquadrar = useCallback(
        (quadros: Rect[]) => {
            setViewport(enquadrarGeometria(quadros, tamanho));
        },
        [tamanho]
    );

    const centralizarEm = useCallback(
        (rect: Rect, escala = 1) => {
            setViewport(centralizarEmGeometria(rect, tamanho, escala));
        },
        [tamanho]
    );

    /** Converte um ponto do evento (clientX/Y) para coordenadas do mundo. */
    const clienteParaMundo = useCallback(
        (clientX: number, clientY: number): Ponto => telaParaMundo(pontoRelativo(clientX, clientY), viewport),
        [pontoRelativo, viewport]
    );

    return {
        containerRef,
        viewport,
        tamanho,
        arrastandoFundo,
        handlers: {
            onPointerDown: aoPressionarFundo,
            onPointerMove: aoMoverPonteiro,
            onPointerUp: aoSoltarPonteiro,
            onPointerCancel: aoSoltarPonteiro,
        },
        aproximar: () => zoomPorBotao(PASSO_DOS_BOTOES),
        afastar: () => zoomPorBotao(1 / PASSO_DOS_BOTOES),
        enquadrar,
        centralizarEm,
        clienteParaMundo,
    };
}
