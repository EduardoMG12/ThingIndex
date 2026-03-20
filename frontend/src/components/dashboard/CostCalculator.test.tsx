import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import CostCalculator from "@/components/dashboard/CostCalculator";
import { createTestWrapper } from "@/test/test-utils";

describe("CostCalculator", () => {
    it("describes runtime-dependent cost visibility without fake fixed estimates", () => {
        render(<CostCalculator />, {
            wrapper: createTestWrapper(),
        });

        expect(screen.getByText("Derived from configured provider")).toBeInTheDocument();
        expect(screen.getByText("Requires synced prices and real usage")).toBeInTheDocument();
        expect(screen.getByText("No fixed amount shown")).toBeInTheDocument();
        expect(screen.queryByText(/vertex ai/i)).not.toBeInTheDocument();
        expect(screen.queryByText(/\$0\./)).not.toBeInTheDocument();
    });
});
