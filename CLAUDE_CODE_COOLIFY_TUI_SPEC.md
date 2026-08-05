# Claude Code zadání: Coolify TUI

> Kompletní implementační specifikace pro vytvoření moderní, vizuálně kvalitní a bezpečné terminálové aplikace pro monitoring a základní správu aplikací nasazených přes Coolify.

---

## 0. Instrukce pro Claude Code

Jsi seniorní Go vývojář se zkušeností s tvorbou produkčních CLI/TUI aplikací, návrhem modulární architektury, bezpečnou integrací REST API a testováním terminálových uživatelských rozhraní.

Tvým úkolem je navrhnout a implementovat open-source TUI aplikaci pro Coolify podle této specifikace.

### Způsob práce

1. Nejprve analyzuj celý dokument.
2. Prohlédni aktuální repozitář a respektuj existující strukturu, pokud už nějaká existuje.
3. Pokud je repozitář prázdný, inicializuj nový Go projekt.
4. Implementuj práci po malých, logických a samostatně testovatelných krocích.
5. Po každém významném kroku spusť relevantní testy, `go test ./...`, `go vet ./...` a formatter.
6. Nevytvářej placeholdery, které vypadají jako hotová implementace.
7. Nepoužívej pseudokód tam, kde lze dodat funkční řešení.
8. Nezakrývej chyby tichým fallbackem. Chyby musí být buď zobrazeny uživateli, zalogovány, nebo explicitně ošetřeny.
9. Nedělej rozsáhlé změny mimo právě řešený úkol.
10. Při nejasnosti zvol rozumné technické rozhodnutí, zapiš jej do `docs/decisions/` a pokračuj.
11. Neimplementuj MCP server v první verzi. Architekturu ale připrav tak, aby šlo aplikační služby později použít z TUI, CLI i MCP adaptéru.
12. Před použitím konkrétního endpointu ověř jeho aktuální podobu v oficiální Coolify API dokumentaci nebo OpenAPI schématu. API klient nesmí být postaven pouze na domněnkách z této specifikace.

### Hlavní zásada

Aplikace nesmí být pouze funkční. Musí působit jako profesionální terminálový produkt s úrovní zpracování podobnou nástrojům `lazygit`, `k9s`, `lazydocker` nebo moderním aplikacím z ekosystému Charm.

---

# 1. Produkt

## 1.1 Pracovní název

Používej pracovní název:

```text
cooldeck
```

Název musí být centralizovaný tak, aby jej bylo možné později snadno změnit.

Binární soubor:

```text
cooldeck
```

Go module použij podle skutečného GitHub repozitáře. Pokud není známý, použij dočasně:

```text
github.com/resetnak/cooldeck
```

Před publikací musí být možné module path jednoduše změnit.

## 1.2 Jednověté vymezení

> A fast and polished terminal dashboard for monitoring and operating Coolify deployments.

## 1.3 Primární cíl

Uživatel má být schopný z terminálu:

- rychle zjistit stav všech aplikací,
- rozpoznat probíhající nebo neúspěšný deployment,
- zobrazit detail aplikace,
- zobrazit runtime logy,
- zobrazit historii deploymentů a build logy,
- spustit deploy,
- restartovat, zastavit nebo spustit aplikaci,
- otevřít doménu nebo repozitář,
- pracovat s více Coolify instancemi,
- provádět běžné operace bez otevírání webového dashboardu.

## 1.4 Necíle první verze

V první verzi neimplementuj:

- plnohodnotnou náhradu administračního rozhraní Coolify,
- tvorbu a mazání aplikací,
- editaci environment variables,
- práci se secrets,
- SSH přístup na server,
- přímou komunikaci s Docker socketem,
- historické CPU/RAM metriky,
- Prometheus integraci,
- AI analýzu logů,
- MCP server,
- plugin systém,
- mobilní nebo webové UI.

Architektura však nesmí budoucímu rozšíření zbytečně bránit.

---

# 2. Technologický stack

## 2.1 Jazyk

Použij Go.

Požadavky:

- aktuální stabilní Go verze podporovaná CI,
- minimální podporovaná verze jasně uvedená v `go.mod` a README,
- idiomatický Go kód,
- žádné zbytečné frameworky nebo runtime dependency injection kontejnery.

## 2.2 TUI stack

Použij aktuální stabilní generaci Charm v2:

```text
charm.land/bubbletea/v2
charm.land/bubbles/v2
charm.land/lipgloss/v2
```

Nepoužívej staré importy `github.com/charmbracelet/...` pro Bubble Tea v1, pokud k tomu není doložený technický důvod.

Bubble Tea v2 využij jako aplikační runtime a event loop. Lip Gloss v2 použij pro layout, border, padding, zarovnání a styly. Bubbles v2 použij tam, kde přináší kvalitní hotové komponenty, například:

- viewport,
- text input,
- spinner,
- help,
- progress,
- list nebo table, pokud vyhovuje požadovanému UX.

Pokud vestavěný `table` nebo `list` komponent neposkytne potřebný vzhled, navigaci nebo výkon, vytvoř vlastní komponentu nad Bubble Tea a Lip Gloss. Nevynucuj použití komponenty pouze proto, že existuje.

## 2.3 Doporučené knihovny

Preferované knihovny:

```text
github.com/spf13/cobra               CLI příkazy a flags
github.com/spf13/viper               konfigurace, pokud skutečně pomůže
github.com/go-resty/resty/v2         HTTP klient, nebo kvalitní vlastní net/http wrapper
github.com/zalando/go-keyring        OS keychain
github.com/charmbracelet/glamour     volitelné renderování Markdownu
github.com/mattn/go-runewidth        práce se šířkou Unicode, pokud ji nepokryje stack
github.com/stretchr/testify          testovací assertions, pouze pokud přináší hodnotu
go.uber.org/zap nebo log/slog        strukturované logování; preferuj standardní slog
github.com/google/go-cmp/cmp         porovnání v testech
github.com/golang/mock nebo mockery  pouze pokud jsou opravdu potřeba
github.com/adrg/xdg                  XDG cesty, pokud není použit čistý os.UserConfigDir
```

Pro konfiguraci zvaž jednoduchý TOML nebo YAML formát. Preferuj TOML, protože je čitelný a vhodný pro několik pojmenovaných instancí.

Nevkládej knihovnu do projektu pouze kvůli jediné triviální funkci.

