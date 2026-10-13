// caveman — opencode plugin (V2)
//
// Provides dynamic caveman mode tracking for opencode:
// - Writes the mode flag on each session start (via the `event` stream)
// - Parses user messages for /caveman commands and natural-language toggles
// - Injects per-turn reinforcement into the system prompt
//
// ESM module; loads the existing security-hardened helpers from
// caveman-config.cjs via new Function so the symlink-safe flag-write code
// lives in one place. Same trick loads caveman-parse.cjs (#602) so the mode-
// change parsing is a single shared source with caveman-mode-tracker.js.
//
// Layout once installed:
//   opencode/plugins/caveman/
//   ├── package.json
//   ├── plugin.js              ← this file
//   ├── caveman-config.cjs
//   └── caveman-parse.cjs
//
// The always-on caveman ruleset is provided separately via AGENTS.md
// (Tier-3 base). This plugin handles dynamic state only: flag writes,
// slash-command parsing, natural-language activation, and per-turn
// reinforcement.
//
// V2 hook mapping (from https://opencode.ai/v2/docs/build/plugins/migrate-v1):
//   - event (event.type === 'session.created') → ctx.event.subscribe() loop
//   - chat.message → ctx.session.hook("prompt", ...) — runs at prompt
//     admission, before attachment/skill resolution; slash commands submitted
//     through session.prompt pass through it
//   - experimental.chat.system.transform → ctx.session.hook("context", ...)
//     and edit event.system (SystemPart[] of {type:"text", text})

import { Plugin } from "@opencode/plugin";
import { createRequire } from 'node:module';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { dirname, join } from 'node:path';
import { existsSync, unlinkSync, readFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));

