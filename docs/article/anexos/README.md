# Anexos do artigo

Material citado no Apêndice A do artigo (`docs/article/artigo.tex`). Por
orientação do colegiado (limite de 30 páginas), diagramas, telas e tabelas
completas ficam aqui, e não no corpo do texto.

## Diagramas (`diagramas/`)

| Arquivo | O que mostra | Seção do artigo |
|---|---|---|
| `casos-de-uso.png` (`.puml`) | Casos de uso, derivados das rotas da API | 6.1 Requisitos |
| `sequencia.png` (`.puml`) | Envio e indexação de um documento e uma pergunta à coleção | 6.2 Arquitetura |
| `esquema-proveniencia.png` (`.puml`) | Recorte do esquema: do usuário até as seis tabelas de estrutura extraída. Linha contínua é chave estrangeira; tracejada, posse por `user_id` | 6.3 Modelo de dados |
| `recuperacao.png` (`.mmd`) | Recuperação: carga em número fixo de consultas, BM25 por tipo de unidade e seleção de fontes | 6.5 Recuperação |

## Telas (`telas/`)

| Arquivo | O que mostra |
|---|---|
| `workspace.png` | *Workspace* de uma coleção: *chat* com a fonte expandida, PDF na página citada com o parágrafo destacado e resumo do documento |
| `operacoes.png` | Tela de operações: modelo ativo, cobertura da indexação, *tokens* estimados, novas tentativas, fila de mensagens mortas, falhas e órfãos |

## Requisitos funcionais

| Id | Requisito | Implementação |
|---|---|---|
| RF01 | Cadastro, autenticação, redefinição de senha e edição de perfil | rotas `/auth/*` e `/user/me` |
| RF02 | Criar, listar e renomear categorias; criar coleções e mover entre categorias | rotas de categoria e coleção |
| RF03 | Enviar documento para uma coleção (PDF, HTML e XLSX), com validação de tipo e tamanho | `POST /documents/upload` |
| RF04 | Indexar de forma assíncrona páginas, parágrafos, tabelas, sumário, trechos e metadados | *worker* `indexer` |
| RF05 | Enriquecer o índice com LLM, opcional e configurável por usuário | preferências de indexação |
| RF06 | Recuperar o contexto de uma coleção por API | rota de contexto da coleção |
| RF07 | Responder perguntas sobre a coleção com as fontes citadas, em *streaming* | `POST /chat/stream` (SSE) |
| RF08 | Abrir cada fonte no PDF, na página e na região de origem | *workspace* do *frontend* |
| RF09 | Configurar provedores de LLM e de armazenamento, com credencial cifrada | rotas `/user/providers` |
| RF10 | Acompanhar status, tentativas, *tokens* e custo da indexação | painel de operações |
| RF11 | Registrar trilha de auditoria de documentos e provedores | tabelas de eventos de auditoria |
| RF12 | Reprocessar documento com falha e marcar documento cujo arquivo sumiu | `POST /documents/:id/retry` |

## Tabelas completas e dados

- `../tables/` — tabelas geradas a partir dos CSV: recuperação com intervalos de
  confiança, P@3 e R@5 (`retrieval-summary*.tex`), consultas negativas e limiar
  (`retrieval-negatives*.tex`, `threshold.tex`), as 75 comparações pareadas
  (`retrieval-significance*.tex`), TPC por tipo (`provenance.tex`) e
  determinismo do denso (`dense-determinism.tex`).
- `../results/` — resultados brutos, uma linha por pergunta e repetição.
- `../results/alteracoes-pos-medicao.md` — toda alteração de gabarito ou
  critério feita depois de medir, com data, motivo e efeito nos números.
- `../../schema/thingindex.dbml` — esquema completo (25 tabelas), gerado dos
  modelos da aplicação.