## 2.4 Build a distribuce

Připrav:

- `Makefile`,
- případně `Taskfile.yml`, ale Makefile musí existovat,
- GoReleaser konfiguraci,
- GitHub Actions pro testy a release,
- cross-platform build pro Linux, macOS a Windows,
- architektury minimálně `amd64` a `arm64`,
- release archivy a checksum soubor.

---

# 3. Coolify integrace

## 3.1 Základní princip

TUI musí komunikovat přímo s Coolify REST API.

MCP není transportní vrstva TUI. Budoucí MCP server bude pouze další adaptér nad stejnými aplikačními službami.

Základní URL:

```text
https://<coolify-host>/api/v1
```

Autorizace:

```http
Authorization: Bearer <token>
Accept: application/json
```

## 3.2 Bezpečnostní oprávnění

Podporuj princip nejnižšího oprávnění.

Očekávaná oprávnění:

- `read` pro seznamy a detaily,
- `read:sensitive` jen tehdy, pokud je podle aktuální Coolify verze nutné pro logy,
- `deploy` pro spuštění deploymentu,
- `write` pro start, stop nebo restart, pokud je endpoint vyžaduje,
- `root` aplikace nikdy aktivně nevyžaduje ani nedoporučuje.

Při chybě `403` zobraz:

- jaká operace selhala,
- které oprávnění pravděpodobně chybí,
- stručnou možnost nápravy,
- bez vypsání tokenu nebo citlivého response body.

## 3.3 API klient

Vytvoř rozhraní, například:

```go
type CoolifyClient interface {
    Health(ctx context.Context) error
    ListApplications(ctx context.Context) ([]Application, error)
    GetApplication(ctx context.Context, uuid string) (Application, error)
    GetApplicationLogs(ctx context.Context, uuid string, lines int) (string, error)

    ListDeployments(ctx context.Context, filter DeploymentFilter) ([]Deployment, error)
    GetDeployment(ctx context.Context, uuid string) (Deployment, error)
    GetDeploymentLogs(ctx context.Context, uuid string) (string, error)

    DeployApplication(ctx context.Context, uuid string, options DeployOptions) (DeploymentRequest, error)
    RestartApplication(ctx context.Context, uuid string) (OperationResult, error)
    StartApplication(ctx context.Context, uuid string, options StartOptions) (OperationResult, error)
    StopApplication(ctx context.Context, uuid string) (OperationResult, error)

    ListProjects(ctx context.Context) ([]Project, error)
    ListServers(ctx context.Context) ([]Server, error)
    ListServices(ctx context.Context) ([]Service, error)
    ListDatabases(ctx context.Context) ([]Database, error)
}
```

Konkrétní podpisy uprav podle skutečného API a potřeb domény. Rozhraní nesmí kopírovat každou vlastnost HTTP klienta. Má reprezentovat schopnosti aplikace.

## 3.4 Endpointy

Při implementaci ověř aktuální dokumentaci. Minimálně očekávej potřebu následujících kategorií:

```text
GET  /applications
GET  /applications/{uuid}
GET  /applications/{uuid}/logs?lines=<n>
GET/POST /applications/{uuid}/start
GET/POST /applications/{uuid}/stop
GET/POST /applications/{uuid}/restart

GET  /deployments
GET  /deployments/{uuid}
GET  /deployments/{uuid}/logs nebo odpovídající aktuální endpoint
POST /deploy nebo odpovídající aktuální deploy endpoint

GET  /projects
GET  /servers
GET  /services
GET  /databases
```

Názvy a HTTP metody neber jako definitivní. Zdroj pravdy je oficiální API dokumentace/OpenAPI schéma konkrétní podporované verze Coolify.

## 3.5 Odolnost API klienta

Implementuj:

- kontext a cancellation,
- timeout pro každý request,
- rozumný connection pooling,
- validaci base URL,
- přidání `/api/v1` bez duplikace,
- bezpečnou práci s trailing slash,
- omezené retry pouze pro idempotentní read requesty,
- exponential backoff s jitterem,
- respektování `Retry-After`, pokud je přítomný,
- explicitní ošetření `401`, `403`, `404`, `409`, `422`, `429` a `5xx`,
- limit velikosti response body,
- dekódování JSON chyb,
- redakci tokenu a secrets z logů,
- volitelný debug HTTP log bez Authorization hlavičky.

Neretryuj automaticky mutační operace, pokud by mohly být provedeny dvakrát.

## 3.6 Rate limiting a refresh

Coolify má ve výchozím stavu globální API limit. Aplikace nesmí provádět N+1 dotazy při každém refreshi.

Požadavky:

- výchozí refresh seznamu: 10 sekund,
- konfigurovatelné minimum: 3 sekundy,
- detail aktivní aplikace načítej odděleně,
- polling logů dělej pouze při otevřeném log view,
- při minimalizaci nebo ztrátě focusu můžeš polling zpomalit,
- při `429` použij backoff a zobraz neinvazivní stav,
- manuální refresh přes `R` musí být okamžitý, ale nesmí vytvářet paralelní duplicitní requesty,
- starší request zruš, pokud jej nahrazuje novější request stejného typu.

## 3.7 Verzování a capability detection

Pokud Coolify poskytuje endpoint s verzí systému, načti jej při připojení.

Vytvoř koncept capabilities:

```go
type Capabilities struct {
    ApplicationLogs bool
    DeploymentLogs  bool
    Restart         bool
    StartStop       bool
    Deploy          bool
    Services        bool
    Databases       bool
}
```

UI nesmí nabízet akci, kterou instance nebo token neumí. Zakázaná akce musí být:

- skrytá, nebo
- viditelná jako disabled s vysvětlením.

Preferuj disabled stav v command palette a help panelu, protože uživatel lépe pochopí, že funkce existuje, ale není dostupná.

---

# 4. Doménový model

Nepoužívej HTTP response structs přímo v UI.

Rozděl:

1. API DTO,
2. doménové modely,
3. view modely/formátované hodnoty.

Minimální doménové typy:

```go
type ResourceStatus string

const (
    StatusRunning    ResourceStatus = "running"
    StatusStopped    ResourceStatus = "stopped"
    StatusStarting   ResourceStatus = "starting"
    StatusStopping   ResourceStatus = "stopping"
    StatusRestarting ResourceStatus = "restarting"
    StatusBuilding   ResourceStatus = "building"
    StatusDeploying  ResourceStatus = "deploying"
    StatusFailed     ResourceStatus = "failed"
    StatusDegraded   ResourceStatus = "degraded"
    StatusUnknown    ResourceStatus = "unknown"
)
```

