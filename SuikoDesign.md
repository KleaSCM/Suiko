# Suiko — Design Document

**Status:** Draft v0.2 · 2026-08-24
**Authors:** KleaSCM 
**Stack:** Go engine · Wails desktop app · Vite + TypeScript + Svelte + Tailwind

---

## 1. Overview

Suiko is an MCP-native roleplay engine. The world lives on disk as plain JSON files.
A keyword index maps aliases → entries. On every turn, matched lore is injected into
context automatically (**push**), and the model can dig deeper through MCP tools
(**pull**). Sessions write back into the world, so canon grows while you play.

The engine has **no UI coupling and no model coupling**:

- The Wails app is *one* frontend — the built-in player.
- The MCP server is *the* interface — any MCP host (opencode, Zed, Claude Desktop…)
  can drive or browse the same world with zero extra work.

**Core bet:** long-context models + structured JSON lore + deterministic keyword
injection + tool-based pull = persistent, consistent RP worlds with no fine-tuning,
no vector DB, no proprietary format. One world per session keeps context pressure
low — exactly what modern models are good at.

---

## 2. Why MCP Is The Key

| Without MCP | With MCP |
|---|---|
| Lore injection is passive; model only knows what we guess it needs | Model has **agency**: pulls lore mid-scene when *it* decides it's relevant |
| Engine locked to one chat UI | Any MCP client becomes a frontend — desktop app is optional |
| Custom protocol to invent, document, maintain | Standard protocol: tools/resources discovered automatically |
| Worlds trapped inside the app | `suiko serve` exposes a world to *anything* |

Two retrieval paths working together is the design centerpiece:

1. **Push (deterministic):** user message → alias scan → ranked injection.
   Works even if the model never calls a single tool.
2. **Pull (agentic):** MCP tools let the model search/read/browse on its own.
   This is what makes scenes feel like the world has depth beyond the last message.

---

## 3. Architecture

```
┌────────────────────────────── Suiko (Go) ──────────────────────────────┐
│                                                                        │
│  worlds/<name>/            WorldStore          KeywordIndex            │
│  ├─ world.json      ──▶    load, validate,     map[alias][]EntryRef   │
│  ├─ canon.json             watch fs                   ▲                │
│  ├─ player.json                               match │ rank           │
│  ├─ characters/*.json                         score │ budget         │
│  ├─ locations/*.json                                │                │
│  └─ events/                                   Injector               │
│                                                      │                │
│  McpServer ◀──────── tools/resources ───────┐        ▼                │
│  (stdio, later http)                        │   PromptCompiler       │
│                                             │   + Guardrails         │
│  Provider (chat completions, SSE) ◀────────┴── Session               │
│                                                                        │
│  Wails App ── bindings/events ──▶ Svelte UI (play, browse, edit)      │
└────────────────────────────────────────────────────────────────────────┘

External clients (opencode, Zed, Claude Desktop)
        │ stdio / http
        ▼
   McpServer ──▶ same WorldStore, same index, same guardrails
```

| Component | Responsibility |
|---|---|
| `WorldStore` | Load, validate, watch JSON files; single source of truth |
| `KeywordIndex` | Alias → entry refs; built at load, rebuilt on file change |
| `Injector` | Scan turn input, score matches, render lore block under token budget |
| `PromptCompiler` | Assemble system prompt from tiers (§7) |
| `Guardrail` | Player-character sovereignty enforcement (§9) |
| `Session` | Turn loop: inject → provider call → stream → persist transcript |
| `Provider` | Chat-completions client with streaming; backends: opencode server (§12) or direct OpenAI-compatible |
| `McpServer` | Tools + resources over stdio (http/sse later) |
| `Wails App` | Desktop shell; binds engine methods, emits stream events |

---

## 4. World Directory Layout

One directory = one world. Plain JSON, git-friendly, hand-editable.

