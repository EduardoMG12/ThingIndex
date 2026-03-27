import type { Meta, StoryObj } from "@storybook/nextjs-vite";
import { ArrowRight, Download, Sparkles } from "lucide-react";

import { Button } from "@/components/ui/button";

const meta = {
    title: "UI/Button",
    component: Button,
    tags: ["autodocs"],
    args: {
        children: "Launch flow",
        variant: "primary",
        size: "md",
    },
    argTypes: {
        onClick: { action: "clicked" },
    },
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Primary: Story = {};

export const Secondary: Story = {
    args: {
        variant: "secondary",
        children: "Review details",
    },
};

export const Outline: Story = {
    args: {
        variant: "outline",
        children: "Open panel",
    },
};

export const Ghost: Story = {
    args: {
        variant: "ghost",
        children: "Dismiss",
    },
};

export const WithIcons: Story = {
    render: () => (
        <div className="flex flex-wrap gap-3">
            <Button>
                <Sparkles className="mr-2 h-4 w-4" />
                Enable readiness
            </Button>
            <Button variant="outline">
                Download report
                <Download className="ml-2 h-4 w-4" />
            </Button>
            <Button variant="secondary">
                Continue
                <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
        </div>
    ),
};
