# Rework log

A running record of the client rework. Each step lists, per file, what was
**removed**, **edited**, and **added** — so a rename can never be mistaken for a
deletion, and a signature change is always visible to whoever maintains callers.

Read alongside `git diff`. Formatting-only changes are called out as such so they
can be skipped.

---

## Step 1 — Foundation fixes

**Goal:** fix the defects found while mapping the architecture, and make the
binary runnable from any directory. No feature work, no visual change.

**Status:** complete, uncommitted. `go vet`, `gofmt`, `go test ./...` all clean.

### `internal/models/problems.go`

The substance of the step. Four separate concerns, all in one file.

#### Removed

| Symbol | Note |
|---|---|
| `FormatProblemFromProblemStruct() (string, error)` | **Deleted and replaced by `Format() string`.** This is the only function in the codebase that was renamed — see *Edited* for why. |
| inline `runtime.Caller(1)` block inside `ValidateProblem` | Moved into the new `callerName` helper. The behaviour survives; only its location changed. |
| the single compound `if` in `ValidateProblem` | One boolean OR across six conditions, replaced by six separate checks. |
| `available := make([]string, ...)` in `GetCodeStub` | The slice was built and then discarded — the function returned `""` regardless. Its intent is now realised by `AvailableLanguages()`. |
| `p.Examples == nil` check | Redundant: `len()` on a nil slice is already `0`, so the adjacent `len(p.Examples) == 0` already covered it. |

#### Edited

**`Example.Explanition` → `Example.Explanation`** *(bug fix)*

```go
- Explanition string `json:"explanition"`
+ Explanation string `json:"explanation"`
```

Every seed file writes `"explanation"`. The tag never matched, so the field
always unmarshalled empty and the render branch that checked it was unreachable.
Explanations now appear in the description pane.

**`ValidateProblem() error`** — *signature unchanged, body rewritten.*

Still a method on `*Problem`, still returns `error`, still called from
`loader.go:56`. What changed is the failure path: it now accumulates the names of
the missing fields, logs the caller to `clicode.log`, and returns a typed
`*ValidationError` instead of a formatted string. The stack context stays in the
log; the UI gets a clean message.

**`GetCodeStub(string) string` → `GetCodeStub(string) (string, bool)`**
*(**breaking signature change** — one caller updated)*

Previously an absent stub and an empty stub were both `""`, indistinguishable.
The bool now says which. Caller updated at `internal/screens/problem.go`.

**`Format()` body** — `t.Explanition` → `t.Explanation`, and it returns one value
instead of two. Output text is otherwise byte-identical to before.

#### Added

| Symbol | Purpose |
|---|---|
| `ValidationError{ID int; Missing []string}` + `Error() string` | A typed error naming the problem and the missing fields. Lets callers use `errors.As` instead of matching strings. |
| `callerName(skip int) string` | Six lines: turns "N frames up the stack" into `"pkg.Func:line"` for the log. **Called from exactly one place** — `ValidateProblem`. It does not replace anything; the old inline version did the same job. Flagged as a candidate to inline back, since the `skip: 2` it requires is fragile. |
| `AvailableLanguages() []string` | Sorted list of languages this problem ships a stub for. Realises the intent of the discarded `available` slice. |
| imports `log`, `sort` | For the above. |

#### Unchanged

`FormatTestCase`, and the `TestCase` / `Submission` / `Problem` struct
definitions. `Problem` gained no fields.

---

### `internal/screens/problem.go`

Consumes the model changes above. One user-visible behaviour change.

#### Removed

- **`knownLanguages = []string{"python", "cpp", "go"}`** — the hardcoded list the
  editor cycled through regardless of what the problem offered. Replaced by
  `fallbackLanguages` (see *Added*), which is a floor rather than the list.
- **`if language == "" { language = "python" }`** in `NewProblemScreen` — an
  unconditional default that ignored the problem's actual stubs. Replaced by
  `initialLanguage`.
- **the `fmtErr` handling block** in `initViewports`:
  ```go
  - formatted, fmtErr := s.problem.FormatProblemFromProblemStruct()
  - if fmtErr != nil {
  -     formatted = fmt.Sprintf("Error formatting problem: %v", fmtErr)
  - }
  ```
  This was the landmine: a validation failure replaced the entire description
  pane with an error string, even though the formatted text was built fine. Now
  one line: `s.problemDescription.SetContent(s.problem.Format())`.