```
worlds/
	<world-name>/
		world.json            # manifest: identity + tuning defaults
		canon.json            # permanent core lore — always in context
		player.json           # THE player character — sovereign (§9)
		characters/*.json     # NPCs
		locations/*.json
		items/*.json
		factions/*.json
		lore/*.json           # free-form topics: magic system, history…
		events/               # append-only session history (write-back)
			2026-08-23-session-01.jsonl
```

### 4.1 `world.json`

```json
{
	"name": "<world-name>",
	"description": "A short pitch of the world — shown to the model every turn.",
	"starting_scene": "Opening narration for session one.",
	"narrator_rules": [
		"Slow-burn pacing. Never rush emotional beats.",
		"NPCs have their own agendas and continue them off-screen."
	],
	"budget": {
		"inject_max_tokens": 3000,
		"top_k_entries": 8,
		"recency_boost_turns": 20
	},
	"provider": {
		"backend": "openai",
		"base_url": "http://localhost:11434/v1",
		"model": "whatever-fits"
	}
}
```

All fields optional except `name`; missing budgets fall back to engine defaults.

### 4.2 Entry schema (characters, locations, items, factions, lore)

```json
{
	"id": "char/mara",
	"type": "character",
	"name": "Mara",
	"aliases": ["Mara", "the smith", "smith girl"],
	"summary": "Village blacksmith, secretly exiled nobility.",
	"body": "Full lore paragraphs here…",
	"links": ["loc/forge", "lore/exile"],
	"tags": ["craft", "hidden-past"],
	"alias_weight": { "Mara": 0.6 },
	"updated": "2026-08-24T00:00:00Z"
}
```

| Field | Meaning |
|---|---|
| `id` | `<type>/<slug>` — unique, stable, referenced by `links` |
| `aliases` | Keywords triggering injection; case-insensitive; multi-word allowed |
| `summary` | One-liner for compact contexts (search hits, related lists) |
| `body` | Full lore injected when the entry is selected |
| `links` | Graph edges — powers `GetRelated`, traversal, future features |
| `tags` | Free-form grouping/filtering |
| `alias_weight` | Per-alias multiplier (default 1.0) to tame common words; values < 0 suppress injection on that alias |
| `updated` | ISO timestamp; conflict resolution = later write wins |

### 4.3 `canon.json`

Not an entry — a document. Always present in Tier 0. Holds what must *never* be
forgotten: world laws, tone, hard facts. Keep it tight (< ~2k tokens); if it grows
past that, the content should become entries instead.

### 4.4 `events/*.jsonl`

Append-only JSON lines written during sessions:

```json
{"t":"2026-08-23T22:14:03Z","turn":41,"kind":"scene","text":"Mara confessed she's been feeding the forge-fire with old sigils.","participants":["char/mara","loc/forge"],"location":"loc/forge"}
```

Events power recency boosts and "previously on…" digests. Never edited, only
appended — history is load-bearing.

### 4.5 Scene state (derived, not authored)

There is no `scene.json` — scene state is **derived** from events so it can never
contradict history:

```
SceneState {
	now          time    # world clock: advances one turn per user message
	location     string  # last event with kind "scene" or "move"
	present      []id    # participants of recent events at current location
	open_threads []text  # events flagged kind "thread" not yet closed by kind "resolution"
}
```

- The session derives this incrementally while replaying today's event log; no
  full rescan per turn.
- NPCs continuing agendas off-screen (narrator rules) is modeled as a world tick:
  when the scene location changes or N turns pass, the engine emits an
  `"offscreen"` event summarizing what absent NPCs did — written by the model via
  `LogEvent`, prompted by Tier 0 ("NPCs you left behind kept living; mention it").
- `GetScene` returns the derived struct; Tier 0 renders a compact form of it.

---

## 5. Retrieval — Push Path

Deterministic; runs every turn before the provider call:

1. **Normalize** the user message: lowercase, strip punctuation, collapse spaces.
   For CJK text (Japanese, Chinese), word-boundary normalization does not apply —
   instead the scanner emits fixed-width **character bigrams** over CJK runs and
   indexes aliases as bigrams too (`iron sigil` → `iron`, `sigil`; `紅茶` → `紅茶`).
   Mixed-script messages scan both paths; results merge into one candidate set.
