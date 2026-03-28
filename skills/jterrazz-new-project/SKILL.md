---
name: jterrazz-new-project
description: Scaffolds a new @jterrazz project with all conventions — package.json, tsconfig, Makefile, CI workflows, linting. Use when creating a new app or library from scratch.
---

# Create a new @jterrazz project

## For a new library (package-{name})

```bash
mkdir package-{name} && cd package-{name}
git init
npm init -y
```

**package.json:**
```json
{
  "name": "@jterrazz/{name}",
  "version": "0.1.0",
  "author": "Jean-Baptiste Terrazzoni <contact@jterrazz.com>",
  "type": "module",
  "files": ["dist"],
  "exports": {
    ".": {
      "require": "./dist/index.cjs",
      "import": "./dist/index.js"
    }
  },
  "publishConfig": { "registry": "https://registry.npmjs.org/" },
  "repository": { "type": "git", "url": "https://github.com/jterrazz/package-{name}" },
  "scripts": {
    "build": "typescript bundle",
    "lint": "codestyle check",
    "lint:fix": "codestyle fix",
    "test": "vitest --run"
  },
  "devDependencies": {
    "@jterrazz/codestyle": "latest",
    "@jterrazz/test": "latest",
    "@jterrazz/typescript": "latest",
    "@types/node": "latest",
    "vitest": "latest"
  }
}
```

**tsconfig.json:**
```json
{ "extends": "@jterrazz/typescript/presets/tsconfig/node" }
```

**.oxlintrc.json:**
```json
{ "extends": ["@jterrazz/codestyle/oxlint/node"] }
```

**.gitignore:**
```
node_modules/
dist
.DS_Store
.idea
```

**Makefile:**
```makefile
.PHONY: build lint test install

node_modules/.install: package-lock.json
	npm ci
	@touch node_modules/.install

install: node_modules/.install

build: node_modules/.install
	npm run build

lint: node_modules/.install
	npm run lint

test: node_modules/.install
	npm test
```

**Create `src/index.ts`**, then `npm install && npm run build`.

## For a new application ({product}-{role})

Same as library but with these differences:

**package.json scripts:**
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

No `exports` or `publishConfig` needed.

## CI workflows

**.github/workflows/validate.yaml:**
```yaml
name: Validate
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  validate:
    uses: jterrazz/jterrazz-workflows/.github/workflows/validate.yaml@main
    with:
      node-version: "24"
```

For libraries, also add **.github/workflows/publish.yaml:**
```yaml
name: Publish
on:
  release:
    types: [created]
permissions:
  contents: read
  id-token: write
jobs:
  publish:
    uses: jterrazz/jterrazz-workflows/.github/workflows/publish-package.yaml@main
    with:
      node-version: "24"
    secrets: inherit
```

## Never

- Never use pnpm or yarn
- Never skip the Makefile
- Never put business logic in `src/index.ts` — it's only for exports
