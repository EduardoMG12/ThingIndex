import type { StorybookConfig } from "@storybook/nextjs-vite";

const config: StorybookConfig = {
    stories: [
        "../src/**/*.mdx",
        "../src/**/*.stories.@(ts|tsx)",
    ],
    addons: [
        "@storybook/addon-docs",
        "@storybook/addon-a11y",
        "@storybook/addon-vitest",
    ],
    framework: {
        name: "@storybook/nextjs-vite",
        options: {},
    },
    staticDirs: ["../public"],
    docs: {
        autodocs: true,
    },
    typescript: {
        reactDocgen: "react-docgen-typescript",
    },
};

export default config;