2. **N-gram scan** up to the longest alias word-count (cap 4). Longest-match-first,
   so `"iron sigil"` wins over bare `"sigil"`.
3. **Alias lookup** in the index (`map[string][]EntryRef`) — O(1) per n-gram.
4. **Score** each candidate entry:
   ```
   Score = Σ over matched aliases of:
       alias_weight        (default 1.0)
     × tier_bonus          multi-word 2.0 · exact word 1.5 · prefix 0.75 (len ≥ 4)
     × recency_boost       +0.5 if entry appears in recent-events window
     × link_bonus          +0.25 per already-matched linked neighbor
   ```
   Aliases marked in a world-level stopword list are skipped entirely. Worlds may
   also set per-alias negative weights (< 0 suppresses injection on that alias) —
   this tames common words like "line" without deleting them from lore.
5. **Dedup**: skip entries injected too recently. The session tracks, per entry,
   the turn number of its last injection; an entry is skipped if it fired within
   the last N turns (N = `dedup_window_turns`, default 10). This survives Tier 3
   compression — the engine remembers injections even after old turns are digested.
6. **Budget fill**: sort by score, take top-K, pack until `inject_max_tokens`.
   Token counting is abstracted behind a `TokenCounter` interface on the Provider;
   the default implementation estimates `bytes / 3` (conservative for mixed EN/JA),
   but a provider-backed tokenizer can replace it per-model without touching the
   injector.
7. **Render** as a fenced `[LORE]` block: id, type, summary, body per entry.
   The UI shows which entries fired — injection is transparent, not magic.

Failure-mode honesty: keyword matching misses paraphrases. Acceptable for v1 —
the pull path covers gaps because the model can go looking.

---

## 6. Retrieval — Pull Path (MCP Surface)

Tools exposed by `suiko`, identical for the built-in app and external hosts:

| Tool | Signature | Purpose |
|---|---|---|
| `SearchWorld` | `(query, type?) → [{id,type,name,summary}]` | Keyword search over index. Cheap first stop |
| `GetEntry` | `(id) → {full entry}` | Deliberate deep-read of one entry |
| `GetRelated` | `(id, depth?) → [{id,summary}]` | Walk `links` graph, summaries only |
| `RecentEvents` | `(limit?, participant?) → [event]` | "What happened lately" |
| `AddEntry` | `(type, name, aliases[], summary, body) → {id}` | Write-back: new canon (player type blocked — §9) |
| `UpdateEntry` | `(id, patch) → {updated}` | Amend entry (**refuses sovereign ids**) |
| `LogEvent` | `(text, participants[], location?) → ok` | Append to today's event log |
| `GetScene` | `() → scene state` | Current scene, present characters, open threads |

Resources (browsable without tool calls):

| URI | Content |
|---|---|
| `suiko://world/tree` | Full entry tree grouped by type |
| `suiko://entry/{id}` | Raw entry JSON |
| `suiko://canon` | Canon document |
| `suiko://events/today` | Today's event log |

Tool results are compact by default (`summary`, not `body`) so a curious model can
scan widely without blowing context; `GetEntry` is the deep read.

---

## 7. Context Tiers

Every request is assembled from four tiers:

| Tier | Content | When | Budget |
|---|---|---|---|
| 0 — Always | Narrator contract + sovereignty rules, world blurb, canon.json, PC identity card, scene state | Every request | ~3–4k |
| 1 — Injected | Keyword matches of this turn (push path) | Every request, ranked | `inject_max_tokens` |
| 2 — Pulled | Tool results the model fetched mid-turn | On demand | Model's choice |
| 3 — History | Rolling transcript; oldest turns compress to digest lines | Every request | Configurable |

