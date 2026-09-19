# Acompanhamento de TCC II — setembro de 2026

**Tempo total: ~15 minutos | 23 slides + demonstração ao vivo**

> **A fala de cada slide está nas notas do apresentador, dentro do próprio deck.**
> Este arquivo não repete essa fala. Ele tem só o que o deck não pode ter: o
> roteiro da demonstração ao vivo, as respostas prontas para a arguição e a
> checagem antes de entrar na sala.

> Este roteiro substitui, para esta aula, o `apresentacao.md`, que é do TCC I e
> responde "o sistema já funciona?" com "não, a implementação será no TCC II".
> Agora funciona e está medido, e essa é a notícia da apresentação.

> Regra que atravessa a apresentação inteira: **apresente o erro antes que
> perguntem**. Banca não pune falha medida com causa identificada. Pune falha
> que ela descobre sozinha.

---

## Ritmo

| Trecho | Slides | Tempo |
|---|---|---|
| Problema, pergunta de pesquisa, como funciona | 1–5 | 3 min |
| Demonstração ao vivo e estado do trabalho | 6–7 | 2,5 min |
| Custo, teto de contexto, citação | 8–11 | 2,5 min |
| Como o número foi produzido, e os números | 13–16 | 3,5 min |
| As duas falhas, o que não funcionou e o que funcionou | 17–20 | 3 min |
| Limitações e próximos passos | 21–23 | 1 min |

Se o tempo apertar, corte o slide do índice vetorial aproximado (determinismo).
É o único que sustenta sozinho a defesa da escolha de arquitetura, e a banca
desta aula não está julgando arquitetura.

---

## A demonstração ao vivo (slide 6)

Único trecho sem rede de segurança. Ensaie duas vezes antes.

1. Abrir a coleção **"Licitações: Lei 8.666 e Lei 14.133"**
2. Perguntar: *"A Lei 8.666 ainda está em vigor?"*
3. Esperar a resposta com as fontes
4. **Clicar na fonte** e mostrar a página abrindo no trecho citado

A frase que justifica o clique:

> "Esse clique é o ponto do trabalho inteiro: a citação não é o modelo dizendo
> de onde tirou. É a coordenada do trecho que o banco realmente entregou para ele."

**A máquina roda com a expansão da pergunta ligada** (`THINGINDEX_QUERY_EXPANSION=on`),
que é o braço medido no slide 20. A pergunta da demo foi testada nesse estado e
responde certo, citando a revogação. Se alguém perguntar por que o sistema ao vivo
está melhor que a tabela do slide 14: porque a tabela é o braço de controle, e o
slide 20 mede a diferença.

**Se falhar:** não improvise conserto e não peça desculpas duas vezes. Diga
*"tenho o resultado medido disso nos próximos slides"* e siga. Nenhum número da
apresentação depende da demonstração funcionar na hora.

**Segunda pergunta, só se sobrar tempo:** *"Como aciono os três anos de
garantia?"* na coleção de produto. A garantia é de um ano, e a resposta certa é
corrigir a premissa. É o comportamento mais difícil de acreditar sem ver.

---

## Perguntas prováveis, com a resposta pronta

**"Você tem certeza dessa taxa de acerto?"** — é o slide 13, mostre-o.
> "São 71 perguntas em três acervos, três repetições cada, 213 respostas. Das 71,
> 69 deram exatamente o mesmo resultado nas três repetições: quando ele erra, erra
> sempre, e dá para investigar. As perguntas estão travadas por hash desde antes
> de eu rodar, o CSV bruto está versionado, e o acerto de 0,817 se repete nos três
> acervos, entre 0,783 e 0,841. O que eu não tenho é gabarito escrito por outra
> pessoa, e isso está declarado."

**"Qual desses números você defende como significativo?"**
> "Um só: o efeito da expansão da pergunta. Dez perguntas passaram a acertar e
> uma deixou de acertar, teste de sinais p igual a 0,012. A queda de vocabulário,
> que é o efeito maior, dá p igual a 0,109: é direcional e eu apresento como
> direcional. Os outros números são descritivos."

**"Quem escreveu essas perguntas?"**
> "Eu, a partir dos PDF, sem consultar o sistema. É uma limitação que eu declaro:
> foi exatamente por isso que as primeiras deram 100%. Na versão atual eu escrevo
> cada pergunta em duas formas para controlar esse viés, e revisor externo está
> no plano."

