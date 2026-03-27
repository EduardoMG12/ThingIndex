# Article Workspace

Este diretório concentra os artefatos versionados do artigo derivado do ThingIndex.

## Arquivos

- `sbc_template (1).doc`: template original da SBC em formato Word, mantido como referência de formatação final.
- `thingindex-sbc-article.md`: rascunho principal do artigo com estrutura alinhada ao template e ao estado atual do runtime.
- `figures/`: diagramas versionados usados como base para as figuras do artigo.

## Fluxo recomendado

1. Evoluir o conteúdo em `thingindex-sbc-article.md`.
2. Exportar os diagramas em `figures/` para PNG ou SVG quando for montar a versão final no Word.
3. Validar alinhamento com `rfc/0001-product-focus.md` e `docs/proposta-continuacao-tcc.md`.
4. Na etapa final de submissão, migrar o conteúdo revisado para o arquivo `.doc` seguindo a formatação do template da SBC.

## Premissa importante

O texto do artigo precisa descrever:

- o que já existe no repositório;
- o que foi medido ou demonstrado;
- o que continua sendo pesquisa e trabalho futuro.

Evite vender GraphRAG, vetores ou extensões acadêmicas como se fossem parte obrigatória do runtime principal atual.
