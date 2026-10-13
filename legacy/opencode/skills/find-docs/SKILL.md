---
name: find-docs
description: Retrieves current documentation via Context7 MCP for libraries, frameworks, SDKs, APIs, CLIs, and cloud services. Use for API syntax, setup, config, migrations, and library-specific debugging.
---

# Documentation Lookup

Use Context7 MCP, not CLI, when current docs matter.

## Use When

- User asks about library/framework/SDK/API/CLI/cloud service.
- API syntax, config, setup, migration, or version-specific behavior matters.
- Debugging depends on library behavior.

## Skip When

- Refactoring local code.
- Writing scripts from scratch with stdlib knowledge.
- Debugging business logic.
- Code review without library/API uncertainty.
- General programming concepts.

## Workflow

1. Resolve library using Context7 MCP with official name and user's full question.
2. Pick best `/org/project` match by exact name, relevance, source reputation, snippet count, and benchmark score.
3. Fetch docs using Context7 MCP for selected ID and question.
4. Answer from fetched docs.

Use `/org/project/version` when user names a version and Context7 offers one.
Never include secrets in queries.

## Query Quality

| Quality | Example |
|---------|---------|
| Good | `"How to set up authentication with JWT in Express.js"` |
| Bad | `"auth"` / `"hooks"` |

Use the full question, not a single vague word.

## Failure

If Context7 quota/auth fails: tell the user, do not silently fall back to training data, and ask whether to continue from training knowledge (note it may be outdated).
