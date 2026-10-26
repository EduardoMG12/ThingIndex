#!/usr/bin/env python3
"""Confere os numeros afirmados no artigo contra o CSV de resultados.

Prosa se descola do artefato que ela descreve. O caso mais perigoso nao e o
numero errado obvio -- e o numero que ficou desatualizado depois de uma
remedicao: continua plausivel, ninguem relê a frase, e sobrevive ate a defesa.

Este script nao interpreta o texto. Ele mantem uma lista EXPLICITA de
afirmacoes (numero, onde aparece, como e calculado) e recalcula cada uma a
partir de docs/article/results/results.csv. Alterar um numero no artigo sem
atualizar aqui quebra o CI, que e exatamente a intencao.

Uso:
    python3 scripts/check-article-numbers.py
"""

from __future__ import annotations

import collections
import csv
import json
import importlib.util
import math
import statistics
import sys
import re
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CSV = RAIZ / "docs" / "article" / "results" / "results.csv"
ARTIGO = RAIZ / "docs" / "article" / "artigo.tex"
# Tabelas e frases que sairam do artigo na simplificacao das metricas. O
# arquivo e um documento LaTeX independente (nao entra no artigo por \input),
# e os numeros dele continuam conferidos aqui contra os CSV.
APOIO = RAIZ / "docs" / "article" / "tables" / "apoio-resultados.tex"
PROVENIENCIA = RAIZ / "docs" / "article" / "tables" / "provenance.tex"

# Tolerancia: o artigo publica 3 casas decimais.
TOL = 0.0005


def carregar() -> list[dict]:
    if not CSV.exists():
        sys.exit(f"erro: {CSV} nao existe -- rode 'go run ./cmd/retrieval-eval' primeiro")
    with CSV.open(encoding="utf-8") as f:
        linhas = list(csv.DictReader(f))
    if not linhas:
        sys.exit(f"erro: {CSV} nao tem dados")
    return linhas


def media(rows: list[dict], gran: str, arm: str, metrica: str) -> float | None:
    vals = [
        float(r[metrica])
        for r in rows
        if r["negative"] == "false" and r["granularity"] == gran and r["arm"] == arm
    ]
    return sum(vals) / len(vals) if vals else None


