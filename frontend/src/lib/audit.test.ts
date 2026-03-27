import { describe, expect, it } from "vitest";

import { formatAuditActor, formatAuditActorType, parseAuditActorId } from "@/lib/audit";

describe("audit helpers", () => {
    it("formats actor labels with localized defaults", () => {
        expect(formatAuditActorType("USER", "pt-BR")).toBe("Usuario");
        expect(formatAuditActorType("SYSTEM", "pt-BR")).toBe("Sistema");
        expect(formatAuditActor("USER", 7, "pt-BR")).toBe("Usuario #7");
        expect(formatAuditActor("SERVICE", undefined, "pt-BR")).toBe("Servico");
        expect(formatAuditActorType("USER", "en")).toBe("User");
        expect(formatAuditActor("SYSTEM", 3, "en")).toBe("System #3");
    });

    it("parses only valid positive actor ids", () => {
        expect(parseAuditActorId("52")).toBe(52);
        expect(parseAuditActorId(" 9 ")).toBe(9);
        expect(parseAuditActorId("0")).toBeUndefined();
        expect(parseAuditActorId("-1")).toBeUndefined();
        expect(parseAuditActorId("")).toBeUndefined();
    });
});
