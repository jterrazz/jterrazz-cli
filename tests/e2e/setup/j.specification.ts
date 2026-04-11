import { execSync } from "node:child_process";
import { existsSync } from "node:fs";
import { resolve } from "node:path";

import { cli } from "@jterrazz/test";

const REPO_ROOT = resolve(import.meta.dirname, "../../..");
const J_BIN = resolve(REPO_ROOT, "tests/e2e/j_test_bin");

// Build once before any spec runs.
if (!existsSync(J_BIN) || process.env.J_FORCE_REBUILD === "1") {
  execSync(`go build -o ${J_BIN} ./src/cmd/j`, {
    cwd: REPO_ROOT,
    stdio: "inherit",
  });
}

export const jSpec = await cli({
  command: J_BIN,
  root: resolve(import.meta.dirname, "../fixtures"),
});
