# Apresentação TCC I — ThingIndex
**Tempo total: ~10 minutos | 11 slides**

> Nota: prototipação de telas, diagramas completos e detalhes técnicos estão no artigo entregue.

---

## Slide 1 — Apresentação
**Tempo:** 20s

**O que mostrar na tela:**
- Título: ThingIndex
- Subtítulo: Plataforma Open Source para Inteligência Documental com Baseline Relacional
- Seu nome, curso, IFPR, orientador

**O que falar:**
> "Boa [dia/tarde]. Meu nome é Charles Eduardo, sou aluno de Sistemas de Informação no IFPR, e este é o meu TCC I. O trabalho se chama **ThingIndex**: uma plataforma open source para inteligência documental com baseline relacional. Meu orientador é o professor Aldo."

> **Transição para o Slide 2:** *"E pra entender por que esse projeto existe, preciso começar pelo problema que ele resolve."*

---

## Slide 2 — Introdução / Problema
**Tempo:** 1min10s

**O que mostrar na tela:**
- Documentos espalhados em pastas sem organização
- Para achar uma informação: achar o arquivo → abrir → Ctrl+F → ler página por página
- Sem rastreabilidade: não dá pra saber de onde veio a informação

**O que falar:**
> "Vou começar pelo problema. Imagine que você precisa confirmar uma informação que está em algum documento — pode ser uma cláusula de contrato, um parecer técnico, ou uma decisão registrada em ata. O que você faz hoje? Primeiro, precisa achar o arquivo certo no meio de dezenas de arquivos espalhados em pastas. Depois de abrir, precisa dar Ctrl+F e buscar palavras-chave, porque ler página por página demora demais. E mesmo assim, você não tem certeza se achou tudo que precisava, nem de onde veio cada informação. Se tiver que confirmar a fonte, lá vai você de novo: abrir arquivo, procurar, conferir. Isso é lento, gera erro e em áreas como jurídico e compliance, onde rastreabilidade é obrigatória, é um risco real."

> **Transição para o Slide 3:** *"E é pra resolver exatamente isso que o ThingIndex existe."*

---

## Slide 3 — Solução
**Tempo:** 1min

**O que mostrar na tela:**
- Organiza documentos em categorias e coleções
- Extrai conteúdo de cada PDF e guarda estruturado no banco
- Faz uma pergunta → recebe resposta com fontes

**O que falar:**
> "O ThingIndex resolve isso organizando documentos em categorias e coleções. Pensa no IFPR: você cria uma categoria 'Alunos', dentro dela uma coleção para cada RA, e dentro de cada coleção você coloca os tipos de documento — histórico, declaração, ata. Quando você envia um PDF, o sistema lê o arquivo, extrai parágrafos, tabelas e metadados, e guarda tudo num banco de dados estruturado. Quando você precisa de uma informação, em vez de procurar, você faz uma pergunta. O sistema busca no conteúdo de todos os documentos daquela coleção e te dá uma resposta — junto com as fontes: exatamente de qual documento, qual página e qual parágrafo aquela informação veio."

> **Transição para o Slide 4:** *"Mas antes de entrar em detalhes, vale olhar o que já existe no mercado e onde essas ferramentas não chegam."*

---

## Slide 4 — Trabalhos Relacionados
**Tempo:** 50s

**O que mostrar na tela:**
- Notion: workspace all-in-one, mas conhecimento preso na plataforma
- LangChain / LlamaIndex: frameworks para RAG com vetores, mas opacos
- Lacuna: nenhuma oferece rastreabilidade total + custo previsível + pipeline assíncrono

**O que falar:**
> "Existem ferramentas que atuam em áreas parecidas. O **Notion** é um workspace completo, mas o conhecimento fica preso na plataforma e não dá pra auditar de onde veio cada informação. O **LangChain** e o **LlamaIndex** são frameworks populares para construir sistemas de IA com documentos, mas o pipeline de recuperação é opaco — você configura, mas não tem visibilidade de como o contexto foi montado. Além disso, dependem de serviços externos caros. Nenhuma dessas oferece ao mesmo tempo: rastreabilidade total, custo previsível e pipeline de processamento com retry automático alem disso nenhum é open source. É essa lacuna que o ThingIndex ocupa."

