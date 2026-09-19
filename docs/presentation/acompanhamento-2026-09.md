# Acompanhamento de TCC II — setembro de 2026

**Tempo total: ~8 minutos | 6 slides + demonstração**

> Este roteiro substitui, para esta aula, o `apresentacao.md`, que é do TCC I e
> responde "o sistema já funciona?" com "não, a implementação será no TCC II".
> Agora funciona e está medido, e essa é justamente a notícia da apresentação.

> Regra que atravessa o roteiro inteiro: **apresente o erro antes que perguntem**.
> Banca não pune falha medida com causa identificada. Pune falha que ela descobre
> sozinha.

---

## Slide 1 — Onde eu parei

**Tempo:** 30s

**O que mostrar na tela:**
- Título: ThingIndex — onde o trabalho está
- Três marcadores: *sistema implementado e rodando* / *dois experimentos medidos* / *artigo em revisão*

**O que falar:**
> "No TCC I eu apresentei a proposta. Hoje o sistema está implementado, rodando, e eu tenho dois experimentos medidos em cima dele. O que eu vou mostrar em oito minutos é o que os números dizem, inclusive onde ele falha, porque a parte mais útil do que eu descobri nas últimas semanas foi uma falha."

> **Transição:** *"Começo pelo que o sistema faz, em uma frase."*

---

## Slide 2 — O que é, em uma frase, e a demonstração

**Tempo:** 90s (a maior parte é a tela, não o slide)

**O que mostrar na tela:**
- Uma frase só: *"Responde perguntas sobre uma pilha de documentos e mostra a página de onde tirou a resposta."*
- Depois troque para o sistema ao vivo.

**O que fazer na demonstração:**
1. Abrir a coleção "Licitações: Lei 8.666 e Lei 14.133"
2. Perguntar: *"A Lei 8.666 ainda está em vigor?"*
3. Esperar a resposta com as fontes
4. **Clicar na fonte** e mostrar a página abrindo no trecho

**O que falar:**
> "É isso. Eu subo os PDFs, pergunto em português, e a resposta vem com o documento e a página. E esse clique aqui é o ponto do trabalho inteiro: a citação não é o modelo dizendo de onde tirou. É a coordenada do trecho que o banco realmente entregou para ele."

**Se a demonstração falhar:** siga em frente sem improvisar conserto. Diga
*"tenho o resultado medido disso nos próximos slides"* e vá para o slide 3. Os
números não dependem da demonstração funcionar na hora.

> **Transição:** *"A pergunta óbvia é: por que não jogar o PDF inteiro no ChatGPT? Eu medi isso."*

---

## Slide 3 — O número que sustenta o trabalho

**Tempo:** 60s

**O que mostrar na tela:**

| | Contexto recuperado | Documento inteiro | Coleção inteira |
|---|---|---|---|
| Acerto | 1,000 | 0,986 | — |
| Tokens por pergunta | **4.215** | 72.067 | 250.504 |

**O que falar:**
> "Eu comparei o meu sistema com a alternativa de jogar o documento inteiro no modelo. Em acerto, empata: a diferença não é estatisticamente distinguível. O que muda é o custo: 17 vezes menos que o documento inteiro, 59 vezes menos que a coleção inteira. E tem um limite que não é de preço: numa das execuções a coleção não coube no contexto, o provedor recusou com 227 mil tokens contra um limite de 131 mil. Acima de certo acervo, a alternativa deixa de ser possível a qualquer preço. A minha não muda de regime."

> **Transição:** *"Só que medir num acervo só não prova nada, então eu montei mais três."*

---

## Slide 4 — Três temas diferentes, e o que eles mostram

**Tempo:** 60s

**O que mostrar na tela:**

| Tema | PDFs | Perguntas | Acerto |
|---|---|---|---|
| Segurança do trabalho (NRs) | 5 | 12 | 0,750 |
| Produto e consumidor (CDC, manual, garantia) | 3 | 12 | 0,917 |
| Licitações (lei revogada e vigente) | 3 | 14 | 0,643 |
| **Total** | **11** | **38** | **0,763** |

**O que falar:**
> "Três acervos de assuntos bem diferentes: normas de segurança do trabalho, documentos de consumidor, e legislação de licitação. Em cada um eu coloquei de propósito documentos que se contradizem, porque acervo real tem isso. O resultado agregado é 76%. E antes de eu fazer isso, o mesmo sistema dava 100% em três dos quatro temas que eu tinha. Ou seja, o teste anterior era fácil demais, e eu vou explicar por quê."

> **Transição:** *"O motivo do 100% anterior é o achado mais importante que eu tenho."*

---

## Slide 5 — O achado: a pergunta feita do jeito que a pessoa fala

**Tempo:** 90s

