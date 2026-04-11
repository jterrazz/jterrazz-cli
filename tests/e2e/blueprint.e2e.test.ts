import { describe, expect, test } from "vitest";

import { copierSpec, TEMPLATE_PATH } from "./setup/copier.specification.js";

interface BlueprintVariant {
  data: Record<string, string>;
  name: string;
  wantFiles?: string[];
  excludedFiles?: string[];
}

const COMMON_FILES = [".editorconfig", ".gitignore", "LICENSE"];
const TS_FILES = [".nvmrc", "tsconfig.json"];
const TS_PKG_FILES = ["package.json", "vitest.config.ts"];
const GO_FILES = ["go.mod", "Makefile", ".golangci.yml"];
const DOCKER_FILES = ["Dockerfile", ".dockerignore"];
const CI_WORKFLOW = ".github/workflows/ci.yml";
const RELEASE_WORKFLOW = ".github/workflows/release.yml";
const DEPLOY_WORKFLOW = ".github/workflows/deploy.yml";

async function runBlueprint(variant: BlueprintVariant): Promise<void> {
  const dataArgs = Object.entries(variant.data)
    .map(([k, v]) => `--data ${k}=${v}`)
    .join(" ");

  // Generate the project into a fresh temp working directory.
  const result = await copierSpec(`scaffold ${variant.name}`)
    .exec(`copy --trust --defaults --quiet ${dataArgs} ${TEMPLATE_PATH} .`)
    .run();

  expect(result.exitCode, result.stderr || result.stdout).toBe(0);

  // Snapshot the entire generated tree against the committed fixture.
  await result.directory(".").toMatchFixture(variant.name);

  // Bonus: explicit presence/absence checks against the fixture file list.
  if (variant.wantFiles || variant.excludedFiles) {
    const files = new Set(await result.directory(".").files());
    for (const want of variant.wantFiles ?? []) {
      expect(files.has(want), `expected file ${want}`).toBe(true);
    }
    for (const not of variant.excludedFiles ?? []) {
      expect(files.has(not), `expected file ${not} NOT to exist`).toBe(false);
    }
  }
}

describe("blueprint — none (license variants)", () => {
  test("none-mit", async () => {
    await runBlueprint({
      name: "none-mit",
      data: {
        project_name: "my-config",
        language: "none",
        project_type: "none",
        license: "MIT",
        ci: "false",
        docker: "false",
        deploy: "none",
      },
      wantFiles: COMMON_FILES,
      excludedFiles: [
        ...TS_FILES,
        ...TS_PKG_FILES,
        ...GO_FILES,
        ...DOCKER_FILES,
        CI_WORKFLOW,
        RELEASE_WORKFLOW,
        DEPLOY_WORKFLOW,
      ],
    });
  });

  test("none-proprietary", async () => {
    await runBlueprint({
      name: "none-proprietary",
      data: {
        project_name: "my-config",
        language: "none",
        project_type: "none",
        license: "proprietary",
        ci: "false",
        docker: "false",
        deploy: "none",
      },
      wantFiles: COMMON_FILES,
    });
  });
});

describe("blueprint — typescript", () => {
  test("typescript-none", async () => {
    await runBlueprint({
      name: "typescript-none",
      data: {
        project_name: "my-ts",
        language: "typescript",
        project_type: "none",
        license: "MIT",
        ci: "false",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...TS_FILES, ...TS_PKG_FILES],
      excludedFiles: [...GO_FILES, ...DOCKER_FILES, CI_WORKFLOW],
    });
  });

  test("typescript-library", async () => {
    await runBlueprint({
      name: "typescript-library",
      data: {
        project_name: "my-lib",
        language: "typescript",
        project_type: "library",
        license: "MIT",
        ci: "true",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...TS_FILES, ...TS_PKG_FILES, CI_WORKFLOW, RELEASE_WORKFLOW],
      excludedFiles: [...GO_FILES, ...DOCKER_FILES, DEPLOY_WORKFLOW],
    });
  });

  test("typescript-api", async () => {
    await runBlueprint({
      name: "typescript-api",
      data: {
        project_name: "my-api",
        language: "typescript",
        project_type: "api",
        license: "MIT",
        ci: "true",
        docker: "true",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...TS_FILES, ...TS_PKG_FILES, ...DOCKER_FILES, CI_WORKFLOW],
    });
  });

  test("typescript-web", async () => {
    await runBlueprint({
      name: "typescript-web",
      data: {
        project_name: "my-web",
        language: "typescript",
        project_type: "web",
        license: "MIT",
        ci: "true",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...TS_FILES, ...TS_PKG_FILES, CI_WORKFLOW],
    });
  });

  test("typescript-mobile", async () => {
    await runBlueprint({
      name: "typescript-mobile",
      data: {
        project_name: "my-mobile",
        language: "typescript",
        project_type: "mobile",
        license: "MIT",
        ci: "true",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...TS_FILES, CI_WORKFLOW],
    });
  });

  test("typescript-api-deploy", async () => {
    await runBlueprint({
      name: "typescript-api-deploy",
      data: {
        project_name: "my-api-deploy",
        language: "typescript",
        project_type: "api",
        license: "MIT",
        ci: "true",
        docker: "true",
        deploy: "kubernetes",
      },
      wantFiles: [
        ...COMMON_FILES,
        ...TS_FILES,
        ...TS_PKG_FILES,
        ...DOCKER_FILES,
        CI_WORKFLOW,
        DEPLOY_WORKFLOW,
      ],
    });
  });
});

