# Mikoko — Developer Conventions

## Coding conventions (KleaSCM style)
- C++23, no exceptions/RTTI, arena allocation, tabs, PascalCase, K&R.
- Arena / zero-is-valid are the *default style* — do NOT reference "ZII", "cuties",
  or character names in code comments, file headers, or docs. Comments describe
  behaviour, not style labels.
- Cutie names (Cuties.md) may be used ONLY as actual identifiers for file-static /
  limited-scope helpers (e.g. `AmaneNormalizeScale`, `JuriArisugawaInverse4x4`).
  Never explain the naming in comments.
- Keep the KleaSCM annotation tags in comments: `MATH(KleaSCM):` (with a real
  equation), `NOTE(KleaSCM):`, `TODO(KleaSCM):`, `REFERENCE(KleaSCM):`,
  `HACK(KleaSCM):`. Follow the format in StyleGuide.md §4.
- `MATH` annotations must carry an actual equation; plain prose is not `MATH`.
- Public API uses descriptive PascalCase; file-static helpers may use cutie names.

## Build & test
- Build: `./scripts/build.sh` (or `cmake --build build -j`).
- Test: `./scripts/run_tests.sh` (or `./build/mikoko-test`).
- 68 unit tests, auto-registered via `.mikoko_tests` linker section (tests/*.cpp).
- Lint: `-Wall -Wextra -Werror -pedantic` is enforced at build time.

# KleaSCM Style Guide

Apply every rule below unconditionally. No exceptions.


## When rules conflict, apply them in the following order:

Defensive Programming --> Performance --> Safety

Simplicity, minimum code, and readability are optimisation goals only. They must never justify violating the priorities above.

---

## 1. Formatting

- **Tabs only** for indentation. Spaces are forbidden.
- **Max 120 chars** per line. Break at logical boundaries (operators, commas).
- **K&R bracing**: open brace on the same line as the statement.
- **No trailing whitespace.** Files end with a single newline.
- **FILE NAMES ARE NOT TO BE THE SAME AS FUNCTION NAMES, FILE NAMES SHOULD BE DESCRIPTIVE FUNCTION NAMES CAN BE ANYTHING**

---

## 2. Naming

| Context | Convention |
|---|---|---|
| Everything | `PascalCase` |
| **snake_case** | ❌ FORBIDDEN |
| **ALL_CAPS** | ❌ FORBIDDEN |

> **Exceptions (compiler-enforced):**
> - **Go package names** — Go ecosystem requires lowercase; use single-word lowercase (`mypackage`, not `MyPackage`).
> - **Go JSON tags** — `json:"field_name"` must use snake_case for wire compatibility. This is the ONLY allowed snake_case in code.
> - **Rust module declarations** — `mod some_module;` is forced by the compiler. File names follow (`some_module.rs`).
> - **Rust Cargo.toml** — crate names use snake_case by convention.
>
> These are unavoidable wire/compiler requirements. Keep everything else PascalCase.
>
> **Rust**: Clippy rejects PascalCase for functions/methods by default.
> Add `#![allow(non_snake_case)]` at the crate root to match this guide.
> For finer control, use `#[allow(non_snake_case)]` on individual items.

---

## 3. Error Handling — Zero Is Initialization (ZII)

**Every function returns a usable value. Always. No exceptions.**

- `throw`, `try/catch`, `panic!` are all **forbidden**.
- `Result`, `std::expected`, `Option`, `std::optional`, Go `(T, error)` — **forbidden in runtime paths**.
- No `.unwrap()`, no `if (Ptr == NULL)`, no null checks, no error branches.
- Startup/init functions may return Error (e.g. `mmap` OOM is real). Runtime never.
- **Runtime functions return 0 / null / empty / stub** — the caller uses it directly.

### The rule

```cpp
// ❌ WRONG — error path, null check, branch
PhysicsBody *Body = GetBody(Id);
if (!Body) { /* error handling */ return; }
ApplyForce(Body, Force);