// caveman-config.cjs sits next to plugin.js (this directory's package.json
// declares "type": "module", so the helpers are named .cjs to stay CommonJS).
//
// Loaded by evaluating the file as CommonJS by hand, NOT via the module
// loader: keeping the V1 loader untouched, which hand-evaluates the file with
// base require() on the loaded file itself — caveman-parse.js does a relative
// require('./caveman-config') that must resolve against the plugin dir when
// installed. One source of truth either way.
function loadConfig() {
  const installed = join(here, 'caveman-config.cjs');
  const dev = join(here, '..', '..', '..', 'hooks', 'caveman-config.js');
  const target = existsSync(installed) ? installed : dev;
  const code = readFileSync(target, 'utf8').replace(/^#![^\n]*\n/, '');
  const mod = { exports: {} };
  new Function('module', 'exports', 'require', '__dirname', '__filename', code)(
    mod, mod.exports, createRequire(pathToFileURL(target).href), dirname(target), target
  );
  return mod.exports;
}
const config = loadConfig();

const { getDefaultMode, safeWriteFlag, readFlag } = config;

// Resolved defensively, NOT destructured with the three above. loadConfig()
// reads whatever caveman-config.cjs sits in the installed plugin directory,
// which can predate this file (#848). handleSessionCreated() runs at setup
// time, outside any try — so destructuring an absent one would throw during
// plugin construction and take caveman from "mode works, history missing" to
// "plugin does not load at all". The history log is best-effort by design
// (its own body silent-fails), so the no-op stub is the honest fallback.
const recordModeChange = config.recordModeChange || function () {};

// Load the shared mode-change parser (#602) the same way loadConfig() loads
// caveman-config.js — hand-evaluated CommonJS, one loader mechanism shared.
function loadParse() {
  const installed = join(here, 'caveman-parse.cjs');
  const dev = join(here, '..', '..', '..', 'hooks', 'caveman-parse.js');
  const target = existsSync(installed) ? installed : dev;
  const code = readFileSync(target, 'utf8').replace(/^#![^\n]*\n/, '');
  const mod = { exports: {} };
  new Function('module', 'exports', 'require', '__dirname', '__filename', code)(
    mod, mod.exports, createRequire(pathToFileURL(target).href), dirname(target), target
  );
  return mod.exports;
}
const { parseModeChange, INDEPENDENT_MODES } = loadParse();

// opencode resolves its config dir from $XDG_CONFIG_HOME, else ~/.config/opencode
// on every platform — including Windows (see V1 note on homedir semantics).
function opencodeConfigDir() {
  if (process.env.XDG_CONFIG_HOME) {
    return path.join(process.env.XDG_CONFIG_HOME, 'opencode');
  }
  return path.join(os.homedir(), '.config', 'opencode');
}

const opencodeDir = opencodeConfigDir();
const flagPath = path.join(opencodeDir, '.caveman-active');

function removeFlag() {
  try {
    unlinkSync(flagPath);
  } catch (error) {
    if (process.env.CAVEMAN_DEBUG === '1' && error.code !== 'ENOENT') {
      console.error(`caveman: failed to remove flag ${flagPath}: ${error.message}`);
    }
  }
}

function reinforcementBanner(mode) {
  return 'CAVEMAN MODE ACTIVE (' + mode + ') — session ruleset applies.';
}

function escapeRegExp(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// Derived from reinforcementBanner() itself (split on a sentinel) rather than
// re-spelling the banner text as a second regex literal: one source of truth,
// and it stays in sync if the wording above ever changes.
const [bannerPrefix, bannerSuffix] = reinforcementBanner('\0').split('\0');
const staleBlock = new RegExp(
  escapeRegExp(bannerPrefix) + '[a-z-]+' + escapeRegExp(bannerSuffix) + '[\\s\\S]*$'
);

// SKILL.md is the single source of truth for caveman behavior, filtered to the
// active level the same way caveman-activate.js and caveman-mode-tracker.js do.
// The filter itself is NOT re-implemented here: it lives in caveman-config.js,
// which loadConfig() already evaluates, so all three loaders share one copy of
// the intensity-table parsing.
//
// Resolved off `config` rather than destructured at module scope because the
// installed caveman-config.cjs is a COPY: a stale copy without these exports
// degrades to the banner alone rather than throwing inside a system-prompt
// hook.
function loadFilteredRuleset(mode) {
  if (typeof config.loadFilteredRuleset !== 'function') return null;
  // The shared loader probes <base>/../../skills and <base>/../skills. Called
  // once per base — `here` resolves the installed tree
  // (opencode/plugins/caveman → opencode/skills) and the parent covers the
  // dev tree layout (plugins/ → repo-root skills).
  for (const base of [here, join(here, '..')]) {
    const ruleset = config.loadFilteredRuleset(mode, base);
    if (ruleset) return ruleset;
  }
  return null;
}

function reinforcementLine(mode) {
  const banner = reinforcementBanner(mode);
  const ruleset = loadFilteredRuleset(mode);
  // No SKILL.md reachable (a standalone plugin install without the skills
  // dir): fall back to the banner alone, the same degrade caveman-activate.js
  // uses for the same case.
  return ruleset ? banner + '\n\n' + ruleset : banner;
}

function applyModeChange(change) {
  if (!change) return;
  if (change.action === 'clear') {
    recordModeChange(opencodeDir, null);
    removeFlag();
    return;
  }
  if (change.action === 'set' && change.mode) {
    recordModeChange(opencodeDir, change.mode);
    safeWriteFlag(flagPath, change.mode);
  }
}

// Session-start logic — extracted so the V2 event stream drives one shared
// implementation. Re-fires on every `session.created` event, so a new session
// in a long-lived plugin process re-asserts the flag.
function handleSessionCreated() {
  const mode = getDefaultMode();
  if (mode === 'off') {
    recordModeChange(opencodeDir, null);
    removeFlag();
    return;
  }
  recordModeChange(opencodeDir, mode);
  safeWriteFlag(flagPath, mode);
}

export default Plugin.define({
  id: 'caveman',
  setup(ctx) {
    // Assert the flag at plugin load as well: in one-shot `opencode run` the
    // first session.created may publish before event dispatch is wired up, so
    // the event subscription alone misses it. The setup-time write covers
    // that race; the subscription re-asserts on every later session.
    handleSessionCreated();

    // V2 events are flat: { type, data: { ... } }.
    const controller = new AbortController();
    void (async () => {
      try {
        for await (const event of ctx.event.subscribe({ signal: controller.signal })) {
          if (event.type === 'session.created') handleSessionCreated();
        }
      } catch {
        // Subscription aborted on plugin unload, or the stream is gone — stop
      }
    })();

    // Intercept user messages to detect /caveman commands and natural-language
    // mode toggles. The V2 prompt hook fires once at prompt admission with the
    // user's text in event.prompt.text (the V1 chat.message hook saw it per
    // message part; V2 collapses to one owned, mutable draft).
    // expandedTpl: parseModeChange recognizes both the raw "/caveman <level>"
    // text AND opencode's expanded command prose ("Activate caveman mode:
    // <level>...") in one call — the expanded-template branch and the literal
    // slash branch are both inside the parser, gated by this option.
    // unwrapQuotes: the non-interactive `run` path delivers the message
    // wrapped in literal quote characters.
    // Return value is ignored — state changes happen via the flag file.
    ctx.session.hook('prompt', (event) => {
      const text = event && event.prompt && event.prompt.text;
      if (!text) return;
      const change = parseModeChange(text, { getDefaultMode, expandedTpl: true, unwrapQuotes: true });
      if (change) applyModeChange(change);
    });

    // Inject the reinforcement line into the system prompt when caveman is
    // active. The V2 context hook fires immediately before every agent-loop
    // model request and expects the hook to mutate event.system (an array of
    // SystemPart objects with .text — V1 mutated an output.system string[]).
    // Applies to the agent loop only; compaction/generate/title are separate
    // hooks and are left alone, as in V1.
    ctx.session.hook('context', (event) => {
      if (!event || !Array.isArray(event.system)) return;
      const active = readFlag(flagPath);
      if (active && !INDEPENDENT_MODES.has(active)) {
        const line = reinforcementLine(active);
        // Idempotent: if the runtime ever reuses the system parts across
        // turns an unguarded append grows the system prompt without bound —
        // silently eating the context window. Rewrite any part we already
        // left instead of stacking another, so a mode switch updates in
        // place rather than accumulating. staleBlock matches to end of
        // part text: `line` now carries the ruleset appended after the
        // banner, and that content is always the last thing this hook
        // writes into an entry, so replacing from the banner on is safe.
        let found = false;
        for (let i = 0; i < event.system.length; i++) {
          const part = event.system[i];
          if (part && typeof part.text === 'string' && staleBlock.test(part.text)) {
            part.text = part.text.replace(staleBlock, line);
            found = true;
          }
        }
        if (found) return;
        if (event.system.length > 0) {
          const last = event.system[event.system.length - 1];
          last.text += '\n\n' + line;
        } else {
          event.system.push({ type: 'text', text: line });
        }
      }
    });

    // Cleanup: stop the event subscription if the plugin unloads. Hook
    // registrations are disposed automatically with the plugin.
    return () => controller.abort();
  },
});