#### Edited

- **`switchLanguage(delta int)`** — iterates `s.languages()` instead of the
  global. Same wrap-around arithmetic.
- **`stubForLang(lang string)`** — uses the new `ok` bool to decide whether to
  fall back to `"// Write your solution here"`.
- **`NewProblemScreen`** — sets `s.language` via `initialLanguage` after the
  struct literal, since the helper needs `s.problem` to already be set.
- **`cmdStyle` / `cmdPrompt` alignment** — *gofmt only, no semantic change.* The
  file was unformatted at HEAD.

#### Added

| Symbol | Purpose |
|---|---|
| `languages() []string` | The problem's available languages, or `fallbackLanguages` if it has none. Single source of truth for both helpers below. |
| `initialLanguage(preferred string) string` | Honours the configured default when the problem has a stub for it, else takes the first available. **This is a behaviour change**, not just a refactor — see below. |
| `fallbackLanguages` | The old `knownLanguages` list, kept only so `langs[0]` can't panic on a problem with zero stubs. |

**Behaviour change to be aware of:** language selection now depends on the open
problem. Invisible with the current seed data — every seed problem ships all
three languages — but it will matter once problems arrive from the API.

---

### `internal/loader/loader.go`

Mechanical: same logic, different byte source.

#### Removed

- the `os.Stat` existence pre-check — the embedded FS reports not-found through
  its read error, so the extra syscall is gone.
- imports `os`, `path/filepath`.

#### Edited

- **`LoadProblem`** — reads via `fs.ReadFile(data.Problems(), "<id>.json")`
  instead of `os.ReadFile("data/problems/<id>.json")`. Not-found is now detected
  with `errors.Is(err, fs.ErrNotExist)`. Still returns the same
  `*ProblemNotFoundError`.
- **`MakeProblemList`** — same substitution for `problems_list.json`.
- the local variable `data` renamed to `raw` in both functions, because `data` is
  now the imported package name.

#### Added

- imports `errors`, `io/fs`, and `github.com/aashish1502/clicode/data`.

#### Unchanged

Both signatures, both error types (`ProblemNotFoundError`,
`InvalidProblemDataError`), and the `ValidateProblem` call.

---

### `internal/screens/title.go`

#### Removed

- **`const artPath = "data/art/current.txt"`** — the relative path that tied art
  loading to the repo root.

#### Edited

- **`loadArt()`** — resolution order changed from
  `data/art/current.txt → banner` to
  `~/.clicode/art/current.txt → embedded → banner`. The art stays swappable
  without recompiling; the swap point moved to the home directory so it survives
  `go install`.
- the `TitleScreen` doc comment, to describe the new order.
- **`artStyle` indentation** — *gofmt only.*

#### Added

| Symbol | Purpose |
|---|---|
| `const userArtPath = "art/current.txt"` | Path relative to `~/.clicode/`. |
| `userArt() string` | Reads the user's override file; returns `""` on any error. |
| `clean(raw string) (string, bool)` | Trims a candidate banner and reports whether anything remains — used to reject a blank file at both the user and embedded steps. |

#### Unchanged

`fallbackArt`, `Init`, `Update`, `View`, and the 4-second auto-advance.

---

### `data/embed.go` — new file (37 lines)

The only new production package. Embeds the seed problems and art into the
binary:

```go
//go:embed problems/*.json art/*.txt
var files embed.FS

func Problems() fs.FS      // rooted at problems/, so entries are "1.json"
func Art() (string, bool)  // false if missing or blank
```

`Problems()` panics if `fs.Sub` fails — unreachable, since the embed directive is
validated at compile time.

---

### `internal/design/theme.go`

**gofmt only. No semantic change.** It appears in the diff solely because the
file was unformatted at HEAD.

---

### `.gitignore`

- **Edited:** uncommented `.idea/` and `.vscode/`.
- **Added:** a `clicode.log` entry under a "Runtime artifacts" heading.
- Both `.idea/` and `clicode.log` were `git rm --cached`'d, so they appear as
  staged deletions.

---

### Tests — 4 new files, 285 lines

Previously zero `_test.go` files existed anywhere in the repo.

