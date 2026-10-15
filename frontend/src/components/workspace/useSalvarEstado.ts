"use client";

import { useEffect, useRef } from "react";

import { putViewState } from "@/lib/api/collections";
import type { WorkspaceViewState } from "@/lib/api/contracts/view-state";

// Debounce de 800 ms, e envio imediato no pagehide (keepalive). `salvarLogo`
// forca o primeiro envio quando o estado veio da migracao do localStorage.

export const DEBOUNCE_MS = 800;

export function useSalvarEstado(
    collectionId: number,
    estado: WorkspaceViewState,
    opcoes: { ativo: boolean; salvarLogo?: boolean; aoFalhar?: (erro: unknown) => void }
) {
    const serializado = JSON.stringify(estado);
    const ultimoEnviado = useRef<string | null>(opcoes.salvarLogo ? null : serializado);
    const pendente = useRef<string | null>(null);
    const timer = useRef<number | null>(null);
    const aoFalhar = useRef(opcoes.aoFalhar);
    useEffect(() => {
        aoFalhar.current = opcoes.aoFalhar;
    }, [opcoes.aoFalhar]);

    useEffect(() => {
        if (!opcoes.ativo || !collectionId || serializado === ultimoEnviado.current) {
            return;
        }
        pendente.current = serializado;
        if (timer.current !== null) {
            window.clearTimeout(timer.current);
        }
        timer.current = window.setTimeout(() => {
            timer.current = null;
            const corpo = pendente.current;
            pendente.current = null;
            if (corpo === null || corpo === ultimoEnviado.current) {
                return;
            }
            ultimoEnviado.current = corpo;
            putViewState(collectionId, JSON.parse(corpo) as WorkspaceViewState).catch((erro: unknown) => {
                ultimoEnviado.current = null;
                aoFalhar.current?.(erro);
            });
        }, DEBOUNCE_MS);
        return () => {
            if (timer.current !== null) {
                window.clearTimeout(timer.current);
                timer.current = null;
            }
        };
    }, [collectionId, serializado, opcoes.ativo]);

    // Ao sair, o que estiver pendente vai na hora.
    useEffect(() => {
        if (!opcoes.ativo || !collectionId) {
            return;
        }
        function descarregar() {
            const corpo = pendente.current;
            if (corpo === null || corpo === ultimoEnviado.current) {
                return;
            }
            pendente.current = null;
            ultimoEnviado.current = corpo;
            void putViewState(collectionId, JSON.parse(corpo) as WorkspaceViewState, { keepalive: true }).catch(() => undefined);
        }
        function aoMudarVisibilidade() {
            if (document.visibilityState === "hidden") {
                descarregar();
            }
        }
        window.addEventListener("pagehide", descarregar);
        document.addEventListener("visibilitychange", aoMudarVisibilidade);
        return () => {
            window.removeEventListener("pagehide", descarregar);
            document.removeEventListener("visibilitychange", aoMudarVisibilidade);
        };
    }, [collectionId, opcoes.ativo]);
}
