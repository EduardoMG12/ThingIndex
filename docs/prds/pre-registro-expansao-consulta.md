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
   de quem não leu. São 31 pares e 10 discordantes, 8 a favor do literal: teste
   de sinais $p = 0{,}109$, direcional e **não** significativo. É o maior efeito
   medido no trabalho, e o mais reproduzível entre acervos.
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

Os valores acima são por resposta (213 respostas). A análise pareada usa o
critério por pergunta, mais exigente: a pergunta conta como acerto quando
acertou nas **três** repetições.

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

> "A queda de vocabulário é o maior efeito que eu medi, e eu ataquei ela
> expandindo a pergunta, não o documento. Em 71 perguntas
> pré-registradas, três repetições, o acerto nas parafraseadas foi de 0,677 para
> X. O prompt e o critério estavam congelados antes de rodar, e o braço de
> controle é o sistema atual. São 71 perguntas de um autor só: é indício
> direcional, e a ressalva de que a primeira ocorrência de cada pergunta depende
> do modelo permanece."

Qualquer afirmação além desta, nesta amostra, é indefensável.

---

# Resultado (2026-09-19, após a execução)

Escrito depois de rodar, contra o pré-registro acima. Nada do que está acima foi
alterado, exceto a correção do $p$ do efeito de vocabulário, descrita ao fim.

## Checagem de mecanismo, feita antes da bateria

Pergunta `sg2P`, congelada por hash antes desta ideia existir:

> "Quem paga o capacete e a luva que eu uso no serviço?"

Termos que a expansão produziu, vendo apenas o texto da pergunta:

> `equipamento proteção individual epi fornecedor empregador custo pagamento`

A resposta passou a citar NR-6 página 2, que é a página-alvo do gabarito. Essa
pergunta errava nas três repetições do braço de controle. A previsão de
mecanismo se confirmou antes de qualquer número agregado.

## Os quatro números

Por resposta, 213 respostas por braço:

|  | Literal | Paráfrase | Todas |
|---|---|---|---|
| C, sem expansão | 0,903 | 0,677 | **0,817** |
| E, com expansão | 0,935 | 0,903 | **0,930** |

Por pergunta, critério de acerto nas três repetições:

|  | Literal | Paráfrase | Todas |
|---|---|---|---|
| C, sem expansão | 27/31 | 21/31 | 57/71 |
| E, com expansão | 29/31 | 28/31 | 66/71 |

**A predição registrada se confirmou na forma específica.** As parafraseadas
subiram 7 perguntas; as literais subiram 2. Não foi uma subida igual nas duas,
que o pré-registro classificava como efeito genérico de acrescentar termos. A
distância entre as duas formulações cai de 0,194 para **0,032**: a lacuna de
vocabulário praticamente fecha.

## Análise pareada entre braços

| | |
|---|---|
| Só E acertou | **10** |
| Só C acertou | **1** |
| Concordantes | 60 de 71 |

Teste de sinais exato, bilateral: **$p = 0{,}0117$**. É significativo ao nível
convencional, e é o único resultado deste trabalho que atinge esse patamar.

Das 10 perguntas ganhas, **8 são parafraseadas**. A única perda também é
parafraseada, e é informativa: `lc3P`, "Aquela lei antiga de licitação ainda vale
hoje?". Os termos expandidos trouxeram vocabulário da lei revogada e puxaram as
páginas dela, que é exatamente o risco declarado no pré-registro — um índice
lexical não distingue termo bom de termo ruim.

## Proveniência e custo

| | C | E |
|---|---|---|
| Proveniência da resposta | 0,371 | **0,532** |
| Tokens de prompt por pergunta | 4.699 | **5.327** |

A recuperação ficou 13,4% mais cara em tokens de prompt, porque a consulta
expandida traz mais páginas candidatas. A isso soma-se **uma chamada de LLM por
pergunta inédita**, com prompt de uma linha, cacheada depois da primeira
ocorrência: na bateria inteira foram 71 chamadas para 213 respostas. O custo
dessa chamada não foi instrumentado e está declarado como não medido; o que está
medido é o aumento no prompt da resposta.

Mesmo com o aumento, o braço E continua uma ordem de grandeza abaixo do braço de
documento inteiro (72.067 tokens por pergunta).

## Determinismo

A ressalva registrada permanece e não foi eliminada: a **primeira** ocorrência de
uma pergunta inédita depende do modelo. Da segunda em diante, o cache por
`sha256` da pergunta normalizada devolve sempre os mesmos termos. Na bateria, as
três repetições de cada pergunta usaram termos idênticos, por construção.

## Correção de um erro na análise anterior

Ao escrever a comparação pareada entre braços, apareceu que o script
`resumo-bateria.py` montava os pares literal/paráfrase dentro do laço de linhas
do CSV, de modo que a **última repetição sobrescrevia as outras**. O teste de
sinais do efeito de vocabulário saía, portanto, de uma repetição só.

Com o critério correto, agregando as três repetições, o efeito de vocabulário do
braço C é de 10 pares discordantes, 8 a favor do literal, **$p = 0{,}109$**:
direcional e **não** significativo. O valor $p = 0{,}021$ reportado antes estava
errado e foi corrigido no artigo, no roteiro de apresentação e nos slides.

O efeito continua sendo o maior medido no trabalho e se repete nos três acervos;
o que muda é que ele não pode ser chamado de significativo com 31 pares. O
resultado que passa em teste é o desta intervenção.

## Afirmação máxima, revisada com os números na mão

> "A lacuna de vocabulário é o maior efeito que eu medi: 0,903 contra 0,677. Ela
> é direcional e não significativa com 31 pares, p igual a 0,109. Eu ataquei a
> causa expandindo a pergunta em vez do documento, com prompt congelado antes de
> rodar, e o acerto foi de 0,817 para 0,930, com as parafraseadas indo de 0,677
> para 0,903. Pareado por pergunta, dez passaram a acertar e uma deixou de
> acertar: p igual a 0,012. Custa 13% mais tokens na recuperação e uma chamada de
> LLM por pergunta inédita, cacheada depois. São 71 perguntas de um autor só, e a
> primeira ocorrência de cada pergunta depende do modelo."

## Ameaças à validade deste resultado

- Setenta e uma perguntas de um único autor, o mesmo do gabarito.
- A mesma família de modelo expande a pergunta e redige a resposta.
- A expansão entra na consulta como texto, então também alimenta a regra de
  linha de tabela, que exige dois termos significativos. Parte do ganho pode vir
  desse caminho, e este experimento não o separa.
- A única perda mostra que o risco de puxar vocabulário errado é real e não
  hipotético.
