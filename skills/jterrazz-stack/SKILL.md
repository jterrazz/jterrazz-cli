---
name: jterrazz-stack
description: Overview of the @jterrazz ecosystem — shared npm packages, naming conventions, project patterns, and how everything composes together. Activates when working on any jterrazz project, choosing packages, or understanding the stack.
---

# @jterrazz Stack

All projects share a composable set of npm packages under the `@jterrazz` scope.

## Packages

| Package | Purpose | npm script |
|---------|---------|------------|
| `@jterrazz/typescript` | Build tooling (tsdown) | `typescript build`, `typescript bundle`, `typescript dev`, `typescript start` |
| `@jterrazz/codestyle` | Linting + formatting (tsgo, oxlint, oxfmt) | `codestyle check`, `codestyle fix` |
| `@jterrazz/test` | Test utilities (vitest, MSW, mockDate) | `vitest --run` |
| `@jterrazz/logger` | Structured logging (pino) | — |
| `@jterrazz/intelligence` | AI toolkit (OpenRouter, Langfuse) | — |
| `@jterrazz/broadcast` | Multi-channel announcements (App Store, push) | — |

## Project types

**Library** (`package-*`):
```json
{
  "build": "typescript bundle",
  "lint": "codestyle check",
  "lint:fix": "codestyle fix",
  "test": "vitest --run"
}
```

**Application** (`signews-api`, `signews-broadcast`, etc.):
```json
{
  "build": "typescript build",
  "start": "typescript start",
  "dev": "typescript dev",
  "lint": "codestyle check",
  "lint:fix": "codestyle fix",
  "test": "vitest --run"
}
```

## Naming conventions

```
{product}-{role}
├── signews-api          # Backend API
├── signews-web          # Web client
├── signews-mobile       # iOS/Android app
├── signews-broadcast    # Event broadcaster
├── signews-blueprint    # Architecture docs
└── package-{name}       # Shared @jterrazz/* packages
```

Roles: `-api`, `-web`, `-mobile`, `-broadcast`, `-blueprint`, `-landing`

## Required files

Every project must have:
- `Makefile` with `build`, `lint`, `test` targets
- `tsconfig.json` extending `@jterrazz/typescript/tsconfig/node`
- `.oxlintrc.json` extending `@jterrazz/codestyle/oxlint/node`
- `.github/workflows/validate.yaml` using shared workflow

## CI/CD

Shared workflows from `jterrazz/jterrazz-workflows`:
- `validate.yaml` — runs `make build`, `make lint`, `make test`
- `build-and-deploy.yaml` — Docker build + Helm deploy for apps
- `publish-package.yaml` — npm publish with OIDC provenance for libraries

## Architecture pattern

Libraries use **ports & adapters**:
- `src/ports/` — interfaces
- `src/adapters/` — implementations
- `src/index.ts` — public API exports

Apps use **hexagonal architecture**:
- `src/domain/` — pure business logic
- `src/application/` — use cases, ports
- `src/infrastructure/` — adapters (HTTP, DB, external APIs)

## Always

- Use `npm` (not pnpm/yarn) — all repos have `package-lock.json`
- Node.js 24
- ESM only (`"type": "module"`)
- `.js` extensions in imports for Node.js projects
- Author: `Jean-Baptiste Terrazzoni <contact@jterrazz.com>`
