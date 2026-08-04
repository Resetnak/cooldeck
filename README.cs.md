# CoolDeck

**Rychlý terminálový dashboard pro [Coolify](https://coolify.io) ovládaný klávesnicí.**

Sledujte deploymenty, otevírejte logy, restarujte aplikace a přepínejte instance — aniž byste opustili terminál.

[![CI](https://github.com/resetnak/cooldeck/actions/workflows/ci.yml/badge.svg)](https://github.com/resetnak/cooldeck/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/resetnak/cooldeck)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> 🇬🇧 **English version:** [README.md](README.md)

```text
┌ cooldeck · production · online ────────────────────────── 12 apps · 3s ago ┐
│ Applications │ Deployments │ Instances │ Diagnostics                        │
│                                                                             │
│  STATUS      APPLICATION         PROJECT / ENV     BRANCH    DEPLOYED       │
│  ▶ Running   stimustop-api       StimuStop / prod  main      3m ago         │
│    Failed    weektraq-web        Weektraq / stage  develop   2h ago         │
│                                                                             │
└ ↑↓ pohyb  enter detail  / filtr  d deploy  : příkazy  ? nápověda ──────────┘
```

## Proč CoolDeck?

| | |
|--|--|
| **Rychlý** | Přímo Coolify REST API — bez prohlížeče, bez MCP |
| **Bezpečný** | Tokeny v OS keyringu; nebezpečné akce vždy s potvrzením |
| **Vychytaný** | Charm v2 UI, témata, command palette, responzivní layout |
| **Na demo** | `cooldeck --demo` běží offline s realistickými daty |

## Instalace

**Požadavek:** Go **1.26.5+**

```bash
git clone https://github.com/resetnak/cooldeck.git
cd cooldeck
make build
./bin/cooldeck --demo
```

Release binárky (Linux / macOS / Windows) vznikají přes GoReleaser na tagech `v*`.

```bash
go install github.com/resetnak/cooldeck/cmd/cooldeck@latest   # po publikaci
```

## Start za minutu

```bash
# 1) Prohlídka bez Coolify
cooldeck --demo

# 2) Skutečná instance — interaktivní průvodce
cooldeck setup

# 3) Denní použití
cooldeck
cooldeck --instance production
cooldeck --theme catppuccin
```

**Coolify token:** vytvořte API token v Coolify (profil → API tokens). Preferujte nejmenší nutná oprávnění (`read` + jen ty write scope, které opravdu potřebujete).

## Funkce

- **Aplikace** — stav, projekt/env, branch, poslední deploy, doména; filtr a řazení
- **Detail** — přehled, historie deploymentů, runtime logy, konfigurace
- **Logy** — follow / pauza / wrap / hledání / kopírování / clear / `+/-` počet řádků
- **Akce** — deploy, force deploy, restart, start/stop (s potvrzením)
- **Deployments** — nedávná historie + filtr jen aktivní (`a`)
- **Instance** — přepnutí v session, test spojení, **přidání / úprava / smazání** lokální config
- **Diagnostics** — výpis prostředí; kopírování nebo export (bez secretů)
- **Command palette** (`:` / `Ctrl+K`) a **nápověda** (`?`)
- **Témata** — auto, dark, light, Dracula, Catppuccin, Nord, Gruvbox, Tokyo Night

## Klávesy (základ)

| Klávesy | Akce |
|:--------|:-----|
| `j` `k` · `↑` `↓` | Pohyb |
| `enter` · `esc` | Otevřít · zpět |
| `/` | Filtr nebo hledání v logách |
| `:` · `Ctrl+K` | Command palette |
| `?` | Nápověda |
| `1`–`4` | Apps · Deploys · Instance · Diagnostics |
| `d` `D` `r` `s` | Deploy · force · restart · start/stop |
| `l` `L` | Runtime logy · build log |
| `c` | Kopírovat UUID / logy |
| `Ctrl+T` · `Ctrl+W` | Téma · kompaktní režim |

Kompletní mapa: v aplikaci `?` nebo [docs/keybindings.md](docs/keybindings.md).

## Konfigurace

```bash
cooldeck config path
cooldeck config validate
```

```toml
version = 1
default_instance = "production"
theme = "auto"
refresh_interval = "10s"

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"    # doporučeno
token_key = "production"
```

Tokeny: **keyring** (výchozí přes setup/auth) · `command` · `env` · `plaintext` (nedoporučeno).  
Jednorázově: `COOLDECK_TOKEN=… cooldeck`.

Podrobnosti: [docs/configuration.md](docs/configuration.md)

## CLI

```text
cooldeck                 TUI dashboard
cooldeck --demo          offline demo data
cooldeck setup           průvodce prvním spuštěním
cooldeck theme           výběr tématu
cooldeck auth add|status|delete <instance>
cooldeck config path|validate
cooldeck version
```

## Bezpečnost

- Tokeny se nikdy nezobrazují v UI, logách, toastech ani diagnostics
- Výstup logů se sanitizuje (žádné syrové ANSI řídicí sekvence)
- V prohlížeči se otevírají jen `http`/`https` URL
- Smazání instance maže jen **lokální** config (+ keyring), ne data v Coolify

Viz [SECURITY.md](SECURITY.md).

## Vývoj

```bash
make check               # fmt + vet + test + build
make test-race
make test-update-golden  # obnovit UI snapshoty
make run                 # --demo
make bench
```

| Dokument | |
|----------|--|
| [Architektura](docs/architecture.md) | Vrstvy a tok zpráv |
| [Coolify API](docs/coolify-api.md) | Integrace |
| [Řešení problémů](docs/troubleshooting.md) | Časté chyby |
| [Contributing](CONTRIBUTING.md) | Workflow pro PR |
| [Changelog](CHANGELOG.md) | Co se změnilo |
| [Specifikace](CLAUDE_CODE_COOLIFY_TUI_SPEC.md) | Kompletní produktová specifikace |

## Roadmapa

**Hotovo:** dashboard, logy, mutace, instance (switch/add/edit/delete), diagnostics, goldeny, CI.

**Další:** další token sources v TUI, volitelné read-only services/databases/servers, MCP adaptér nad stejným `app.Service`.

## Licence

[MIT](LICENSE) · příspěvky vítány.
