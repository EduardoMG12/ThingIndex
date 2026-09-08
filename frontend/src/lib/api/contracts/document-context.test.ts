import { describe, expect, it } from "vitest";

import { normalizeOutlineNodes, normalizeParagraphNodes, readBox } from "@/lib/api/contracts/document-context";

const BOX = { x_min: 72, x_max: 603.8, y_min: 85.9, y_max: 98.9, page_width: 612, page_height: 792 };

describe("normalizeOutlineNodes", () => {
    it("normaliza start_page em startPage e deixa undefined quando a API nao manda", () => {
        const [com, sem] = normalizeOutlineNodes([
            { node_id: "node-001", document_id: 1, sequence_order: 0, level: 1, title: "A", start_page: 3 },
            { node_id: "node-002", document_id: 1, sequence_order: 1, level: 1, title: "B" },
        ]);
        expect(com.startPage).toBe(3);
        expect(sem.startPage).toBeUndefined();
    });
});

describe("readBox", () => {
    it("devolve a caixa tipada quando o metadata traz box valido", () => {
        const [node] = normalizeParagraphNodes([
            { document_id: 1, sequence_order: 0, content: "x", page_number: 1, metadata: { box: BOX } },
        ]);
        expect(readBox(node)).toEqual({ xMin: 72, xMax: 603.8, yMin: 85.9, yMax: 98.9, pageWidth: 612, pageHeight: 792 });
    });

    it("devolve null para box ausente ou malformado", () => {
        expect(readBox({ metadata: {} })).toBeNull();
        expect(readBox({ metadata: undefined })).toBeNull();
        expect(readBox({ metadata: { box: { ...BOX, x_min: "72" } } })).toBeNull();
        expect(readBox({ metadata: { box: { ...BOX, page_width: 0 } } })).toBeNull();
        expect(readBox({ metadata: { box: { ...BOX, x_max: 10 } } })).toBeNull();
        expect(readBox({ metadata: { box: [72, 603] } })).toBeNull();
    });
});
