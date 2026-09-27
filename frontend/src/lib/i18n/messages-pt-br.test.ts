import { describe, expect, it } from "vitest";

import { getMessages, type TranslationTree } from "@/lib/i18n/messages";

// Chaves cujo valor e comparado com texto de outro sistema (nao e UI).
const CHAVES_IGNORADAS = new Set(["notFoundToken", "city"]);

// Palavras sem acento que so existem como erro de digitacao no pt-BR da UI.
const SEM_ACENTO = [
    "nao", "Nao", "possivel", "configuracao", "configuracoes", "Configuracoes", "indexacao", "Indexacao",
    "operacao", "operacoes", "Operacoes", "informacao", "duracao", "Duracao", "automatico", "automatica",
    "intervencao", "Intervencao", "acessivel", "voce", "tambem", "orfao", "Orfaos", "sessao", "historico",
    "Historico", "indisponivel", "Indisponivel", "confirmacao", "exclusao", "Exclusao", "Usuario", "usuario",
];

function valores(arvore: TranslationTree, caminho = ""): Array<[string, string]> {
    return Object.entries(arvore).flatMap(([chave, valor]) => {
        const atual = caminho ? `${caminho}.${chave}` : chave;
        if (typeof valor === "string") {
            return CHAVES_IGNORADAS.has(chave) ? [] : [[atual, valor] as [string, string]];
        }
        return valores(valor, atual);
    });
}

describe("catalogo pt-BR acentuado", () => {
    it("nenhum texto de UI em pt-BR usa as palavras comuns sem acento", () => {
        const padrao = new RegExp(`(?<![\\p{L}{])(${SEM_ACENTO.join("|")})(?![\\p{L}}])`, "u");
        const erros = valores(getMessages("pt-BR"))
            .filter(([, texto]) => padrao.test(texto))
            .map(([chave, texto]) => `${chave}: ${texto}`);

        expect(erros).toEqual([]);
    });
});
