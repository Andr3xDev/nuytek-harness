---
name: md-style-guide
description: Apply personal markdown formatting standards when producing any document, report, planning file, or written artifact in markdown. Trigger whenever the user requests a document, guide, report, planning file, README, or any structured markdown output. Also trigger when the user asks to format, clean up, or rewrite an existing markdown file. This skill governs visual structure, typography rules, table formatting, and separator conventions — ensuring all output is clean, readable, and consistent across all agents and contexts.
---
# MD Style Guide

Defines the personal markdown formatting standard for all written artifacts.
Applies to any document, report, planning file, or structured output in markdown.

---

## Title and Header Structure

- The document title uses `#` — only one per document, always at the top
- The title is centered using HTML: `<h1 align="center">Title</h1>`
- Immediately below the title: either a subtitle or shield.io badges — never both
  - Badges: only when explicitly requested
  - Subtitle: plain text, no formatting, centered: `<p align="center">Subtitle here</p>`
- All other sections use `##`, `###`, `####` — never skip levels
- Section headers are plain text — no bold, no italics, no inline code unless the header is a term

<br>

## Separators and Spacing

- Never use `---` as a horizontal rule anywhere in the document
- Use `<br>` to create visual breathing room between sections
- Use two consecutive `<br>` tags on separate lines to separate major sections that need stronger visual separation
- Blank lines between paragraphs — do not crowd content

<br>

## Typography

- **Bold** only when the term or phrase has real relevance in context — not for decoration
- *Italics* for emphasis on a specific word when tone requires it — use sparingly
- Never bold entire sentences or paragraph openers as a habit
- Never use bold and italics together unless absolutely necessary
- Inline `code` for technical terms, commands, file paths, and values — not for emphasis

<br>

## Bullets and Lists

- Use `-` for all bullet points — never `*`
- Nested bullets only when hierarchy genuinely exists in the content
- Do not bullet-ize content that reads better as prose
- Ordered lists (`1.`) only when sequence or priority matters

<br>

## Tables

- Column headers in **bold**
- Header row content: left-aligned labels, centered if the column is a label or category
- Data rows: left-aligned by default, centered when the column contains short categorical values
- Keep tables narrow — split into multiple tables if too many columns reduce readability
- No decoration inside cells — no bold on regular data, no inline code unless it is actual code

Example:

| **Category** | **Rule** | **When** |
|:---|:---|:---:|
| Separators | Use `<br>` | Always |
| Bold | Key terms only | Sparingly |

<br>

## Emojis

- No emojis in body text or headers
- Allowed only as warning or status indicators in specific contexts (e.g., `⚠️`, `✓`, `✗`)
  and only when the document format calls for it explicitly
- Shield.io badges are not emojis — they follow their own rule above

<br>

## General Principles

- Prioritize readability over density — do not pack too much into one section
- If content feels cluttered, break it into subsections or separate paragraphs
- Consistency across the document matters more than local cleverness
- When in doubt: less formatting, more clarity

<br>

## Control Points

- If the document type is ambiguous (report vs. reference vs. planning file), confirm before structuring
- If badges are needed but no badge content was specified, ask before generating them
- Do not apply this style retroactively to existing files without explicit instruction