Tier 3 compression is **deterministic, not model-generated**: when the transcript
passes its cap, the oldest turns collapse into digest lines extracted from events
(participants, kind, first sentence). No provider call mid-turn — no latency, no
cost, no risk of the model rewriting history. Events captured what matters;
history is convenience, not canon. A richer abstractive "story so far" can be a
session-end option (§8), not an in-loop step.

---

## 8. Write-Back — Worlds That Grow

The difference between a museum and a world is that play changes it.

- During play the model may call `AddEntry` / `UpdateEntry` / `LogEvent`.
- Writes land on disk immediately (atomic: temp file + rename); fs watcher rebuilds
  the index, so external editors stay in sync too.
- **Human veto by default:** model writes enter a pending queue in the UI
  (accept / edit / reject). `auto_accept_writes: true` flips this for fully
  agentic growth.
- Accepted writes go through the **same atomic path as direct edits** (temp file
  + rename, `updated` bumped at write time), so pending-queue accepts and manual
  editor saves behave identically to the fs watcher. The UI's entry editor warns
  before overwriting a file whose `updated` changed since it was loaded.
- Session end: optional digest summarizes the session into a few event lines —
  "previously on" material for next time.

Conflict policy: last-write-wins per field via `updated`. Deep reconciliation is a
human job — files are right there, git handles the archaeology.

Canon overflow: if `canon.json` exceeds ~2k tokens, `suiko validate` emits a
warning (not an error) listing oversized sections; the PromptCompiler includes it
in full regardless — canon is never silently truncated.

---

## 9. Player Character Sovereignty — Hard Guardrails

**Rule zero: the AI NEVER controls the player character. Ever.**
Enforced in layers — prompt alone is never trusted.

### Layer 1 — Schema
- Exactly one `player.json` per world with `sovereign: true`.
- Validator rejects any other file carrying `sovereign: true`.

### Layer 2 — Prompt contract (engine-generated, always Tier 0)

```
You are the Narrator. You control the world and every NPC.
You NEVER control {PC.name}. Never write {PC.name}'s actions, dialogue,
thoughts, feelings, or decisions. If asked to, refuse in-fiction and wait.
The human's messages ARE {PC.name}'s actions and words.
```

Generated by the engine from `player.json` — not editable per-world, so world
content can't weaken it.

### Layer 3 — Injection policy
- The PC card enters context as **identity only**: name, appearance, how others
  perceive them. Secrets/goals/inner-life stay out unless the player reveals them —
  the narrator shouldn't know what the player hasn't shown.

### Layer 4 — Tool lockout
- `UpdateEntry` / `AddEntry` hard-refuse any id resolving to the sovereign entry;
  refusal returns a structured result ("sovereign — player-owned") the model reads.
- No tool exists that can append to the transcript as the player.

### Layer 5 — Output advisory filter (v1.x)
- Post-process model output for quoted PC dialogue / action verbs near the PC's
  name or aliases. Inherently unreliable → **advisory only**: flags the message in
  the UI ("possible PC control detected"), offers one-click regenerate. Never silent.

Turn contract — the whole game in one sentence:
**user message = what the PC says/does · assistant message = everything else.**

---

## 10. Frontend — Wails + Svelte

Desktop shell wrapping the Go engine; same process, no REST layer between them.

| View | Contents |
|---|---|
| Play | Chat stream with streaming tokens, inline lore cards showing which entries fired this turn, "dig deeper" affordance |
| World | Tree browser by type, fuzzy filter, entry editor (form fields + raw JSON toggle), link editor with autocomplete |
| Sessions | Session list, event timeline, digest generation, resume |
| Settings | Provider config (base URL/key/model), budgets, auto-accept writes, MCP server toggle |

Implementation notes:

- Wails v3 preferred for modern bindings; fallback to v2 stable if v3 blocks us —
  the binding surface used is small either way.
- Svelte 5 runes, TypeScript strict, Tailwind v4, Vite build.
- Streaming: goroutine reads SSE from provider → `EventsEmit("token", …)` →
  Svelte store appends. Trivial backpressure at chat rates.
- State: one Svelte store mirroring session + selected entry; lore cards derive
  from injection metadata returned with each completed turn.
