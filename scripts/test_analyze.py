#!/usr/bin/env python3
"""Testes de scripts/analyze.py.

Uma funcao estatistica errada nao quebra nada: devolve um numero plausivel e
invalida a conclusao do artigo em silencio. Por isso os valores esperados aqui
sao calculados A MAO ou tirados de exemplos publicados, nunca da saida da
propria implementacao.

Rodar:  python3 scripts/test_analyze.py
"""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from analyze import (  # noqa: E402
    bootstrap_ci,
    build_negative_table,
    build_significance_table,
    build_summary_table,
    build_variant_table,
    ceiling_key,
    cliffs_delta,
    filter_granularity,
    mean,
    wilcoxon_signed_rank,
)


class TestCliffsDelta(unittest.TestCase):
    def test_separacao_completa_da_delta_um(self):
        # Todo valor de `a` supera todo valor de `b`: delta = +1 por definicao.
        delta, magnitude = cliffs_delta([5, 6, 7], [1, 2, 3])
        self.assertAlmostEqual(delta, 1.0)
        self.assertEqual(magnitude, "grande")

    def test_separacao_completa_invertida_da_menos_um(self):
        delta, _ = cliffs_delta([1, 2, 3], [5, 6, 7])
        self.assertAlmostEqual(delta, -1.0)

    def test_amostras_identicas_dao_zero(self):
        delta, magnitude = cliffs_delta([1, 2, 3], [1, 2, 3])
        self.assertAlmostEqual(delta, 0.0)
        self.assertEqual(magnitude, "desprezivel")

    def test_valor_conferido_a_mao(self):
        # a=[1,3], b=[2,2]: pares (1,2)(1,2)(3,2)(3,2)
        #   maiores = 2, menores = 2  ->  (2-2)/4 = 0
        delta, _ = cliffs_delta([1, 3], [2, 2])
        self.assertAlmostEqual(delta, 0.0)

        # a=[3,4], b=[1,5]: (3>1)(3<5)(4>1)(4<5) -> maiores=2, menores=2 -> 0
        delta, _ = cliffs_delta([3, 4], [1, 5])
        self.assertAlmostEqual(delta, 0.0)

        # a=[3,4], b=[1,2]: todos maiores -> +1
        delta, _ = cliffs_delta([3, 4], [1, 2])
        self.assertAlmostEqual(delta, 1.0)

    def test_limiares_seguem_romano(self):
        # Fronteiras publicadas: <0.147 desprezivel, <0.33 pequeno,
        # <0.474 medio, senao grande.
        casos = [
            ([1] * 10, [1] * 10, "desprezivel"),
            ([2] * 8 + [0] * 12, [1] * 20, "pequeno"),
        ]
        for a, b, esperado in casos:
            _, magnitude = cliffs_delta(a, b)
            self.assertEqual(magnitude, esperado, f"a={a[:3]}... b={b[:3]}...")


class TestWilcoxon(unittest.TestCase):
    def test_amostras_identicas_nao_tem_diferenca(self):
        valores = [0.5] * 20
        _, p, n = wilcoxon_signed_rank(valores, valores)
        self.assertEqual(n, 0, "todos os pares sao empate e devem ser descartados")
        self.assertEqual(p, 1.0)

    def test_pares_empatados_sao_descartados(self):
        # 12 pares no total, mas so 10 diferem. O procedimento classico de
        # Wilcoxon descarta os empates -- manter os pares iguais diluiria o
        # sinal, o que em RI acontece direto (bracos acertam as mesmas
        # consultas faceis).
        a = [1.0] * 10 + [0.5, 0.5]
        b = [0.0] * 10 + [0.5, 0.5]
        _, _, n = wilcoxon_signed_rank(a, b)
        self.assertEqual(n, 10)

    def test_diferenca_consistente_e_significativa(self):
        # `a` supera `b` em TODOS os 20 pares: o menor W possivel, portanto o
        # p-valor tem que ser pequeno.
        a = [x / 20 for x in range(1, 21)]
        b = [x - 0.05 for x in a]
        _, p, n = wilcoxon_signed_rank(a, b)
        self.assertEqual(n, 20)
        self.assertLess(p, 0.001, "diferenca sistematica em 20 pares deveria ser altamente significativa")

    def test_amostra_pequena_nao_finge_precisao(self):
        # Abaixo de n=10 a aproximacao normal nao vale. Devolver p=1.0 e
        # honesto; devolver um numero qualquer daria falsa confianca.
        a = [1.0, 1.0, 1.0]
        b = [0.0, 0.0, 0.0]
        _, p, n = wilcoxon_signed_rank(a, b)
        self.assertEqual(p, 1.0)
        self.assertEqual(n, 3)

    def test_p_valor_esta_no_intervalo_valido(self):
        a = [0.9, 0.8, 0.7, 0.95, 0.6, 0.85, 0.75, 0.65, 0.55, 0.5, 0.45, 0.4]
        b = [0.1, 0.2, 0.3, 0.05, 0.4, 0.15, 0.25, 0.35, 0.45, 0.5, 0.55, 0.6]
        _, p, _ = wilcoxon_signed_rank(a, b)
        self.assertGreaterEqual(p, 0.0)
        self.assertLessEqual(p, 1.0)