**"Como você sabe que a resposta está certa?"**
> "Cada pergunta tem resposta esperada e página-alvo escritas antes. O harness
> confere as duas coisas: o conteúdo da resposta e se a página citada é a do
> gabarito. Não é avaliação por impressão."

**"E se o modelo variar de uma execução para a outra?"**
> "Foi o que eu medi com as três repetições: 69 das 71 perguntas estáveis. E o
> acervo de segurança do trabalho eu rodei duas vezes inteiro, em execuções
> separadas, com resultado idêntico nas 23 perguntas."

**"Você mexeu no gabarito depois de ver o resultado?"**
> "Mexi numa coisa e declaro qual: o meu corretor marcava como erro recusas
> corretas escritas com outras palavras. Corrigi o padrão e rodei a bateria inteira
> de novo, do zero, nos três acervos. A execução antiga está versionada em
> `antes-correcao-negativas/` para comparação. Corrigir só os itens afetados daria
> o mesmo número por um caminho que ninguém pode auditar."

**"Esse número é baixo, não?"**
> "Depende do que compara. O acervo foi montado de propósito com documentos que
> se contradizem, que é o caso difícil. O número baixo é onde eu quero trabalhar,
> não onde eu quero esconder."

**"E a janela de contexto grande não resolve tudo?"**
> "Resolve o acerto, não resolve o custo nem a citação. E tem limite físico: numa
> das execuções a coleção não coube, o provedor recusou com 227 mil tokens contra
> um limite de 131 mil. Além disso existe efeito medido de perda de desempenho
> quando a informação está no meio de um contexto longo, que é o trabalho do Liu
> e outros, de 2024."

**"Funciona sem IA?"**
> "Indexa, recupera e cita sem IA nenhuma; eu testei numa conta separada. O que
> não funciona sem modelo é redigir o texto final da resposta: o chat recusa."

**"Por que não banco vetorial?"**
> "Medi o preço do índice aproximado: em todas as escalas até mil documentos,
> 100% das consultas mudaram de resposta só trocando a semente. E ordem de
> inserção não é configuração, é histórico de uso."

**"Você vai nichar o sistema?"**
> "Pensei e decidi que não, por enquanto. Tenho três domínios medidos e resultado
> diferente em cada um. Fechar num nicho agora jogaria fora essa evidência."

**"Você conseguiu consertar a queda de vocabulário?"**
> "Consegui, e é o slide 20. Em vez de enriquecer o documento, que já tinha
> falhado, eu expando a pergunta com os termos técnicos que ela descreve sem
> nomear. O acerto vai de 0,817 para 0,930, e as parafraseadas, que eram o
> problema, de 0,677 para 0,903. Dez perguntas passaram a acertar e uma deixou de
> acertar, p igual a 0,012. Custa 13% mais tokens na busca."

**"E a pergunta que piorou?"**
> "É a que eu mais gosto de mostrar: 'aquela lei antiga de licitação ainda vale
> hoje'. Os termos expandidos puxaram as páginas da lei revogada. Eu registrei
> esse risco no pré-registro antes de rodar, justamente para não poder escondê-lo
> se acontecesse. É a mesma causa do slide 17, e é por isso que a frente 1 do
> plano é vigência de norma."

**"A expansão não quebra o determinismo?"**
> "A primeira vez que uma pergunta aparece, sim: é uma chamada de LLM. Depois
> disso ela fica em cache pela hash da pergunta normalizada, e a mesma pergunta
> devolve sempre os mesmos termos. Eu declaro essa ressalva."

**"O passo de IA na indexação serve para quê, então?"**
> "Para leitura na tela, não para busca. Eu medi, está no slide 18, e o resultado
> foi nulo nas quatro versões do índice. Deixei registrado como resultado
> negativo em vez de tirar do artigo."

---

## Checagem antes de entrar na sala

- [ ] Stack de pé: `curl localhost:8000/api/v1/health` responde
- [ ] Login da conta de demonstração funciona
- [ ] Provedor de LLM com crédito (conferir em `ollama.com/api/usage`)
- [ ] A coleção de licitações abre e responde
- [ ] `docker compose exec api printenv THINGINDEX_QUERY_EXPANSION` devolve `on`
- [ ] O deck abre e passa slide (testar no navegador da sala, não só no seu)
- [ ] Números impressos em papel, para o caso de a tela falhar
