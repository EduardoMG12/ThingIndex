// Coordenadas:
//  - PDF: pontos, origem no topo esquerdo.
//  - mundo: px do canvas em zoom 1.
//  - tela: px do viewport, depois de translate(x, y) scale(escala).

import type { ParagraphBox } from "@/lib/api/contracts/document-context";

export type Ponto = { x: number; y: number };
export type Rect = { x: number; y: number; largura: number; altura: number };
export type Viewport = { x: number; y: number; escala: number };
export type Tamanho = { largura: number; altura: number };

export const ZOOM_MINIMO = 0.1;
export const ZOOM_MAXIMO = 4;

export function boxParaPixels(
    box: ParagraphBox,
    larguraRenderizada: number,
    alturaRenderizada: number
): { left: number; top: number; width: number; height: number } {
    const fx = larguraRenderizada / box.pageWidth;
    const fy = alturaRenderizada / box.pageHeight;
    return {
        left: box.xMin * fx,
        top: box.yMin * fy,
        width: (box.xMax - box.xMin) * fx,
        height: (box.yMax - box.yMin) * fy,
    };
}

export function limitarZoom(escala: number, minimo = ZOOM_MINIMO, maximo = ZOOM_MAXIMO): number {
    return Math.min(maximo, Math.max(minimo, escala));
}

export function telaParaMundo(ponto: Ponto, viewport: Viewport): Ponto {
    return {
        x: (ponto.x - viewport.x) / viewport.escala,
        y: (ponto.y - viewport.y) / viewport.escala,
    };
}

export function mundoParaTela(ponto: Ponto, viewport: Viewport): Ponto {
    return {
        x: ponto.x * viewport.escala + viewport.x,
        y: ponto.y * viewport.escala + viewport.y,
    };
}

/** Zoom mantendo sob o cursor o mesmo ponto do mundo. */
export function aplicarZoom(
    viewport: Viewport,
    fator: number,
    pontoNaTela: Ponto,
    limites: { min: number; max: number } = { min: ZOOM_MINIMO, max: ZOOM_MAXIMO }
): Viewport {
    const escala = limitarZoom(viewport.escala * fator, limites.min, limites.max);
    if (escala === viewport.escala) {
        return viewport;
    }
    const fixo = telaParaMundo(pontoNaTela, viewport);
    return {
        escala,
        x: pontoNaTela.x - fixo.x * escala,
        y: pontoNaTela.y - fixo.y * escala,
    };
}

export function deslocar(viewport: Viewport, deltaTela: Ponto): Viewport {
    return { ...viewport, x: viewport.x + deltaTela.x, y: viewport.y + deltaTela.y };
}

export function uniaoDeRects(rects: Rect[]): Rect | null {
    if (rects.length === 0) {
        return null;
    }
    let x0 = Infinity;
    let y0 = Infinity;
    let x1 = -Infinity;
    let y1 = -Infinity;
    for (const r of rects) {
        x0 = Math.min(x0, r.x);
        y0 = Math.min(y0, r.y);
        x1 = Math.max(x1, r.x + r.largura);
        y1 = Math.max(y1, r.y + r.altura);
    }
    return { x: x0, y: y0, largura: x1 - x0, altura: y1 - y0 };
}

/**
* "Ajustar a tela" so pela largura: quadros de PDF sao altos demais (~19 mil px
* num edital de 24 paginas). Centraliza na vertical se couber.
*/
export function enquadrar(quadros: Rect[], tela: Tamanho, margem = 48): Viewport {
    const caixa = uniaoDeRects(quadros);
    if (!caixa || caixa.largura <= 0 || caixa.altura <= 0) {
        return { x: margem, y: margem, escala: 1 };
    }
    const escala = limitarZoom(Math.min((tela.largura - 2 * margem) / caixa.largura, 1));
    const cabeNaAltura = caixa.altura * escala <= tela.altura - 2 * margem;
    return {
        escala,
        x: (tela.largura - caixa.largura * escala) / 2 - caixa.x * escala,
        y: cabeNaAltura ? (tela.altura - caixa.altura * escala) / 2 - caixa.y * escala : margem - caixa.y * escala,
    };
}

/** Viewport que poe o centro do rect no centro da tela, na escala pedida. */
export function centralizarEm(rect: Rect, tela: Tamanho, escala: number): Viewport {
    const e = limitarZoom(escala);
    return {
        escala: e,
        x: tela.largura / 2 - (rect.x + rect.largura / 2) * e,
        y: tela.altura / 2 - (rect.y + rect.altura / 2) * e,
    };
}

export function rectVisivel(rect: Rect, viewport: Viewport, tela: Tamanho): boolean {
    const a = mundoParaTela({ x: rect.x, y: rect.y }, viewport);
    const b = mundoParaTela({ x: rect.x + rect.largura, y: rect.y + rect.altura }, viewport);
    return b.x >= 0 && b.y >= 0 && a.x <= tela.largura && a.y <= tela.altura;
}

/**
 * Pontos de saida e entrada de uma linha entre dois retangulos do mundo. Sai
 * pela borda lateral voltada ao destino e entra pelo lado mais proximo, para a
 * linha nao atravessar o proprio grifo de origem.
 */
export function ancorasDeLinha(origem: Rect, destino: Rect): { de: Ponto; para: Ponto } {
    const centroOrigem = { x: origem.x + origem.largura / 2, y: origem.y + origem.altura / 2 };
    const centroDestino = { x: destino.x + destino.largura / 2, y: destino.y + destino.altura / 2 };
    const destinoADireita = centroDestino.x >= centroOrigem.x;

    const de = {
        x: destinoADireita ? origem.x + origem.largura : origem.x,
        y: centroOrigem.y,
    };

    const dx = centroDestino.x - de.x;
    const dy = centroDestino.y - de.y;
    let para: Ponto;
    if (Math.abs(dx) * destino.altura >= Math.abs(dy) * destino.largura) {
        // Entra pela lateral.
        para = { x: dx >= 0 ? destino.x : destino.x + destino.largura, y: centroDestino.y };
    } else {
        // Entra por cima ou por baixo.
        para = { x: centroDestino.x, y: dy >= 0 ? destino.y : destino.y + destino.altura };
    }
    return { de, para };
}

/**
 * Zoom para ler um trecho: o rect passa a ocupar ~55% da largura da tela,
 * entre 0,9 (nao afasta para ler) e 1,8 (mais que isso vira borrao).
 */
export function escalaDeFoco(larguraDoRect: number, larguraDaTela: number): number {
    return limitarZoom((larguraDaTela * 0.55) / Math.max(larguraDoRect, 1), 0.9, 1.8);
}