```go
type Application struct {
    UUID            string
    Name            string
    Description     string
    Status          ResourceStatus
    FQDNs           []string
    RepositoryURL   string
    Branch          string
    CommitSHA       string
    BuildPack       string
    Project         ResourceRef
    Environment     ResourceRef
    Server          ResourceRef
    HealthCheck     HealthCheck
    LastDeployment  *DeploymentSummary
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

```go
type Deployment struct {
    UUID          string
    ApplicationID string
    Status        DeploymentStatus
    CommitSHA     string
    CommitMessage string
    Trigger       string
    StartedAt     time.Time
    FinishedAt    *time.Time
    Duration      time.Duration
    CreatedAt     time.Time
}
```

Všechny parsované statusy normalizuj přes jednu mapovací vrstvu. Neznámý status nesmí způsobit pád aplikace.

---

# 5. Architektura

## 5.1 Vrstvy

Použij pragmatickou clean/hexagonal architekturu bez přehnaného množství abstrakcí.

Doporučené vrstvy:

```text
cmd/cooldeck/             entrypoint
internal/app/             aplikační orchestrace/use cases
internal/domain/          doménové modely a pravidla
internal/coolify/         REST klient, DTO, mapování
internal/config/          konfigurace a validace
internal/credentials/     keychain/env/command token providers
internal/tui/             Bubble Tea root model
internal/tui/components/  opakovaně použitelné UI komponenty
internal/tui/views/       obrazovky
internal/tui/theme/       design tokens a styly
internal/platform/        browser, clipboard, OS integrace
internal/logging/         bezpečné logování
internal/version/         build info
pkg/                      pouze pokud vznikne skutečně veřejné API
```

## 5.2 Aplikační služby

UI nesmí volat Coolify HTTP klient přímo.

Vytvoř například:

```go
type ApplicationService interface {
    Dashboard(ctx context.Context, instanceID string) (DashboardSnapshot, error)
    ApplicationDetail(ctx context.Context, instanceID, appUUID string) (ApplicationDetail, error)
    RuntimeLogs(ctx context.Context, instanceID, appUUID string, lines int) (LogSnapshot, error)
    Deployments(ctx context.Context, instanceID, appUUID string) ([]Deployment, error)
    DeploymentLogs(ctx context.Context, instanceID, deploymentUUID string) (LogSnapshot, error)
    Deploy(ctx context.Context, instanceID, appUUID string, force bool) (OperationResult, error)
    Restart(ctx context.Context, instanceID, appUUID string) (OperationResult, error)
    Start(ctx context.Context, instanceID, appUUID string) (OperationResult, error)
    Stop(ctx context.Context, instanceID, appUUID string) (OperationResult, error)
}
```

To umožní později přidat:

```text
TUI adapter
CLI adapter
MCP adapter
```

bez duplikace business logiky.

## 5.3 Stav TUI

Root model musí vlastnit:

- aktivní instanci,
- aktivní obrazovku,
- focus,
- dimensions,
- theme,
- globální loading stav,
- toast/notifikace,
- modal stack,
- command palette,
- background jobs,
- error state,
- poslední úspěšný refresh.

Nevytvářej jeden gigantický `model.go`. Rozděl view models a komponenty podle odpovědnosti.

## 5.4 Zprávy a příkazy

Používej explicitní Bubble Tea messages, například:

```go
type applicationsLoadedMsg struct {
    RequestID string
    Apps      []domain.Application
    LoadedAt  time.Time
}

type applicationsLoadFailedMsg struct {
    RequestID string
    Err       error
}

type tickMsg time.Time

type operationCompletedMsg struct {
    Operation string
    Resource  string
    Result    app.OperationResult
}
```

Každá async operace musí být identifikovatelná. Pozdní odpověď starého requestu nesmí přepsat novější data.

---

# 6. Konfigurace a credentials

## 6.1 Umístění konfigurace

Použij standardní OS/XDG cesty.

Příklady:

```text
Linux:  ~/.config/cooldeck/config.toml
macOS:  ~/Library/Application Support/cooldeck/config.toml
Windows: %AppData%\cooldeck\config.toml
```

Povol také:

```bash
cooldeck --config /custom/path/config.toml
```

## 6.2 Formát konfigurace

Navrhni například:

```toml
version = 1
default_instance = "production"
refresh_interval = "10s"
log_refresh_interval = "2s"
log_lines = 300
theme = "auto"
confirm_destructive_actions = true

[ui]
show_header = true
show_footer = true
nerd_font = "auto"
compact_mode = "auto"
mouse = true

[instances.production]
name = "Production"
url = "https://coolify.example.com"
token_source = "keyring"
token_key = "production"