| File | Covers |
|---|---|
| `internal/models/problems_test.go` | The `explanation` tag fix, `ValidateProblem`'s missing-field list (table-driven), the UI error message carrying no stack detail, `GetCodeStub`'s bool, `AvailableLanguages` ordering, `Format` on an incomplete problem. |
| `internal/loader/loader_test.go` | `TestEverySeedProblemLoads` walks the index and fails if a seed file drifts from it. `TestLoadsFromAnyWorkingDirectory` uses `t.Chdir` and would have failed before the embed. |
| `data/embed_test.go` | Art and problem files are actually present in the binary. |

---

### Corrections to the earlier survey

- **`data/problems/110.json` was never unreachable.** It is present in
  `problems_list.json`; all 11 ids match their files. The initial report was
  wrong and no fix was needed.

---

## Step 1b — Language catalog (`internal/languages`)

**Goal:** fix a regression introduced in step 1 and put the seam for
server-supplied languages in place.

**Why:** step 1 made the editor cycle over `AvailableLanguages()` — the languages
a problem ships *starter code* for. That silently capped what a user could write
in. A judge that accepts Kotlin accepts it whether or not anyone wrote a Kotlin
stub, so a problem shipping only a Python stub became Python-only. Strictly worse
than the hardcoded list it replaced, which at least let you reach a language with
no stub.

**The distinction now enforced:**

| Question | Answered by |
|---|---|
| Which languages may I write in? | the judge's supported set — config today, the API later |
| Do I get starter code? | `AvailableLanguages()`, else a per-language comment |

**Status:** complete, uncommitted. `go vet`, `gofmt`, `go test ./...` clean.

### `internal/languages/languages.go` — new package (150 lines)

#### Added

| Symbol | Purpose |
|---|---|
| `Language{ID, DisplayName, CommentPrefix}` | One selectable language. |
| `Language.FallbackStub() string` | The placeholder buffer when a problem ships no starter code — a comment *in that language*. |
| `Catalog` interface | `Supported(ctx) ([]Language, error)`. The seam the API-backed implementation plugs into. |
| `Static` + `NewStatic(ids)` | Config-backed implementation. Empty config → `DefaultIDs()`. |
| `Get(id) Language` | Metadata lookup that **tolerates unknown ids**. |
| `Union(supported, ids)` | Supported set plus any stubbed language the catalog hasn't heard of. |
| `DefaultIDs()`, `IDs()` | Default set (`python, cpp, java, go`), and a flattener. |

**The one forward-compat decision:** `ID` is a plain `string`, never a Go
constant or iota. An enum would need a client release for every language the
server adds. With strings, the server starts returning `"kotlin"` and an
already-shipped client renders it — displayed under its own id, commented with
`//`. `TestGetToleratesUnknownLanguage` is the guard on that property.

Everything genuinely dependent on the API's shape — pagination, whether languages
are global or per-platform, version distinctions like `python3` vs `python` — is
deferred until the response shape is real.

### `internal/config/config.go`

#### Added

- **`Languages []string`** with a `json:"languages"` tag. Empty means
  `languages.DefaultIDs()`. This is the interim stand-in for the server's
  capabilities response.

#### Edited

- `Default()` sets `Languages: nil` explicitly, so the zero value is the
  documented "use defaults" case rather than an accident.

### `internal/screens/router.go`

#### Added

- **`Router.supported []languages.Language`** — resolved once in `NewRouter` from
  `languages.NewStatic(cfg.Languages)`. A resolution failure logs and falls back
  to defaults; a static catalog can't fail, but the API-backed one can and the
  editor still needs something to open.
- imports `context`, `log`, `internal/languages`.

#### Edited

- `NewProblemScreen(...)` call now passes `r.supported`.

### `internal/screens/problem.go`

#### Removed

- **`fallbackLanguages`** *(added in step 1, now superseded)* — the hardcoded
  `{python, cpp, go}` floor. `languages.DefaultIDs()` does this job.
- **`languages() []string`** *(added in step 1)* — renamed and re-specified as
  `languageSet()`. The rename avoids shadowing confusion with the new package
  name; the change of meaning is the actual fix.
- the hardcoded `"// Write your solution here\n"` fallback string.

#### Edited

