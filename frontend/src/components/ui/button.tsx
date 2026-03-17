"use client";

import * as React from "react";
import { motion, HTMLMotionProps } from "framer-motion";
import { cn } from "@/lib/utils";

interface ButtonProps extends HTMLMotionProps<"button"> {
    asChild?: boolean;
    variant?: "primary" | "secondary" | "ghost" | "outline";
    size?: "sm" | "md" | "lg" | "icon";
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    ({ className, variant = "primary", size = "md", asChild = false, children, ...props }, ref) => {
        const variants = {
            primary: "bg-primary text-primary-foreground hover:opacity-90 shadow-gold",
            secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
            ghost: "bg-transparent text-foreground hover:bg-secondary/50",
            outline: "bg-transparent border border-border text-foreground hover:border-primary/50",
        };

        const sizes = {
            sm: "px-3 py-1.5 text-xs",
            md: "px-4 py-2 text-sm",
            lg: "px-6 py-3 text-base",
            icon: "h-8 w-8",
        };

        const classes = cn(
            "inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50",
            variants[variant],
            sizes[size],
            className
        );

        if (asChild && React.isValidElement(children)) {
            const child = React.Children.only(children);

            return React.cloneElement(child, {
                ...props,
                className: cn(classes, (child.props as { className?: string }).className),
            });
        }

        return (
            <motion.button
                ref={ref}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                className={classes}
                {...props}
            >
                {children}
            </motion.button>
        );
    }
);

Button.displayName = "Button";