[instances.staging]
name = "Staging"
url = "https://staging-coolify.example.com"
token_source = "command"
token_command = ["gopass", "show", "coolify/staging"]
refresh_interval = "5s"
```

## 6.3 Zdroje tokenu

Podporuj v tomto pořadí:

1. explicitní CLI environment variable,
2. OS keychain,
3. externí příkaz vracející token na stdout,
4. environment variable definovanou v configu,
5. plaintext token pouze jako explicitní opt-in a s výrazným bezpečnostním warningem.

Příklady:

```bash
COOLDECK_TOKEN=... cooldeck
cooldeck auth add production
cooldeck auth delete production
cooldeck auth status
```

Token:

- nikdy nezobrazuj,
- nevypisuj do logu,
- nevkládej do chyb,
- při debug HTTP logu rediguj,
- nenechávej v crash reportu,
- pokud je spouštěn `token_command`, trimuj pouze okolní whitespace.

## 6.4 První spuštění

Když konfigurace neexistuje, spusť interaktivní onboarding:

1. název instance,
2. Coolify URL,
3. výběr zdroje tokenu,
4. bezpečné zadání tokenu bez echo,
5. test připojení,
6. uložení konfigurace,
7. otevření dashboardu.

Onboarding musí působit jako součást stejného vizuálního produktu, ne jako obyčejná sekvence `fmt.Scan`.

---

# 7. UX a informační architektura

## 7.1 Hlavní obrazovky

Implementuj minimálně:

```text
Dashboard / Applications
Application detail
Deployments
Runtime logs
Deployment logs
Instances
Settings / diagnostics
Help
```

V MVP mohou být Services, Databases a Servers pouze v read-only přehledu nebo připravené jako následný milestone. Applications musí být dotažené.

## 7.2 Hlavní layout

Pro běžný terminál použij layout:

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Logo / instance / connection / last refresh                  global actions │
├──────────────────────┬───────────────────────────────────────────────────────┤
│ Navigation/sidebar   │ Main content                                          │
│                      │                                                       │
│ Applications         │ Resource table / detail / logs                        │
│ Deployments          │                                                       │
│ Services             │                                                       │
│ Databases            │                                                       │
│ Servers              │                                                       │
│                      │                                                       │
├──────────────────────┴───────────────────────────────────────────────────────┤
│ Contextual help / status / operation progress                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

Na užších terminálech sidebar skryj a navigaci převeď na tabs nebo command palette.

## 7.3 Responsivní breakpointy

Navrhni minimálně tři režimy:

### Wide

```text
šířka >= 120 znaků
```

- sidebar,
- tabulka s více sloupci,
- vedlejší preview/detail panel,
- plný footer.

### Standard

```text
šířka 80–119 znaků
```

- kompaktní sidebar nebo horní tabs,
- redukované sloupce,
- detail na samostatné obrazovce.

### Compact

```text
šířka < 80 znaků nebo výška < 24 řádků
```

- single-column,
- minimum borderů,
- zkrácené labels,
- žádné horizontální scrollování pro základní navigaci,
- jasná hláška, pokud je terminál už nepoužitelně malý.

Minimální podporovaná velikost může být například 60×18. Pod touto hranicí zobraz elegantní obrazovku s požadavkem na zvětšení terminálu.

## 7.4 Navigace

Podporuj:

```text
j / ↓            další položka
k / ↑            předchozí položka
g                první položka
G                poslední položka
Ctrl+d / PgDn    posun dolů
Ctrl+u / PgUp    posun nahoru
Enter            otevřít detail / potvrdit
Esc              zpět / zavřít modal
Tab              další panel
Shift+Tab        předchozí panel
/                filtrovat
: nebo Ctrl+k    command palette
?                help
R                manuální refresh
q                zpět nebo ukončit
Ctrl+c           bezpečné ukončení
```

Klávesy musí být centralizované v key mapě. Nevkládej string comparisons nahodile po celém projektu.

## 7.5 Command palette

Command palette je povinná.

Po otevření musí nabídnout kontextové příkazy, například:

```text
> deploy
  Deploy selected application
  Force deploy selected application
  Open deployments
  Open runtime logs
  Restart application
  Stop application
  Switch instance
  Toggle compact mode
  Toggle theme
  Open help
```

Požadavky:

- fuzzy filtrování,
- disabled příkazy s důvodem,
- klávesová zkratka u položky,
- bezpečné potvrzení nebezpečných akcí,
- příkazy registrované centralizovaně, ne hardcoded v rendereru.

## 7.6 Filtrace a vyhledávání

Dashboard filtruje minimálně podle:

- názvu,
- domény,
- projektu,
- environmentu,
- branch,
- statusu.

Podporuj jednoduché tokeny:

```text
status:failed
status:running
project:shipyard
env:production
branch:main
```

Běžný text prohledává všechny relevantní sloupce.

## 7.7 Modaly

Použij modaly pro:

- potvrzení deploye,
- force deploy,
- restart,
- stop,
- start,
- zobrazení chyby s detailem,
- výběr instance,
- command palette.

Modal musí vizuálně vystoupit nad pozadí, ale nepoužívej skutečnou průhlednost, pokud ji terminál spolehlivě nepodporuje. Pozadí může být vizuálně ztlumené přestylováním.

## 7.8 Toasty a stavové zprávy

Použij nenásilné toasty:

```text
✓ Deployment queued
✓ Application restarted
! API rate limit reached; retrying in 12s
× Failed to load deployments
```

Typy:

- success,
- info,
- warning,
- error.

Toast:

- nesmí překrývat kritické ovládací prvky,
- má automaticky zmizet,
- error může zůstat déle,
- detail chyby musí jít otevřít,
- stack toastů musí mít limit.

---

# 8. Vizuální design

## 8.1 Designový směr

Aplikace má působit:

- čistě,
- technicky,
- moderně,
- profesionálně,
- mírně futuristicky,
- bez přeplácaných ASCII dekorací.

Inspirace:

- moderní Charm aplikace,
- lazygit,
- k9s,
- GitHub CLI,
- Linear,
- Vercel dashboard,
- Coolify branding, ale bez kopírování chráněných assetů.

## 8.2 Design tokens

Vytvoř vlastní theme package. Styly nesmí být rozházené po view souborech.

Příklad tokenů:

```go
type Palette struct {
    Background       lipgloss.Color
    Surface          lipgloss.Color
    SurfaceRaised    lipgloss.Color
    Border           lipgloss.Color
    BorderFocused    lipgloss.Color
    Text             lipgloss.Color
    TextMuted        lipgloss.Color
    TextSubtle       lipgloss.Color
    Primary          lipgloss.Color
    Secondary        lipgloss.Color
    Success          lipgloss.Color
    Warning          lipgloss.Color
    Error            lipgloss.Color
    Info             lipgloss.Color
    Selection        lipgloss.Color
    SelectionText    lipgloss.Color
}
```

Dále definuj:

- spacing scale,
- border styles,
- typography roles,
- status styles,
- badge styles,
- focus styles,
- selected row styles.

## 8.3 Dark/light theme

Podporuj:

```text
auto
dark
light
```

V režimu `auto` použij Bubble Tea v2 background color detection a na základě výsledku vyber odpovídající theme.

Nepředpokládej tmavé pozadí.

## 8.4 Barvy

Použij barvy významově, ne dekorativně.

Doporučené významy:

```text
green    running / success
orange   building / warning
red      failed / stopped kvůli chybě / destructive action
blue     info / selected neutral action
purple   primary accent / deployments
muted    secondary metadata
```

Stav nesmí být vyjádřen pouze barvou. Vždy přidej symbol nebo text.

## 8.5 Status indikátory

Například:

```text
● Running
◐ Deploying
◌ Queued
■ Stopped
▲ Degraded
× Failed
? Unknown
```

Pokud Nerd Font není dostupný nebo je vypnutý, použij bezpečné Unicode/ASCII symboly.

Nevyžaduj Nerd Font pro základní použitelnost.

## 8.6 Tabulka aplikací

Wide varianta:

```text
 STATUS      APPLICATION        PROJECT / ENV       BRANCH       DEPLOYED       DOMAIN
 ● Running   shipyard-api      Shipyard / prod    main         3m ago         shipyard.example
 ◐ Building  landing-web       Personal / prod     main         21s            landing.example
 × Failed    billing-worker   Billing / prod    main         2h ago         -
