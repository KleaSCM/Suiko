# Suiko Frontend — Implementation Plan

Full Wails + Svelte 5 + TypeScript + Tailwind v4 frontend for the Suiko MCP roleplay engine.

---

## Background

The backend is a complete Go engine (`internal/`) with a working MCP server, world store, injector,
session loop, and provider abstraction. The design document (§10) defines exactly what the UI needs:

- **Play view** — streaming chat, inline lore cards (which entries fired), "dig deeper" affordance
- **World view** — tree browser by type, fuzzy filter, entry editor (form + raw JSON), link editor
- **Sessions view** — session list, event timeline, digest generation, resume
- **Settings view** — provider config, budgets, auto-accept writes, MCP server toggle

The UI talks to the engine **only through Wails bindings** that mirror the MCP tool surface.

---

## Stack

| Layer | Choice | Reason |
|---|---|---|
| Desktop shell | Wails v2 (stable) | v3 still in flux; v2 well-documented |
| UI framework | Svelte 5 runes | Design doc §10 — runes give fine-grained reactivity without a vdom |
| Language | TypeScript strict | Design doc §10 — strict mode |
| Styling | Tailwind v4 | Design doc §10 — explicit requirement |
| Bundler | Vite | Design doc §10 — explicit requirement |
| Fonts | Google Fonts — Outfit + Noto Sans JP | Premium feel, bilingual support |

---

## Proposed Changes

### Wails scaffolding + Go App struct

#### [NEW] `wails.json`
Top-level Wails manifest. Points at `frontend/` as the Vite app, names the binary `suiko`.

#### [MODIFY] `cmd/suiko/main.go`
Add `suiko desktop` verb that boots the Wails runtime instead of the CLI loop.
The existing `validate` and `serve` verbs are untouched.

#### [NEW] `app.go` (project root)
Wails `App` struct — the bridge between Go engine and Svelte UI.

Bound methods (all callable from TypeScript via `wailsjs/go/`):

| Go method | Maps to | Cutie name |
|---|---|---|
| `LoadWorld(path string)` | `world.Load` | — |
| `GetEntries(worldPath string)` | `Store.Entries()` | `TiltyClaret` |
| `GetEntry(id string)` | index lookup | `YuuKoito` |
| `SearchWorld(query, entryType string)` | injector matcher | `AnisphiaWynnPalettia` |
| `GetRelated(id string, depth int)` | link graph walk | `MiyakoKodama` |
| `GetScene(worldPath string)` | `scene.SumikaIzumino` | — |
| `GetRecentEvents(limit int)` | `world.IliaCoral` | — |
| `SendTurn(userText string)` | `session.YukiFukuzawa` | `KanadeAmou` |
| `AddEntry(type,name,aliases,summary,body)` | `world.WriteEntry` | `ClaireFrancois` |
| `UpdateEntry(id string, patch)` | `world.WriteEntry` | `RaeTaylor` |
| `GetManifest(worldPath string)` | manifest load | — |
| `SaveManifest(manifest)` | manifest write | — |
| `ListWorlds()` | scan `worlds/` | `TomaoSuzumi` |

Streaming events emitted via `runtime.EventsEmit`:
- `"token"` — `{delta: string, turn: int}`
- `"turn-done"` — `{text: string, fired: string[], turn: int}`
- `"write-pending"` — `{entry: Entry}` (pending queue item from model)

---

### Frontend scaffolding

#### [NEW] `frontend/` (Vite + Svelte 5 + Tailwind v4 + TypeScript)

```
frontend/
  index.html
  vite.config.ts
  svelte.config.ts
  tailwind.config.ts         ← Tailwind v4
  tsconfig.json
  src/
    app.css                  ← design tokens, base styles, Tailwind imports
    App.svelte               ← root: nav + router
    lib/
      types.ts               ← TS mirrors of Go structs (Entry, Manifest, Event, SceneState, TurnResult)
      bindings.ts            ← typed wrappers around wailsjs/go/main/ auto-generated stubs
      stores/
        WorldStore.svelte.ts ← rune-based reactive world state (entries, manifest, scene)
        SessionStore.svelte.ts ← rune-based turn history, streaming token buffer, fired lore
        PendingStore.svelte.ts ← pending model-write queue
        NavStore.svelte.ts   ← active view, active entry id, filter state
      util/
        EntryFormatting.ts   ← type→icon, type→label, alias formatting helpers (no logic in render)
        TimeFormat.ts        ← RFC3339 → human-readable display
    views/
      PlayView.svelte        ← chat stream + lore cards
      WorldView.svelte       ← tree browser + entry editor
      SessionsView.svelte    ← session list + event timeline
      SettingsView.svelte    ← provider + budget + toggles
    components/
      NavBar.svelte          ← left sidebar navigation
      ChatBubble.svelte      ← single user or narrator message
      LoreCard.svelte        ← fired-entry card (inline in chat)
      EntryTree.svelte       ← tree browser panel (WorldView)
      EntryEditor.svelte     ← form + raw-JSON toggle (WorldView)
      LinkEditor.svelte      ← autocomplete link picker
      EventTimeline.svelte   ← chronological event list (SessionsView)
      PendingReviewPanel.svelte ← pending model writes (accept/edit/reject)
      ProviderForm.svelte    ← provider config inputs (SettingsView)
      BudgetForm.svelte      ← budget sliders/inputs (SettingsView)
      WorldSelector.svelte   ← world-picker modal on first launch
      TokenStream.svelte     ← live streaming token display (Play)
      DigestCard.svelte      ← compressed history digest display
```

