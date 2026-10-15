// Quantas paginas de PDF podem ter <canvas> vivo somando todos os quadros.
// Teto 12 e piso de zoom 0,35 foram chutados com o corpus de teste.

import { rectVisivel, type Rect, type Tamanho, type Viewport } from "@/lib/workspace/geometry";

export const TETO_DE_PAGINAS_VIVAS = 12;
export const ZOOM_MINIMO_PARA_PAGINAS = 0.35;

export type PaginaCandidata<Id> = { id: Id; rect: Rect };

export function escolherPaginasVivas<Id>(
    paginas: PaginaCandidata<Id>[],
    viewport: Viewport,
    tela: Tamanho,
    teto = TETO_DE_PAGINAS_VIVAS,
    zoomMinimo = ZOOM_MINIMO_PARA_PAGINAS
): Set<Id> {
    if (viewport.escala < zoomMinimo || teto <= 0) {
        return new Set();
    }
    const centroTela = { x: tela.largura / 2, y: tela.altura / 2 };
    const visiveis = paginas
        .filter((p) => rectVisivel(p.rect, viewport, tela))
        .map((p) => {
            const cx = (p.rect.x + p.rect.largura / 2) * viewport.escala + viewport.x;
            const cy = (p.rect.y + p.rect.altura / 2) * viewport.escala + viewport.y;
            return { id: p.id, distancia: Math.hypot(cx - centroTela.x, cy - centroTela.y) };
        })
        .sort((a, b) => a.distancia - b.distancia);
    return new Set(visiveis.slice(0, teto).map((p) => p.id));
}

/** Escala de rasterizacao do pdf.js: nitido no zoom atual, sem exagero em telas HiDPI. */
export function escalaDeRenderizacao(zoom: number, devicePixelRatio: number): number {
    return Math.min(zoom, 2) * Math.min(Math.max(devicePixelRatio, 1), 2);
}