```

Požadavky:

- sticky header v rámci viewportu, pokud to komponenta umožňuje,
- jasně vyznačený výběr,
- přirozené zkracování sloupců,
- žádné rozbití layoutu kvůli dlouhé doméně,
- Unicode-aware width,
- stav a název nikdy neschovávej dříve než méně důležité sloupce,
- volitelný sort podle statusu, názvu nebo posledního deploye.

## 8.7 Header

Header může vypadat například:

```text
 COOLDECK   Production  ● connected                  8 apps   refreshed 4s ago
```

Zobraz:

- název produktu,
- aktivní instanci,
- connection state,
- počet zdrojů,
- poslední refresh,
- případně aktivní filtr.

Header nesmí být zbytečně vysoký.

## 8.8 Footer

Footer má být kontextový:

```text
 ↑↓ navigate   enter details   / filter   d deploy   l logs   : commands   ? help
```

Na malém terminálu zobraz pouze nejdůležitější zkratky a `? more`.

## 8.9 Empty states

Připrav kvalitní empty states:

### Žádné aplikace

```text
No applications found on this Coolify instance.

Check the active team or open the Coolify dashboard to create one.
```

### Filtr bez výsledků

```text
No applications match “status:failed”.
Press Esc to clear the filter.
```

### Offline

```text
Cannot reach Production.

Last successful refresh: 10:32:14
Retrying automatically…

[R] Retry now   [i] Switch instance   [e] View error
```

## 8.10 Loading states

Nepoužívej pouze globální spinner.

Použij:

- skeleton-like řádky nebo placeholder text při prvním načtení,
- malý spinner v headeru při background refreshi,
- progress/status u mutačních operací,
- zachování starých dat při refreshi, pokud jsou dostupná.

Při background refreshi nevyprázdňuj obrazovku.

---

# 9. Obrazovky

## 9.1 Dashboard / Applications

### Obsah

- seznam aplikací,
- status,
- project/environment,
- branch,
- poslední deployment,
- primární doména,
- indikace aktivního deploymentu,
- connection state.

### Akce

```text
Enter    detail aplikace
d        deploy
D        force deploy
r        restart
s        start/stop podle stavu
l        runtime logy
L        poslední deployment log
b        otevřít doménu v browseru
g        otevřít Git repository
/        filtr
R        refresh
```

Mutační akce musí být dostupné pouze při správném oprávnění.

## 9.2 Detail aplikace

Rozděl na sekce nebo tabs:

```text
Overview
Deployments
Runtime Logs
Configuration
```

Overview:

- název a status,
- UUID s možností copy,
- domény,
- repository,
- branch,
- commit SHA,
- build pack,
- project,
- environment,
- server,
- health check,
- poslední deployment,
- created/updated.

Citlivá data nezobrazuj.

## 9.3 Deployments

Tabulka:

```text
STATUS      STARTED       DURATION    COMMIT      TRIGGER
✓ Finished  3m ago        1m 24s      a91fc80     git push
× Failed    2h ago        42s         7c201ab     manual
◐ Running   18s ago       18s         bf80aa1     manual
```

Akce:

```text
Enter    deployment detail/log
l        log
c        copy deployment UUID
R        refresh
```

## 9.4 Runtime logs

Požadavky:

- monospaced viewport,
- tail-like režim,
- pauza/resume,
- autoscroll pouze když uživatel stojí na konci,
- při ručním scrollu nezatahuj uživatele zpět dolů,
- vyhledávání v aktuálně načtených logách,
- přepnutí wrap/no-wrap,
- změna počtu řádků,
- copy vybraného řádku nebo viditelného bloku,
- clear local buffer,
- timestamp highlighting,
- zvýraznění `ERROR`, `WARN`, `INFO`, `DEBUG` bez závislosti na přesném formátu,
- žádné vykonávání ANSI escape sekvencí z nedůvěryhodných logů.

Bezpečně sanitizuj control characters a terminálové escape sekvence.

Klávesy:

```text
space    pause/resume
f        follow on/off
w        wrap on/off
/        search
n/N      next/previous match
+/-      zvýšit/snížit počet řádků nebo refresh rate dle kontextu
```

## 9.5 Deployment logs

Stejná log komponenta jako runtime logs, ale s:

- deployment metadata headerem,
- duration,
- commit,
- triggerem,
- výsledkem,
- možností znovu spustit deploy.

API může vracet log jako celý string, ne stream. UI tomu musí přizpůsobit polling a diffování bez blikání.

## 9.6 Instances

Zobraz:

- název,
- URL,
- connection state,
- Coolify verzi, pokud je dostupná,
- team, pokud je dostupný,
- auth status bez tokenu,
- poslední latency,
- poslední error.

Akce:

```text
Enter    přepnout instanci
T        test connection
a        add instance
e        edit non-secret settings
d        delete instance po potvrzení
```

Mazání instance nesmí smazat Coolify data, pouze lokální konfiguraci a volitelně credential z keychainu.

## 9.7 Diagnostics

Implementuj read-only diagnostickou obrazovku:

- cooldeck verze,
- Go verze,
- OS/arch,
- config path,
- log path,
- active instance URL bez tokenu,
- connection status,
- detected capabilities,
- terminal dimensions,
- color profile,
- dark/light detection,
- mouse enabled,
- Nerd Font mode,
- posledních několik bezpečně redigovaných aplikačních chyb.

Přidej možnost exportovat diagnostiku do souboru bez secrets.

---

# 10. Akce a bezpečnost

## 10.1 Potvrzování

Vyžaduj potvrzení minimálně pro:

- force deploy,
- restart production aplikace,
- stop aplikace,
- odstranění lokální instance,
- jakoukoli budoucí destruktivní operaci.

Běžný deploy může mít potvrzení konfigurovatelné.

Modal příklad:

```text
Restart application?