- The UI talks to the engine **only through operations MCP also exposes** —
  the MCP surface is the API, so behavior is identical in any frontend.

---

## 11. Go Package Layout

```
suiko/
	cmd/
		suiko/          # wails app entry + CLI verbs (serve, validate)
	internal/
		world/          # Store, Loader, Validator, Watcher, Index
		inject/         # Matcher, Scorer, Budgeter, Renderer
		narrate/        # PromptCompiler, tier assembly, digests
		guard/          # Sovereignty checks shared by tools + compiler
		mcpserver/      # JSON-RPC framing, tool registry, resources
		provider/       # Provider interface, opencode client, OpenAI-compatible client, TokenCounter
		scene/          # Derived scene state from event replay
		session/        # Turn loop, transcript, pending-write queue
	frontend/           # Vite + Svelte + TS + Tailwind
	worlds/             # user worlds (default dir, configurable)
```

---

## 12. Opencode Integration

[Opencode](https://opencode.ai) slots into Suiko on **both sides** of the engine:

```
Svelte UI ──▶ Suiko engine ──▶ opencode serve (HTTP) ──▶ 75+ providers/models
opencode TUI / IDE / any MCP host ──▶ suiko serve (MCP) ──▶ same world
```

### 12.1 Opencode as the provider layer

Suiko's `Provider` is an interface; opencode is a first-class implementation
alongside the plain OpenAI-compatible client:

| Suiko need | Opencode server endpoint |
|---|---|
| List available models | `GET /config/providers` |
| Provider auth incl. OAuth | `GET /provider/auth`, `POST /provider/{id}/oauth/authorize` |
| Create an RP session | `POST /session` |
| Send compiled turn | `POST /session/:id/message` (`system` = Tier 0+1 prompt, per-message `tools` config) |
| Stream tokens | `GET /event` (SSE bus), filtered by session/message ID |
| Abort turn | `POST /session/:id/abort` |

Why this wins: every provider opencode supports — API keys, OAuth flows, model
catalogs — works in Suiko with zero plumbing. Model switching becomes a dropdown
fed by `/config/providers`.

**Bare-session discipline:** opencode is a coding *agent*, so RP sessions must
strip it down. Per message we pass our compiled prompt via the `system` field and
disable built-in tools (`tools: {}`); no `AGENTS.md`, no project context leaks
into play. The Suiko world is the only context that matters.

### 12.2 Suiko as opencode's MCP server

Already covered by §6: register `suiko serve` as an MCP server in opencode's
config and any opencode client (TUI, IDE, web) can browse and drive a world.
The one-writer lock (§13) applies — an opencode-driven session locks the world,
the desktop app observes.

### 12.3 Configuration

```json
{
	"provider": {
		"backend": "opencode",              // or "openai"
		"server_url": "http://127.0.0.1:4096",
		"provider_id": "anthropic",
		"model_id": "whatever-fits"
	}
}
```

`backend: "openai"` keeps direct chat-completions for users who don't run
opencode; the injector, guardrails, and session loop are identical under both.

---

## 13. Concurrency & Edge Cases

- **One writer per world.** A world directory is locked by a single active
  Session (lock file `worlds/<name>/.suiko-lock`); a second session (desktop app
  *or* external MCP host) attaches read-only and the UI shows "observing". This
  keeps write-back deterministic without distributed coordination.
- **Stream cancellation.** Aborting a turn cancels the provider request context;
  partial output is discarded (not appended to the transcript), and any pending
  writes from that turn are dropped from the queue. The injection metadata of a
  cancelled turn is not recorded — dedup counters don't advance.
- **Watcher debounce.** fs events are debounced (~200ms) before index rebuild so
  editor save-storms don't thrash; rebuilds are atomic under a read-write lock.
- **Malformed files.** The validator isolates bad entries: an invalid JSON file is
  skipped with a warning surfaced in the UI and `suiko validate`, never crashing
  the store. The rest of the world stays live.
