#!/usr/bin/env node
/**
 * release-notes.mjs — generate professional GitHub release notes via AI.
 *
 * Usage:
 *   node release-notes.mjs <tag> <prevTag> <changelog> <apiKey>
 *
 * Prints markdown to stdout. Uses the 9Router OpenAI-compatible endpoint.
 */
const [, , tag, prevTag, changelog, apiKey] = process.argv;

const AI_BASE = process.env.AI_BASE || "https://ai.jefripunza.com/v1";
const AI_MODEL = process.env.AI_MODEL || "hermes";

if (!apiKey) {
  console.error("[release-notes] no AI_API_KEY — emitting raw changelog");
  console.log(`## What Changed\n\n${changelog}\n`);
  process.exit(0);
}

const system = `You are a professional release engineer writing release notes for Golify, a self-hosted application deployment platform built with Go (Fiber) and Vue 3. Write in a professional, confident tone. Output GitHub-flavored Markdown with these sections:

## Highlights
2-3 sentence executive summary of what this release delivers.

## What's New
Bullets for new features, each with a short user-value explanation.

## Improvements
Bullets for enhancements, fixes, and internal changes.

## Breaking Changes (only if any)
Omit this section entirely when there are no breaking changes.

Keep bullets concise (under 20 words). Use present tense. Do NOT invent features absent from the changelog. Respond ONLY with the markdown body — no preamble, no code fences.`;

const user = `Release tag: ${tag || "?"}
Previous tag: ${prevTag || "(first release)"}

Changelog (commit hash + subject):
${changelog || "(empty)"}

Write the release notes markdown now.`;

async function main() {
  const res = await fetch(`${AI_BASE}/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: AI_MODEL,
      messages: [
        { role: "system", content: system },
        { role: "user", content: user },
      ],
      temperature: 0.4,
      max_tokens: 1200,
    }),
  });

  if (!res.ok) {
    const err = await res.text().catch(() => "");
    console.error(`[release-notes] AI HTTP ${res.status}: ${err.slice(0, 300)}`);
    console.log(`## What Changed\n\n${changelog}\n`);
    process.exit(0);
  }

  const data = await res.json();
  const notes = data?.choices?.[0]?.message?.content?.trim() || "";
  if (!notes) {
    console.error("[release-notes] AI returned empty content");
    console.log(`## What Changed\n\n${changelog}\n`);
    process.exit(0);
  }
  console.log(notes);
}

main().catch((e) => {
  console.error(`[release-notes] ${e.message}`);
  console.log(`## What Changed\n\n${changelog}\n`);
});
