// notifier-shim — V2 bridge for the V1-only @mohak34/opencode-notifier plugin.
//
// The package's notify/sound/bell/focus/config internals (osascript, node-notifier,
// notify-send, paplay chain, KDE/Hyprland focus jump, permission dedupe, idle
// debounce, sub-agent tracking) are all closures inside dist/index.js and are far
// too deep to rewrite faithfully, so this shim BRIDGES the package's exported
// `NotifierPlugin({ client, directory })` V1 factory:
//
//   event (session.created/updated/deleted, permission.asked, session.idle,
//          session.status, session.error, message.updated)
//     → ctx.event.subscribe + V1-shape event adapter
//   permission.ask → ctx.permission path via the permission.asked feed (V1
//     also deduped it against the event stream; both are fed here and the
//     package's own 1s dedupe suppresses the duplicate)
//   tool.execute.before (question / plan_exit tools)
//     → ctx.tool.hook("execute.before", { tool: event.tool })
//
// Its own client calls are adapted to the V2 session domain. Config
// (opencode-notifier.json) and state (opencode-notifier-state.json) keep their
// original formats and lookup rules.

import { Plugin } from "@opencode/plugin";
import { createRequire } from "module";
import { existsSync, readFileSync, writeFileSync } from "fs";
import { homedir } from "os";
import { dirname, isAbsolute, join, resolve } from "path";
import { pathToFileURL } from "url";

// V2 events are flat under `data` (V1 nested under `properties`, and session
// lifecycle info under `properties.info`). Renames applied for the bridge:
//   session.execution.failed → session.error (error.type → error.name)
//   session.execution.interrupted → session.error (name "MessageAbortedError")
//   session.inbox.delivered → message.updated (role "user")
// V1-only hooks there is no V2 event for: session.updated (only used for
// sub-agent detection; the V2 session.created parentID covers it), and the
// `session.status` non-busy branches (notifier ignores them anyway).
export function adaptEvent(event) {
  const d = event?.data ?? {};
  switch (event?.type) {
    case "session.created":
      return {
        type: "session.created",
        properties: { info: { id: d.sessionID, parentID: d.parentID, title: d.title } },
      };
    case "session.deleted":
      return {
        type: "session.deleted",
        properties: { info: { id: d.sessionID, parentID: d.parentID, title: d.title } },
      };
    case "session.idle":
      return { type: "session.idle", properties: { sessionID: d.sessionID } };
    case "session.status":
      if (d.status?.type !== "busy") return null;
      return {
        type: "session.status",
        properties: { sessionID: d.sessionID, status: { type: "busy" } },
      };
    case "permission.asked":
      return { type: "permission.asked", properties: { sessionID: d.sessionID } };
    case "session.execution.failed":
      return {
        type: "session.error",
        properties: { sessionID: d.sessionID, error: { name: d.error?.type } },
      };
    case "session.execution.interrupted":
      return {
        type: "session.error",
        properties: { sessionID: d.sessionID, error: { name: "MessageAbortedError" } },
      };
    case "session.inbox.delivered":
      return {
        type: "message.updated",
        properties: { info: { role: "user", sessionID: d.sessionID } },
      };
    default:
      return null;
  }
}

const defaultConfigPath = join(homedir(), ".config", "opencode", "opencode-notifier.json");
export function resolveConfigPath(configDirectory) {
  // Keep the package's own lookup rules; only fall back to this config
  // directory's file when the shipped default path has no config (repo layout).
  if (process.env.OPENCODE_NOTIFIER_CONFIG_PATH) return process.env.OPENCODE_NOTIFIER_CONFIG_PATH;
  if (existsSync(defaultConfigPath)) return defaultConfigPath;
  const local = join(configDirectory, "opencode-notifier.json");
  if (existsSync(local)) return local;
  return defaultConfigPath;
}

export default Plugin.define({
  id: "opencode-notifier",
  async setup(ctx) {
    const require = createRequire(import.meta.url);
    const root = require.resolve("@mohak34/opencode-notifier/package.json");
    const notifier = await import(pathToFileURL(join(root, "..", "dist", "index.js")).href);

    // Point the package's config/state lookup at this config directory when
    // its default path has nothing (works for both repo-local and deployed
    // ~/.config/opencode layouts without changing the file formats).
    if (!process.env.OPENCODE_NOTIFIER_CONFIG_PATH && !existsSync(defaultConfigPath)) {
      const local = join(ctx.location.directory, "opencode-notifier.json");
      if (existsSync(local)) process.env.OPENCODE_NOTIFIER_CONFIG_PATH = local;
    }

    // The package plays custom sound paths verbatim via existsSync(), so
    // config-relative paths ("./sounds/…") resolve against the process cwd.
    // When the effective config uses relative sound paths, materialize a sibling
    // copy with them resolved against the config dir and point the package at it
    // (same directory → state file location is unchanged).
    const effectiveConfigPath = process.env.OPENCODE_NOTIFIER_CONFIG_PATH ?? defaultConfigPath;
    if (existsSync(effectiveConfigPath)) {
      try {
        const table = JSON.parse(readFileSync(effectiveConfigPath, "utf-8"));
        const sounds = table?.sounds;
        if (sounds && typeof sounds === "object") {
          const configDir = dirname(effectiveConfigPath);
          let touched = false;
          for (const [key, value] of Object.entries(sounds)) {
            if (typeof value === "string" && value && !isAbsolute(value)) {
              sounds[key] = resolve(configDir, value);
              touched = true;
            }
          }
          if (touched) {
            const resolvedPath = join(configDir, "opencode-notifier.resolved.json");
            writeFileSync(resolvedPath, JSON.stringify(table, null, 2) + "\n");
            process.env.OPENCODE_NOTIFIER_CONFIG_PATH = resolvedPath;
          }
        }
      } catch {
        // Unreadable/malformed config: fall through, let the package handle it
      }
    }

    // V1 client (client.session.get/messages) adapter over the V2 session domain.
    const client = {
      session: {
        get: async ({ path }) => ({ data: await ctx.session.get({ sessionID: path.id }) }),
        messages: async ({ path }) => {
          const messages = await ctx.session.context({ sessionID: path.id });
          return {
            data: messages.map((m) => ({
              info: { ...m, role: m.type, time: { created: m.time?.created } },
            })),
          };
        },
      },
    };

    const v1 = notifier.NotifierPlugin({
      client,
      directory: ctx.location.directory,
    });

    const controller = new AbortController();
    const consume = async () => {
      try {
        for await (const event of ctx.event.subscribe({ signal: controller.signal })) {
          const shimmed = adaptEvent(event);
          if (!shimmed) continue;
          await v1.event({ event: shimmed });
          // The V1 permission.ask hook also ran alongside the stream for
          // promptless asks; the package dedupes, so both callbacks per
          // permission.asked event reproduce the V1 combined behavior.
          if (shimmed.type === "permission.asked" && v1["permission.ask"]) {
            await v1["permission.ask"]();
          }
        }
      } catch {
        // Subscription aborted on plugin unload — stop quietly
      }
    };
    void consume();

    const toolRegistration = await ctx.tool.hook("execute.before", (event) => {
      if (v1["tool.execute.before"]) {
        return v1["tool.execute.before"]({ tool: event.tool });
      }
    });

    // Hook/transform registrations are plugin-scoped and auto-disposed; the
    // event subscription needs the explicit abort.
    return () => {
      controller.abort();
      toolRegistration?.dispose?.();
    };
  },
});
