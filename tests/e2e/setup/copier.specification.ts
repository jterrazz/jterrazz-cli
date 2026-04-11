import { resolve } from "node:path";

import { cli } from "@jterrazz/test";

export const TEMPLATE_PATH = resolve(import.meta.dirname, "../../../dotfiles/blueprints");

export const copierSpec = await cli({
  command: "copier",
  root: resolve(import.meta.dirname, "../fixtures"),
});