def _numeros_geracao():
    spec = importlib.util.spec_from_file_location("numeros_geracao", RAIZ / "scripts" / "numeros-geracao.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _proveniencia_geracao():
    spec = importlib.util.spec_from_file_location("proveniencia_geracao", RAIZ / "scripts" / "proveniencia-geracao.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _br(valor: float, casas: int = 3) -> str:
    return f"{valor:.{casas}f}".replace(".", ",")


def _pct(valor: float) -> str:
    """Proporcao como o artigo a publica: em porcentagem, com uma casa.
    Parte do valor ja arredondado em 3 casas (_br), para que 0,8125 vire 81,2
    exatamente como no texto, e nao 81,3 por arredondamento binario."""
    milesimos = round(float(_br(valor).replace(",", ".")) * 1000)
    return f"{milesimos / 10:.1f}".replace(".", ",")


def _milhar(valor: float) -> str:
    return f"{round(valor):,}".replace(",", ".")


def conferir_geracao(texto: str, apoio: str) -> int:
    """Recalcula dos CSV os numeros das secoes de geracao, bateria,
    enriquecimento e expansao e exige que cada um apareca no artigo como
    esta escrito. Um numero que mude no CSV, ou que suma do texto, falha.
    As chaves de NO_APOIO sao conferidas contra apoio-resultados.tex, para
    onde as tabelas e frases correspondentes foram movidas."""
    ng = _numeros_geracao()
    g = ng.geracao()
    c, e = ng.bateria("off"), ng.bateria("on")
    en = ng.enriquecimento()
    ef = ng.efeito_formulacao(c)
    ex = ng.efeito_expansao(c, e)

    def taxa(par):
        return par[0] / par[1]

    def p_texto(p: float) -> str:
        return r"p < 0{,}001" if p < 0.001 else "p = " + _br(p).replace(",", "{,}")

    pg = _proveniencia_geracao()
    prov_t = pg.resumo("generation-results-T-ordem.csv", "T")["todas"]
    prov_f = pg.resumo("generation-results-F-document.csv", "F")["todas"]
    ic_c = ng.ic_conglomerado("off", ("seguranca-trabalho", "produto", "licitacoes"))
    ic_e = ng.ic_conglomerado("on", ("seguranca-trabalho", "produto", "licitacoes"))
    # O teste de redacao com o gabarito original, citado na Secao 8.7 e na
    # tabela-resumo.
    antigo = redacao_com_gabarito_original()

    esperados = {
        "tokens T": _milhar(g["T"]["todas"]["tokens"]),
        "tokens F": _milhar(g["F"]["todas"]["tokens"]),
        "pagina exibida T em respostas": f"{prov_t['exibida']} de {prov_t['respostas']}",
        "pagina citada T": f"correta em {prov_t['prosa_com_folio']} de {prov_t['respostas']} respostas em T",
        "pagina citada F": f"{prov_f['prosa_com_folio']} de {prov_f['respostas']} em F",
        # Sem aceitar o numero impresso (folio), so pela posicao da pagina no PDF.
        "pagina citada sem folio (8.5)": (f"Sem aceitar o número impresso, F teria {prov_f['prosa']}, "
                                          f"e T, {prov_t['prosa']}."),
        # Cada numero ancorado na frase em que aparece: um valor solto casaria
        # com qualquer outra ocorrencia (a banca mostrou que 0,685 -> 0,686
        # passava).
        "acerto da bateria (resumo)": f"com perguntas novas, o acerto foi de {_pct(taxa(c['total']))}\\%, e",
        "acerto E (resumo)": f"experimental, o acerto chegou a {_pct(taxa(e['total']))}\\%.",
        "acerto da bateria (secao)": f"a taxa de acerto foi de {_pct(taxa(c['total']))}\\%, cerca de 7 em cada 10 respostas",
        "acerto da bateria (conclusao)": f"a taxa de acerto foi de {_pct(taxa(c['total']))}\\% com a pergunta original. As par",
        "acerto E (conclusao)": f"elevou o acerto a {_pct(taxa(e['total']))}\\% (experimental)",
        "literal x parafrase C": f"o acerto foi {_pct(taxa(c['variante']['literal']))}\\% nas literais e {_pct(taxa(c['variante']['parafrase']))}\\% nas par",
        # A lacuna literal-parafrase e dita pelas duas taxas, sem e com expansao.
        "parafrase sem e com expansao": f"paráfrases subiu de {_pct(taxa(c['variante']['parafrase']))}\\% para {_pct(taxa(e['variante']['parafrase']))}\\%",
        "literal sem e com expansao": f"literais, que passou de {_pct(taxa(c['variante']['literal']))}\\% para {_pct(taxa(e['variante']['literal']))}\\%",
        "proveniencia C": _pct(taxa(c["prov"])),
        "literal E": _pct(taxa(e["variante"]["literal"])),
        "parafrase E": _pct(taxa(e["variante"]["parafrase"])),
        "acerto E": _pct(taxa(e["total"])),
        "proveniencia E": _pct(taxa(e["prov"])),
        "tokens C": _milhar(c["tokens"]),
        "tokens E": _milhar(e["tokens"]),
        # Sairam do corpo para apoio-resultados.tex (NO_APOIO); o 49 tambem
        # segue no corpo, na linha da Tabela da expansao.
        "sempre certa": f"{c['sempre_certa']} das 71 acertaram nas tr",
        "sempre errada": f"{c['sempre_errada']} erraram nas tr",
        "pares a favor da literal": f"{ef['a_favor_literal']} foram discordantes, todos a favor da literal",
        "p da formulacao": p_texto(ef["p"]),
        "expansao ganhou": f"{len(ex['ganhou'])} passaram a acertar",
        "expansao perdeu": "nenhuma deixou de acertar" if not ex["perdeu"] else f"{len(ex['perdeu'])} deixou de acertar",
        "p da expansao": p_texto(ex["p"]),
        "enriquecimento, fontes mudam": f"em {en['fontes_mudam']} das 12 perguntas",
        # Intervalo do total da bateria, na Secao 8.7 e na tabela-resumo.
        "ic C (8.7)": f"intervalo de 95\\% de {_pct(ic_c[0])}\\% a {_pct(ic_c[1])}\\%",
        "resumo acerto": f"{_pct(taxa(c['total']))}\\% das respostas, cerca de 7 em 10",
        # "todos a favor da literal" so vale enquanto nenhum par discordante
        # favorecer a parafrase.
        "resumo redacao": (f"{_pct(taxa(c['variante']['literal']))}\\% com as palavras do documento e "
                           f"{_pct(taxa(c['variante']['parafrase']))}\\% com outras; "
                           f"{ef['a_favor_literal']} pares discordantes, todos a favor da literal, com o gabarito revisado"
                           if ef["a_favor_parafrase"] == 0 else "<algum par discordante passou a favorecer a parafrase>"),
        # O "pode ser acaso" do gabarito original saiu da tabela-resumo na
        # simplificacao das metricas; a comparacao continua conferida, mas
        # contra o Anexo A7/A8 (apoio-resultados.tex), nao contra o corpo.
        "resumo redacao, gabarito original": (f"{antigo[0]} a {antigo[1]}, que pode ser acaso, com o gabarito original"
                           if antigo[2] >= 0.05 and ef["p"] < 0.05 else "<teste da redacao mudou de conclusao>"),
        "resumo expansao": (f"de {_pct(taxa(c['total']))}\\% para {_pct(taxa(e['total']))}\\% no total, e as "
                            f"paráfrases chegam a {_pct(taxa(e['variante']['parafrase']))}\\%; "
                            f"{len(ex['ganhou'])} perguntas ganhas"),
        # Comparacao com o gabarito original, como o corpo da Secao 8.7 a escreve.
        "redacao, gabarito original (corpo)": (
            f"eram {antigo[0]} pares a favor da literal e {antigo[1]} da paráfrase, sem diferença estatisticamente significativa"
            if antigo[2] >= 0.05 else "<redacao com o gabarito original passou a ser significativa>"),
        "premissa falsa (corpo)": (f"em só {c['forma']['premissa-falsa'][0]} de {c['forma']['premissa-falsa'][1]} "
                                   "respostas o sistema corrigiu o dado errado"),
        "negativas da bateria": (f"em todas as {c['forma']['negativa'][1]} respostas negativas"
                                 if c["forma"]["negativa"][0] == c["forma"]["negativa"][1]
                                 else "<negativas da bateria deixaram de acertar todas>"),
    }
    for braco in ("content", "content-summary", "content-questions", "max"):
        k, n = en[braco]["geral"]
        esperados[f"enriquecimento {braco}"] = f"{k} de {n}"

    # Linhas inteiras das tabelas: um numero isolado pode reaparecer em outro
    # paragrafo e mascarar a troca; a linha da tabela, nao.
    def linha(*celulas):
        return " & ".join(celulas) + r" \\"

    t, f = g["T"], g["F"]
    # Tabela de geracao: acerto e pagina citada, trechos (T) e PDF inteiro (F).
    for est, rot in (("prosa", "Prosa (30)"), ("tabela", "Tabela (21)"), ("calendario", "Calendário (16)")):
        esperados[f"tabela geracao {est}"] = linha(
            f"{rot:<15}", _pct(t[est]["acerto"]), _pct(f[est]["acerto"]),
            _pct(pg.taxa_estrato("generation-results-T-ordem.csv", est)),
            _pct(pg.taxa_estrato("generation-results-F-document.csv", est)))
    esperados["tabela geracao todas"] = linha(
        f"{'Todas (72)':<15}", _pct(t["todas"]["acerto"]), _pct(f["todas"]["acerto"]),
        _pct(prov_t["prosa_com_folio"] / prov_t["respostas"]), _pct(prov_f["prosa_com_folio"] / prov_f["respostas"]))
    # Tabela da expansao, virada: cada linha compara sem e com expansao.
    for rotulo, chave_c, chave_e in (
            ("literal", taxa(c["variante"]["literal"]), taxa(e["variante"]["literal"])),
            ("parafrase", taxa(c["variante"]["parafrase"]), taxa(e["variante"]["parafrase"])),
            ("todas", taxa(c["total"]), taxa(e["total"])),
            ("pagina", taxa(c["prov"]), taxa(e["prov"]))):
        esperados[f"tabela expansao {rotulo}"] = f"& {_pct(chave_c)} & {_pct(chave_e)} \\\\"
    esperados["tabela expansao tokens"] = f"& {_milhar(c['tokens'])} & {_milhar(e['tokens'])} \\\\"
    esperados["tabela expansao perguntas"] = f"& {c['sempre_certa']} de 71 & {e['sempre_certa']} de 71 \\\\"
    temas = (("seguranca-trabalho", "Segurança do trabalho"), ("produto", "Produto e consumidor "), ("licitacoes", "Licitações           "))
    for tema, rot in temas:
        k, n = c["tema"][tema]
        lo, hi = ng.ic_conglomerado("off", (tema,))
        certas = sum(all(v) for q, v in ng.por_pergunta("off", (tema,)).items())
        total_q = len(ng.por_pergunta("off", (tema,)))
        esperados[f"tabela bateria {tema}"] = linha(rot, _pct(k / n), f"{_pct(lo)}--{_pct(hi)}", f"{certas} de {total_q}")
    esperados["tabela bateria total"] = linha(
        f"{'Total':<21}", _pct(taxa(c["total"])), f"{_pct(ic_c[0])}--{_pct(ic_c[1])}", f"{c['sempre_certa']} de 71")
    esperados["ic E (texto)"] = f"intervalo de {_pct(ic_e[0])}\\% a {_pct(ic_e[1])}\\%, contra {_pct(ic_c[0])}\\% a {_pct(ic_c[1])}\\% sem ela"
    for forma, rot in (("negativa", "Negativa      "), ("existencia", "Existência    "), ("confirmacao", "Confirmação   "),
                       ("aberta", "Aberta        "), ("premissa-falsa", "Premissa falsa")):
        k, n = c["forma"][forma]
        esperados[f"tabela forma {forma}"] = linha(f"{rot:<14}", f"{k}/{n}", _pct(k / n))
    esperados["redacao, gabarito original"] = (
        f"eram {antigo[0]} pares a favor da literal e {antigo[1]} a favor da paráfrase, diferença que pode ser acaso"
        if antigo[2] >= 0.05 else "<redacao com o gabarito original passou a ser significativa>")
    esperados["gpt-oss sem falhas de API"] = gpt_oss_sem_falhas()
    esperados.update(expansao_por_variante())
    esperados.update(acerto_por_pergunta_ensino())
    esperados.update(origem_c1())
    esperados.update(legislacao())
    esperados.update(busca_da_bateria())
    esperados.update(enriquecimento_repeticoes())
    esperados.update(sim_constante())
    esperados.update(ancoras_auditoria(c, e, ef, g))
    esperados.update(acerto_com_gabarito_original())
    esperados.update(custo_invertido())
    esperados.update(existencia_acerto())
    esperados.update(mediana_tokens())
    esperados.update(continuacao_e_latencia())

    falhas = 0
    # A razao com as negativas (26,5) so aparece com a ressalva da Secao 8.5.
    for velho in ("1/26", "26 vezes", "26 times"):
        if velho in texto:
            print(f"FALHA  tokens: '{velho}' voltou ao artigo; o numero principal e o das perguntas com resposta",
                  file=sys.stderr)
            falhas += 1
    for descricao, trecho in esperados.items():
        alvo, onde = (apoio, "apoio-resultados.tex") if descricao in NO_APOIO else (texto, "artigo")
        if trecho not in alvo:
            print(f"FALHA  {descricao}: o CSV da '{trecho}', que nao aparece no {onde}", file=sys.stderr)
            falhas += 1
    return falhas


# Ancoras cujas tabelas ou frases sairam do artigo e estao, palavra por
# palavra, em docs/article/tables/apoio-resultados.tex. Continuam conferidas
# contra o CSV, mas no arquivo de apoio. (O detalhe de C1 e as correcoes de
# gabarito tambem estao nos anexos A7 e A8.)
NO_APOIO = {
    "tokens T", "tokens F", "pagina citada sem folio (8.5)", "tokens, mediana com negativas (8.5)", "acerto sem negativas (8.7)", "tokens C", "tokens E", "ic E (texto)", "expansao por variante",
    "tabela expansao pagina", "tabela expansao tokens",
    "tabela geracao prosa", "tabela geracao tabela", "tabela geracao calendario", "tabela geracao todas",
    "tabela bateria seguranca-trabalho", "tabela bateria produto", "tabela bateria licitacoes", "tabela bateria total",
    "tabela forma negativa", "tabela forma existencia", "tabela forma confirmacao", "tabela forma aberta",
    "tabela forma premissa-falsa",
    # Saíram do corpo na simplificação das métricas (Seção 8.4/8.7/8.9);
    # continuam conferidos contra o CSV, mas em apoio-resultados.tex (Anexo A7/A8).
    "proveniencia C", "proveniencia E", "ic C (8.7)", "resumo redacao, gabarito original",
    "redacao, gabarito original", "gpt-oss sem falhas de API",
    "origem C1, evolucao (apoio)", "origem C1, trechos", "origem C1, metadados",
    "sempre certa", "sempre errada",
    "ganho por pergunta (apoio)", "pares concordantes (apoio)", "existencia, Sim. (apoio)",
}

def acerto_por_pergunta_ensino() -> dict[str, str]:
    """Acervo de ensino: perguntas certas nas tres repeticoes, T contra F
    (tabela-resumo e Conclusao). Pergunta certa e a que acertou em todas as
    linhas do CSV."""
    res = RAIZ / "docs" / "article" / "results"
    def contar(nome):
        d: dict[str, list[bool]] = {}
        for r in csv.DictReader(open(res / nome, encoding="utf-8")):
            d.setdefault(r["question_id"], []).append(r["correct"] == "true")
        return sum(all(v) for v in d.values()), len(d)
    kt, nt = contar("generation-results-T-ordem.csv")
    kf, nf = contar("generation-results-F-document.csv")
    return {
        "ensino por pergunta (resumo)": f"com {kt} de {nt} perguntas certas contra {kf} de {nf}",
        "ensino por pergunta (conclusao)": (f"({kt} de {nt} perguntas contra {kf} de {nf}, com configurações "
                                            "diferentes entre os dois braços; Seção~\\ref{sec:res-geracao})"),
    }


def ancoras_auditoria(c: dict, e: dict, ef: dict, g: dict) -> dict[str, str]:
    """Numeros que mudaram na auditoria de falsos positivos (registro 8 de
    alteracoes-pos-medicao.md) e que ate entao nao tinham ancora: o ganho da
    expansao por pergunta com o intervalo de Wald da diferenca pareada, os
    pares concordantes, a forma existencia, o acerto por resposta de T no
    acervo de ensino e o resumo em ingles."""
    n = len(c["por_q"])
    b = sum(e["por_q"][q] and not c["por_q"][q] for q in c["por_q"])
    k = sum(c["por_q"][q] and not e["por_q"][q] for q in c["por_q"])
    d = (b - k) / n
    se = math.sqrt((b + k) / n - d * d) / math.sqrt(n)
    ex_k, ex_n = c["forma"]["existencia"]
    t = g["T"]["todas"]
    return {
        "ganho por pergunta (apoio)": (
            f"o ganho foi de {_pct(d)} pontos percentuais (de {c['sempre_certa']} para {e['sempre_certa']} das "
            f"{n} perguntas certas nas três repetições), com intervalo de 95\\% de {_pct(d - 1.96 * se)} a "
            f"{_pct(d + 1.96 * se)} pontos"),
        "pares concordantes (apoio)": f"{ef['concordantes']} dos 31 pares tiveram o mesmo resultado",
        "existencia, Sim. (apoio)": f"de modo que o {_pct(ex_k / ex_n)}\\% dessa forma",
        "ensino por resposta (limitacoes)": f"O acervo de desenvolvimento (ensino) chegou a {_pct(t['ok'] / t['n'])}\\%",
        "abstract": f"accuracy was {_pct(c['total'][0] / c['total'][1]).replace(',', '.')}\\%, and",
    }


# Contagens que o texto escreve por extenso no inicio de frase.
EXTENSO = {19: "Dezenove"}
# Contagens que o texto escreve por extenso no meio da frase.
EXTENSO_MIN = {4: "quatro"}


def origem_c1() -> dict[str, str]:
    """Origem completa em C1, lida da tabela que o CI regera a partir do Go
    (provenance.tex). As contagens intermediarias (Secao 8.4 e apoio) saem das linhas
    por tipo: a primeira medicao nao tinha pagina nos trechos nem nos
    metadados; a correcao do esquema resolveu os trechos."""
    tab = PROVENIENCIA.read_text(encoding="utf-8")
    m = re.search(r"\\textbf\{(\d+)\} & \\textbf\{(\d+)\}", tab)
    total, completos = int(m.group(1)), int(m.group(2))
    tipo = {t: int(n) for t, n in re.findall(r"^(Trecho|Metadado) & (\d+) &", tab, re.M)}
    primeira = total - tipo["Trecho"] - tipo["Metadado"]
    return {
        # A tabela-resumo so da o total; as contagens intermediarias ficam na
        # Secao 8.4, conferidas nas ancoras abaixo.
        # A correcao do sistema (pagina nos trechos) levou de 51 a 70; a
        # redefinicao da metrica (metadados do documento inteiro), de 70 a 155.
        "origem C1 (resumo)": (f"Sim: o documento é exigido pelo esquema, e a página, gravada na indexação; em C1 (sem páginas), {completos} de {total}, "
                               f"{tipo['Metadado']} delas metadados"),
        "origem C1, correcao do sistema": (f"Dar coluna de página aos trechos levou a {primeira + tipo['Trecho']}. "
                                           f"As outras {tipo['Metadado']} vieram de ajustar a métrica"),
        "origem C1, versao atual": (f"Na versão atual, as {total} unidades indexadas de C1 têm a origem completa"
                                    if completos == total else "<nem todas as unidades de C1 tem origem completa>"),
        "origem C1, primeira medicao": f"A primeira medição deu {primeira} de {total}",
        # Tabela de resposta (a) e Conclusao: quantas das unidades completas
        # sao metadados do documento inteiro, sem pagina.
        "origem C1, metadados (tabela a)": (f"Em C1, as {total} unidades têm origem completa no banco; "
                                            f"{tipo['Metadado']} delas são metadados do documento inteiro, sem página"),
        # Conclusao: so o total; o detalhe dos metadados fica na tabela (a).
        "origem C1 (conclusao)": f"as {total} unidades de C1 têm a origem gravada no banco.",
        # As contagens intermediarias sairam do corpo e estao, com a evolucao
        # 51 -> 70 -> 155, em apoio-resultados.tex (NO_APOIO).
        "origem C1, trechos": f"{EXTENSO.get(tipo['Trecho'], tipo['Trecho'])} unidades sem página eram um defeito de modelagem",
        "origem C1, metadados": f"As outras {tipo['Metadado']} não eram defeito",
        "origem C1, evolucao (apoio)": (f"Unidades completas & {primeira} & {primeira + tipo['Trecho']} & {total} \\\\"),
    }


def legislacao() -> dict[str, str]:
    """Acervo de legislacao (Secao 8.6): acerto total e nas divergentes."""
    rows = list(csv.DictReader(open(RAIZ / "docs" / "article" / "results" / "generation-results-T-licitacoes.csv",
                                    encoding="utf-8")))
    div = [r for r in rows if r["stratum"] == "divergente"]
    k = sum(r["correct"] == "true" for r in rows)
    kd = sum(r["correct"] == "true" for r in div)
    por_id = {r["question_id"]: r for r in rows}
    g = json.load(open(RAIZ / "api" / "internal" / "evalcorpus" / "testdata" / "gabarito-legislacao.json", encoding="utf-8"))
    gab = {q["id"]: q for q in g.get("questions", g)}
    # p05: resposta certa ("ate 10 anos", art. 108) marcada errada porque o
    # padrao exige "10 (dez) anos" ou "dez anos"; nao foi corrigida.
    p05, g05 = por_id["p05"], gab["p05"]
    p05_ok = (p05["correct"] == "false" and g05["answer_patterns"] == [r"10 \(dez\) anos|dez anos"]
              and "até 10 anos" in p05["answer"] and "Art. 108" in p05["answer"]
              and not re.search(g05["answer_patterns"][0], p05["answer"].lower()))
    # p01: errada, citou a pagina da Lei 8.666 com o art. 24 (pagina 18 do
    # PDF) e a pagina-alvo da Lei 14.133 nao esta entre as fontes.
    p01, g01 = por_id["p01"], gab["p01"]
    alvo01 = {f"{t['document_id']}:{pg}" for t in g01["targets"] for pg in t["pages"]}
    fontes01 = set(p01["cited_pages"].split(";"))
    p01_ok = p01["correct"] == "false" and not (alvo01 & fontes01) and "lei-8666:18" in fontes01 and "Lei 8.666" in p01["answer"]
    # d03: recusa ("nao ha informacoes"), sem a pagina-alvo entre as fontes.
    d03 = por_id["d03"]
    d03_ok = (d03["correct"] == "false" and d03["retrieval_hit"] == "false"
              and d03["answer"].lower().startswith("com base nos documentos fornecidos, não há informações"))
    erradas = len(rows) - k
    return {
        "legislacao d03 (8.6)": ("A quarta, sobre quem conduz a licitação, foi uma recusa: a página-alvo não chegou às "
                                 "fontes, e o sistema respondeu que a informação não constava nos documentos."
                                 if d03_ok and erradas == 4 else "<d03 da legislacao mudou>"),
        "legislacao": f"acertou {k} das {len(rows)} perguntas; nas {len(div)} divergentes, acertou {kd}",
        "legislacao p05 (8.6)": (f"Das {EXTENSO_MIN.get(erradas, erradas)} respostas contadas como erradas, uma estava correta (``até 10 anos'', art.~108 da Lei 14.133), "
                                 f"mas o gabarito exigia ``10 (dez) anos'' ou ``dez anos''; o número não foi corrigido, e o "
                                 f"acerto oficial continua {k} de {len(rows)}"
                                 if p05_ok else "<p05 da legislacao mudou>"),
        "legislacao p01 (8.6)": ("(a resposta citou a lei revogada, na página certa dela; a página da lei vigente não chegou às fontes)"
                                 if p01_ok else "<p01 da legislacao mudou>"),
    }


def busca_da_bateria() -> dict[str, str]:
    """Pagina certa entre as fontes na bateria (retrieval_hit), sem e com
    expansao, e as respostas certas sem a pagina certa entre as fontes
    (Secoes 8.7 e 8.9 e Tabela de resposta). As negativas nao tem
    pagina-alvo (retrieval_hit vazio) e ficam fora do denominador."""
    res = RAIZ / "docs" / "article" / "results" / "expansao"
    def linhas(arm):
        return [r for t in ("seguranca-trabalho", "produto", "licitacoes")
                for r in csv.DictReader(open(res / f"{arm}-{t}.csv", encoding="utf-8"))]
    off, on = linhas("off"), linhas("on")
    def hit(rs):
        alvo = [r for r in rs if r["retrieval_hit"] != ""]
        return sum(r["retrieval_hit"] == "true" for r in alvo), len(alvo)
    (ko, no), (ke, ne) = hit(off), hit(on)
    # So as certas com pagina-alvo: as negativas certas nao tem pagina a achar.
    certas = [r for r in off if r["correct"] == "true" and r["retrieval_hit"] != ""]
    sem_fonte = sum(r["retrieval_hit"] == "false" for r in certas)
    return {
        "busca da bateria (8.7)": (f"a página certa esteve entre as fontes em {ko} de {no} respostas com página-alvo ({_pct(ko / no)}\\%, "
                                   "com a métrica corrigida depois da medição; Anexo~A8)"),
        "busca com expansao (8.9)": (f"a página certa passou a estar entre as fontes em {ke} de {ne} respostas com "
                                     f"página-alvo ({_pct(ke / ne)}\\%), contra {ko} ({_pct(ko / no)}\\%) sem ela"),
        "busca da bateria (tabela b)": (f"Na bateria, a página certa esteve entre as fontes em {_pct(ko / no)}\\% das respostas com página-alvo"),
        "busca com expansao (tabela b)": f"A expansão por LLM leva a página certa a {_pct(ke / ne)}\\%",
        "certas sem a pagina (8.7)": f"Em {sem_fonte} das {len(certas)} respostas certas com página-alvo, a página certa não estava entre as fontes",
        "certas e conferiveis (8.7)": f"foram {len(certas) - sem_fonte} de {no} ({_pct((len(certas) - sem_fonte) / no)}\\%)",
        "certas sem a pagina (tabela c)": f"em {sem_fonte} das {len(certas)} respostas certas da bateria com página-alvo a página certa não estava entre as fontes",
    }


def sim_constante() -> dict[str, str]:
    """A resposta fixa "Sim." contra as perguntas de existencia da bateria
    (Secao 8.7; registro 6 de alteracoes-pos-medicao.md): conta as que ela
    acertaria com o gabarito atual."""
    gab = RAIZ / "api" / "internal" / "evalcorpus" / "testdata"
    total = acerta = 0
    for tema in ("seguranca-trabalho", "produto", "licitacoes"):
        g = json.load(open(gab / f"gabarito-{tema}.json", encoding="utf-8"))
        for q in g.get("questions", g):
            if q.get("stratum") != "existencia":
                continue
            total += 1
            # Mesma dobra do corretor (fold em api/cmd/generation-eval/metrics.go).
            ok = all(re.search(p, "sim.") for p in q.get("answer_patterns", []))
            ok = ok and not any(re.search(p, "sim.") for p in q.get("reject_patterns") or [])
            acerta += ok
    if acerta != total:
        return {"Sim. constante": f"<Sim. acerta {acerta} de {total} perguntas de existencia>"}
    return {"Sim. constante": f"Nas {total} perguntas de existência, a resposta certa é sempre ``sim''"}


def enriquecimento_repeticoes() -> dict[str, str]:
    """O enriquecimento (Secao 8.8) foi executado uma vez por braco."""
    d = RAIZ / "docs" / "article" / "results" / "enriquecimento-reexecucao"
    reps = {r["repeat"] for b in ("content", "content-summary", "content-questions", "max")
            for r in csv.DictReader(open(d / f"braco-seguranca-trabalho-{b}.csv", encoding="utf-8"))}
    return {"enriquecimento, uma repeticao": "e 2 negativas) e uma repetição," if reps == {"1"} else f"<enriquecimento com repeticoes {sorted(reps)}>"}


def queda_literal_parafrase(rows: list[dict], gran: str, arm: str) -> float | None:
    """P@1(parafrase) - P@1(literal): a dependencia de casamento lexical."""
    por_variante = collections.defaultdict(list)
    for r in rows:
        if r["negative"] == "false" and r["granularity"] == gran and r["arm"] == arm:
            por_variante[r["variant"]].append(float(r["p_at_1"]))

    if not por_variante.get("literal") or not por_variante.get("paraphrase"):
        return None

    lit = sum(por_variante["literal"]) / len(por_variante["literal"])
    par = sum(por_variante["paraphrase"]) / len(por_variante["paraphrase"])
    return par - lit


def acertos_variante(rows: list[dict], gran: str, arm: str, variante: str) -> tuple[int, int]:
    """Consultas com o texto certo em primeiro lugar (P@1 = 1), e o total."""
    vals = [float(r["p_at_1"]) for r in rows
            if r["negative"] == "false" and r["granularity"] == gran and r["arm"] == arm
            and r["variant"] == variante]
    return round(sum(vals)), len(vals)


def conferir_variantes(rows: list[dict], texto: str, apoio: str) -> int:
    """Literais e parafrases de C1 como o texto as afirma, em contagem e em
    porcentagem (Secao 8.1). Substitui a ancora '85,7': a queda da contagem por
    secao (-0,857, conferida em main) aparece no texto como 5 de 5 literais e
    1 de 7 parafrases.

    As quatro frases por tipo de consulta sairam do artigo e estao, palavra
    por palavra, em apoio-resultados.tex (e no Anexo A7); sao conferidas la.
    O artigo guarda o resumo (literais nos dois niveis, literais da densa) e
    a tabela da Secao 8.1 em contagem, conferidos contra o texto."""
    def kn(gran, arm, var):
        return acertos_variante(rows, gran, arm, var)
    lit_a, par_a = kn("document", "A-relacional", "literal"), kn("document", "A-relacional", "paraphrase")
    lit_b, par_b = kn("document", "B-bm25", "literal"), kn("document", "B-bm25", "paraphrase")
    lit_c, par_c = kn("document", "C-denso", "literal"), kn("document", "C-denso", "paraphrase")
    slit_a, spar_a = kn("section", "A-relacional", "literal"), kn("section", "A-relacional", "paraphrase")
    slit_b, spar_b = kn("section", "B-bm25", "literal"), kn("section", "B-bm25", "paraphrase")
    noi_b, noi_c = kn("document", "B-bm25", "noisy"), kn("document", "C-denso", "noisy")

    def erradas(arm):
        return {r["query_id"] for r in rows if r["negative"] == "false" and r["granularity"] == "document"
                and r["arm"] == arm and float(r["p_at_1"]) < 1}
    comum = len(erradas("B-bm25") & erradas("C-denso"))
    # O texto diz "as 5 literais" do BM25 e "as 4 ruidosas" da densa: so vale
    # se acertaram todas.
    por_tipo = (f"o BM25 acertou as {lit_b[1]} literais, {par_b[0]} das {par_b[1]} paráfrases e {noi_b[0]} das "
                f"{noi_b[1]} ruidosas; a densa, {lit_c[0]} das {lit_c[1]} literais, {par_c[0]} das {par_c[1]} "
                f"paráfrases e as {noi_c[1]} ruidosas, e os dois erraram juntos em só {comum} consulta"
                if lit_b[0] == lit_b[1] and noi_c[0] == noi_c[1] and comum == 1 else "<C1 por tipo de consulta mudou>")
    # "nao e estatisticamente significativa" entre as secoes so vale enquanto o P@1 contagem x BM25
    # nao for significativo (tabela gerada por analyze.py, Anexo A7).
    sig = (RAIZ / "docs" / "article" / "tables" / "retrieval-significance-section.tex").read_text(encoding="utf-8")
    m = re.search(r"A, contagem vs B, BM25 & P@1 & [^&]+& \d+ & 0\{,\}(\d+) &", sig)
    acaso = m is not None and float("0." + m.group(1)) >= 0.05
    falhas = 0
    if not (lit_a[0] == lit_a[1] == lit_b[0] == lit_b[1] == slit_a[0] == slit_a[1] == slit_b[0]):
        print("FALHA  C1: o texto diz que as buscas por palavras acertam todas as literais", file=sys.stderr)
        falhas += 1
    def n(gran, arm):
        return round(media(rows, gran, arm, "p_at_1") * 16)

    no_texto = {
        "C1 tabela contagem": f"Contagem de palavras (versão anterior) & {n('document', 'A-relacional')} de 16 & {n('section', 'A-relacional')} de 16 \\\\",
        "C1 tabela BM25": f"BM25 (usado no sistema) & {n('document', 'B-bm25')} de 16 & {n('section', 'B-bm25')} de 16 \\\\",
        "C1 tabela densa": f"Busca densa & {n('document', 'C-denso')} de 16 & -- \\\\",
        "C1 tabela hibrida": f"BM25 + busca densa (híbrida) & {n('document', 'E-rrf-denso')} de 16 & -- \\\\",
        "C1 densa aproximada": f"tantas vezes quanto a exata ({n('document', 'C-denso-ivf')} de 16)",
        "C1 resumo": f"Sem diferença detectável: {n('document', 'B-bm25')} de 16 contra {n('document', 'C-denso')} de 16",
        "C1 conclusao": f"({n('document', 'B-bm25')} de 16 consultas contra {n('document', 'C-denso')} de 16)",
        "C1 BM25 texto": f"o BM25 acertou o primeiro resultado em {n('document', 'B-bm25')} das 16 consultas",
        "C1 secoes texto": (f"o BM25 acertou {n('section', 'B-bm25')} e a contagem de palavras, {n('section', 'A-relacional')}; "
                            "essa diferença também não é estatisticamente significativa" if acaso else "<contagem x BM25 em secoes ficou significativo>"),
        "C1 por tipo (8.1)": por_tipo,
        # Discussao: a hibrida nao acertou mais que o BM25 em C1.
        "C1 hibrida (discussao)": (f"a combinação dos dois acertou tantas quanto o BM25 ({n('document', 'B-bm25')} de 16)"
                                   if n("document", "E-rrf-denso") == n("document", "B-bm25") else "<hibrida diferente do BM25 em C1>"),
    }
    no_apoio = {
        "C1 literais, dois niveis": f"acertaram as {lit_a[1]} consultas literais nos dois níveis",
        "C1 denso, literais erradas": f"a densa errou {lit_c[1] - lit_c[0]} das {lit_c[1]} literais",
        "C1 literais, palavras": f"As duas buscas por palavras acertam as {lit_a[1]} consultas literais",
        "C1 parafrases, documento": (f"nas {par_a[1]} paráfrases, a contagem acerta {par_a[0]} "
                                     f"({_pct(par_a[0] / par_a[1])}\\%) e o BM25, {par_b[0]} "
                                     f"({_pct(par_b[0] / par_b[1])}\\%)"),
        "C1 denso": (f"A busca densa acerta {par_c[0]} das {par_c[1]} paráfrases, mas só "
                     f"{lit_c[0]} das {lit_c[1]} literais ({_pct(lit_c[0] / lit_c[1])}\\%)"),
        "C1 secao": (f"a contagem continua acertando as {slit_a[1]} literais, mas só {spar_a[0]} das "
                     f"{spar_a[1]} paráfrases ({_pct(spar_a[0] / spar_a[1])}\\%), e o BM25 acerta "
                     f"{spar_b[0]} das {spar_b[1]}"),
    }
    for alvo, onde, esperados in ((texto, "artigo", no_texto), (apoio, "apoio-resultados.tex", no_apoio)):
        for descricao, trecho in esperados.items():
            if trecho not in alvo:
                print(f"FALHA  {descricao}: o CSV da '{trecho}', que nao aparece no {onde}", file=sys.stderr)
                falhas += 1
    return falhas


def custo_invertido() -> dict[str, str]:
    """Secao 8.5: perguntas do acervo de ensino em que T (trechos) enviou mais
    tokens que F (documento inteiro), na media das repeticoes, e o tamanho dos
    documentos-alvo dessas perguntas ("documentos curtos": ate 5 paginas)."""
    res = RAIZ / "docs" / "article" / "results"

    def tok(arq):
        d: dict[str, list[int]] = {}
        for r in csv.DictReader(open(res / arq, encoding="utf-8")):
            d.setdefault(r["question_id"], []).append(int(r["prompt_tokens"] or 0))
        return {q: sum(v) / len(v) for q, v in d.items()}
    t, f = tok("generation-results-T-ordem.csv"), tok("generation-results-F-document.csv")
    g = json.load(open(RAIZ / "api" / "internal" / "evalcorpus" / "testdata" / "gabarito-si.json", encoding="utf-8"))
    paginas = {d["id"]: d["pages"] for d in g["documents"]}
    alvo = {q["id"]: {x["document_id"] for x in q.get("targets") or []} for q in g["questions"]}
    inv = [q for q in t if t[q] > f[q]]
    curtos = all(alvo[q] and max(paginas[d] for d in alvo[q]) <= 5 for q in inv)
    return {"custo invertido (8.5)": (f"Em {len(inv)} das {len(t)} perguntas, sobre documentos curtos, a busca enviou "
                                      "mais \\emph{tokens} que o documento inteiro"
                                      if curtos else "<custo invertido fora de documentos curtos>")}


def mediana_tokens() -> dict[str, str]:
    """Secao 8.5, Resumo, Abstract, tabela-resumo e Conclusao: razao F/T de
    tokens por pergunta (media das repeticoes), na mediana. O numero principal
    e o das perguntas com resposta; com as 5 negativas, em que F recebe cinco
    documentos, a mediana sobe."""
    res = RAIZ / "docs" / "article" / "results"

    def tok(arq):
        d: dict[str, list[int]] = {}
        est: dict[str, str] = {}
        for r in csv.DictReader(open(res / arq, encoding="utf-8")):
            d.setdefault(r["question_id"], []).append(int(r["prompt_tokens"] or 0))
            est[r["question_id"]] = r["stratum"]
        return {q: sum(v) / len(v) for q, v in d.items()}, est
    (t, est), (f, _) = tok("generation-results-T-ordem.csv"), tok("generation-results-F-document.csv")
    razao = {q: f[q] / t[q] for q in t}
    com = statistics.median(v for q, v in razao.items() if est[q] != "negativa")
    todas = statistics.median(razao.values())
    n_com = sum(est[q] != "negativa" for q in razao)
    n_neg = len(razao) - n_com
    k = round(com)
    # Faixa da razao nas perguntas com resposta: o minimo (abaixo de 1) e o
    # maximo, que dependem do tamanho do documento inteiro.
    com_vals = [v for q, v in razao.items() if est[q] != "negativa"]
    return {
        "tokens, faixa da razao (8.5)": (f"numa razão que variou de {_br(min(com_vals), 1)} a {_br(max(com_vals), 1)} "
                                         "vezes conforme o tamanho do documento"),
        "tokens, mediana com resposta (8.5)": (f"nas {n_com} perguntas com resposta, F enviou ao modelo, na mediana, "
                                               f"{_br(com, 1)} vezes mais"),
        "tokens, mediana com negativas (8.5)": (f"contando as {n_neg} negativas, em que o documento inteiro inclui cinco "
                                                f"PDFs, a mediana sobe para {_br(todas, 1)}"),
        "tokens, mediana (tabela-resumo)": (f"cerca de {k} vezes menos \\emph{{tokens}} enviados para responder, na "
                                            "mediana das perguntas com resposta"),
        "tokens, mediana (conclusao)": f"Na mediana das perguntas com resposta, buscar trechos usou cerca de {k} vezes menos",
    }


def continuacao_e_latencia() -> dict[str, str]:
    """Secao 8.5: o limite da continuacao de pagina, lido do codigo do chat,
    e a latencia mediana de T (da requisicao ao fim do stream, medida por
    cmd/generation-eval: inclui a busca e o modelo no provedor)."""
    go = (RAIZ / "api" / "internal" / "usecase" / "chat_usecase.go").read_text(encoding="utf-8")
    lim = re.search(r"^const ChatContinuationCharLimit = (\d+)$", go, re.M)
    lat = [int(r["latency_ms"]) for r in csv.DictReader(open(
        RAIZ / "docs" / "article" / "results" / "generation-results-T-ordem.csv", encoding="utf-8")) if r["latency_ms"]]
    return {
        "continuacao de pagina (8.5)": (f"leva junto até {_milhar(int(lim.group(1)))} caracteres da página seguinte"
                                        if lim else "<ChatContinuationCharLimit sumiu>"),
        "latencia T (8.5)": f"na mediana, cerca de {_br(statistics.median(lat) / 1000, 1)} s, da pergunta ao fim do texto",
    }


def existencia_acerto() -> dict[str, str]:
    """Secao 8.7: acerto do sistema (sem expansao) nas respostas de existencia."""
    res = RAIZ / "docs" / "article" / "results" / "expansao"
    rs = [r for t in ("seguranca-trabalho", "produto", "licitacoes")
          for r in csv.DictReader(open(res / f"off-{t}.csv", encoding="utf-8")) if r["stratum"] == "existencia"]
    k = sum(r["correct"] == "true" for r in rs)
    # Acerto sem as negativas (todas certas), ao lado dos 68,1%.
    todas = [r for t in ("seguranca-trabalho", "produto", "licitacoes")
             for r in csv.DictReader(open(res / f"off-{t}.csv", encoding="utf-8"))]
    neg = [r for r in todas if r["stratum"] == "negativa"]
    com = [r for r in todas if r["stratum"] != "negativa"]
    kc = sum(r["correct"] == "true" for r in com)
    sem_neg = (f"Sem as {len(neg)} respostas negativas, todas certas, o acerto com a pergunta original foi de {_pct(kc / len(com))}\\% ({kc} de {len(com)})"
               if all(r["correct"] == "true" for r in neg) else "<negativas da bateria deixaram de acertar todas>")
    return {"existencia, acerto do sistema (8.7)": f"Responder ``Sim.'' a todas acertaria as {len(rs)} respostas desse tipo; o sistema acertou {k}",
            "acerto sem negativas (8.7)": sem_neg}


def _gabarito_original() -> dict:
    spec = importlib.util.spec_from_file_location("redacao_gabarito_original",
                                                  RAIZ / "scripts" / "redacao-gabarito-original.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod.calcula()


def redacao_com_gabarito_original() -> tuple[int, int, float]:
    """Pares a favor da literal, da parafrase e p exato com o gabarito
    original (54a4d70^), recorrigindo as respostas gravadas hoje nos CSV da
    expansao (scripts/redacao-gabarito-original.py). Nao usa a coluna
    `correct` de CSV antigo nem docs/article/results/resumo-bateria.md, que
    saiu do criterio com bug (so a ultima repeticao)."""
    red = _gabarito_original()["redacao"]
    return len(red["so_literal"]), len(red["so_parafrase"]), red["p"]


def acerto_com_gabarito_original() -> dict[str, str]:
    """O 81,7% da primeira medicao (Secao 8.7), recalculado com o gabarito
    original sobre as mesmas respostas."""
    k, n = _gabarito_original()["off"]
    return {"acerto com o gabarito original (8.7)":
            f"o acerto caiu de {_pct(k / n)}\\% para"}


def gpt_oss_sem_falhas() -> str:
    """Acerto do F com gpt-oss:120b, descontadas as falhas da API do provedor,
    contra o modelo principal nas MESMAS perguntas (banca, terceira leitura)."""
    res = RAIZ / "docs" / "article" / "results"
    rows = list(csv.DictReader(open(res / "generation-results-F-document-gpt-oss-120b.csv", encoding="utf-8")))
    def erro(r):
        return r.get("answer", "") + r.get("error", "")
    longas = {r["question_id"] for r in rows if re.search(r"prompt is too long", erro(r), re.I)}
    tempo = {r["question_id"] for r in rows if re.search(r"timeout|deadline", erro(r), re.I)} - longas
    falha = longas | tempo
    # As recusadas por tamanho sao as negativas, em que F recebe cinco documentos.
    negativas_longas = {r["question_id"] for r in rows if r["question_id"] in longas and r["stratum"] == "negativa"}
    if negativas_longas != longas:
        return "<recusas por tamanho fora das negativas>"
    validas = [r for r in rows if r["question_id"] not in falha]
    k = sum(r["correct"] == "true" for r in validas)
    ids = {r["question_id"] for r in validas}
    base = [r for r in csv.DictReader(open(res / "generation-results-F-document.csv", encoding="utf-8"))
            if r["question_id"] in ids]
    kb = sum(r["correct"] == "true" for r in base)
    # As erradas que trazem espaco estreito (U+202F) ou hifen inseparavel
    # (U+2011): FoldDiacritics (api/internal/retrieval/lexical.go), usado pelo
    # corretor, nao os normaliza, e a comparacao entre modelos nao vale.
    erradas = [r for r in validas if r["correct"] != "true"]
    especiais = sum(bool(re.search("[\u202f\u2011]", r["answer"])) for r in erradas)
    return (f"acertou {k} das {len(validas)} perguntas que respondeu ({_pct(k / len(validas))}\\%), contra "
            f"{_pct(kb / len(base))}\\% do modelo principal nas mesmas perguntas. A comparação não vale: "
            f"{especiais} das {len(erradas)} respostas marcadas como erradas contêm o espaço estreito "
            f"(U+202F) ou o hífen inseparável (U+2011), que o corretor não normaliza (só tira acentos e "
            f"caixa), e por isso esses números não mostram que o acerto dependa do modelo. Das {len(falha)} "
            f"restantes, {len(longas)} eram negativas cujo texto, com cinco documentos, passou da janela do "
            f"modelo, e {len(tempo)} esgotou o tempo.")


def expansao_por_variante() -> dict[str, str]:
    """Paraframes e literais que erravam sem expansao e quantas ela corrigiu."""
    res = RAIZ / "docs" / "article" / "results" / "expansao"
    def por_pergunta(arm):
        d: dict[str, list[bool]] = {}
        for t in ("seguranca-trabalho", "produto", "licitacoes"):
            for r in csv.DictReader(open(res / f"{arm}-{t}.csv", encoding="utf-8")):
                d.setdefault(r["question_id"], []).append(r["correct"] == "true")
        return d
    c, e = por_pergunta("off"), por_pergunta("on")
    err = [q for q, v in c.items() if not all(v)]
    par = [q for q in err if q.endswith("P")]
    lit = [q for q in err if q.endswith("L")]
    ok = lambda qs: sum(all(e[q]) for q in qs)
    return {"expansao por variante": f"corrigiu {ok(par)} de {len(par)} paráfrases e {ok(lit)} de {len(lit)} literais"}


def main() -> int:
    rows = carregar()

    # (descricao, valor afirmado no artigo, funcao que recalcula)
    afirmacoes = [
        ("nDCG@10 relacional, documento", 0.869, lambda: media(rows, "document", "A-relacional", "ndcg_at_10")),
        ("nDCG@10 denso exato, documento", 0.844, lambda: media(rows, "document", "C-denso", "ndcg_at_10")),
        ("nDCG@10 denso IVF, documento", 0.797, lambda: media(rows, "document", "C-denso-ivf", "ndcg_at_10")),
        ("P@1 relacional, documento", 0.750, lambda: media(rows, "document", "A-relacional", "p_at_1")),
        ("P@1 BM25, documento", 0.812, lambda: media(rows, "document", "B-bm25", "p_at_1")),
        ("P@1 denso exato, documento", 0.688, lambda: media(rows, "document", "C-denso", "p_at_1")),
        ("P@1 relacional, secao", 0.562, lambda: media(rows, "section", "A-relacional", "p_at_1")),
        ("P@1 BM25, secao", 0.750, lambda: media(rows, "section", "B-bm25", "p_at_1")),
        ("R@5 relacional, secao", 0.719, lambda: media(rows, "section", "A-relacional", "r_at_5")),
        ("R@5 BM25, secao", 0.812, lambda: media(rows, "section", "B-bm25", "r_at_5")),
        ("queda literal->parafrase relacional, documento", -0.429, lambda: queda_literal_parafrase(rows, "document", "A-relacional")),
        ("queda literal->parafrase relacional, secao", -0.857, lambda: queda_literal_parafrase(rows, "section", "A-relacional")),
        ("queda literal->parafrase BM25, documento", -0.286, lambda: queda_literal_parafrase(rows, "document", "B-bm25")),
    ]

    falhas = 0
    for descricao, no_artigo, calcular in afirmacoes:
        medido = calcular()
        if medido is None:
            print(f"FALHA  {descricao}: nao ha dados para recalcular", file=sys.stderr)
            falhas += 1
            continue
        if abs(no_artigo - medido) > TOL:
            print(
                f"FALHA  {descricao}: artigo diz {no_artigo:.3f}, medicao da {medido:.3f}",
                file=sys.stderr,
            )
            falhas += 1

    # R@5 saturado em documento e a premissa da Secao de Granularidade: se
    # deixar de ser 1,000, o argumento do artigo muda.
    for arm in ("A-relacional", "B-bm25", "D-rrf"):
        valor = media(rows, "document", arm, "r_at_5")
        if valor is not None and abs(valor - 1.0) > TOL:
            print(
                f"FALHA  R@5 de {arm} em documento deveria ser 1,000 (saturado); veio {valor:.3f}"
                " -- a Secao de Granularidade argumenta a partir dessa saturacao",
                file=sys.stderr,
            )
            falhas += 1

    # A queda precisa ser MAIOR em secao: e a conclusao central da Secao de
    # Granularidade.
    d = queda_literal_parafrase(rows, "document", "A-relacional")
    s = queda_literal_parafrase(rows, "section", "A-relacional")
    if d is not None and s is not None and s >= d:
        print(
            f"FALHA  a queda em secao ({s:+.3f}) deveria ser MAIOR que em documento ({d:+.3f})"
            " -- se a diferenca sumiu, a conclusao sobre dependencia lexical precisa ser revista",
            file=sys.stderr,
        )
        falhas += 1

    # As porcentagens de P@1 (81,2; 68,8; 75,0; 56,2) sairam do corpo na
    # simplificacao da Secao 8: o texto e a Tabela da Secao 8.1 dao as mesmas
    # medidas em contagem (k de 16), conferidas em conferir_variantes. Os
    # valores acima continuam conferidos contra o CSV porque estao no Anexo A7.
    texto = ARTIGO.read_text(encoding="utf-8")

    # As quedas literal->parafrase sao ditas no texto como acertos por tipo de
    # consulta (k de n), recalculados aqui do CSV.
    apoio = APOIO.read_text(encoding="utf-8")
    falhas += conferir_variantes(rows, texto, apoio)

    falhas += conferir_geracao(texto, apoio)

    print(f"afirmacoes conferidas: {len(afirmacoes)} de recuperacao, mais as de geracao")

    if falhas:
        print(f"\n{falhas} divergencia(s) entre o artigo e a medicao.", file=sys.stderr)
        return 1

    print("\nOK: todo numero afirmado no artigo bate com o CSV de resultados.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
