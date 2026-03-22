import { describe, expect, it } from "vitest";

import { formatAuditActor, formatAuditActorType, parseAuditActorId } from "@/lib/audit";

describe("audit helpers", () => {
    it("formats actor labels with localized defaults", () => {
        expect(formatAuditActorType("USER")).toBe("Usuario");
        expect(formatAuditActorType("SYSTEM")).toBe("Sistema");
        expect(formatAuditActor("USER", 7)).toBe("Usuario #7");
        expect(formatAuditActor("SERVICE")).toBe("Servico");
    });

    it("parses only valid positive actor ids", () => {
        expect(parseAuditActorId("52")).toBe(52);
        expect(parseAuditActorId(" 9 ")).toBe(9);
        expect(parseAuditActorId("0")).toBeUndefined();
        expect(parseAuditActorId("-1")).toBeUndefined();
        expect(parseAuditActorId("")).toBeUndefined();
    });
});