---

### Per-view design notes

#### Play view
- Left panel: session metadata (turn count, active world, PC name).
- Centre: scrollable message list. Each assistant turn has a "Lore fired this turn" expandable section
  showing `LoreCard` components for each entry id in `fired[]`.
- Bottom: text input with send button + abort button during streaming.
- Streaming: `runtime.EventsOn("token", …)` appends to a live buffer; `runtime.EventsOn("turn-done", …)` finalises.
- PC sovereignty advisory: if `turn-done` carries `advisoryFlag: true`, show a banner with one-click regenerate.
- "Dig deeper" affordance on each lore card — calls `GetEntry(id)` and opens the entry in a side panel.

#### World view
- Left panel: `EntryTree` grouped by type (player, character, location, item, faction, lore), with fuzzy text filter.
- Right panel: `EntryEditor` — two tabs: **Form** (name, aliases, summary, body, tags, links) and **Raw JSON**.
  Both are fully editable; switching tabs round-trips through the typed `Entry` shape.
- `LinkEditor` autocompletes from the full entry list (id + name).
- New entry button at the top of each type section.
- Pending-queue badge on the World view icon when there are pending model writes.

#### Sessions view
- Top: session file list (events/ directory, one item per `.jsonl` file).
- Middle: `EventTimeline` for the selected session — chronological cards with kind badge, participants, text.
- Bottom: "Generate digest" button (calls `GetRecentEvents` + formats).
- "Resume" button loads the world and opens Play view with the session's history pre-loaded.

#### Settings view
- Provider section: `ProviderForm` (backend selector, URL, model, API key).
- Budget section: `BudgetForm` (token budget, top-K, recency window, dedup window).
- Auto-accept writes toggle.
- MCP server status (read-only: is stdio server running?).
- World directory path picker.

---

### Design system

**Palette — dark-first:**
```
Background:   hsl(225, 15%, 9%)    ← near-black blue-grey
Surface:      hsl(225, 12%, 14%)   ← card/panel base
Surface-alt:  hsl(225, 10%, 18%)   ← raised elements
Border:       hsl(225, 10%, 22%)
Text-primary: hsl(220, 20%, 92%)
Text-muted:   hsl(220, 10%, 55%)
Accent:       hsl(270, 70%, 68%)   ← violet — matches Suiko's mythological feel
Accent-dim:   hsl(270, 40%, 30%)
Fired:        hsl(48, 90%, 60%)    ← warm gold for lore-injection highlight
Warning:      hsl(30, 90%, 58%)
Danger:       hsl(355, 75%, 58%)
```

**Typography:**
- Heading: `Outfit`, weight 600/700
- Body: `Outfit`, weight 400
- Japanese: `Noto Sans JP` fallback
- Monospace (JSON editor): `JetBrains Mono`

**Animation:** 150ms ease transitions on interactive elements; smooth scroll in chat; streaming text
uses `requestAnimationFrame` batching to avoid layout thrash on rapid token deltas.

---

### Cutie name assignments (frontend)

Unticked names used as TypeScript function/store identifiers (file-static helpers):

| Identifier | Used as | Cutie |
|---|---|---|
| `TiltyClaret` | Svelte store action: loads all entries | Tilty Claret |
| `YuuKoito` | lookup helper: id → Entry from store | Yuu Koito |
| `AnisphiaWynnPalettia` | fuzzy-filter function for EntryTree | Anisphia Wynn Palettia |
| `MiyakoKodama` | walks the link graph for "dig deeper" | Miyako Kodama |
| `TomaoSuzumi` | scans worlds/ directory listing | Tamao Suzumi |
| `ClaireFrancois` | wraps AddEntry binding call | Claire François |
| `RaeTaylor` | wraps UpdateEntry binding call | Rae Taylor |

These are file-static helpers only. Public store exports and component props use descriptive PascalCase.

---

## Verification Plan

### Automated
- `npm run build` (Vite) — zero TS errors, zero Tailwind warnings.
- `wails build` — binary links and runs on Linux.

### Manual
- Open the app with no world → world selector modal appears.
- Load a world → Play view renders, manifest name in sidebar.
- Type a message → streaming tokens appear, turn-done fires lore cards.
- Switch to World view → entry tree, fuzzy filter, form/JSON toggle all work.
- Sessions view → event timeline renders for a `.jsonl` file.
- Settings view → changing provider backend and saving persists to `world.json`.