class TestBootstrap(unittest.TestCase):
    def test_valores_constantes_dao_intervalo_degenerado(self):
        lo, hi = bootstrap_ci([0.7] * 30)
        self.assertAlmostEqual(lo, 0.7)
        self.assertAlmostEqual(hi, 0.7)

    def test_intervalo_contem_a_media(self):
        valores = [0.1, 0.9, 0.5, 0.4, 0.7, 0.3, 0.8, 0.2, 0.6, 0.55]
        lo, hi = bootstrap_ci(valores)
        media = mean(valores)
        self.assertLessEqual(lo, media)
        self.assertGreaterEqual(hi, media)

    def test_e_reproduzivel(self):
        # Semente fixa: sem isso as tabelas do artigo mudariam de valor a cada
        # execucao do script.
        valores = [0.1, 0.9, 0.5, 0.4, 0.7, 0.3, 0.8, 0.2]
        self.assertEqual(bootstrap_ci(valores), bootstrap_ci(valores))


def _sample_rows() -> list[dict]:
    rows = []
    for i in range(12):
        for arm, base in (("A-relacional", 0.6), ("B-bm25", 0.9)):
            rows.append(
                {
                    "query_id": f"q{i:02d}",
                    "arm": arm,
                    "variant": ["literal", "paraphrase", "noisy"][i % 3],
                    "negative": "false",
                    "p_at_1": f"{base:.6f}",
                    "p_at_3": f"{base / 3:.6f}",
                    "r_at_5": "1.000000",
                    "reciprocal_rank": f"{base:.6f}",
                    "ndcg_at_10": f"{base:.6f}",
                    "false_positive": "0.000000",
                    "result_count": "3",
                    "latency_ms": "0.100000",
                }
            )
    for arm in ("A-relacional", "B-bm25"):
        rows.append(
            {
                "query_id": "q-neg",
                "arm": arm,
                "variant": "literal",
                "negative": "true",
                "p_at_1": "0.000000",
                "p_at_3": "0.000000",
                "r_at_5": "0.000000",
                "reciprocal_rank": "0.000000",
                "ndcg_at_10": "0.000000",
                "false_positive": "1.000000",
                "result_count": "2",
                "latency_ms": "0.100000",
            }
        )
    return rows