billing-api · Production
Active requests may be interrupted.

[Esc] Cancel                 [Enter] Restart
```

Pro stop produkční aplikace zvaž explicitní napsání názvu jen jako volitelný „strict safety mode“, ne výchozí UX.

## 10.2 Production awareness

Environment s názvem odpovídajícím `prod`, `production`, `live` apod. označ jako production.

Production akce:

- mají výraznější warning,
- vždy ukazují instance + project + environment + app,
- nesmí být spuštěny omylem opakováním klávesy,
- během probíhající operace stejnou akci disableuj.

## 10.3 Secrets

Nikdy nezobrazuj:

- environment variable values,
- private keys,
- tokeny,
- passwords,
- raw compose obsah, pokud může obsahovat secrets,
- response headers s credentials.

Pokud API omylem vrátí citlivá data, mapovací vrstva je nesmí ukládat do doménového modelu, pokud je UI nepotřebuje.

## 10.4 Otevírání URL

Před otevřením URL:

- akceptuj pouze `http` a `https`,
- repository může podporovat také bezpečně rozpoznanou Git URL převedenou na web URL,
- nepředávej nevalidovanou hodnotu shellu,
- používej `exec.Command` s argumenty, nikdy shell string concatenation.

---

# 11. Chybové stavy

Vytvoř vlastní typy chyb:

```go
type ErrorKind string

const (
    ErrorNetwork       ErrorKind = "network"
    ErrorTimeout       ErrorKind = "timeout"
    ErrorUnauthorized  ErrorKind = "unauthorized"
    ErrorForbidden     ErrorKind = "forbidden"
    ErrorNotFound      ErrorKind = "not_found"
    ErrorRateLimited   ErrorKind = "rate_limited"
    ErrorConflict      ErrorKind = "conflict"
    ErrorValidation    ErrorKind = "validation"
    ErrorServer        ErrorKind = "server"
    ErrorDecode        ErrorKind = "decode"
    ErrorUnsupported   ErrorKind = "unsupported"
    ErrorUnknown       ErrorKind = "unknown"
)
```

Uživatelská chyba musí obsahovat:

- krátký title,
- lidsky čitelnou zprávu,
- volitelný technický detail,
- retryability,
- suggested action,
- bezpečný request ID, pokud existuje.

Příklad:

```text
Permission denied