> **Transição para o Slide 5:** *"E é nessa lacuna que entra a proposta do nosso trabalho."*

---

## Slide 5 — Nosso Projeto / Objetivo
**Tempo:** 1min10s

**O que mostrar na tela:**
- **Pergunta de pesquisa:** É possível fazer inteligência documental com auditabilidade usando só modelagem relacional — sem banco vetorial ou grafo?
- **Objetivo geral:** Propor e especificar o ThingIndex como plataforma open source que organiza, processa, indexa e recupera contexto documental de forma auditável
- **Diferencial:** RAG relacional — contexto montado por SQL, não por vetores

**O que falar:**
> "A pergunta de pesquisa do trabalho é: **é possível construir um sistema de inteligência documental com auditabilidade usando apenas modelagem relacional, sem depender de bancos vetoriais ou grafos?** O objetivo geral é propor e especificar o ThingIndex como uma plataforma open source que organiza, processa, indexa e recupera contexto documental de forma auditável. O diferencial é o que chamamos de **RAG relacional**: em vez de quebrar documentos em pedaços genéricos e buscar por similaridade vetorial — que é uma caixa-preta — o ThingIndex preserva a estrutura original dos documentos em tabelas separadas e monta o contexto com SQL, que é determinístico e totalmente rastreável."

> **Transição para o Slide 6:** *"Pra tirar isso do papel, o sistema precisa atender a um conjunto de requisitos. Vou mostrar os principais."*

---

## Slide 6 — Requisitos Funcionais
**Tempo:** 40s

**O que mostrar na tela:**
- RF01: Autenticação (cadastro, login, perfil)
- RF02-03: Categorias e coleções documentais
- RF04-05: Upload de PDF + armazenamento em object storage
- RF06: Processamento assíncrono (extrai parágrafos, tabelas, metadados)
- RF08: Recuperação de contexto via API
- RF09: Chat grounded com citação de fontes
- RF11: Painel de monitoramento
- *Lista completa: 15 RFs — ver artigo*

**O que falar:**
> "Os requisitos funcionais cobrem todo o fluxo do sistema. Autenticação de usuários, criação de categorias e coleções, upload de PDFs com armazenamento separado, processamento assíncrono que extrai parágrafos, tabelas e metadados, recuperação de contexto via API, chat com citação de fontes e painel de monitoramento. São 15 requisitos no total — a lista completa está no artigo."

> **Transição para o Slide 7:** *"Além do que o sistema faz, tem também como ele faz — os requisitos não funcionais."*

---

## Slide 7 — Requisitos Não Funcionais
**Tempo:** 40s

**O que mostrar na tela:**
- RNF01: Interface responsiva
- RNF02: Integridade referencial e ACID no PostgreSQL
- RNF05: Segurança (hash de senhas, tokens, criptografia, rate limiting)
- RNF06: Reprodutível via Docker Compose
- RNF07: Testes automatizados
- *Lista completa: 8 RNFs — ver artigo*

**O que falar:**
> "Os requisitos não funcionais garantem a qualidade do sistema. Interface responsiva, integridade dos dados com chaves estrangeiras e transações ACID, segurança com hash de senhas, tokens e criptografia de credenciais, ambiente reproduzível via Docker Compose e cobertura de testes automatizados. São 8 requisitos no total — também no artigo."

> **Transição para o Slide 8:** *"E pra atender a tudo isso, a arquitetura do sistema é dividida em cinco componentes."*

---

## Slide 8 — Arquitetura / Visão Geral do Sistema
**Tempo:** 1min40s

**O que mostrar na tela:**
- Diagrama da arquitetura (visao-geral-sistema.png do artigo)
- Frontend (Next.js) → API/Worker (Go) → PostgreSQL + RabbitMQ + Object Storage
- Padrão outbox: evento na mesma transação do banco → sem perda de mensagens