class TestTabelas(unittest.TestCase):
    def setUp(self):
        self.rows = _sample_rows()

    def test_tabelas_sao_latex_balanceado(self):
        for nome, tabela in (
            ("summary", build_summary_table(self.rows)),
            ("variants", build_variant_table(self.rows)),
            ("significance", build_significance_table(self.rows)),
            ("negatives", build_negative_table(self.rows)),
        ):
            self.assertIn(r"\begin{table}", tabela, nome)
            self.assertIn(r"\end{table}", tabela, nome)
            self.assertEqual(tabela.count(r"\begin{tabular}"), tabela.count(r"\end{tabular}"), nome)
            self.assertIn("GERADO POR", tabela, f"{nome}: falta o aviso de arquivo gerado")

    def test_casos_negativos_nao_entram_nas_metricas_de_ranking(self):
        # Um caso negativo tem P@1 = 0 por construcao. Se ele entrasse na
        # media, puniria o braco por acertar -- e a tabela reportaria numero
        # menor que o real.
        tabela = build_summary_table(self.rows)
        self.assertIn("0.600", tabela, "a media de A deveria ser 0.600, ignorando o caso negativo")
        self.assertIn("0.900", tabela, "a media de B deveria ser 0.900, ignorando o caso negativo")

    def test_tabela_de_negativos_reporta_falso_positivo(self):
        tabela = build_negative_table(self.rows)
        self.assertIn("1.000", tabela)
        self.assertIn("retrieval-negatives", tabela)

    def test_bonferroni_multiplica_pelo_numero_de_comparacoes(self):
        # 2 bracos = 1 par; 5 metricas => 5 comparacoes.
        tabela = build_significance_table(self.rows)
        self.assertIn("5 comparações", tabela)

    def test_sem_casos_negativos_a_tabela_e_omitida_com_explicacao(self):
        positivos = [r for r in self.rows if r["negative"] == "false"]
        tabela = build_negative_table(positivos)
        self.assertIn("omitida", tabela)


class TestExecucaoFimAFim(unittest.TestCase):
    def test_script_roda_e_escreve_os_quatro_arquivos(self):
        script = Path(__file__).resolve().parent / "analyze.py"
        with tempfile.TemporaryDirectory() as tmp:
            tmpdir = Path(tmp)
            csv_path = tmpdir / "results.csv"

            rows = _sample_rows()
            header = list(rows[0].keys())
            linhas = [",".join(header)]
            linhas += [",".join(str(row[col]) for col in header) for row in rows]
            csv_path.write_text("\n".join(linhas) + "\n", encoding="utf-8")

            outdir = tmpdir / "tables"
            result = subprocess.run(
                [sys.executable, str(script), str(csv_path), "--outdir", str(outdir)],
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)

            esperados = {
                "retrieval-summary.tex",
                "retrieval-variants.tex",
                "retrieval-significance.tex",
                "retrieval-negatives.tex",
            }
            self.assertEqual({p.name for p in outdir.glob("*.tex")}, esperados)


class TestGranularidade(unittest.TestCase):
    """O CSV traz as duas unidades de recuperacao na mesma execucao.

    Misturar as duas produz uma media entre grandezas diferentes -- um numero
    plausivel e sem significado, que e o modo de falha mais perigoso de uma
    analise: nao levanta erro nenhum.
    """

    def _linhas_das_duas_granularidades(self):
        linhas = []
        for granularity, base in (("document", 0.8), ("section", 0.4)):
            for i in range(12):
                linhas.append(
                    {
                        "query_id": f"q{i:02d}",
                        "arm": "A-relacional",
                        "granularity": granularity,
                        "variant": ["literal", "paraphrase", "noisy"][i % 3],
                        "negative": "false",
                        "p_at_1": f"{base:.6f}",
                        "p_at_3": f"{base / 3:.6f}",
                        "r_at_5": f"{base:.6f}",
                        "reciprocal_rank": f"{base:.6f}",
                        "ndcg_at_10": f"{base:.6f}",
                        "false_positive": "0.000000",
                        "result_count": "3",
                        "latency_ms": "0.100000",
                    }
                )
        return linhas

    def test_filtra_uma_granularidade_por_vez(self):
        linhas = self._linhas_das_duas_granularidades()

        so_documento = filter_granularity(linhas, "document")
        self.assertEqual(len(so_documento), 12)
        self.assertTrue(all(r["granularity"] == "document" for r in so_documento))

        so_secao = filter_granularity(linhas, "section")
        self.assertEqual(len(so_secao), 12)
        self.assertTrue(all(r["granularity"] == "section" for r in so_secao))

    def test_media_nao_mistura_as_duas(self):
        linhas = self._linhas_das_duas_granularidades()

        tabela_doc = build_summary_table(filter_granularity(linhas, "document"))
        tabela_sec = build_summary_table(filter_granularity(linhas, "section"))

        # 0.800 e 0.400 sao os valores reais de cada granularidade. A media
        # entre elas seria 0.600 -- um numero que nao corresponde a medicao
        # nenhuma e que apareceria se o filtro nao existisse.
        self.assertIn("0.800", tabela_doc)
        self.assertNotIn("0.600", tabela_doc)
        self.assertIn("0.400", tabela_sec)
        self.assertNotIn("0.600", tabela_sec)

    def test_granularidade_inexistente_falha_em_vez_de_devolver_vazio(self):
        linhas = self._linhas_das_duas_granularidades()
        with self.assertRaises(SystemExit):
            filter_granularity(linhas, "paragrafo")

    def test_csv_sem_a_coluna_e_tratado_como_documento(self):
        # Retrocompatibilidade com resultados gerados antes da issue #16.
        antigas = [
            {
                "query_id": "q00",
                "arm": "A-relacional",
                "variant": "literal",
                "negative": "false",
                "p_at_1": "1.000000",
                "p_at_3": "0.333333",
                "r_at_5": "1.000000",
                "reciprocal_rank": "1.000000",
                "ndcg_at_10": "1.000000",
                "false_positive": "0.000000",
                "result_count": "3",
                "latency_ms": "0.100000",
            }
        ]
        resultado = filter_granularity(antigas, "document")
        self.assertEqual(len(resultado), 1)


