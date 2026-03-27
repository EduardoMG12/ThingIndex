import { addons } from "storybook/manager-api";
import { create } from "storybook/theming";

addons.setConfig({
    theme: create({
        base: "light",
        brandTitle: "ThingIndex Storybook",
        brandTarget: "_self",
        colorPrimary: "#b7791f",
        colorSecondary: "#0f766e",
        appBg: "#f5f2e8",
        appContentBg: "#ffffff",
        appBorderColor: "#d6d0c3",
        appBorderRadius: 14,
        barBg: "#ffffff",
        barTextColor: "#1f2937",
        barSelectedColor: "#0f766e",
        inputBg: "#ffffff",
        inputBorder: "#d6d0c3",
        inputTextColor: "#111827",
        textColor: "#111827",
        textInverseColor: "#ffffff",
    }),
});