- **`NewProblemScreen`** — gained a sixth parameter, `supported
  []languages.Language`. *(**breaking signature change** — one caller, the
  router, updated.)*
- **`switchLanguage`** — iterates `languageSet()` and compares `l.ID`. Same
  wrap-around arithmetic.
- **`stubForLang`** — the no-stub path now returns
  `languages.Get(lang).FallbackStub()`. **This fixes a latent bug:** the old
  fallback was `//` for every language, which is a syntax error in Python. It was
  unreachable only because all 11 seed problems ship all three stubs — fixing the
  cycling regression is exactly what would have made it reachable.
- **`initialLanguage`** — reordered: configured default if supported → else a
  language with starter code → else the first supported. Preferring a stubbed
  language is a convenience and never restricts what can be selected afterwards.
- **`View`** — the header shows `DisplayName` ("C++", not "cpp") and appends
  `· no starter code` when the current language has none, so an empty-looking
  buffer is explained rather than mysterious.

#### Added

- **`ProblemScreen.supported []languages.Language`** field.
- **`languageSet() []languages.Language`** — `Union(supported, stubbed)`, with a
  defaults fallback if `supported` arrives empty.

### Tests — 2 new files

| File | Covers |
|---|---|
| `internal/languages/languages_test.go` | Unknown-id tolerance, per-language comment prefixes, static catalog + config fallback, `Union` ordering and dedup. |
| `internal/screens/problem_test.go` | **`TestLanguageSetIsNotLimitedToStubs` is the regression guard** — a Python-only problem must still reach Kotlin. Plus unknown stubbed languages surviving, full cycling coverage, per-language fallback stubs, `initialLanguage` precedence, and an empty supported set. |

First tests to exist for the `screens` package.

### Still open from step 1

`callerName`'s `skip: 2` in `internal/models/problems.go`. Unchanged by this step.

---

## Step 1c — Language picker modal

**Goal:** replace `ctrl+l`/`ctrl+h` cycling with a list you choose from, with the
languages that ship starter code marked in green.

**Why:** cycling was tolerable across three hardcoded languages. Once the
selectable set is judge-supplied it can be a dozen or more, and stepping through
them one keypress at a time to reach the last one is the wrong interaction.

**Design decision — a modal, not a routed screen.** A router screen would need
the problem threaded into it plus a return path, and the router's back navigation
is hardcoded (`router.go:73-80`). Keeping the picker inside `ProblemScreen` means
the editor buffer and description scroll position are untouched by construction.
While open it replaces the split-pane view rather than floating over it;
compositing over live ANSI output is a layout concern and belongs with the layout
work in step 4.

**Status:** complete, uncommitted. `go vet`, `gofmt`, `go test ./...` clean.

### `internal/screens/language_picker.go` — new file (130 lines)

#### Added

| Symbol | Purpose |
|---|---|
| `languagePicker{open, langs, stubbed, cursor}` | Modal state. `stubbed` is a set, so colouring is a map lookup per row. |
| `openPicker(langs, stubbed, current)` | Builds the modal with the cursor already on the current language. |
| `(languagePicker) Update(tea.KeyMsg) (languagePicker, string)` | One key. The returned id is non-empty **only** on an actual selection; cancelling returns `""`, so the caller cannot confuse the two. |
| `(languagePicker) View(width, height, current) string` | Centred box. Green for stubbed, normal foreground for the rest, `(current)` on the active one, plus a legend explaining the colour. |
| `pickerBox`, `pickerTitle`, `pickerCursor` styles | Local for now; they fold into the theme in step 4. |

Keys: `j`/`k`/arrows/`ctrl+n`/`ctrl+p` move, `g`/`G` jump to ends, `enter`/`space`
select, `esc`/`q`/`ctrl+l` cancel.

### `internal/design/theme.go`

#### Added

- **`LanguageStubbed`** (green, 82) and **`LanguageBare`** (250, normal
  foreground). Named for what they mean rather than reusing `Solved`, which is
  the same green but a different concept.

### `internal/screens/problem.go`

#### Removed

- **`switchLanguage(delta int)`** *(added in step 1, replaced by
  `setLanguage(id)`)* — cycling is gone entirely.
- **the `ctrl+h` binding** in both panes. It only existed to cycle backwards and
  has no meaning without cycling.

#### Edited

