# Alterações feitas depois de ver os resultados

Registro exigido pela banca simulada de 2026-09-27. Cada linha diz o que mudou
depois de uma medição, por quê, e o efeito nos números. Os números que o
artigo publica são os da última coluna de cada linha.

| # | Data | O quê | Por quê | Efeito |
|---|------|-------|---------|--------|
| 1 | 2026-09-10 a 09-18 | Quatro versões do gabarito do acervo de ensino (v1 a v4): regex afrouxadas em t04, t06, c08, c13; páginas-alvo novas em p01, p20, p22, t21. | Respostas certas recusadas por redação diferente; informação presente em mais de uma página. | F (sistema inalterado): 204 → 213/216 respostas. T passou de 67/72 perguntas (1 repetição, 2026-09-09) a 216/216 respostas, somando ajustes do sistema e do gabarito (commits 63ca3d8, e88e452, 74e4ddd). O acervo de ensino foi, na prática, conjunto de desenvolvimento (ver 5). |
| 2 | 2026-09-19 | Padrão das perguntas negativas das baterias ampliado ("nao consta", "nao ha", ...). | Recusas corretas redigidas de outro modo eram marcadas erradas. | Produto 0,783 → 0,841 (C). Ver `correcao-negativas.md`. |
| 3 | 2026-09-27 | Aliases de título de documento no gabarito das baterias (fora do hash das perguntas). | `cited_pages` gravava o título da fonte, não o id do gabarito; a proveniência contava erro quando o título diferia. | Proveniência C 0,371 → 0,581; E 0,532 → 0,823. Nenhum veredito de acerto mudou. |
| 4 | 2026-09-27 | Gabaritos das baterias v4 e do edital v2: existência e lc3 passam a vetar recusa (`reject_patterns`); premissa falsa passa a exigir o dado que corrige a premissa; lc3 deixa de exigir "não"; pr2 aceita o prazo de 2 anos que o termo de garantia também traz. Script: `scripts/gabarito-baterias-v4.py`. | A banca mostrou que uma recusa constante acertava 24 perguntas de existência e premissa falsa, e que lc3 no braço E, correta, era marcada errada. Teste de regressão: `recusa_test.go`. | C 174 → 149/213 (0,817 → 0,700); E 198 → 192/213 (0,930 → 0,901). Premissa falsa C 35 → 13/36. Expansão: 10 ganhos e 1 perda → 15 ganhos e 0 perdas (p 0,012 → 0,0001). Enriquecimento: 9/12 nos quatro braços → 7, 8, 7 e 8/12. Edital: sem mudança. |
| 5 | — | Heurísticas do ranqueador (cobertura mínima de 2 termos, variante de plural com 6+ letras, descarte do reforço por título, regra de continuação de página) foram mantidas ou descartadas pelo efeito medido nas 72 perguntas do acervo de ensino. | Ciclos de DSR. | O acerto e a proveniência de T no acervo de ensino são resultado dentro da amostra de desenvolvimento. As baterias, escritas depois, são o conjunto de teste. |