describe("blueprint — go", () => {
  test("go-none", async () => {
    await runBlueprint({
      name: "go-none",
      data: {
        project_name: "my-go",
        language: "go",
        project_type: "none",
        license: "MIT",
        ci: "false",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...GO_FILES],
    });
  });

  test("go-cli", async () => {
    await runBlueprint({
      name: "go-cli",
      data: {
        project_name: "my-go-cli",
        language: "go",
        project_type: "cli",
        license: "MIT",
        ci: "true",
        docker: "false",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...GO_FILES, CI_WORKFLOW],
    });
  });

  test("go-api", async () => {
    await runBlueprint({
      name: "go-api",
      data: {
        project_name: "my-go-api",
        language: "go",
        project_type: "api",
        license: "MIT",
        ci: "true",
        docker: "true",
        deploy: "none",
      },
      wantFiles: [...COMMON_FILES, ...GO_FILES, ...DOCKER_FILES, CI_WORKFLOW],
    });
  });
});

describe("blueprint — content checks", () => {
  test("MIT license contains author", async () => {
    const result = await copierSpec("license mit")
      .exec(
        `copy --trust --defaults --quiet --data project_name=x --data language=none --data project_type=none --data license=MIT --data ci=false --data docker=false --data deploy=none ${TEMPLATE_PATH} .`,
      )
      .run();

    expect(result.exitCode).toBe(0);
    const license = result.file("LICENSE").content;
    expect(license).toContain("MIT License");
    expect(license).toContain("Jean-Baptiste Terrazzoni");
  });

  test("proprietary license", async () => {
    const result = await copierSpec("license proprietary")
      .exec(
        `copy --trust --defaults --quiet --data project_name=x --data language=none --data project_type=none --data license=proprietary --data ci=false --data docker=false --data deploy=none ${TEMPLATE_PATH} .`,
      )
      .run();

    expect(result.exitCode).toBe(0);
    expect(result.file("LICENSE").content).toContain("All Rights Reserved");
  });

  test("typescript tsconfig variants", async () => {
    const cases: { project_type: string; expectedExtends: string }[] = [
      { project_type: "library", expectedExtends: "@jterrazz/typescript/tsconfig/node" },
      { project_type: "api", expectedExtends: "@jterrazz/typescript/tsconfig/node" },
      { project_type: "web", expectedExtends: "@jterrazz/typescript/tsconfig/next.json" },
      { project_type: "mobile", expectedExtends: "@jterrazz/typescript/tsconfig/expo" },
    ];

    for (const tc of cases) {
      const result = await copierSpec(`tsconfig ${tc.project_type}`)
        .exec(
          `copy --trust --defaults --quiet --data project_name=x --data language=typescript --data project_type=${tc.project_type} --data license=MIT --data ci=true --data docker=false --data deploy=none ${TEMPLATE_PATH} .`,
        )
        .run();

      expect(result.exitCode, result.stderr).toBe(0);
      expect(result.file("tsconfig.json").content).toContain(tc.expectedExtends);
    }
  });

  test("workflow files have no leftover {% raw %} tags", async () => {
    const result = await copierSpec("raw escape")
      .exec(
        `copy --trust --defaults --quiet --data project_name=x --data language=typescript --data project_type=library --data license=MIT --data ci=true --data docker=false --data deploy=none ${TEMPLATE_PATH} .`,
      )
      .run();

    expect(result.exitCode).toBe(0);
    const workflows = await result.directory(".github/workflows").files();
    expect(workflows.length).toBeGreaterThan(0);

    for (const wf of workflows) {
      const content = result.file(`.github/workflows/${wf}`).content;
      expect(content, `${wf} contains {% raw %}`).not.toContain("{% raw %}");
      expect(content, `${wf} contains {% endraw %}`).not.toContain("{% endraw %}");
    }
  });
});
