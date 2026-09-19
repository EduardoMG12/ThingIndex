# Correção do corretor nas perguntas negativas (19/09/2026)

## O defeito

As perguntas do estrato `negativa` são aquelas cuja resposta certa é **recusar**:
a informação não está no acervo, e o sistema tem que dizer isso em vez de
inventar. O acerto é conferido por expressão regular sobre a resposta.

Os três gabaritos novos (`seguranca-trabalho`, `produto`, `licitacoes`) foram
escritos com uma versão **estreita** dessa expressão:

```
nao consta|nao informa|nao ha|nao encontr|nao especific|nao define|nao traz|nao estabelec
```

O gabarito anterior do projeto (`gabarito-si.json`) já usava uma versão ampla,
com `menciona`, `aparece`, `possui`, `nenhuma informac` e as formas em inglês.
Ou seja: a versão estreita é uma regressão introduzida ao escrever os gabaritos
novos, não um critério deliberado.

## A evidência

Duas perguntas de produto foram contadas como erro tendo a resposta correta:

| Pergunta | Resposta do sistema | Por que caiu |
|---|---|---|
| `pr6` "Qual é o preço do monitor?" | "As informações fornecidas nos documentos **não mencionam** o preço do monitor Philips 241V8." | `mencion` fora do padrão |
| `pr7` "Qual é o endereço da assistência técnica mais próxima de mim?" | "As informações fornecidas nos documentos **não contêm** endereços de assistência técnica." | `cont` fora do padrão |

As duas são recusas corretas. O sistema fez o que se pedia dele e o corretor
marcou errado.

## O que foi feito

1. As nove perguntas negativas dos três gabaritos passaram a usar o padrão amplo
   do `gabarito-si.json`, acrescido de `cont` e de `nenhum (endereco|preco|valor|dado)`.
2. `frozen_sha256` recalculado nos três arquivos:

   | Gabarito | Novo hash |
   |---|---|
   | seguranca-trabalho | `8dfd376387d0064dc8ab9d11911eba15065a25883e7d9fe051caafea239e8de4` |
   | produto | `4b5186d6fcb40d6857020f1211e0b8794468ae2ea86d1c0be0fa6e32a7a85b63` |
   | licitacoes | `6d1033608edb060e9d7efa7f404aa2083a8c94e610f35424526a9f2f6169b0c7` |

3. A bateria inteira foi rodada de novo, do zero, nos três temas.

## O que isso muda no que pode ser afirmado

Esta correção só pode **subir** o número, e por isso precisa ser declarada em vez
de embutida: quem lê tem que poder separar "o sistema melhorou" de "o corretor
parou de errar". Nada além do estrato negativa foi tocado.

Os CSV da execução anterior, com o padrão estreito, estão preservados em
`antes-correcao-negativas/` para a comparação. Nela, segurança do trabalho e
produto deram **0,783** cada um, com as negativas de produto em 4 de 9.

## O efeito da correção

| | Antes | Depois |
|---|---|---|
| Segurança do trabalho | 0,783 (54/69) | 0,783 (54/69) |
| Produto | 0,783 (54/69) | 0,841 (58/69) |
| Negativas, agregado nos três acervos | --- | 1,000 (27/27) |

Segurança do trabalho não mudou: as três negativas dele já casavam o padrão
estreito. A comparação item a item das duas execuções desse acervo dá **zero
divergência nas 23 perguntas**, o que serve como verificação independente de
reprodutibilidade: seis repetições concordantes ao todo.

Nenhuma pergunta, resposta esperada, página-alvo ou estrato foi alterado. A
mudança é exclusivamente na expressão que reconhece uma recusa.