**O que mostrar na tela:**
- Duas perguntas lado a lado, mesma resposta esperada:
  - *"Qual é o nível de ação para agentes químicos?"* → **acertou**
  - *"A partir de quanto de produto químico no ar a empresa já tem que tomar providência?"* → **errou**
- E o número: literal **0,875** contra paráfrase **0,625**

**O que falar:**
> "As duas perguntas querem a mesma informação e têm a mesma resposta na mesma página. A de cima usa as palavras do documento. A de baixo usa as palavras que uma pessoa usaria. Ele acerta a primeira e erra a segunda. Quando eu escrevi todas as perguntas nas duas versões, o acerto caiu de 87% para 62%."
>
> "E aqui está o que isso me ensinou sobre o meu próprio teste anterior: as perguntas do gabarito antigo foram escritas por mim, depois de eu ter lido a página. Sem perceber, eu herdei o vocabulário do documento. Os 100% não mediam compreensão, mediam coincidência de palavras."

**Se perguntarem "e como conserta":** *"Expansão da consulta por sinônimos antes
da busca, com cache para não perder o determinismo. Está mapeado, não está feito."*

> **Transição:** *"Tem um segundo modo de falha, e esse é o mais grave."*

---

## Slide 6 — O erro que eu não resolvo ainda

**Tempo:** 90s

**O que mostrar na tela:**
- A pergunta: *"A modalidade tomada de preços existe?"*
- A resposta do sistema: *"Sim, conforme o Art. 22 da Lei 8.666, página 16."*
- Embaixo, em vermelho: *a Lei 8.666 foi revogada em 30/12/2023*

**O que falar:**
> "Essa resposta está certa e é enganosa ao mesmo tempo. O artigo existe, a página existe, quem clicar na fonte vai encontrar exatamente o que ele disse. Só que essa lei foi revogada, e ele não avisa. Num acervo de normas, isso tem consequência: alguém usaria uma regra que não vale mais."
>
> "Isso é o que eu considero o achado mais importante do trabalho, porque ele mostra que **citação conferível é condição necessária e não suficiente**. Não adianta eu provar de onde veio a resposta se eu não sei dizer se aquilo ainda vale."

> **Transição:** *"E é daí que sai o que eu vou fazer agora."*

---

## Slide 7 — O que vem a seguir

**Tempo:** 45s

**O que mostrar na tela:**
1. Relação temporal entre documentos: distinguir *um revoga o outro* de *os dois valem, em épocas diferentes*
2. Expansão da pergunta por sinônimos, com cache
3. Revisor externo para o gabarito

**O que falar:**
> "Três frentes. A primeira é tratar a relação entre documentos: saber que um revogou o outro, e que contrato antigo continua na lei antiga. Dá para fazer com data de vigência e comparação de intervalos em SQL, sem precisar de banco de grafo, o que preserva a tese. A segunda é a expansão da pergunta, que ataca a queda de vocabulário. A terceira é pedir para outra pessoa escrever parte do gabarito, porque o meu foi escrito por mim, e eu já vi que isso vicia o resultado."

---

## Perguntas prováveis, com a resposta pronta

**"Quem escreveu essas perguntas?"**
> "Eu, a partir dos PDFs, sem consultar o sistema. E é uma limitação que eu declaro: foi exatamente por isso que as primeiras deram 100%. Na versão atual eu escrevo cada pergunta em duas formas para controlar esse viés, e revisor externo está no plano."

**"76% é pouco, não?"**
> "Depende do que compara. O acervo foi montado de propósito com documentos que se contradizem, que é o caso difícil. Num acervo comum ele vai a 92%. O número baixo é onde eu quero trabalhar, não onde eu quero esconder."

**"E a janela de contexto grande não resolve tudo?"**
> "Resolve o acerto, não resolve o custo nem a citação. E tem limite físico: a minha coleção não coube em 131 mil tokens. Além disso existe efeito medido de perda de desempenho quando a informação está no meio de um contexto longo, que é o trabalho do Liu e outros, de 2024."

**"Funciona sem IA?"**
> "Indexa, recupera e cita sem IA nenhuma. Só o texto final da resposta depende do modelo."

**"Você vai nichar o sistema?"**
> "Pensei nisso e decidi que não, por enquanto. Eu tenho três domínios medidos e resultado diferente em cada um. Fechar num nicho agora jogaria fora essa evidência. Se aparecer um objetivo claro para nichar, eu nicho com dado na mão."

---

## Checagem antes de entrar na sala

- [ ] Stack de pé: `curl localhost:8000/api/v1/health` responde
- [ ] Login da conta de demonstração funciona
- [ ] Provedor de LLM com crédito (a chave principal, conferir em `ollama.com/api/usage`)
- [ ] A coleção de licitações abre e responde
- [ ] Ter os números impressos, para o caso de a tela falhar
