package main

import (
	"errors"
	"testing"

	"thing-index/internal/evalcorpus"
)

// Regressao: o status "certa/errada" lia row[7] ("repeat") no lugar da coluna
// "correct" (indice 9), e toda resposta aparecia como errada no terminal.
func TestStatusImpressoLeAColunaCorrectDaLinhaGerada(t *testing.T) {
	q := evalcorpus.GabaritoQuestion{ID: "q1", AnswerPatterns: []string{"15 dias"}}

	certa := buildRow(q, "F", "document", 1, "m", answer{Text: "O prazo e de 15 dias."}, 0, 0)
	if certa[colCorrect] != "true" {
		t.Fatalf("resposta certa lida como %q na coluna correct", certa[colCorrect])
	}
	errada := buildRow(q, "F", "document", 1, "m", answer{Text: "nao sei"}, 0, 0)
	if errada[colCorrect] != "false" {
		t.Fatalf("resposta errada lida como %q na coluna correct", errada[colCorrect])
	}
	comErro := buildRow(q, "F", "document", 1, "m", answer{Err: errors.New("x")}, 0, 0)
	if comErro[colCorrect] != "false" {
		t.Fatalf("resposta com erro lida como %q na coluna correct", comErro[colCorrect])
	}
	if certa[colModel] != "m" {
		t.Fatalf("coluna model desalinhada: %q", certa[colModel])
	}
}

// buildRow e csvHeader precisam ter o mesmo numero de colunas; senao todo
// indice derivado do cabecalho aponta para o campo errado.
func TestLinhaGeradaTemAsMesmasColunasDoCabecalho(t *testing.T) {
	row := buildRow(evalcorpus.GabaritoQuestion{ID: "q"}, "F", "", 1, "m", answer{}, 0, 0)
	if len(row) != len(csvHeader) {
		t.Fatalf("linha com %d colunas, cabecalho com %d", len(row), len(csvHeader))
	}
	if colCorrect != 9 || colModel != 8 {
		t.Fatalf("csvHeader mudou de ordem: correct=%d model=%d; revise os CSVs ja publicados", colCorrect, colModel)
	}
}
