import { describe, expect, test } from "vitest";

import { jSpec } from "./setup/j.specification.js";

describe("j CLI — help and metadata", () => {
  test("install --help", async () => {
    const result = await jSpec("install help").exec("install --help").run();
    expect(result.exitCode).toBe(0);
    expect(result.stdout + result.stderr).toContain("install");
  });

  test("clean --help", async () => {
    const result = await jSpec("clean help").exec("clean --help").run();
    expect(result.exitCode).toBe(0);
    expect(result.stdout + result.stderr).toContain("clean");
  });

  test("upgrade --help", async () => {
    const result = await jSpec("upgrade help").exec("upgrade --help").run();
    expect(result.exitCode).toBe(0);
    expect(result.stdout + result.stderr).toContain("upgrade");
  });

  test("run git --help lists subcommands", async () => {
    const result = await jSpec("run git help").exec("run git --help").run();
    const output = result.stdout + result.stderr;
    for (const sub of ["feat", "fix", "chore", "push", "sync"]) {
      expect(output).toContain(sub);
    }
  });

  test("run docker --help lists subcommands", async () => {
    const result = await jSpec("run docker help").exec("run docker --help").run();
    const output = result.stdout + result.stderr;
    for (const sub of ["rm", "rmi", "clean", "reset"]) {
      expect(output).toContain(sub);
    }
  });

  test("host --help", async () => {
    const result = await jSpec("host help").exec("host --help").run();
    const output = result.stdout + result.stderr;
    expect(result.exitCode).toBe(0);
    expect(output).toContain("host");
    expect(output).toContain("--profile");
  });
});

describe("j sync", () => {
  test("sync --help lists subcommands and flags", async () => {
    const result = await jSpec("sync help").exec("sync --help").run();
    const output = result.stdout + result.stderr;
    for (const sub of ["init", "status", "diff"]) {
      expect(output).toContain(sub);
    }
    expect(output).toContain("--all");
  });

  test("sync init --help describes the command", async () => {
    const result = await jSpec("sync init help").exec("sync init --help").run();
    expect(result.stdout + result.stderr).toContain("Initialize project from template");
  });

  test("sync status — unlinked directory", async () => {
    // Given — empty temp dir, no .copier-answers.yml
    const result = await jSpec("sync status unlinked")
      .exec("sync status")
      .run();

    // Then — reports unlinked and hints at init
    expect(result.exitCode).toBe(0);
    const output = result.stdout + result.stderr;
    expect(output).toContain("Not linked");
    expect(output).toContain("j sync init");
  });

  test("sync status — linked directory", async () => {
    // Given — fixture project with a .copier-answers.yml
    const result = await jSpec("sync status linked")
      .project("linked-project")
      .exec("sync status")
      .run();

    // Then — reports linked with project metadata
    expect(result.exitCode).toBe(0);
    const output = result.stdout + result.stderr;
    expect(output).toContain("Linked");
    expect(output).toContain("project_name");
    expect(output).toContain("test-project");
    expect(output).toContain("language");
  });

  test("sync (no subcommand, unlinked) — warns to run init", async () => {
    const result = await jSpec("sync unlinked").exec("sync").run();
    const output = result.stdout + result.stderr;
    expect(output).toContain("No .copier-answers.yml");
    expect(output).toContain("j sync init");
  });

  test("sync diff — unlinked", async () => {
    const result = await jSpec("sync diff unlinked")
      .exec("sync diff")
      .run();
    expect(result.stdout + result.stderr).toContain("No .copier-answers.yml");
  });

  test("sync --all with no projects", async () => {
    // Given — isolated HOME with an empty Developer/ dir
    const result = await jSpec("sync all empty")
      .project("empty-home")
      .env({ HOME: "$WORKDIR" })
      .exec("sync --all")
      .run();

    // Then — reports no projects (or copier missing in CI)
    const output = result.stdout + result.stderr;
    expect(output).toMatch(/No projects|copier not installed/);
  });
});