- **`ctrl+l`** now opens the picker instead of advancing the language. Same key,
  so existing muscle memory still lands somewhere sensible.
- **key routing** — the picker is checked *before* command mode and before the
  insert-mode passthrough, so it swallows every key while open. `ctrl+c` is the
  one exception and still quits.
- **`View`** — returns the modal early while it is open.
- **the help bar** — `ctrl+l/h: lang` → `ctrl+l: language`.

#### Added

| Symbol | Purpose |
|---|---|
| `ProblemScreen.picker` field | Modal state. |
| `setLanguage(id string) ProblemScreen` | Stashes the current buffer into `langEdits` before swapping, so switching away and back preserves work. No-ops on `""` or the current language, which is what makes a cancelled picker safe. |
| `openLanguagePicker() ProblemScreen` | Builds the modal from `languageSet()` and the problem's stubs. |

### Tests

#### Removed

- **`TestSwitchLanguageReachesEveryLanguage`** — tested cycling, which no longer
  exists. Its coverage is now spread across the picker tests.

#### Added

Seven tests in `internal/screens/problem_test.go`, plus `key()` and `press()`
helpers that drive the screen through `Update` the way bubbletea does — so these
exercise real key routing rather than calling methods directly:

| Test | Guards |
|---|---|
| `TestCtrlLOpensThePicker` | The binding is wired. |
| `TestPickerListsEverySelectableLanguage` | All selectable languages listed, and `stubbed` correctly marks which have starter code. |
| `TestPickerOpensOnTheCurrentLanguage` | Cursor starts where you are, not at the top. |
| `TestPickerSelectsAnUnstubbedLanguage` | **The Kotlin case** — selecting a language with no starter code works. |
| `TestPickerCancelKeepsCurrentLanguage` | `esc` after moving the cursor changes nothing. |
| `TestSetLanguagePreservesEdits` | Switching away and back does not lose typed work. |
| `TestPickerViewShowsEveryLanguageAndLegend` | Every display name and the legend actually render. |

### Not done

The picker replaces the pane view while open rather than floating over a dimmed
background. That is deliberate — see the design note above.

---

## Step 1d — Navigation stack

**Goal:** replace the router's hardcoded transitions with a real screen stack
that defines the app's lifecycle, and make navigation keys impossible to spam
into unbounded growth.

**Why:** `router.go` kept screens in a `map[screenID]tea.Model` and hardcoded
"back" as *test-case → problem → menu* (`router.go:73-80`). There was no history,
so every new screen needed another hardcoded arm, and nothing bounded how often a
screen could be re-entered.

**Status:** complete, uncommitted. `go vet`, `gofmt`, `go test ./...` clean.

### Model

Screen identity is `Key{ID, Arg}`. An empty `Arg` means a singleton — there is
only ever one menu. A non-empty `Arg` separates instances of the same screen
holding different content, so problem 1 and problem 2 are distinct entries with
distinct cached state.

Each push declares how it interacts with what is already there:

| Mode | Behaviour | Used by |
|---|---|---|
| `Single` | Already on the stack? Pop back to it. Else push. | list, problem, test case, settings |
| `Root` | Clears the stack; becomes the only entry. | menu |
| `Replace` | Swaps the current entry, depth unchanged. | title → menu |
| `Fresh` | Always a new instance, never cached. | nothing yet — the escape hatch |

`Single` is what bounds the stack: it can never grow past the number of distinct
screens. `Root` on the menu means reaching it from any depth collapses to one
frame. `Replace` on the title means going back from the menu quits rather than
returning to the splash.

**Lifecycle:** the app's lifetime *is* the stack's. Popping the last frame
returns `ok == false`, and the router turns that into `tea.Quit`.

### `internal/screens/stack.go` — new file (180 lines)

#### Added

| Symbol | Purpose |
|---|---|
| `Key{ID, Arg}` | Screen instance identity. |
| `Mode` + `Single`/`Root`/`Replace`/`Fresh` | Push semantics. |
| `stack{frames, cache}` | The history, plus the LRU of popped screens. |
| `push(key, mode, build) tea.Cmd` | Returns the new screen's `Init` — **nil when an instance was reused or restored**, so a cached title screen does not restart its timer. |
| `pop() (tea.Cmd, bool)` | `ok == false` means the stack is empty: quit. |
| `broadcast(msg) tea.Cmd` | Delivers to every frame, not just the visible one. |
| `reset`, `current`, `replaceCurrent`, `depth`, `truncate`, `recache`, `uncache` | Supporting operations. |
| `const cacheSize = 8` | LRU bound. |