class TestTetos(unittest.TestCase):
    """A tabela precisa declarar o maximo alcancavel de cada metrica.

    P@k divide por k, entao uma consulta com um unico item relevante limita P@3
    a 1/3 mesmo sob ranking perfeito. Publicar 0,312 sem dizer que o teto e
    0,375 leva o leitor a concluir que o sistema acerta 31%, quando o real e 83%
    do maximo possivel.
    """

    def _linhas_com_teto(self, teto_p3="0.333333"):
        return [
            {
                "query_id": f"q{i:02d}",
                "arm": "A-relacional",
                "granularity": "section",
                "variant": "literal",
                "negative": "false",
                "p_at_1": "1.000000",
                "p_at_3": "0.333333",
                "p_at_5": "0.200000",
                "r_at_5": "1.000000",
                "r_at_10": "1.000000",
                "reciprocal_rank": "1.000000",
                "ndcg_at_10": "1.000000",
                "false_positive": "0.000000",
                "result_count": "3",
                "latency_ms": "0.100000",
                "p_at_1_ceiling": "1.000000",
                "p_at_3_ceiling": teto_p3,
                "p_at_5_ceiling": "0.200000",
                "r_at_5_ceiling": "1.000000",
                "r_at_10_ceiling": "1.000000",
                "relevant_count": "1",
            }
            for i in range(12)
        ]

    def test_tabela_publica_a_linha_de_teto(self):
        tabela = build_summary_table(self._linhas_com_teto())
        self.assertIn("Teto alcançável", tabela)
        self.assertIn("0.333", tabela, "o teto de P@3 precisa aparecer na tabela")

    def test_teto_de_ndcg_nao_e_publicado(self):
        # nDCG e MRR ja sao normalizados pelo ideal do proprio gabarito: o teto
        # e 1,000 por construcao e publica-lo nao informa nada.
        self.assertIsNone(ceiling_key("ndcg_at_10"))
        self.assertIsNone(ceiling_key("reciprocal_rank"))
        self.assertEqual(ceiling_key("p_at_3"), "p_at_3_ceiling")

    def test_csv_antigo_sem_teto_nao_quebra(self):
        # Retrocompatibilidade: resultados gerados antes da issue #16 nao tem as
        # colunas de teto, e a tabela precisa continuar sendo gerada.
        linhas = self._linhas_com_teto()
        for linha in linhas:
            for chave in list(linha):
                if chave.endswith("_ceiling"):
                    del linha[chave]

        tabela = build_summary_table(linhas)
        self.assertIn(r"\begin{table}", tabela)
        self.assertNotIn("Teto alcançável", tabela)

    def test_teto_reflete_o_gabarito_e_nao_o_braco(self):
        # O teto e propriedade do gabarito: mudar o numero de relevantes muda o
        # teto, mesmo com as metricas medidas identicas.
        tabela_um = build_summary_table(self._linhas_com_teto(teto_p3="0.333333"))
        tabela_dois = build_summary_table(self._linhas_com_teto(teto_p3="0.666667"))
        self.assertIn("0.333", tabela_um)
        self.assertIn("0.667", tabela_dois)


if __name__ == "__main__":
    unittest.main(verbosity=2)
