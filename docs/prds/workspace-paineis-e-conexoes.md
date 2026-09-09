# Workspace: painel que fecha e redimensiona, virar página pelas bordas da folha, conexões só com termo raro

**Status:** entregue em PR sem merge (2026-09-09), no PR #60
**Data:** 2026-09-09
**Base:** `docs/prds/workspace-unificado.md` e `docs/prds/chat-contexto-e-custo.md` (PRs #55 e #60, sem merge). Mesma branch do PR #60.
**Origem:** pedido do autor com foto em 1.900 px: fechar ou redimensionar o painel de chat/documentos; passar página "por cima ou por baixo" da folha; e a observação de que "grande parte das coisas não tem conexão", com o canvas desenhando arestas de "2 termos" entre anexos que só compartilham palavras comuns.

## Goals

(1) O painel esquerdo fecha por um botão nele mesmo e a divisa entre painel e canvas arrasta, entre 280 e 640 px, com a largura lembrada no navegador. (2) No modo "uma página", a folha tem uma faixa clicável em cima (página anterior) e outra embaixo (próxima), além dos botões do cabeçalho do quadro. (3) Uma conexão entre seções só existe quando os termos casados são raros no acervo: termo presente em mais de 5% das seções da coleção não conta como evidência. A ordem, o determinismo e os termos casados continuam iguais; o que muda é quais termos podem sustentar a aresta.

Pronto quando: E2E com mock fecha o painel pelo botão e arrasta a divisa mudando a largura; unitário do `PdfFrame` vira página pelas faixas; unitário de `BuildNodeConnections` prova que termo onipresente não sustenta conexão e termo raro sustenta; na stack real, o número de conexões do grifo verde do Edital 56 cai e as que sobram são justificadas por termos discriminativos.

## Requirements

Painel
- [ ] Botão "Fechar painel" (`data-testid="fechar-painel"`) no fim da linha de abas; faz o mesmo que o botão "Painel" do cabeçalho.
- [ ] Divisa arrastável (`role="separator"`, `aria-orientation="vertical"`, `data-testid="redimensionar-painel"`) só em `lg`; pointer capture; largura entre `PAINEL_LARGURA_MIN = 280` e `PAINEL_LARGURA_MAX = 640`; salva em `localStorage` (`thingindex.workspace.painel.largura`) e lê ao montar. A largura entra por variável CSS (`--painel`) na classe `lg:w-[var(--painel)]`.

Folha
- [ ] No modo página, faixas de 28 px no topo e no rodapé da folha (`data-testid="virar-pagina-anterior"` / `"virar-pagina-proxima"`), com chevron, visíveis ao passar o mouse (sempre visíveis no toque), que chamam `irParaPagina(±1)`; a do topo some na primeira página e a do rodapé na última. `onPointerDown` para a propagação para não arrastar o quadro.

Conexões
- [ ] `BuildNodeConnections` calcula, para cada termo significativo da origem, em quantos candidatos (de qualquer documento, exceto a origem) ele aparece. Termo com `df > max(2, ceil(NodeConnectionMaxTermShare × N))`, com `NodeConnectionMaxTermShare = 0.05`, é descartado antes da pontuação e da justificativa. `minScore` continua 2.
- [ ] Testes: termo presente em todos os candidatos não sustenta conexão; dois termos raros sustentam; conjunto pequeno continua funcionando pelo piso de 2.

## Non-Goals

- Enriquecimento por LLM na indexação: com IDF, resumos gerados só ajudam se trouxerem termos raros; sem medir, não se liga. Fica como opção que já existe em Configurações.
- Limpar o sumário (861 seções do PPC): PRD próprio.
- Persistir a largura do painel no servidor (estado de visualização); localStorage basta para preferência de tela.
- Virar página com a roda do mouse nas bordas: a roda é pan do canvas, e mudar isso confunde.

## Constraints

- `BuildNodeConnections` é função pura; o cálculo de df entra dentro dela sobre `candidates`, sem repositório nem persistência (constituição II).
- O texto do candidato é `outlineNodeSearchText` (título + retrieval_text ou resumo); df conta presença por substring, a mesma regra do casamento.
- `PdfFrame` já tem `irParaPagina`, `pararPonteiro` e o mapa de páginas com `modoPagina`.
- A página já tem `chatAberto`; o `aside` usa `lg:w-[340px]` hoje.

## Decisões tomadas sem entrevista (autor presente, mas decisões de padrão)

- 5% como teto de frequência: com 1.853 seções no demo, um termo em mais de 92 seções é vocabulário do acervo, não relação. Constante exportada; ajustar com dado.
- Faixas nas bordas em vez de roda do mouse.
- Largura do painel no navegador, não no servidor.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (a queda de conexoes no demo e registro, nao criterio)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (roda x faixas; localStorage x servidor)
  Assumptions:  0.0   ✓ clear (verificado: irParaPagina, pararPonteiro, chatAberto, assinatura pura de BuildNodeConnections)
  ──────────────────────────────
  Aggregate:    0.10  ✓ below threshold (0.2 spec)
```

## Discoveries

- Conexões no demo (coleção de SI, 1.076 seções, amostra de 60): antes, praticamente toda seção tinha aresta por "curso", "edital", "campus"; depois, 23 das 60 têm conexão, e as que sobram são justificadas por termos como "estágio, supervisionado", "conclusão, trabalho", "engenharia, requisitos". Sobra ruído de nomes próprios e números de resolução ("bruno, guaringue", "nº, resolução"), que só a limpeza do sumário resolve.
- O piso do teto de raridade precisou ser 2 seções, não 1: com 32 seções e piso 1, um termo em 2 delas (6%) já não contava, e o unitário com dois termos raros falhava.
- Na primeira visita a tela de abertura (SplashScreen) cobre a página por segundos e engole o arrasto do mouse no E2E; o teste marca o splash como já visto por `addInitScript`, como `public-smoke` já fazia.
- O cabeçalho do quadro e a faixa da borda têm o mesmo rótulo acessível ("Próxima página"); o unitário antigo passou a pegar o primeiro.
- Enriquecimento por LLM continua fora: com IDF, resumos gerados só ajudam se trouxerem termos raros, e isso é o experimento, não uma feature.

## Rodada 2 (2026-09-09): sumário limpo e corpus ampliado

O autor pediu para verificar se a indexação "faz sentido". Não fazia: a página de sumário do PPC virava 124 seções duplicadas, células de tabela viravam seção, valores de capa ("Reitor", "Presencial") e pedaços de título quebrado ("DE ENSINO", "Programação de") também. Cinco regras entraram no indexador, cada uma com teste: página de sumário (5+ linhas terminadas em número de página) não gera título; célula em caixa mista com número solto não é título; linha terminada em ";" ou "," não é título; fragmento que começa por preposição ou termina em preposição não é título (com "DA/DO/DAS/DOS" no início preservados, porque abrem capítulo em texto legal); data por extenso no início não é título. Em PDF, uma palavra só em caixa mista não é título.

| Documento | Seções antes | Seções depois |
|---|---|---|
| PPC Sistemas de Informação (206 p.) | 861 | 564 |
| Matriz curricular | 50 | 16 |
| Regimento Interno dos campi | 54 | 32 |
| Calendário | 21 | 18 |
| Ementas | 144 | 133 |
| Regimento Geral | 86 | 73 |
| Editais 56 e 57 | 321 / 316 | 308 / 300 (cláusulas numeradas, como devem ser) |

O que sobra no PPC (101 títulos de até duas palavras) são nomes de pessoas e cargos das tabelas de comissão, que só uma heurística de tabela resolveria. Registrado, não feito.

Corpus: entraram a Resolução CONSUP 82/2022 (política de estágios, 15 p., conversa com o Apêndice A do PPC e com o Regimento) e o PPC do Técnico em Informática de Palmas (160 p., cita o curso de SI e compartilha laboratórios e regulamentos). Os dois têm camada de texto e caixas em todos os parágrafos. A coleção de SI passou a 5 documentos; o seed segue idempotente e verificou conexões em todas as coleções. O stub de 273 bytes da Resolução 36/2019 (404 salvo por engano) saiu.

## Rodada 3 (2026-09-09): conexões legíveis

O autor mostrou três fotos: pílulas amarelas truncadas na margem do PDF ("3.7 Redistrib…"), linhas cruzando o canvas com um "4 termos" solto no vazio, e nenhuma explicação do que é uma conexão. Três mudanças, sem tocar no cálculo:

- **Marcador de seção** virou uma barra de 3 px na margem; o título só aparece ao passar o mouse. Continua na mesma posição (âncora do parágrafo inicial da seção), com o mesmo `data-testid`.
- **Rótulo da linha** saiu do meio da curva e foi para a ponta de destino, colado ao trecho apontado; ao passar o mouse, mostra os termos que justificam a aresta em vez da contagem.
- **Sidebar** explica em uma frase o que é uma conexão e como navegar, acima da lista.

Verificado na stack real com a coleção do processo seletivo: 49 grifos, 9 marcadores discretos, 12 linhas com o rótulo na ponta.
