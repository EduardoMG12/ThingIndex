/**
 * Marcas coloridas e legenda do workspace, como a API devolve. Normalizadores
 * tolerantes a snake_case e camelCase, como os outros contratos; cor fora da
 * paleta e descartada em vez de quebrar a tela.
 */

export const CORES = ["amarelo", "verde", "azul", "vermelho", "roxo"] as const;
export type HighlightColor = (typeof CORES)[number];

export function ehCor(valor: unknown): valor is HighlightColor {
    return typeof valor === "string" && (CORES as readonly string[]).includes(valor);
}

export type Highlight = {
    paragraphNodeId: number;
    documentId: number;
    color: HighlightColor;
};

export type Legend = Partial<Record<HighlightColor, string>>;

function asRecord(value: unknown): Record<string, unknown> | null {
    return typeof value === "object" && value !== null && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

function asNumber(value: unknown): number | null {
    return typeof value === "number" && Number.isFinite(value) ? value : null;
}

export function normalizeHighlight(value: unknown): Highlight | null {
    const record = asRecord(value);
    if (!record) {
        return null;
    }
    const paragraphNodeId = asNumber(record.paragraphNodeId ?? record.paragraph_node_id);
    const documentId = asNumber(record.documentId ?? record.document_id);
    const color = record.color;
    if (paragraphNodeId === null || documentId === null || !ehCor(color)) {
        return null;
    }
    return { paragraphNodeId, documentId, color };
}

export function normalizeHighlights(value: unknown): Highlight[] {
    const record = asRecord(value);
    const raw = Array.isArray(record?.highlights) ? record!.highlights : Array.isArray(value) ? value : [];
    return raw.map(normalizeHighlight).filter((h): h is Highlight => h !== null);
}

export function normalizeLegend(value: unknown): Legend {
    const record = asRecord(value);
    const labels = asRecord(record?.labels ?? value);
    const legend: Legend = {};
    if (!labels) {
        return legend;
    }
    for (const [color, label] of Object.entries(labels)) {
        if (ehCor(color) && typeof label === "string" && label.trim() !== "") {
            legend[color] = label;
        }
    }
    return legend;
}
