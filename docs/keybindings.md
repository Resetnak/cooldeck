# Keybindings

All bindings are defined in `internal/tui/keys.go`. Changing them there updates
behaviour, footer hints, and the help overlay together.

## Global

| Keys | Action |
|------|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` / `Home` | First item |
| `G` / `End` | Last item |
| `Ctrl+D` / `PgDn` | Page down |
| `Ctrl+U` / `PgUp` | Page up |
| `h` / `←` | Left / sidebar focus |
| `l` / `→` | Right (context-dependent) |
| `Enter` | Open / confirm |
| `Esc` | Back / cancel / close modal |
| `q` | Back, or quit on the root list |
| `Ctrl+C` | Force quit (cancels in-flight work) |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `/` | Filter applications, or search logs |
| `:` / `Ctrl+K` | Command palette |
| `?` | Help overlay |
| `R` | Manual refresh |
| `Ctrl+T` | Cycle theme |
| `Ctrl+W` | Toggle compact layout |
| `1` | Applications |
| `2` | Deployments (active queue) |
| `3` | Instances |
| `4` | Diagnostics |

## Applications

| Keys | Action |
|------|--------|
| `Enter` | Application detail |
| `d` / `D` | Deploy / force deploy (confirm) |
| `r` | Restart (confirm) |
| `s` | Start or stop (confirm) |
| `l` / `L` | Runtime logs / open build log path |
| `b` / `o` | Open primary domain / repository |
| `c` | Copy application UUID |
| `S` | Cycle sort (status → name → last deploy) |
| `space` | Mark / unmark for the fleet tail |
| `t` | Tail the marked applications |
| `/` | Filter (`status:`, `project:`, `env:`, `branch:`, free text) |

## Detail & logs

| Keys | Action |
|------|--------|
| `←` / `→` | Previous / next tab |
| `l` | Runtime logs tab |
| `L` / `Enter` on deployment | Build log |
| `Space` | Pause / resume log polling |
| `f` | Follow tail on/off |
| `w` | Wrap long lines |
| `/` · `n` · `N` | Search · next · previous match |
| `c` | Copy buffer or current search match |
| `Ctrl+L` | Clear local log buffer |
| `+` / `-` | Fetch more / fewer lines (session only) |

## Deployments (top-level)

| Keys | Action |
|------|--------|
| `Enter` | Open build log for selected deployment |
| `a` | Toggle active-only / recent history |
| `c` | Copy deployment UUID |

## Fleet tail

| Keys | Action |
|------|--------|
| `space` | Pause / resume polling |
| `f` | Follow the newest line |
| `w` | Wrap long lines |
| `/` | Search the merged buffer |
| `c` | Copy the buffer |
| `esc` / `q` | Back to applications |

## Instances

| Keys | Action |
|------|--------|
| `Enter` | Switch session to selected instance |
| `T` | Test active connection |
| `a` | Add instance (modal; token → keyring) |
| `e` | Edit name and URL |
| `d` | Delete local config entry (confirm) |

## Diagnostics

| Keys | Action |
|------|--------|
| `c` | Copy diagnostics text |
| `e` | Export to state directory |

## Demo only

| Keys | Action |
|------|--------|
| `F2` | Toggle simulated outage |

## Modals

| Keys | Action |
|------|--------|
| `Enter` | Confirm dangerous action |
| `Esc` / `q` | Cancel |

Command palette: type to filter, `↑`/`↓` or `Ctrl+N`/`Ctrl+P` to move,
`Enter` to run, `Esc` to close. Disabled commands show a reason and do not run.
