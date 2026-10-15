"use client";

import type { PDFDocumentLoadingTask, PDFDocumentProxy } from "pdfjs-dist";

// pdf.js carregado sob demanda, um worker so e cache de documentos por id.
// O PDF vem de /api/documents/{id}/preview (mesma origem, cookie vai junto).

export type TamanhoDePagina = { largura: number; altura: number };

type Entrada = {
    promessa: Promise<PDFDocumentProxy>;
    // No pdf.js 6 quem fecha e a tarefa de carregamento, nao o documento.
    tarefa: PDFDocumentLoadingTask | null;
};

// Build legacy: o padrao do pdf.js 6 usa Map.prototype.getOrInsertComputed
// e quebrava (TypeError) em navegadores de poucos meses atras.
let biblioteca: Promise<typeof import("pdfjs-dist/legacy/build/pdf.mjs")> | null = null;
const abertos = new Map<number, Entrada>();

export function urlDePreview(documentId: number): string {
    return `/api/documents/${documentId}/preview`;
}

export function carregarPdfjs(): Promise<typeof import("pdfjs-dist/legacy/build/pdf.mjs")> {
    if (!biblioteca) {
        biblioteca = import("pdfjs-dist/legacy/build/pdf.mjs").then((modulo) => {
            if (!modulo.GlobalWorkerOptions.workerSrc) {
                modulo.GlobalWorkerOptions.workerSrc = new URL(
                    "pdfjs-dist/legacy/build/pdf.worker.min.mjs",
                    import.meta.url
                ).toString();
            }
            return modulo;
        });
    }
    return biblioteca;
}

export function abrirDocumento(documentId: number): Promise<PDFDocumentProxy> {
    const existente = abertos.get(documentId);
    if (existente) {
        return existente.promessa;
    }
    const entrada: Entrada = { promessa: Promise.resolve(null as unknown as PDFDocumentProxy), tarefa: null };
    entrada.promessa = carregarPdfjs()
        .then((pdfjs) => {
            entrada.tarefa = pdfjs.getDocument({ url: urlDePreview(documentId) });
            return entrada.tarefa.promise;
        })
        .catch((erro: unknown) => {
            abertos.delete(documentId);
            throw erro;
        });
    abertos.set(documentId, entrada);
    return entrada.promessa;
}

/** Fecha e esquece o documento; a proxima chamada a abrirDocumento reabre. */
export function fecharDocumento(documentId: number): void {
    const entrada = abertos.get(documentId);
    if (!entrada) {
        return;
    }
    abertos.delete(documentId);
    void entrada.promessa
        .then(() => entrada.tarefa?.destroy())
        .catch(() => undefined);
}

/** Dimensoes de cada pagina em pontos PDF (escala 1). Barato: nao rasteriza. */
export async function medirPaginas(documento: PDFDocumentProxy): Promise<TamanhoDePagina[]> {
    const tamanhos: TamanhoDePagina[] = [];
    for (let numero = 1; numero <= documento.numPages; numero += 1) {
        const pagina = await documento.getPage(numero);
        const viewport = pagina.getViewport({ scale: 1 });
        tamanhos.push({ largura: viewport.width, altura: viewport.height });
    }
    return tamanhos;
}
