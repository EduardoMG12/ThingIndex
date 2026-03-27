import type { ComponentProps } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import ProviderRuntimeCallout from "@/components/runtime/ProviderRuntimeCallout";
import buttonMeta, { Primary } from "@/components/ui/button.stories";
import runtimeMeta, { ChatActiveRuntime } from "@/components/runtime/ProviderRuntimeCallout.stories";
import { Button } from "@/components/ui/button";
import { I18nProvider } from "@/lib/i18n/provider";

type ButtonStoryArgs = ComponentProps<typeof Button>;
type RuntimeStoryArgs = ComponentProps<typeof ProviderRuntimeCallout>;

const primaryButtonArgs = {
    ...buttonMeta.args,
    ...Primary.args,
} satisfies ButtonStoryArgs;

const activeRuntimeArgs = {
    ...runtimeMeta.args,
    ...ChatActiveRuntime.args,
} satisfies RuntimeStoryArgs;

describe("storybook smoke", () => {
    it("renders the primary button story args", () => {
        render(<Button {...primaryButtonArgs} />);

        expect(screen.getByRole("button", { name: "Launch flow" })).toBeInTheDocument();
    });

    it("renders the active runtime story within the shared i18n provider", () => {
        render(
            <I18nProvider locale="en">
                <ProviderRuntimeCallout {...activeRuntimeArgs} />
            </I18nProvider>
        );

        expect(screen.getByRole("heading", { name: "Active chat LLM" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Review in settings/i })).toHaveAttribute("href", "/en/settings");
    });
});
