package indexing

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"thing-index/internal/models"
)

func TestStructureLegalDocument(t *testing.T) {
	idx := NewDeterministicIndexer()

	doc := &models.Document{
		ID:       1,
		Title:    "Petição Inicial",
		MimeType: "text/plain",
	}

	content := `EXCELENTÍSSIMO(A) SENHOR(A) DOUTOR(A) JUIZ(A) DE DIREITO DA __ª VARA CÍVEL DA COMARCA DE SÃO PAULO/SP

Processo nº 1234567-89.2024.8.26.0100

AUTOR: João da Silva, brasileiro, casado, empresário, portador do RG nº 12.345.678-9 e CPF nº 123.456.789-00, residente e domiciliado na Rua das Flores, nº 123, Centro, São Paulo/SP.

RÉU: Empresa XYZ Ltda., pessoa jurídica de direito privado, inscrita no CNPJ sob o nº 12.345.678/0001-90, com sede na Av. Paulista, nº 1000, Bela Vista, São Paulo/SP.

I - DA NATUREZA DA AÇÃO

Trata-se de ação de indenização por danos materiais e morais.

II - DOS FATOS

No dia 15 de janeiro de 2024, o autor adquiriu um veículo zero quilômetro da ré.

III - DO DIREITO

A responsabilidade civil da ré é objetiva nos termos do art. 927, parágrafo único, do Código Civil.

IV - DOS PEDIDOS

Diante do exposto, requer-se:

a) A condenação da ré ao pagamento de indenização por danos materiais;

b) A condenação da ré ao pagamento de indenização por danos morais;
c) A condenação da ré em custas e honorários advocatícios.

São Paulo, 20 de fevereiro de 2024.

Advogado: Dr. José Santos
OAB/SP nº 123.456`

	result, err := idx.Index(context.Background(), doc, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n=== Documento Legal ===\n")
	fmt.Printf("Pages: %d | Paragraphs: %d | Outline: %d | Chunks: %d\n",
		len(result.DocumentPages), len(result.ParagraphNodes),
		len(result.OutlineNodes), len(result.Chunks))

	fmt.Printf("\n=== Outline ===\n")
	for _, node := range result.OutlineNodes {
		indent := strings.Repeat("  ", node.Level-1)
		fmt.Printf("%s[%d] %s\n", indent, node.Level, node.Title)
	}

	// Verificar se boilerplate foi filtrado
	hasBoilerplate := false
	for _, node := range result.OutlineNodes {
		title := strings.ToLower(node.Title)
		if strings.Contains(title, "excelentissimo") ||
			strings.Contains(title, "processo nº") ||
			strings.Contains(title, "são paulo, ") {
			hasBoilerplate = true
			fmt.Printf("⚠️  BOILERPLATE DETECTADO: %s\n", node.Title)
		}
	}
	if hasBoilerplate {
		t.Error("Boilerplate não foi filtrado corretamente")
	}

	// Verificar se field lines foram filtrados
	hasFieldLine := false
	for _, node := range result.OutlineNodes {
		title := strings.ToLower(node.Title)
		if strings.Contains(title, "advogado:") {
			hasFieldLine = true
			fmt.Printf("⚠️  FIELD LINE DETECTADO: %s\n", node.Title)
		}
	}
	if hasFieldLine {
		t.Error("Field line não foi filtrado corretamente")
	}
}

func TestStructureWithMarkdownTable(t *testing.T) {
	idx := NewDeterministicIndexer()

	doc := &models.Document{
		ID:       2,
		Title:    "Relatório Financeiro",
		MimeType: "text/plain",
	}

	content := `RELATÓRIO FINANCEIRO - JANEIRO 2024

1. RESUMO EXECUTIVO

Este relatório apresenta os resultados financeiros do mês de janeiro.

2. RECEITAS

As receitas totais foram de R$ 500.000,00.

2.1 Receitas por Segmento

Abaixo a distribuição por segmento:

Segmento      | Valor      | Percentual
--------------|------------|------------
Varejo        | R$ 300.000 | 60%
Atacado       | R$ 150.000 | 30%
Serviços      | R$ 50.000  | 10%

2.2 Crescimento Mensal

O crescimento foi de 15% comparado ao mês anterior.

3. DESPESAS

As despesas totais foram de R$ 350.000,00.

3.1 Despesas Operacionais

Incluem salários, aluguel e utilities.

4. RESULTADO LÍQUIDO

O resultado líquido foi de R$ 150.000,00.

4.1 Margem de Lucro

A margem de lucro atingiu 30% no período.`

	result, err := idx.Index(context.Background(), doc, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n=== Documento com Tabela ===\n")
	fmt.Printf("Pages: %d | Paragraphs: %d | Outline: %d | Chunks: %d\n",
		len(result.DocumentPages), len(result.ParagraphNodes),
		len(result.OutlineNodes), len(result.Chunks))

	fmt.Printf("\n=== Outline ===\n")
	for _, node := range result.OutlineNodes {
		indent := strings.Repeat("  ", node.Level-1)
		fmt.Printf("%s[%d] %s\n", indent, node.Level, node.Title)
	}

	// Verificar se linhas de tabela foram detectadas como headings (falso positivo)
	falsePositiveCount := 0
	for _, node := range result.OutlineNodes {
		title := node.Title
		if strings.Contains(title, "|") || strings.Contains(title, "---") {
			falsePositiveCount++
			fmt.Printf("⚠️  FALSO POSITIVO (tabela como heading): %s\n", title)
		}
	}
	if falsePositiveCount > 0 {
		t.Errorf("%d linhas de tabela detectadas como headings", falsePositiveCount)
	}
}
