import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import AccountStatus from "@/components/dashboard/AccountStatus";
import { createTestWrapper } from "@/test/test-utils";

describe("AccountStatus", () => {
    it("renders honest operational copy instead of static telemetry placeholders", () => {
        render(<AccountStatus />, {
            wrapper: createTestWrapper(),
        });

        expect(screen.getByText("Session-protected routes")).toBeInTheDocument();
        expect(screen.getByText("Runtime, cost, and audit")).toBeInTheDocument();
        expect(screen.getByText("Sensitive config stays in backend")).toBeInTheDocument();
        expect(screen.queryByText("1.2k / 5k tokens")).not.toBeInTheDocument();
        expect(screen.queryByText("Expires in 4h")).not.toBeInTheDocument();
    });
});