The Production token cannot restart applications.
Create a token with the required write permission or use a read-only action.
```

Nezobrazuj Go stack trace běžnému uživateli.

---

# 12. Cache a offline chování

## 12.1 In-memory cache

Udržuj poslední úspěšný snapshot:

- applications,
- selected app detail,
- deployments,
- runtime logs.

Při dočasném výpadku zobraz stará data s jasným označením:

```text
OFFLINE · showing data from 4m ago
```

## 12.2 Persistent cache

Pro MVP je persistent cache volitelná.

Pokud ji implementuješ:

- ukládej pouze necitlivé snapshoty,
- používej verzovaný formát,
- nastav rozumnou expiraci,
- atomické zápisy,
- oprávnění souboru omez podle OS,
- cache nesmí obsahovat token nebo raw secrets.

---

# 13. CLI rozhraní

TUI je výchozí režim:

```bash
cooldeck
```

Připrav také:

```bash
cooldeck --instance production
cooldeck --config /path/config.toml
cooldeck --debug
cooldeck --no-mouse
cooldeck --theme dark
cooldeck version
cooldeck doctor
cooldeck auth add <instance>
cooldeck auth delete <instance>
cooldeck auth status [instance]
cooldeck config path
cooldeck config validate
```

`doctor` musí:

- ověřit config,
- ověřit credential source,
- ověřit URL,
- otestovat API,
- zobrazit capabilities,
- nesmí vypsat token.

Exit codes musí být deterministické a dokumentované pro non-TUI subcommands.

---

# 14. Výkon

Požadavky:

- input reakce bez znatelného lagu,
- žádný HTTP request v `View()`,
- žádná těžká práce v renderovací cestě,
- nepřepočítávat statické Lip Gloss styly při každém renderu,
- filtrovaná data memoizovat nebo přepočítávat pouze při změně dat/filtru,
- velké logy držet v limitovaném ring bufferu,
- nedržet neomezenou historii toastů a chyb,
- žádné goroutine leaks,
- context cancellation při odchodu z view,
- graceful shutdown.

Cílové scénáře:

- 200 aplikací bez znatelného zpomalení,
- 1 000 deploymentů ve view,
- 20 000 log řádků s rozumnou navigací,
- práce přes SSH s omezenou šířkou pásma.

Bubble Tea v2 renderer využij bez obcházení jeho optimalizací zbytečnými full-screen změnami.

---

# 15. Přístupnost a kompatibilita

Podporuj:

- light i dark terminal,
- true color, 256 color i omezenější profily s graceful downsampling,
- klávesnici bez myši,
- volitelnou myš,
- ASCII-safe fallback symboly,
- čitelnost bez Nerd Fontu,
- zobrazení bez color-only významu,
- Windows Terminal,
- iTerm2,
- macOS Terminal,
- Kitty,
- WezTerm,
- Alacritty,
- GNOME Terminal,
- běh přes SSH a tmux.

Terminálové capabilities detekuj, ale vždy měj fallback.

---

# 16. Testování

## 16.1 Unit testy

Pokryj minimálně:

- status mapping,
- DTO → domain mapping,
- URL normalization,
- config validation,
- duration/time formatting,
- truncation a column sizing,
- filter parser,
- command availability podle capabilities,
- credential redaction,
- error mapping,
- retry rozhodování,
- log sanitization,
- production environment detection.

## 16.2 API testy

Použij `httptest.Server`.

Pokryj:

- Bearer header,
- správnou URL,
- úspěšné response,
- malformed JSON,
- prázdné body,
- 401,
- 403,
- 404,
- 429 + Retry-After,
- 500,
- timeout,
- cancellation,
- příliš velké response body,
- redakci tokenu.

Nevyžaduj reálnou Coolify instanci pro běžný test suite.

## 16.3 TUI testy

Testuj `Update` jako deterministický state machine.

Pokryj:

- navigaci,
- focus switching,
- otevření/zavření modalu,
- command palette,
- potvrzení/cancel akce,
- resize,
- compact mode,
- stale response ignorování,
- loading/error/offline state,
- autoscroll logů,
- pause/resume logů.

## 16.4 Golden tests

Použij golden/snapshot testy pro vybrané pohledy:

- dashboard wide dark,
- dashboard standard light,
- dashboard compact,
- empty state,
- offline state,
- app detail,
- deployment modal,
- logs view.

Golden testy musí normalizovat dynamický čas a další nestabilní hodnoty.

Přidej explicitní příkaz pro aktualizaci snapshots:

```bash
make test-update-golden
```

## 16.5 Race a quality checks

CI musí spouštět:

```bash
gofmt -w / gofmt check
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
govulncheck ./...
```

Pokud race test na některé platformě není vhodný, zdokumentuj výjimku.

---

# 17. Observabilita aplikace

## 17.1 Logování

Použij strukturované logování přes `log/slog`.

Výchozí režim:

- žádné logy do stdout/stderr během fullscreen TUI,
- logy do souboru v cache/state directory,
- rotace nebo velikostní limit,
- bezpečná redakce.

Debug režim:

```bash
cooldeck --debug
```

Loguj:

- start aplikace,
- verzi,
- aktivní instanci,
- request method + sanitizovanou URL,
- status code,
- duration,
- retry,
- state transitions významných operací,
- chyby.

Nikdy neloguj tokeny ani raw secrets.

## 17.2 Panic recovery

Na nejvyšší úrovni zachyť panic:

- obnov terminál do normálního stavu,
- ulož bezpečný crash log,
- zobraz stručnou zprávu a cestu k logu,
- vrať nenulový exit code.

---

# 18. Dokumentace

Vytvoř:

```text
README.md
CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
LICENSE
CODE_OF_CONDUCT.md                 volitelné
docs/configuration.md
docs/keybindings.md
docs/architecture.md
docs/coolify-api.md
docs/troubleshooting.md
docs/screenshots/                  později
docs/decisions/                    ADR záznamy
```

## 18.1 README

README musí obsahovat:

- krátký pitch,
- screenshot nebo VHS/GIF placeholder pouze pokud bude skutečně doplněn,
- features,
- instalaci,
- quick start,
- vytvoření Coolify API tokenu,
- doporučená oprávnění,
- config příklad,
- keybindings,
- security poznámky,
- roadmap,
- contribution instrukce.

Nevkládej nepravdivé badges ani tvrzení o podporovaných funkcích.

## 18.2 Architektura

`docs/architecture.md` musí vysvětlit:

- vrstvy,
- datový tok,
- Bubble Tea message flow,
- lifecycle requestů,
- cancellation,
- cache,
- credentials,
- capability detection,
- budoucí MCP adaptér.

## 18.3 ADR

Vytvoř minimálně:

```text
0001-use-go-and-charm-v2.md
0002-direct-coolify-api-not-mcp.md
0003-separate-domain-model-from-api-dto.md
0004-credential-storage-strategy.md
0005-tui-responsive-layout.md
```

---

# 19. CI/CD a release

## 19.1 GitHub Actions

Vytvoř workflows:

```text
.github/workflows/ci.yml
.github/workflows/release.yml
.github/workflows/security.yml
```

CI:

- checkout,
- setup Go,
- dependency cache,
- format check,
- vet,
- test,
- race test,
- staticcheck,
- build.

Security:

- govulncheck,
- dependency review pro PR, pokud je dostupný,
- CodeQL je volitelný.

Release:

- tag-based,
- GoReleaser,
- checksums,
- release notes,
- více OS/arch.

## 19.2 Makefile

Minimálně:

```makefile
make build
make run
make test
make test-race
make test-update-golden
make lint
make fmt
make vet
make vuln
make check
make clean
make snapshot
```

`make check` má spustit vše relevantní před commitem.

## 19.3 Version info

Binary musí podporovat:

```bash
cooldeck version
```

Výstup:

```text
cooldeck 0.1.0
commit: abc1234
built: 2026-08-04T08:00:00Z
go: go1.xx.x
platform: darwin/arm64
```

Hodnoty injectuj přes `-ldflags`.

---

# 20. Milestones

Implementuj v tomto pořadí.

## Milestone 0 - Repository foundation

- Go module,
- základní adresářová struktura,
- Cobra entrypoint,
- Makefile,
- CI,
- logging,
- version command,
- základní README,
- test harness.

### Acceptance criteria

```bash
go test ./...
go vet ./...
go build ./cmd/cooldeck
./cooldeck version
```

musí fungovat.

## Milestone 1 - Configuration and authentication

- config loader,
- config validation,
- instance model,
- keychain provider,
- env provider,
- command provider,
- auth commands,
- doctor command,
- redaction tests.

### Acceptance criteria

Uživatel dokáže bezpečně uložit token, ověřit připojení a token se nikde nevypíše.

## Milestone 2 - Coolify read API

- API client,
- list applications,
- get application,
- application logs,
- deployments list/detail/logs podle dostupného API,
- projects/servers mapping podle potřeby,
- error model,
- retry/backoff,
- capability detection.

### Acceptance criteria

API vrstva je kompletně testovatelná přes `httptest.Server` bez reálného Coolify.

## Milestone 3 - TUI shell and design system

- root model,
- responsive layout,
- theme auto/dark/light,
- header,
- footer,
- sidebar/tabs,
- modal system,
- toast system,
- command palette,
- help.

### Acceptance criteria

Aplikace se spustí s fake daty a je vizuálně konzistentní ve wide, standard a compact režimu.

## Milestone 4 - Applications dashboard

- seznam aplikací,
- selection,
- sorting,
- filtering,
- refresh,
- offline snapshot,
- empty/loading/error states,
- app detail.

### Acceptance criteria

Dashboard funguje s reálným API i fake clientem a zvládne 200 aplikací.

## Milestone 5 - Logs and deployments

- deployments table,
- runtime logs,
- deployment logs,
- follow/pause/wrap/search,
- sanitizace logů,
- polling a cancellation.

### Acceptance criteria

Log view nebliká, respektuje ruční scroll a nezpracuje escape sekvence jako terminálové příkazy.

## Milestone 6 - Mutating operations

- deploy,
- force deploy,
- restart,
- start,
- stop,
- permissions/capabilities,
- confirmation modals,
- operation progress,
- result toasts,
- duplicate operation guard.

### Acceptance criteria

Žádná nebezpečná operace se nespustí jediným náhodným stiskem bez jasného kontextu a potvrzení.

## Milestone 7 - Onboarding and polish

- first-run wizard,
- instances management,
- diagnostics,
- browser/clipboard integration,
- polished empty states,
- full docs,
- golden tests,
- performance review,
- release pipeline.

### Acceptance criteria

Nový uživatel dokáže aplikaci nainstalovat, přidat instanci a dostat se na dashboard bez ruční editace configu.

## Milestone 8 - Optional read-only resources

Až po kvalitním dokončení Applications:

- services,
- databases,
- servers,
- jednotný resource navigation model.

Neobětuj kvalitu hlavního dashboardu kvůli šíři funkcí.

---

# 21. Fake/demo režim

Implementuj lokální demo režim:

```bash
cooldeck --demo
```

Demo režim musí:

- nepotřebovat API ani token,
- poskytovat realistické aplikace a deploymenty,
- simulovat loading,
- simulovat success/error/offline stav,
- simulovat probíhající deployment,
- umožnit prezentaci TUI a snapshot testy,
- neprovádět skutečné mutace.

Fake client musí implementovat stejné aplikační rozhraní jako reálný klient.

Přidej deterministický seed pro testy.

---

# 22. Budoucí MCP integrace

MCP server není součástí MVP, ale kód připrav následovně:

```text
internal/app služby neznají Bubble Tea
internal/coolify nezná TUI
TUI používá aplikační rozhraní
mutace mají explicitní input/output typy
chyby jsou strojově klasifikovatelné
```

Budoucí nástroje:

```text
list_applications
get_application
get_application_logs
list_deployments
get_deployment_logs
deploy_application
restart_application
start_application
stop_application
```

Při budoucím MCP adaptéru musí být mutační tools opt-in a navázané na přísnější oprávnění. MCP nesmí být nutný pro běh TUI.

---

# 23. Definice hotového MVP

MVP je hotové pouze tehdy, když splňuje všechny následující body:

- lze připojit minimálně jednu reálnou Coolify instanci,
- lze bezpečně uložit credential,
- lze zobrazit seznam aplikací,
- lze zobrazit detail aplikace,
- lze filtrovat a řadit,
- lze zobrazit runtime logy,
- lze zobrazit deploymenty a dostupné deployment logy,
- lze spustit deploy,
- lze restartovat/start/stop aplikaci podle oprávnění,
- mutace mají potvrzení a viditelný výsledek,
- funguje automatický i manuální refresh,
- funguje offline/error state,
- UI je responsivní,
- funguje dark/light mode,
- aplikace nevyžaduje Nerd Font,
- token se nikde nezobrazí ani nezaloguje,
- testy běží bez reálného serveru,
- CI je zelené,
- release build funguje pro Linux/macOS/Windows,
- dokumentace odpovídá skutečnému stavu aplikace.

---

# 24. Kvalitativní požadavky

Před označením práce za dokončenou proveď vlastní review:

## Kód

- Je kód idiomatický?
- Nejsou rozhraní příliš široká?
- Neexistují cyklické závislosti?
- Neunikají goroutines?
- Je cancellation správná?
- Neprovádí se I/O v rendereru?
- Jsou chyby zachovány přes `%w`?
- Jsou secrets redigovány?

## UX

- Je vždy jasné, která instance a aplikace je aktivní?
- Je vždy jasné, co udělá Enter?
- Lze se vrátit pomocí Esc?
- Jsou destructive actions jednoznačné?
- Je aplikace použitelná jen klávesnicí?
- Neztrácí se selection po refreshi?
- Neodskakují logy při ručním scrollu?
- Funguje layout při resize?

## Vzhled

- Je spacing konzistentní?
- Jsou border a accent barvy střídmé?
- Je selection dostatečně viditelná?
- Funguje light theme?
- Jsou statusy čitelné bez barev?
- Nejsou obrazovky přeplněné?
- Nepůsobí komponenty jako slepené nesouvisející knihovny?

## Robustnost

- Co se stane při 401?
- Co se stane při 429?
- Co se stane při timeoutu?
- Co se stane při neznámém statusu?
- Co se stane při dlouhém názvu/doméně?
- Co se stane při 20 000 řádcích logu?
- Co se stane, když uživatel rychle přepne view?
- Co se stane při Ctrl+C během requestu?

---

# 25. Požadovaný první výstup Claude Code

Nezačínej okamžitě psát celou aplikaci naslepo.

Nejdříve vrať stručný implementační plán obsahující:

1. zjištěný stav repozitáře,
2. navrženou výslednou adresářovou strukturu,
3. ověřené verze klíčových dependencies,
4. ověřené Coolify endpointy pro MVP,
5. identifikovaná API omezení nebo nejasnosti,
6. pořadí prvních implementačních kroků,
7. rizika,
8. co přesně bude součástí prvního commitu.

Poté pokračuj implementací Milestone 0 a Milestone 1. Nečekej na další potvrzení, pokud nenarazíš na skutečně blokující informaci, kterou nelze rozumně odvodit.

---

# 26. Referenční zdroje ověřené při přípravě zadání

Stav ověřen k 4. srpnu 2026.

- Coolify API Authorization: `https://coolify.io/docs/api-reference/authorization`
- Coolify API Reference: `https://coolify.io/docs/api-reference/api/`
- Coolify application logs endpoint: `GET /api/v1/applications/{uuid}/logs`
- Charm v2 announcement: `https://charm.land/blog/v2/`
- Bubble Tea v2: `https://github.com/charmbracelet/bubbletea`
- Bubbles v2: `https://github.com/charmbracelet/bubbles`
- Lip Gloss v2: `https://github.com/charmbracelet/lipgloss`

Při implementaci vždy preferuj aktuální oficiální dokumentaci a aktuální stabilní verze před čísly nebo endpointy uvedenými v tomto dokumentu.

---

# 27. Finální instrukce

Postav aplikaci tak, aby:

- byla příjemná pro každodenní používání,
- byla bezpečná i proti nechtěným produkčním zásahům,
- byla rychlá přes lokální terminál i SSH,
- měla promyšlené stavy loading/error/offline,
- nabízela kvalitní klávesové ovládání,
- vypadala jako soudržný produkt,
- šla později rozšířit o služby, databáze, metriky a MCP,
- nepřerostla kvůli abstrakcím dříve, než vznikne fungující MVP.

Funkčnost, vizuální kvalita, bezpečnost a testovatelnost mají stejnou prioritu.
