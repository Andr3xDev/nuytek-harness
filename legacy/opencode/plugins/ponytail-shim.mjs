// ponytail-shim — V2 bridge for the V1-only @dietrichgebert/ponytail plugin.
//
// V1 hooks (ponytail v4.9.0 .opencode/plugins/ponytail.mjs) → V2:
//   config (commands + skills.paths)            → ctx.command.transform + ctx.skill.transform
//   experimental.chat.system.transform          → ctx.session.hook("context")
//   command.execute.before (mode persistence)   → the "ponytail" command executor
//
// Approach: REUSE. The shared instruction builder / config helpers are required
// directly from the installed package by absolute path (its exports map only
// "." and "./plugin", so the internals are not importable as subpaths). Only
// the thin V1 glue is reimplemented. If the package cannot be resolved, the
// shim disables itself rather than forking the ruleset.

import { Plugin } from "@opencode/plugin";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { fileURLToPath } from "url";

// ponytail package.json exports map blocks subpaths like ./package.json, so
// resolve the main entry and walk up to the package root instead.
export function loadPonytail() {
  const req = createRequire(import.meta.url);
  let dir = path.dirname(req.resolve("@dietrichgebert/ponytail"));
  let root = null;
  for (let i = 0; i < 6; i++) {
    if (fs.existsSync(path.join(dir, "hooks", "ponytail-instructions.js"))) {
      root = dir;
      break;
    }
    dir = path.dirname(dir);
  }
  if (!root) {
    throw new Error("ponytail hooks/ not found above " + req.resolve("@dietrichgebert/ponytail"));
  }
  const instructions = req(path.join(root, "hooks", "ponytail-instructions.js"));
  const config = req(path.join(root, "hooks", "ponytail-config.js"));
  const { parseCommandFile } = req(
    path.join(root, ".opencode", "plugins", "ponytail-frontmatter.cjs")
  );
  return { root, rootReq: req, instructions, config, parseCommandFile };
}

let ponytail = null;
try {
  ponytail = loadPonytail();
} catch (e) {
  console.error(`[ponytail-shim] could not load @dietrichgebert/ponytail: ${e?.message ?? e}`);
}

// Same flag file as the V1 plugin: mode beside the opencode config.
const statePath = path.join(
  process.env.XDG_CONFIG_HOME || path.join(os.homedir(), ".config"),
  "opencode",
  ".ponytail-active",
);

function readMode() {
  try {
    return (
      ponytail.config.normalizePersistedMode(fs.readFileSync(statePath, "utf8").trim()) ||
      ponytail.config.getDefaultMode()
    );
  } catch {
    return ponytail.config.getDefaultMode();
  }
}

function writeMode(mode) {
  fs.mkdirSync(path.dirname(statePath), { recursive: true });
  fs.writeFileSync(statePath, mode);
}

function appendToSystem(event, text) {
  if (event.system.length > 0) {
    event.system[event.system.length - 1].text += "\n\n" + text;
  } else {
    event.system.push({ type: "text", text });
  }
}

export default Plugin.define({
  id: "ponytail",
  setup(ctx) {
    if (!ponytail) return () => {};

    // ── Prompt injection: same single-system-message append as the V1 plugin ──
    // "review" is handled inside get PonytailInstructions (session-only pointer
    // text); "off" stays silent, exactly like the V1 transform.
    ctx.session.hook("context", (event) => {
      const mode = readMode();
      if (mode === "off") return;
      const instructions = ponytail.instructions.getPonytailInstructions(mode);
      appendToSystem(event, instructions);
    });

    // ── Slash commands (V1 config hook registered these as config.command) ──
    const commandDir = path.join(ponytail.root, ".opencode", "command");
    const skillDir = path.join(ponytail.root, "skills");
    const commands = [];
    try {
      for (const file of fs.readdirSync(commandDir).filter((f) => f.endsWith(".md"))) {
        const name = path.basename(file, ".md");
        const parsed = ponytail.parseCommandFile(path.join(commandDir, file));
        if (parsed) commands.push({ name, ...parsed });
      }
    } catch {}
    const skills = [];
    try {
      for (const entry of fs.readdirSync(skillDir, { withFileTypes: true })) {
        if (!entry.isDirectory()) continue;
        const skillPath = path.join(skillDir, entry.name, "SKILL.md");
        const content = fs.readFileSync(skillPath, "utf8");
        const fm = content.match(/^---\r?\n([\s\S]*?)\r?\n---/)?.[1] ?? "";
        const name = fm.match(/^name:\s*(.+)\s*$/m)?.[1]?.trim() || entry.name;
        // Multi-line folded YAML descriptions are flattened to one line.
        const description =
          fm.match(/^description:\s*>-?\s*\n([\s\S]*?)(?=^\s*\w|^\s*argument)/m)?.[1];
        const flat =
          description ? description.replace(/\s*\n\s*/g, " ").trim() :
          fm.match(/^description:\s*(.+)\s*$/m)?.[1]?.trim().replace(/("|')\s*$/, "");
        skills.push({ id: entry.name, name, description: flat, path: skillPath, content });
      }
    } catch {}

    ctx.command.transform((editor) => {
      for (const { name, description, template } of commands) {
        editor.add({
          name,
          description,
          execute: async (invocation) => {
            const args = (invocation.prompt?.text ?? "").trim();
            // Persist /ponytail <level> exactly like the V1 command.execute.before
            // hook did; the injected instructions follow from the next turn.
            if (name === "ponytail") {
              const mode = args ? ponytail.config.normalizePersistedMode(args) : ponytail.config.getDefaultMode();
              if (mode) writeMode(mode);
            }
            const text = template.includes("$ARGUMENTS")
              ? template.replace("$ARGUMENTS", args)
              : args
                ? `${template}\n\n${args}`
                : template;
            await ctx.session.prompt({ sessionID: invocation.sessionID, text });
          },
        });
      }
    });

    ctx.skill.transform((editor) => {
      for (const skill of skills) {
        try {
          editor.add(skill);
        } catch {}
      }
    });

    return () => void 0;
  },
});
