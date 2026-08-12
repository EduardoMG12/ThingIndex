import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ProviderBrandMark } from "@/components/providers/ProviderBrandMark";

describe("ProviderBrandMark", () => {
    it("renders a stable provider-specific test id and monogram", () => {
        render(<ProviderBrandMark providerType="OPENAI" />);

        expect(screen.getByTestId("provider-brand-OPENAI")).toBeInTheDocument();
        expect(screen.getByText("OA")).toBeInTheDocument();
    });

    it("falls back gracefully when the provider type is missing", () => {
        render(<ProviderBrandMark providerType={null} showMonogram={false} />);

        expect(screen.getByTestId("provider-brand-unknown")).toBeInTheDocument();
    });
});
