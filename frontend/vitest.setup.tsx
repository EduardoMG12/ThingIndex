import "@testing-library/jest-dom/vitest";

import React from "react";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

vi.mock("next/navigation", () => ({
    usePathname: () => "/en",
    useRouter: () => ({
        push: vi.fn(),
        replace: vi.fn(),
        refresh: vi.fn(),
        prefetch: vi.fn(),
        back: vi.fn(),
    }),
    redirect: (href: string) => {
        throw new Error(`NEXT_REDIRECT:${href}`);
    },
}));

vi.mock("framer-motion", () => {
    const createMotionComponent = (tag: string) =>
        React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(function MotionComponent(
            { children, ...props },
            ref
        ) {
            const domProps = { ...props } as React.HTMLAttributes<HTMLElement> & Record<string, unknown>;
            delete domProps.animate;
            delete domProps.initial;
            delete domProps.transition;
            delete domProps.whileHover;
            delete domProps.whileTap;

            return React.createElement(tag, { ...domProps, ref }, children);
        });

    const motion = new Proxy(
        {},
        {
            get: (_target, property) => createMotionComponent(String(property)),
        }
    );

    return {
        AnimatePresence: ({ children }: { children: React.ReactNode }) => <>{children}</>,
        motion,
    };
});

Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
    })),
});

Object.defineProperty(window, "scrollTo", {
    writable: true,
    value: vi.fn(),
});

class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
}

vi.stubGlobal("ResizeObserver", ResizeObserverMock);
