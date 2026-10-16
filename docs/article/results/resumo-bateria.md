> **Histórico.** Saída de `scripts/resumo-bateria.py` gravada em ae39a1e (2026-09-19), com o gabarito original e a versão do script que tinha um bug: os pares literal/paráfrase guardavam só a última repetição de cada pergunta (corrigido em cb5a373). Por isso a lista abaixo dá 9 pares só-literal e 1 só-paráfrase; com o critério de acerto nas três repetições são 8 e 2 (p = 0,109). Os números atuais estão no artigo; a conta do gabarito original é `python3 scripts/redacao-gabarito-original.py` (`--ultima-repeticao` reproduz esta lista).


## Por tema

| Tema | Acertos | Taxa |
|---|---|---|
| licitacoes | 62/75 | 0.827 |
| produto | 58/69 | 0.841 |
| seguranca-trabalho | 54/69 | 0.783 |
| **todos** | 174/213 | 0.817 |

## Por forma da pergunta

| Forma | Acertos | Taxa |
|---|---|---|
| aberta | 49/78 | 0.628 |
| confirmacao | 27/36 | 0.750 |
| existencia | 36/36 | 1.000 |
| premissa-falsa | 35/36 | 0.972 |
| negativa | 27/27 | 1.000 |

## Por formulação

| Formulação | Acertos | Taxa |
|---|---|---|
| literal | 84/93 | 0.903 |
| parafrase | 63/93 | 0.677 |

Queda literal para paráfrase: **0.903 para 0.677** (22.6 pontos).

## Pares em que a formulação decidiu o resultado

- `lc1`: só a versão **literal** acertou
- `lc3`: só a versão **parafrase** acertou
- `pr11`: só a versão **literal** acertou
- `pr8`: só a versão **literal** acertou
- `pr9`: só a versão **literal** acertou
- `sg2`: só a versão **literal** acertou
- `sg3`: só a versão **literal** acertou
- `sg4`: só a versão **literal** acertou
- `sg8`: só a versão **literal** acertou
- `sg9`: só a versão **literal** acertou

## Estabilidade entre repetições

Repetições por pergunta: [3]

| Comportamento | Perguntas |
|---|---|
| Acertou em todas | 57 |
| Errou em todas | 12 |
| Oscilou entre repetições | 2 |
| **Total** | **71** |

Oscilaram: `lc13L`, `pr8L`