**O que falar:**
> "A arquitetura tem cinco componentes. O **frontend** em Next.js é a interface do usuário. A **API e o worker** são em Go — Go foi escolhido porque lida bem com processamento paralelo de vários documentos ao mesmo tempo. O **PostgreSQL** é o banco principal, guardando toda a estrutura dos documentos com integridade referencial. O **Object Storage** armazena os PDFs brutos separadamente, acessados por URLs temporárias. E o **RabbitMQ** gerencia a fila de processamento. Um ponto importante é o padrão **outbox**: quando um documento é enviado, o evento de indexação é gravado na mesma transação do banco. Isso significa que mesmo se a API cair antes de publicar na fila, o evento continua no banco e será reenviado. Zero perda de mensagens. Se o processamento falhar, o sistema tenta de novo automaticamente. Se falhar várias vezes, o documento vai pra uma fila de inspeção manual."

> **Transição para o Slide 9:** *"Pra mostrar como esses componentes se comunicam na prática, vou apresentar o diagrama de sequência."*

---

## Slide 9 — Diagrama de Execução
**Tempo:** 40s

**O que mostrar na tela:**
- Diagrama de sequência (`sequencia.png` do artigo)
- Fluxo: Frontend → API → Storage/Banco → RabbitMQ → Worker → Resposta
- Nota: diagramas completos (ER, classes, casos de uso) no artigo

**O que falar:**
> "Esse diagrama mostra a ordem temporal das chamadas entre os componentes. O usuário faz upload pelo frontend, a API valida e salva no storage e no banco. O evento de indexação vai pro RabbitMQ, o worker processa o PDF e extrai as estruturas. Quando o usuário faz uma pergunta, o contexto é montado por SQL e enviado ao LLM, que responde com as fontes citadas. Esse é o fluxo principal — diagramas mais detalhados como entidade-relacionamento, classes e casos de uso estão no artigo."

> **Transição para o Slide 10:** *"E como isso se traduz na prática, pro usuário? Vou mostrar algumas telas do protótipo."*

---

## Slide 10 — Protótipo de Telas
**Tempo:** 30s

**O que mostrar na tela:**
- 2-3 telas do protótipo (dashboard, documents, settings — do artigo)
- Nota: prototipação completa no artigo

**O que falar:**
> "Essas são algumas telas do protótipo. O **dashboard** mostra categorias, coleções, documentos com status de processamento e telemetria. A tela de **documentos** permite visualizar e gerenciar os arquivos enviados. E as **configurações** permitem gerenciar provedores de IA e storage, com criptografia de credenciais no backend. A prototipação completa com todas as telas está no artigo."

> **Transição para o Slide 11:** *"Para fechar, um resumo de onde estamos e pra onde vamos."*

---

## Slide 11 — Conclusão
**Tempo:** 1min

**O que mostrar na tela:**
- TCC I = proposta e especificação
- Contribuição: baseline relacional como alternativa a abordagens vetoriais
- TCC II = implementação e validação (jun–dez 2026)
- Open source após conclusão
- Obrigado

**O que falar:**
> "Para fechar: este TCC I é a proposta e a especificação do ThingIndex. A contribuição principal é mostrar que uma baseline relacional pode ser uma alternativa real às abordagens vetoriais que dominam hoje — com mais transparência, mais auditabilidade e custo previsível. A implementação e os testes práticos vão acontecer no TCC II, entre junho e dezembro de 2026. O código será open source, para que qualquer pessoa possa usar, testar e contribuir. Obrigado."

---

## Referências Rápidas (para perguntas)

**Por que Go e não Python?**
> Go compila para um binário único, tem concorrência nativa com goroutines e é mais simples de deployar. Python exigiria gerenciar dependências e ambiente.

**Por que não usar pgvector já?**
> O objetivo do TCC I é estabelecer uma baseline relacional pura primeiro. pgvector entra no TCC II como comparação futura.

**Qual a diferença pro LangChain/LlamaIndex?**
> Eles são frameworks genéricos para RAG com vetores. O ThingIndex é uma plataforma completa com rastreabilidade total via SQL.

**O sistema já funciona?**
> Não — este é o TCC I, que é a proposta e especificação. A implementação será no TCC II.

**Como funciona a parte de IA/chat?**
> O contexto é montado por consultas SQL no banco relacional. Esse contexto é enviado para o LLM junto com a pergunta do usuário. O LLM gera a resposta baseado apenas nesse contexto, e o sistema mostra as fontes (documento, página, parágrafo) de onde cada informação veio.