**The cache** holds popped screens keyed by identity, so returning to a problem
restores its editor buffer rather than resetting it. It is a convenience, not
storage — capped at 8, and made redundant once step 3 persists solutions to
SQLite.

### `internal/screens/router.go` — rewritten

#### Removed

- **`Router.screens map[screenID]tea.Model`** — the retained-screen map. Replaced
  by the stack.
- **the hardcoded back path** (`if r.screens[problemScreenID] != nil { ... } else
  { menuScreenID }`). `NavigateBackMsg` is now just `pop()`.
- **the eager construction** of title/menu/list at 0×0 in `NewRouter`. Only the
  title is built at startup; everything else is built on first push.
- **the deliberate rebuilds** of menu and list on every navigation (`router.go:83-87`,
  `89-94`). They exist to keep the last-worked-on marker fresh, which `Refresh`
  now does without discarding cursor position.

#### Edited

- **`Update`** — navigation cases became `navigate(key, mode, build)` calls.
  `NavigateToProblem`/`TestCase` key by problem id.
- **`Init`/`View`** — read through `stack.current()`.
- **resize** — delegates to `stack.broadcast`.

#### Added

| Symbol | Purpose |
|---|---|
| `Router.stack` | Replaces the screens map. |
| `refresher` interface | `Refresh(lastProblemID) tea.Model`, implemented by screens whose data goes stale while hidden. |
| `navigate(key, mode, build)` | Push, then reveal. |
| `reveal()` | Refreshes the now-visible screen and hands it the **current terminal size** — a screen restored from the cache after a resize would otherwise render at its old dimensions. |
| **global `ctrl+q`** | Hard quit, handled before anything else so it works even if a screen stops responding. |

### Key changes across the screens

`ctrl+c` now **pops** rather than quitting. Popping the root quits, so at the
menu it still exits — the behaviour is the same where you'd expect it, and
different where it's useful.

| File | Change |
|---|---|
| `menu.go` | `ctrl+c`/`esc` → back. Added `Refresh`. Help line advertises `ctrl+q: quit`. |
| `problem_list.go` | `ctrl+c`/`q`/`esc` → back. Added `Refresh`. |
| `settings.go` | `ctrl+c`/`q`/`esc` → back, instead of jumping to the menu. Under a stack these differ: back returns wherever you came from. |
| `problem.go` | `ctrl+c` → back in both panes. Help line says `q: back`. |
| `language_picker.go` | `ctrl+c` closes the modal rather than leaving the screen. |
| `editor/vim.go` | **`ctrl+c` now exits insert mode**, as in vim. Without this the app-level back binding would fire mid-typing. |

`testcase.go` needed no change — it already emitted `NavigateBackMsg`.

### Tests — 2 new files, 20 tests

`internal/screens/stack_test.go` (10) covers the stack in isolation with a stub
screen: `Single` not growing over 50 pushes, raising an existing instance without
rebuilding, `Root` collapsing, `Replace` holding depth, `Fresh` always pushing,
pop reporting empty, cached state surviving a round trip, keyed instances staying
independent, and the cache staying bounded.

`internal/screens/router_test.go` (10) covers the lifecycle end to end, with
`HOME` pointed at a temp dir so session writes never touch the real `~/.clicode`:
startup on the title, the title not lingering behind the menu, back-from-root
quitting, back-below-root not quitting, `ctrl+q` from three screens deep,
**50 menu navigations leaving depth at 1**, the same for the list, problems
stacking on the list, the menu being refreshed when revealed, and a revealed
screen picking up a size change it missed.

### Open

`Fresh` mode has no user yet. It exists because the alternative — discovering it
is needed later and retrofitting it — is worse, and it costs four lines.

### Open question

`callerName`'s `skip: 2` is correct only because of where the function currently
sits, and nothing enforces that. Inlining it back into `ValidateProblem` with
`runtime.Caller(1)` keeps the line number and the typed error while removing the
frame-counting footgun. Awaiting a decision.
