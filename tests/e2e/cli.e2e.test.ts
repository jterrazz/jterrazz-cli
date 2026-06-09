import { describe, expect, test } from "vitest";

import { jSpec } from "./setup/j.specification.js";

describe("j CLI — help and metadata", () => {
  test("install --help", async () => {
    const result = await jSpec("install help").exec("install --help").run();
    expect(result.exitCode).toBe(0);
    expect(result.stdout + result.stderr).toContain("install");
  });

  test("install list includes Jump Desktop apps", async () => {
    const result = await jSpec("install list").exec("install").run();
    const output = result.stdout + result.stderr;
    expect(result.exitCode).toBe(0);
    expect(output).toContain("jump-desktop-connect");
    expect(output).toContain("jump-desktop");
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

  test("host status includes homelab readiness checks", async () => {
    const result = await jSpec("host status homelab")
      .exec("host status --profile homelab")
      .run();
    const output = result.stdout + result.stderr;
    expect(result.exitCode).toBe(0);
    expect(output).toContain("Profile: homelab");
    expect(output).toContain("OpenClaw");
    expect(output).toContain("Console");
    expect(output).toContain("Jump Connect app");
    expect(output).toContain("Jump client app");
    expect(output).toContain("Jump audio");
    expect(output).toContain("Auto boot");
    expect(output).toContain("Wake network");
    expect(output).toContain("Power button");
    expect(output).toContain("OrbStack bg agent");
  });

  test("host unlock dry-run prints FileVault SSH command", async () => {
    const result = await jSpec("host unlock dry-run")
      .exec("host unlock --dry-run")
      .run();
    const output = result.stdout + result.stderr;
    expect(result.exitCode).toBe(0);
    expect(output).toContain("FILEVAULT UNLOCK");
    expect(output).toContain("jterrazz.agent@192.168.1.106");
    expect(output).toContain("PreferredAuthentications=password");
    expect(output).toContain("PubkeyAuthentication=no");
  });
});