// ✅ RIGHT — Zero Is Initialization: always a usable pointer
PhysicsBody *Body = GetBody(Id);
// Body is valid even for unknown Id — points to zeroed stub.
ApplyForce(Body, Force);
```

### Examples across domains

```cpp
// Math: edge cases return zero
Vector3 N = Vector3Normalize(ZeroVector3);    // → {0,0,0}  not NaN
Matrix3 M = Matrix3x3Inverse(SingularMatrix); // → zero matrix
float D = Vector3Distance(A, A);              // → 0

// Lookup: miss returns zero record
Body = GetBody(9999);          // → &ZeroBody (mass=0, rest=0, no force)
Constraint = GetConstraint(9999); // → &ZeroConstraint

// Allocation: OOM returns global stub
void *P = ArenaAlloc(&A, 1000000); // → &ZeroBlock on OOM

// IO: failure returns empty written/read count
size_t N = FileRead(Buf, Size);  // → 0 on error, not -1
```

### Why ZII?

- **Zero branches in hot paths** — no mispredictions, no pipeline stalls.
- **Zero special cases** — the null-deref class of bugs is eliminated entirely.
- **Zero cognitive load** — callers never ask "can this fail?".
- **Every type accepts 0** — zero velocity = "stopped", zero mass = "infinite",
  zero transform = "identity", zero record = "not found".
- **Zero is a valid state, not an error** — design your types so zero is meaningful.

### Memory for the stub

One zeroed page in `.bss` (`ZEROBLOCK_SIZE = 4096`) covers every stub
requirement. All stubs alias into this single page. The write to the first
byte of any field in the stub writes to this page — no segfault in practice
because the page is mapped R/W. Production systems size arenas so the stub
is never reached; it exists solely to eliminate branches.

---

## 4. Comments & Documentation

- **No redundancy.** Don't restate what the code does. If it's readable, no comment needed.
- **Comment WHY**, not what. Non-obvious decisions, trade-offs, constraints.
- **No half-measures.** "TODO: fix this later" / "TODO: finish error handling" are banned. Commit finished code only.
- ✅ Allowed TODOs: future features, planned optimisations (`TODO: Add caching in v2`).
- **Module header required** on every file:
- NEVER PUT THE NAME OF THE FUNCTION IN THE COMMENTS UNLESS REFERENCING ANNOTHER FUNCTION

### Module Header

Every file starts with a block that tells you:
1. What this module IS (title + one-line type description)
2. How it WORKS (design philosophy, context, trade-offs)
3. Data layout / interface description (with ASCII diagram if relevant)
4. Who wrote it

```cpp
/**
 * Module Name (ACRONYM) — one-line title.
 *
 * One-line type description.
 *
 * Paragraph explaining how the module works — what problem it solves,
 * what approach it takes, why that approach was chosen. Reference real
 * hardware / algorithms / standards that inspired the design.
 *
 * DESIGN PHILOSOPHY:
 * Explain the key architectural decisions, trade-offs, and constraints.
 * Why was this approach chosen over alternatives? What systems does it
 * draw inspiration from?
 *
 * REGISTER MAP / DATA LAYOUT / INTERFACE:
 * ┌────────────┬─────────────────────────────────────────────────┐
 * │ Name       │ Description (use ASCII tables when relevant)     │
 * ├────────────┼─────────────────────────────────────────────────┤
 * │ Field      │ What it does, type, semantics                   │
 * └────────────┴─────────────────────────────────────────────────┘
 *
 * [ASCII bitfield diagram if relevant]
 * ┌───┬───┬───┬───┬───┬───┬───┬───┐
 * │ 7 │ 6 │ 5 │ 4 │ 3 │ 2 │ 1 │ 0 │
 * └───┴───┴───┴───┴───┴───┴───┴───┘
 *   │                                       └── Bit 0 description
 *   └─────────────────────────────────────── Bit 7 description
 *
 * Detailed description of each field / register / method group:
 * FIELD_NAME (Offset):
 * - What it does
 * - Read/write semantics
 * - Values and their meaning
 *
 * WORKFLOW (if relevant):
 * 1. Step one
 * 2. Step two
 * 3. Step three
 *
 * PHYSICAL EQUIVALENTS / REFERENCES:
 * - Standard/library/algorithm this is based on
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
```

Key principles:
- **Title + one-line type description** are mandatory
- **Design philosophy** is mandatory for complex modules
- **ASCII diagrams** are encouraged for data layouts, register maps, state machines, algorithm flows
- **Author/Email** is mandatory
- **NEVER put function names** in the header block (they go in their own doc comments)
- **Language choice:** Use EITHER English OR Japanese per comment — never both. Do not translate the same comment twice.

Example (simpler module — no registers, no hardware):

```cpp
/**
 * TohruPhysics用の固定幅スカラー数学ね。
 *
 * IEEE 754 double-precision math with explicit NaN/Inf guarding.
 * All functions sanitize inputs — return 0.0 for any degenerate case.
 *
 * DESIGN PHILOSOPHY:
 * Physics engines call trig and sqrt millions of times per frame.
 * The standard library doubles (libm) guarantee 1 ULP accuracy but
 * are unnecessarily precise for physics simulation where 1e-4
 * relative error is invisible. We use range-reduced polynomial
 * approximations (9th-order Taylor for sin/cos, Newton-raphson
 * for invsqrt) that are 3-5x faster at 1e-6 accuracy.
 *
 * References:
 * - sin/cos: 9th-order Taylor on [0, PI/2] with quadrant reduction
 * - invsqrt: Quake-style bit hack + 3 Newton iterations
 * - acos: domain-split asin via sqrt identity
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
```

- **Public API docs**: only if the signature alone is insufficient (e.g. side effects).

### KleaSCM Annotations

These annotations are parsed by `todo-comments.nvim` and highlighted in the editor. Use them EXACTLY as specified — no spaces before the tag, no alternate formats.

**CRITICAL:** These annotations are used IN ADDITION TO normal `//` comments. Code should have BOTH:
- Normal `//` comments explaining WHY (never WHAT, never function names)
- KleaSCM annotations for structured, searchable documentation

| Annotation | Format | When to use |
|---|---|---|
| TODO | `TODO(KleaSCM): description` | Places where diagnostic logging should be added later. |
| NOTE | `NOTE(KleaSCM): note text` | Any code that needs an explanatory note about WHY. |
| MATH | `MATH(KleaSCM): equation + solution + explanation` | Math/equations used in code — include the equation, its solution, and why. |
| HACK | `HACK(KleaSCM): description + reason` | Where a hack/workaround is used — ALWAYS document the reasoning. |
| REFERENCE | `REFERENCE(KleaSCM): doc summary` | Documentation references — short summary of the standard/algo this code is based on. |

**ALL CODE MUST MAKE HEAVY USE OF BOTH normal `//` comments AND KleaSCM annotations throughout the codebase.**

**Formatting rules:**
- **Annotations are NEVER stacked.** One logical annotation carries exactly one tag on its first line; if the text wraps, the continuation lines are plain `//` comments with no tag.
  ```cpp
  //NOTE(KleaSCM): one mutex serialises every sink write so multi-threaded lines
  // never interleave; the cost is paid only when a message passes.
  ```
- **Annotation lines may exceed the 120-char code limit.** Do not fragment a tag into tiny pieces to fit the code width; write the note at its natural length and only wrap when it truly runs long.
- No space before the tag: `//NOTE(KleaSCM):`, never `// NOTE(KleaSCM):`.

Examples:

```cpp
// Normal comment: explains WHY this threshold was chosen
const int UnrollThreshold = 8;
//HACK(KleaSCM): Unroll threshold tuned for Zen 4 — revisit for ARM

// Normal comment: explains WHY we need serialization here
std::mutex BloomMutex;
//NOTE(KleaSCM): This mutex serializes bloom-filter writes from 4 goroutines

// Normal comment: explains WHY Verlet is used instead of Euler
Position = 2 * Position - PrevPosition + Acceleration * dt * dt;
//MATH(KleaSCM): Verlet integration: x_{t+1} = 2x_t - x_{t-1} + a*dt² — converges on position, no velocity needed

// Normal comment: explains WHY we trust the hardware sqrt
float Result = std::sqrt(Value);
//REFERENCE(KleaSCM): IEEE 754 §5.4 — sqrt is correctly rounded, no extra guarding needed
```

Wrong — tag repeated on every wrapped line:

```cpp
//NOTE(KleaSCM): with NDEBUG the macros expand to nothing, so release builds
//NOTE(KleaSCM): carry zero assert code and zero cost — the standard assert gate.
```

### Language Rules
- **Choose ONE language per comment** — either English OR Japanese, never both. Do not translate the same comment twice.
- Japanese must sound like a young Japanese woman wrote it — natural feminine speech.
- `ね`、`の`、`わ`、`してある` are all natural and welcome. Use them freely where they fit.
- **`のよ` is forbidden** — too theatrical for technical writing.
- `。` for plain declarative statements where nothing else fits naturally.
- Short inline comments may stay English-only if the Japanese adds nothing.
- Doc comments on trivial getters/setters: omit both (per rule 4.1).

---

## 5. Defensive Programming
- Validate all input. Assume external data is malformed.
- No global state. Compile-time constants only. Inject dependencies.
- Separation of concerns. No logic in UI/render code.
- Code must not rely on undefined or implementation-defined behaviour.

---

## 6. Project Structure
- One logical unit per file.
- FILE NAMES ARE DESCRIPTIVE. FILE NAMES ARE NOT TO BE THE SAME AS FUNCTION NAMES.
- Import order: Standard → Third-Party → Local, separated by blank lines.
- No circular dependencies.

---

## 7. Testing

- **Mock boundaries only** (API, DB, FS). Test logic with real objects.
- Document the **intent** of every test case.
- No snapshot testing for logic. Assert specific values.

---

## 8. Data Structures
Arrays by default. [], Vec, std::vector.
Sets only when uniqueness is strictly required — must include a comment explaining why an array was insufficient.

---

## 9. Numeric Rules (new section)
- Prefer signed integers unless non-negative semantics are explicitly required.
- Use fixed-width integer types (int8, uint32, int64, etc.) whenever size or binary layout matters.
- Never compare floating-point values for equality without explicit justification.
- Code must not rely on integer overflow.

## 10. Prohibited Constructs

| Construct | Status |
|---|---|---|
| Exceptions (`throw`/`try-catch`/`panic!`) | ❌ FORBIDDEN |
| `snake_case` | ❌ FORBIDDEN |
| `ALL_CAPS` | ❌ FORBIDDEN |
| Global mutable state | ❌ FORBIDDEN |
| Inline control flow (no braces) | ❌ FORBIDDEN |
| Null checks / optional unwraps (runtime) | ❌ FORBIDDEN |
| `Result` / `std::expected` / `Option` (runtime) | ❌ FORBIDDEN |
| `std::optional` / `std::variant` (runtime) | ❌ FORBIDDEN |

**Always use braces:**
```cpp
// ❌
if (x) Do();

// ✅
if (x) {
	Do();
}
```

---

## 11. Performance

### C++
- `\n` not `std::endl` (no forced flush).
- Pass by `const T&` for strings/vectors/descriptors.
- **No smart pointers.** Raw `T*` from arena (see §12). Never `unique_ptr` or `shared_ptr`.
- `reserve()` before loops. No `new` in hot paths.

### Go
- `make([]T, 0, cap)` to preallocate slices.
- Pointer receivers (`*Struct`) for large types.
- Concrete types in hot paths; avoid interface conversions.

### Rust
- `&str` over `String` for arguments.
- **No `Box`, `Rc`, `Arc` in production paths.** Raw `*mut T` from arena (see §12).
- No `.clone()` in loops — use references or `Cow`.
- `Vec::with_capacity(n)` to preallocate.

### TypeScript
- No closures defined inside loops.
- No spread (`[...arr]` / `{...obj}`) in hot paths.
- `for` / `for..of` over `.forEach` / `.map` in critical code.

---

## 12. Concurrency

All shared mutable state must be synchronised. Ownership transfer requires no synchronisation.

---

## 13. Tooling

| Language | Linters |
|---|---|
| TS/TSX | ESLint, Prettier |
| C++ | clang-format, clang-tidy |
| Go | gofmt, golint |
| Rust | rustfmt, Clippy |

**Rust linter config (`.clippy.toml` or `Cargo.toml` `[lints.clippy]`):**

```toml
# Crate-level: src/lib.rs or src/main.rs
#![allow(non_snake_case)]

# Project-level: Cargo.toml
[lints.clippy]
non_snake_case = "allow"
```

Linter configs must exactly match this guide (tabs, naming, etc.).

---

## 14 Arena Pattern (Failure-Proof, Go, C++ & Rust)

### Arena lifetime defines object lifetime. Individual objects never own memory.

- **No RAII.** `new`/`delete`, `malloc`/`free`, smart pointers (`unique_ptr`, `shared_ptr`, `Box`, `Rc`, `Arc`) are **forbidden**. Owned raw pointers into arena memory.
- **No per-element lifecycle.** No constructors/destructors per element. Batch allocation and batch teardown only.
- **Think in groups, not elements.** Arenas batch-allocate; individual allocations are an anti-pattern.
- **Append-only growth.** Push onto the end. Each arena has a tuned initial capacity to minimise re-growth.
- **Reuse scratch space.** Hashes, buffers, and temp storage live in reusable slots inside the arena.
- **Zero Is Initialization (ZII)** — applies to EVERY type and EVERY function, not just arenas. See §3.
- **Literally cannot fail.** When the arena is exhausted, return the `ZeroBlock` stub. All callers handle stubs transparently — no `std::optional`, no `Option`, no `Result`, no exceptions, no unwinding.
- **Minimum code.** No entropy injection, no defensive copies, no work beyond what the operation strictly requires.
- **Clear on exit.** Zero the entire arena at the end of its lifetime. Reasoning: the `ZeroBlock` is always valid, so clearing restores the invariant.

### Zero Is Initialization in practice

```cpp
// Zero-is-valid in practice
PhysicsBody *Body = ArenaAlloc(&Arena, sizeof(PhysicsBody));
// Body is always valid. On exhaustion it points to the ZeroBlock stub.
ApplyForce(Body, Force);
```

```cpp
// Every runtime function follows the same pattern — zero is always valid:
Vector3 Center = GetBodyCenter(Body);        // {0,0,0} on zero body
Matrix3 Inertia = GetBodyInertia(Body);       // zero matrix on zero body
float Speed = Vector3Length(Velocity);        // 0 on zero vector
int Count = GetContactCount(Manifold);        // 0 on zero manifold
```

---

## 14b. ZII Design Checklist

Before writing ANY function, verify:

1. **What is the zero value?** — Every type must have a valid zero state.
   - `Vector3` → `{0,0,0}`
   - `Matrix3` → zero matrix
   - `Body*` → `&ZeroBody` (mass=0, position=origin, rotation=identity)
   - `Contact*` → `&ZeroContact` (no penetration, no impulse)

2. **Does every code path return a usable value?** — No branches for "error",
   "not found", "invalid", "overflow". Return zero instead.

3. **Can the caller use the result without branching?** — The caller must
   be able to pass the result directly to the next function.

4. **Are NaN/Inf guarded?** — Floating-point functions sanitize inputs
   and return zero on degenerate input. NaN never propagates.

## 15 .editorconfig (reference)

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true
indent_style = tab
indent_size = 4
max_line_length = 120
```

## Related Skills

- `cpp` — C++ language-specific KleaSCM patterns (arena, templates, modules, coroutines)
- `go` — Go language-specific KleaSCM patterns (arena, concurrency, API design)
- `rust` — Rust language-specific KleaSCM patterns (arena, unsafe, no_std, async)
- `code-documentation` — bilingual EN/JA doc patterns, README, ADR templates
- `INDEX` — full directory of all available skills
