#!/usr/bin/env python3
"""Gera as fixtures PDF do E2E do workspace, sem dependencia alem da stdlib.

    python3 tests/e2e/fixtures/gerar.py

- uma-pagina.pdf: 1 pagina carta, 4 linhas em Helvetica em posicoes conhecidas.
  As caixas usadas no mock de contexto (workspace.spec.ts) sao as que
  `pdftotext -bbox-layout` mede neste arquivo -- nao invente, meca.
- trinta-paginas.pdf: 30 paginas com uma linha cada, para o teste de orcamento
  de renderizacao (workspace-orcamento.spec.ts).

Os arquivos sao versionados; este script existe para explicar de onde vieram e
para regenera-los se um dia mudarem.
"""

from pathlib import Path

PASTA = Path(__file__).resolve().parent


def pagina(linhas):
    """Objeto de conteudo de uma pagina: lista de (x, y_baseline, tamanho, texto)."""
    return "BT\n" + "".join(f"/F1 {tam} Tf 1 0 0 1 {x} {y} Tm ({txt}) Tj\n" for x, y, tam, txt in linhas) + "ET\n"


def montar(paginas):
    """PDF 1.4 minimo: catalogo, pages, uma fonte, e um par page/contents por pagina."""
    objs = ["<< /Type /Catalog /Pages 2 0 R >>", None, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"]
    kids = []
    for conteudo in paginas:
        n_page = len(objs) + 1
        n_stream = n_page + 1
        objs.append(
            f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 3 0 R >> >> /Contents {n_stream} 0 R >>"
        )
        objs.append(f"<< /Length {len(conteudo.encode('latin-1'))} >>\nstream\n{conteudo}endstream")
        kids.append(f"{n_page} 0 R")
    objs[1] = f"<< /Type /Pages /Kids [{' '.join(kids)}] /Count {len(kids)} >>"

    saida = b"%PDF-1.4\n"
    offsets = []
    for i, obj in enumerate(objs, 1):
        offsets.append(len(saida))
        saida += f"{i} 0 obj\n{obj}\nendobj\n".encode("latin-1")
    xref = len(saida)
    saida += f"xref\n0 {len(objs) + 1}\n0000000000 65535 f \n".encode()
    for off in offsets:
        saida += f"{off:010d} 00000 n \n".encode()
    saida += f"trailer\n<< /Size {len(objs) + 1} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n".encode()
    return saida


UMA_PAGINA = [
    (72, 700, 14, "1 DISPONIBILIDADE"),
    (72, 670, 11, "O fornecedor garante uptime de 99,5 por cento ao mes."),
    (72, 620, 14, "2 PENALIDADES"),
    (72, 590, 11, "Multa de 5 por cento sobre a fatura mensal por descumprimento."),
]

if __name__ == "__main__":
    (PASTA / "uma-pagina.pdf").write_bytes(montar([pagina(UMA_PAGINA)]))
    (PASTA / "trinta-paginas.pdf").write_bytes(
        montar([pagina([(72, 700, 14, f"Pagina {n} de 30")]) for n in range(1, 31)])
    )
    print("ok")
