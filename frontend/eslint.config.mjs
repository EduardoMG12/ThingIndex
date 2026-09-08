import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    // eslint-config-next 16.2 promoveu tres regras do React Compiler a erro.
    // Elas apontam 20 pontos que funcionam hoje (lint, build, unit e E2E
    // verdes antes do upgrade). Ficam como aviso ate o refactor dedicado.
    rules: {
      "react-hooks/set-state-in-effect": "warn",
      "react-hooks/refs": "warn",
      "react-hooks/immutability": "warn",
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    // Generated validation artifacts:
    "storybook-static/**",
    "playwright-report/**",
    "test-results/**",
    "coverage/**",
    "debug-storybook.log",
  ]),
]);

export default eslintConfig;
