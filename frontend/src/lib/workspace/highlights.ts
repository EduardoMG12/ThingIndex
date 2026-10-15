// As chaves das cores vao para o banco, nao renomear. Classes Tailwind
// literais: montadas por template somem no purge.
// Texto preto sobre o fundo a 40%: todas acima de 11:1.

import { CORES, ehCor, type HighlightColor, type Legend } from "@/lib/api/contracts/highlights";

export { CORES, ehCor, type HighlightColor, type Legend };

export type ClassesDaCor = {
    /** Fundo + borda do grifo marcado. */
    grifo: string;
    /** Chip da barra flutuante e da legenda. */
    chip: string;
    /** Grifo secundario (secao destino de uma conexao), mais claro. */
    secundario: string;
};

const CLASSES: Record<HighlightColor, ClassesDaCor> = {
    amarelo: { grifo: "bg-yellow-300/40 border-yellow-500", chip: "bg-yellow-300", secundario: "bg-yellow-200/30 border-yellow-400/70" },
    verde: { grifo: "bg-green-300/40 border-green-600", chip: "bg-green-400", secundario: "bg-green-200/30 border-green-400/70" },
    azul: { grifo: "bg-sky-300/40 border-sky-600", chip: "bg-sky-400", secundario: "bg-sky-200/30 border-sky-400/70" },
    vermelho: { grifo: "bg-red-300/40 border-red-600", chip: "bg-red-400", secundario: "bg-red-200/30 border-red-400/70" },
    roxo: { grifo: "bg-violet-300/40 border-violet-600", chip: "bg-violet-400", secundario: "bg-violet-200/30 border-violet-400/70" },
};

export function classesDaCor(cor: HighlightColor): ClassesDaCor {
    return CLASSES[cor];
}

/** Rotulo da legenda, ou o nome da cor (traduzido por quem chama) quando vazio. */
export function rotuloDaCor(legenda: Legend, cor: HighlightColor, nomeDaCor: (cor: HighlightColor) => string): string {
    const rotulo = legenda[cor]?.trim();
    return rotulo ? rotulo : nomeDaCor(cor);
}

export type Marca = { paragraphNodeId: number | string; documentId: number; color: HighlightColor };

export function contagemPorCor(marcas: Iterable<Marca>, documentId: number): Record<HighlightColor, number> {
    const saida = Object.fromEntries(CORES.map((c) => [c, 0])) as Record<HighlightColor, number>;
    for (const marca of marcas) {
        if (marca.documentId === documentId) {
            saida[marca.color] += 1;
        }
    }
    return saida;
}

export type ModoDeApresentacao = "folhas" | "pagina";

export type Apresentacao = { modo: ModoDeApresentacao; paginaAtual: number };

export const APRESENTACAO_PADRAO: Apresentacao = { modo: "folhas", paginaAtual: 1 };

/**
 * Ir ate uma pagina de um quadro. Em "pagina" o quadro precisa trocar a
 * pagina atual antes de centralizar; em "folhas" basta centralizar, porque
 * todas as paginas ja estao no canvas.
 */
export function regraIrAte(apresentacao: Apresentacao, paginaDestino: number): { trocarPagina: number | null; centralizar: true } {
    if (apresentacao.modo === "pagina" && apresentacao.paginaAtual !== paginaDestino) {
        return { trocarPagina: paginaDestino, centralizar: true };
    }
    return { trocarPagina: null, centralizar: true };
}

/**
 * O botao "Ir para p. N" so aparece quando a pagina destino nao esta a vista:
 * em "pagina", sempre que nao e a atual; em "folhas", quando nao esta entre as
 * paginas visiveis no viewport.
 */
export function precisaDoBotaoIrPara(
    apresentacao: Apresentacao,
    paginasVisiveis: ReadonlySet<number>,
    paginaDestino: number | null
): boolean {
    if (paginaDestino === null) {
        return false;
    }
    if (apresentacao.modo === "pagina") {
        return apresentacao.paginaAtual !== paginaDestino;
    }
    return !paginasVisiveis.has(paginaDestino);
}

export function limitarPagina(pagina: number, total: number): number {
    if (total < 1) {
        return 1;
    }
    return Math.min(Math.max(Math.trunc(pagina), 1), total);
}
