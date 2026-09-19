# Pré-registro: expansão da pergunta antes da busca

**Status:** pré-registrado em 2026-09-19, antes de qualquer implementação
**Branch:** `feat/proveniencia-completa-e-multidominio`

> Escrito **antes** da intervenção. Fixa a hipótese, o prompt exato, os braços e
> o critério de sucesso. O experimento anterior (`pre-registro-perguntas-geradas.md`)
> mostra por que isso importa: sem o registro prévio, um resultado nulo vira
> "ainda não achei o prompt certo" e a medição nunca termina.

## O que já está estabelecido (não é o que se vai testar)

1. O sistema acerta **0,903** quando a pergunta reutiliza o vocabulário do
   documento e **0,677** quando a mesma necessidade é expressa com as palavras
   de quem não leu. São 31 pares, 10 discordantes, 9 para o mesmo lado, teste de
   sinais $p = 0{,}021$. É o único efeito com significância no trabalho.
2. Expandir o **documento** não resolve. Quatro braços de enriquecimento, nulo
   em todos, nenhuma pergunta mudou de resultado.
3. A causa apurada: pedir à LLM que escreva "linguagem de leigo" produz a
   paráfrase do documento, não a da pessoa. Existem muitas formas leigas de
   dizer a mesma coisa e uma só forma técnica.

## Hipótese

A assimetria do item 3 é a razão para tentar o caminho oposto. Existem muitas
maneiras leigas de perguntar e **uma** maneira técnica de nomear a coisa. Ir de
muitas para uma é um problema mais fácil que ir de uma para muitas.

> Se a pergunta for expandida com os termos técnicos que ela descreve sem
> nomear, a recuperação das perguntas parafraseadas melhora, e a das literais
> fica aproximadamente igual, porque estas já trazem os termos do documento.

Caso concreto que o experimento tem que resolver (`sg2P`, congelada por hash
antes desta ideia existir):

> "Quem paga o capacete e a luva que eu uso no serviço?"

A página-alvo (NR-6, p. 2) fala em "equipamento de proteção individual",
"fornecer gratuitamente" e "organização". A pergunta e a página não compartilham
**nenhum** termo significativo. A expansão precisa produzir "EPI", "equipamento
de proteção individual" e "fornecimento" a partir de "capacete" e "luva", sem
jamais ver a página.

## O que garante que não há circularidade

A expansão recebe **apenas o texto da pergunta**. Ela nunca vê o acervo, a
página-alvo, a resposta esperada nem o gabarito. Um termo expandido que acerte o
vocabulário do documento é conhecimento de mundo do modelo, não vazamento.

As 71 perguntas e os 31 pares continuam travados pelos mesmos `frozen_sha256`.
Nenhuma pergunta, resposta esperada ou página-alvo será alterada.

## O prompt, congelado

**Roda uma vez. Não haverá segunda versão nesta medição.** Ajuste posterior
exige pré-registro novo e a medição recomeça.

Prompt de sistema:

```
You expand a search query for a lexical (BM25) index over Portuguese documents.
Return only a single line of terms separated by spaces.
Do not answer the question.
Do not explain.
Write the technical or formal terms that the question describes without naming them.
Include the singular form of each term.
At most 8 terms.
If the question already uses the formal vocabulary, return an empty line.
```

Prompt de usuário:

```
Pergunta: <texto da pergunta>
Termos:
```

O hash deste prompt está travado em
`TestPromptDaExpansaoEstaCongelado`:
`bbc051efd4d6636b2682c7ffd33866845aaf894da6ad0172ccbe289bf44d2250`. Alterá-lo
quebra o build.

## Os dois braços

| Braço | Texto que vai para a recuperação |
|---|---|
| **C** (controle) | a pergunta como o usuário escreveu |
| **E** (tratamento) | a pergunta, mais os termos expandidos |

O braço C é o comportamento atual do sistema, já medido: 174/213.

## Determinismo

A expansão é uma chamada de LLM, e portanto não é determinística por si. O
resultado é congelado por cache com chave `sha256(pergunta normalizada)`: a
mesma pergunta produz sempre os mesmos termos depois da primeira vez. Isso
preserva o princípio II da constituição do projeto (recuperação determinística)
com a ressalva, que será declarada, de que a **primeira** ocorrência de uma
pergunta inédita depende do modelo.

O cache é o de enriquecimento que já existe, com `kind = "query"`. Não há tabela
nova.

## Predição, registrada antes de rodar

- **Se a hipótese estiver certa:** o acerto nas parafraseadas sobe; o das
  literais fica aproximadamente igual; o número de pares discordantes cai.
- **Se subir nas duas igualmente:** o efeito é de ter mais termos no índice, não
  de ponte de vocabulário, e a hipótese específica está errada.
- **Se não subir em nenhuma:** terceira evidência independente de que a ponte de
  vocabulário não se faz com LLM neste sistema, e a direção do trabalho muda.
- **Se cair:** é o desfecho que o pré-registro existe para tornar reportável.
  Termos a mais podem puxar páginas erradas, e um índice lexical não distingue
  termo bom de termo ruim. Uma queda será publicada com o mesmo destaque de uma
  subida.

## Os quatro números

|  | Literal | Paráfrase |
|---|---|---|
| C, sem expansão (já medido) | 0,903 | 0,677 |
| E, com expansão | a medir | a medir |

## Análise estatística

A comparação é **pareada por pergunta entre os braços**, não entre formulações:
as mesmas 71 perguntas, três repetições, nos dois braços. Teste de sinais exato
sobre as perguntas cujo resultado mudou de braço. Perguntas que acertam ou erram
nos dois braços não entram, porque não carregam informação sobre a diferença.

Com 71 perguntas, um resultado só será chamado de significativo se o teste der
$p < 0{,}05$; caso contrário será relatado como direcional e inconclusivo.

## Custo

Uma chamada de LLM por pergunta inédita, com prompt de uma linha. É custo novo
no caminho da pergunta, que hoje não tem nenhum. Será reportado em tokens por
pergunta, ao lado dos 4.215 atuais, porque o argumento central do trabalho é de
custo e piorá-lo em silêncio invalidaria o resto.

## Afirmação máxima permitida na apresentação

> "A queda de vocabulário é o único efeito com significância que eu medi, e eu
> ataquei ela expandindo a pergunta, não o documento. Em 71 perguntas
> pré-registradas, três repetições, o acerto nas parafraseadas foi de 0,677 para
> X. O prompt e o critério estavam congelados antes de rodar, e o braço de
> controle é o sistema atual. São 71 perguntas de um autor só: é indício
> direcional, e a ressalva de que a primeira ocorrência de cada pergunta depende
> do modelo permanece."

Qualquer afirmação além desta, nesta amostra, é indefensável.
