<p align="center">
  <h1 align="center">LaraDev</h1>
  <p align="center">
    <strong>Laravel Development Environment Manager</strong><br>
    A fast, single-binary CLI + interactive TUI for managing your entire Laravel dev workflow.<br>
    No Herd. No Valet. No Docker. Just <code>laradev</code>.
  </p>
  <p align="center">
    <a href="#install">Install</a> &nbsp;&bull;&nbsp;
    <a href="#quick-start">Quick Start</a> &nbsp;&bull;&nbsp;
    <a href="#https-proxy">HTTPS Proxy</a> &nbsp;&bull;&nbsp;
    <a href="#cli-command-reference">CLI Reference</a> &nbsp;&bull;&nbsp;
    <a href="#interactive-tui">TUI Guide</a>
  </p>
</p>

---

## What is LaraDev?

LaraDev is a compiled Go binary that gives you everything you need to run and manage a Laravel project in development — services, migrations, code generation, log tailing, cache management, and more — from a single command.

It works in two ways:
- **Interactive TUI** — run `laradev` for a full terminal UI with menus, live status, and scrollable output
- **CLI mode** — run `laradev <command>` for direct access from your shell or scripts

```
 LaraDev  >  Main Menu
 ● PHP:8007  │  ● Vite:5173  │  ○ Queue  │  ○ Sched  │  ● HTTPS
 ╭──────────────────────────────────────────────────────────────────╮
 │ Taskify [local]  ~/Projects/Laravel/taskify                     │
 │                                                                 │
 │ PHP 8.4.1 │ Node v22.0.0 │ DB sqlite (48K) │ Log 4.0K          │
 │                                                                 │
 │ ● App https://taskify.test          Vite http://localhost:5173   │
 │                                                                 │
 │ Pest · Vite · Breeze                                            │
 ╰──────────────────────────────────────────────────────────────────╯

 --- SERVICES ---
 ▸ Start All Services
   Stop All Services
   Restart All Services
   Manage Services
 --- DEVELOP ---
   Database
   Development
   Cache & Optimize
 --- MONITOR ---
   Logs
 --- SYSTEM ---
   Config
   Exit
```

### Features

- **Service management** — Start, stop, restart PHP server, Vite, queue worker, scheduler, and Reverb with PID/memory tracking
- **HTTPS proxy** — Automatic `.test` domain with trusted TLS certificates (no browser warnings). Powered by a built-in Go reverse proxy and pure-Go certificate authority — no external tools required
- **Database operations** — Migrate, rollback, fresh, seed — all with confirmation dialogs for destructive actions
- **Code generation** — 15 `artisan make:*` generators with smart defaults (Model with `-mfscR`, Event + Listener paired, etc.)
- **Test runner** — Auto-detects Pest vs PHPUnit, run all/unit/feature/filtered
- **Log viewer** — Live-tailing with scrollable viewports, Laravel Pail support, grep search
- **Project detection** — Reads `.env`, `composer.json`, and project structure to show app name, environment, versions, DB type, starter kit, and more
- **Cache tools** — Clear individual or all caches, optimize for production
- **Configurable** — Ports, queue settings, and more via `.laradev.conf` or interactive config editor
- **Zero dependencies** — Single ~5MB binary. No runtime Go, Python, or Node requirements beyond what Laravel itself needs

---

## Install

### One-line install (macOS and Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/DiyRex/laradev-go/main/scripts/install.sh | bash
```

The script:
- Auto-detects your OS and CPU architecture
- Downloads the latest `laradev` binary directly from GitHub Releases
- Creates `~/.laradev/` for storing proxy configs and certificates

That's it — no package manager, no extra tools, no dependencies of any kind.

### Update to latest version

Run the same command again — the script detects an existing installation and replaces only the binary. Your `~/.laradev/` directory (proxy configs, certificates, project state) is **never touched** during an update.

```bash
curl -fsSL https://raw.githubusercontent.com/DiyRex/laradev-go/main/scripts/install.sh | bash
```

### Manual download

Pre-built binaries for every platform are on the [Releases](https://github.com/DiyRex/laradev-go/releases) page:

| Platform | Binary |
|---|---|
| Linux x86_64 | `laradev-linux-amd64` |
| Linux ARM64 | `laradev-linux-arm64` |
| macOS Intel | `laradev-darwin-amd64` |
| macOS Apple Silicon | `laradev-darwin-arm64` |
| Windows x86_64 | `laradev-windows-amd64.exe` |

Download, `chmod +x`, and move to your `PATH`.

### Build from source

```bash
git clone https://github.com/DiyRex/laradev-go.git
cd laradev-go
go mod tidy
make build
```

Requires Go 1.21+. Produces the `laradev` binary in the project directory.

---

## Quick Start

```bash
# Navigate to any Laravel project
cd ~/my-laravel-app

# Launch the interactive TUI
laradev

# Or use CLI commands directly
laradev up          # Start all services
laradev status      # Dashboard with PIDs, memory, versions
laradev down        # Stop everything
```

LaraDev finds your Laravel project automatically by looking for the `artisan` file. It walks up from the current directory if needed.

---

## HTTPS Proxy

LaraDev includes a built-in HTTPS reverse proxy that gives your project a `.test` domain with a trusted TLS certificate — similar to Laravel Herd, but implemented entirely in the `laradev` binary with no Nginx or system daemons required.

All proxy state is stored in `~/.laradev/` — **nothing is added to your project directory**.

### How it works

| Component | What it does |
|---|---|
| **Domain** | Auto-derived from `APP_NAME` in `.env` — `"My Shop"` → `myshop.test` |
| **DNS** | Adds `127.0.0.1 myapp.test` to `/etc/hosts` (one line, sudo once) |
| **TLS cert** | Generated in pure Go, stored in `~/.laradev/certs/`, trusted in the System keychain |
| **Proxy** | Go reverse proxy listening on `127.0.0.1:8443`, forwarding to `localhost:PHP_PORT` |
| **Port redirect** | `pfctl` (macOS) or `iptables` (Linux) persistently routes port `443 → 8443` — set up automatically during `proxy:setup` via a LaunchDaemon/systemd service |
| **HTTP redirect** | Port `8080` → `https://domain.test` redirect |

The proxy daemon runs on port `8443` (no root required). Port `443` is routed there transparently at the OS level — your browser always sees a clean `https://myapp.test` URL with no port number.

### Setup (one time per project)

```bash
cd ~/my-laravel-app

laradev proxy:setup
```

This will:
1. Generate a local CA (pure Go, stored in `~/.laradev/ca/`)
2. Trust the CA in the **System keychain** (macOS: `sudo security add-trusted-cert`; Linux: `update-ca-certificates`) — all browsers will trust it without warnings
3. Generate a TLS certificate for your `.test` domain (pure Go, signed by the local CA)
4. Add the domain to `/etc/hosts`
5. Install a persistent port `443 → 8443` redirect via LaunchDaemon (macOS) or systemd (Linux)
6. Save proxy config to `~/.laradev/projects/{id}/proxy.conf`

After setup, visiting `https://myapp.test` works immediately — no port number in the URL, no browser security warnings. That's the only command you ever need to run manually.

### Automatic start / stop

Once configured, the proxy **starts and stops automatically** with your services — no separate commands needed:

```bash
laradev up        # Starts PHP + Vite + Queue + HTTPS proxy
laradev down      # Stops everything including the proxy
laradev restart   # Restarts everything including the proxy
```

### Toggle from the TUI

Open the TUI (`laradev`), navigate to **Manage Services** — the HTTPS Proxy appears as the last entry:

```
 [ON]  HTTPS Proxy (myapp.test)  --  running
```

Select it to stop the proxy. Select a stopped proxy to start it. If not yet configured, a help message tells you to run `laradev proxy:setup`.

### Status check

```bash
laradev proxy:status   # Show domain, target port, and running state
```

### Re-trusting the CA

If you reinstall the binary or the CA trust is lost (e.g. after a keychain reset), re-run trust without touching the cert:

```bash
laradev proxy:trust
```

This removes the cached trust flag and re-adds the CA to the System keychain. Restart your browser afterwards.

### Port 443 redirect

Port `443 → 8443` forwarding is configured automatically during `proxy:setup` via a system daemon (LaunchDaemon on macOS, systemd on Linux). It persists across reboots.

If the forwarding ever stops working (e.g. after Docker Desktop reloads pfctl rules on macOS), reapply it manually:

```bash
laradev proxy:ports
```

### TUI indicators

The info box and status bar show the proxy state at a glance:

| Indicator | Meaning |
|---|---|
| `● HTTPS` green | Proxy running — HTTPS active |
| `● HTTPS` red | Proxy configured but stopped (`proxy:up` to start) |
| `○ HTTPS` dim | Not configured yet (`proxy:setup` to configure) |

### Proxy configuration

Stored at `~/.laradev/projects/{id}/proxy.conf` — managed by `laradev`, no need to edit manually:

```ini
DOMAIN="myapp.test"
TARGET_PORT="8007"      # PHP server port
PROXY_PORT="8443"       # internal HTTPS listener (443 → this via pfctl/iptables)
HTTP_PORT="8080"        # HTTP → HTTPS redirect listener (80 → this)
ENABLED="true"
PORT_FORWARDING="true"  # true once persistent 443→8443 redirect is installed
```

---

## CLI Command Reference

### Services

| Command | Alias | Description |
|---|---|---|
| `laradev up` | `start` | Start PHP server, Vite, and queue worker |
| `laradev down` | `stop` | Gracefully stop all running services |
| `laradev restart` | | Stop then start all services |
| `laradev status` | `st` | Status dashboard with PIDs, memory, versions, URLs |
| `laradev serve` | `server` | Start PHP dev server only |
| `laradev vite` | | Start Vite HMR server only |
| `laradev queue` | | Start queue worker only |
| `laradev schedule` | | Start Laravel scheduler only |

Default ports: PHP `0.0.0.0:8007`, Vite `localhost:5173` (configurable via `.laradev.conf`).

### HTTPS Proxy

| Command | Description |
|---|---|
| `laradev proxy:setup` | **One-time setup** — generate cert, trust CA, add `/etc/hosts` entry, configure port 443 redirect |
| `laradev proxy:status` | Show domain, target port, and running state |
| `laradev proxy:up` | Manually start the proxy (automatic with `laradev up`) |
| `laradev proxy:down` | Manually stop the proxy (automatic with `laradev down`) |
| `laradev proxy:ports` | Reapply port 443 → 8443 redirect if it was cleared (e.g. by Docker Desktop) |
| `laradev proxy:trust` | Re-trust the CA in the system keychain (run after keychain reset or reinstall) |

### Development

| Command | Description |
|---|---|
| `laradev build` | `npm run build` |
| `laradev test` | Run tests (auto-detects Pest vs PHPUnit) |
| `laradev test --filter=MyTest` | Filter specific tests |
| `laradev test --testsuite=Unit` | Run a test suite |
| `laradev tinker` | Laravel Tinker REPL |
| `laradev routes` | Route list (excluding vendor) |
| `laradev artisan <cmd>` | Any artisan command |

```bash
# Examples
laradev artisan about
laradev artisan make:model Post -m
laradev artisan queue:work --once
```

### Database

| Command | Alias | Description |
|---|---|---|
| `laradev migrate` | `mg` | Run pending migrations |
| `laradev fresh` | | Drop all tables, re-migrate + seed (confirmation required) |
| `laradev seed` | | Run database seeders |
| `laradev rollback` | `rb` | Rollback last migration batch |

### Logs

| Command | Alias | Description |
|---|---|---|
| `laradev logs` | `log:app` | Tail `storage/logs/laravel.log` |
| `laradev log:pail` | `pail` | Laravel Pail (real-time formatted log viewer) |
| `laradev log:server` | | PHP server output |
| `laradev log:vite` | | Vite dev server output |
| `laradev log:queue` | | Queue worker output |
| `laradev log:all` | | All service logs combined |
| `laradev log:clear` | | Truncate laravel.log |

### Cache & Optimization

| Command | Description |
|---|---|
| `laradev cache` / `clear` | Clear all caches (config, route, view, event, app, compiled) |
| `laradev optimize` | Cache config, routes, views for production |

### System

| Command | Description |
|---|---|
| `laradev setup` | First-time setup: `.env`, deps, key, migrate, build, storage link |
| `laradev nuke` | Full reset: remove deps, reinstall, fresh migrate, rebuild (double confirmation) |
| `laradev about` | `php artisan about` |
| `laradev help` | Command reference |

---

## Interactive TUI

Run `laradev` without arguments to launch the interactive terminal UI.

### Navigation

| Key | Action |
|---|---|
| `↑` `↓` or `k` `j` | Navigate menus |
| `Enter` | Select / confirm |
| `Esc` / `Backspace` | Go back |
| `q` / `Ctrl+C` | Quit (from main menu) |
| `↑` `↓` | Scroll output viewports |
| `←` `→` or `Tab` | Toggle confirmation dialogs |

### Pages

**Main Menu** — Title bar, live service status bar (PHP / Vite / Queue / Sched / HTTPS), project info box (name, env, path, versions, DB, URLs with HTTPS indicator, detected tools), and section-based navigation.

**Manage Services** — Per-service control with live PID and memory display. Supports PHP Server, Vite, Queue Worker, Scheduler, Reverb WebSocket, and **HTTPS Proxy** (toggle start/stop directly from the list; shows setup instructions if not yet configured).

**Database** — Run Migrations, Fresh + Seed, Seed, Rollback, Rollback N steps, Reset All. Destructive operations require confirmation.

**Development** — Build, Test (All / Unit / Feature / Filter), Routes, Tinker REPL, Artisan Command, and the **Make** sub-menu with 15 generators:

| Generator | Flags |
|---|---|
| Model | `-mfscR` (migration, factory, seeder, controller, resource) |
| Controller | `--resource` |
| Migration | |
| Middleware | |
| Request | |
| Resource | |
| Seeder | |
| Factory | |
| Job | |
| Event + Listener | Creates both files |
| Mail | |
| Notification | |
| Command | |
| Policy | |
| Test | `--pest` or PHPUnit (auto-detected) |

**Cache & Optimize** — Clear individual caches (config, routes, views, events, app, compiled), clear all at once, or optimize.

**Logs** — Live tail with scrollable viewport (500-line buffer), Laravel Pail, per-service logs, combined view, grep search, and log clearing.

**Config** — Edit settings interactively. Changes persist to `.laradev.conf`:

| Setting | Default |
|---|---|
| PHP Host | `0.0.0.0` |
| PHP Port | `8007` |
| Vite Port | `5173` |
| Queue Tries | `1` |
| Queue Timeout | `90` |
| Queue Sleep | `3` |

---

## Project Detection

LaraDev automatically reads your project environment on launch:

| Info | Source |
|---|---|
| App name & environment | `.env` → `APP_NAME`, `APP_ENV` |
| Database connection & size | `.env` → `DB_CONNECTION` + SQLite file size |
| Queue connection | `.env` → `QUEUE_CONNECTION` |
| PHP & Node versions | Runtime detection |
| Log file size | `storage/logs/laravel.log` |
| Test framework | `vendor/pestphp/` or `phpunit.xml` |
| Build tool | `vite.config.js` / `.ts` or `webpack.mix.js` |
| Starter kit | `composer.json` → Breeze, Jetstream, or Filament |

---

## Configuration

### Per-project config (`.laradev.conf`)

Port and queue overrides are stored in `.laradev.conf` at the project root. Add this file to your `.gitignore`.

```ini
# .laradev.conf
PHP_HOST="0.0.0.0"
PHP_PORT="8007"
VITE_PORT="5173"
QUEUE_TRIES="1"
QUEUE_TIMEOUT="90"
QUEUE_SLEEP="3"
```

Edit via the TUI Config page or any text editor. Service PIDs and logs are stored in `.laradev_pids/`.

### Global config (`~/.laradev/`)

HTTPS proxy state is stored globally — nothing proxy-related is written to the project directory:

```
~/.laradev/
  ca/                           # local CA key and certificate (pure Go generated)
  certs/                        # domain certificates (pure Go generated, signed by local CA)
    myapp.test.pem
    myapp.test-key.pem
  projects/
    {id}/                       # keyed by project path hash
      proxy.conf                # domain, ports, enabled flag
      proxy.pid                 # daemon PID (while running)
```

This directory is **never modified by `laradev update`** — your certificates and proxy configs survive upgrades.

---

## Architecture

- **Single binary** — Compiled Go with no runtime dependencies. ~5MB static binary.
- **Bubble Tea** — Elm-architecture TUI framework for robust terminal handling.
- **Lipgloss** — Terminal styling with the Cerise color theme.
- **Process management** — PID tracking in `.laradev_pids/`, recursive child process kill via process groups and `pgrep` tree walking.
- **Built-in HTTPS proxy** — `net/http/httputil.ReverseProxy` with TLS termination. No Nginx or system daemons.
- **Alternate screen** — TUI runs in the terminal's alternate buffer; your shell stays clean.
- **Interactive handoff** — Tinker and Pail fully take over the terminal, then return to the TUI seamlessly.
- **Graceful shutdown** — Process groups ensure ports are released immediately on stop.

---

## Releases

Releases are automated via GitHub Actions with GoReleaser. Creating a GitHub release triggers cross-compiled builds for all platforms:

| Binary | Platform |
|---|---|
| `laradev-linux-amd64` | Linux x86_64 |
| `laradev-linux-arm64` | Linux ARM64 (Raspberry Pi 4+, AWS Graviton) |
| `laradev-darwin-amd64` | macOS Intel |
| `laradev-darwin-arm64` | macOS Apple Silicon (M1/M2/M3/M4) |
| `laradev-windows-amd64.exe` | Windows x86_64 |

All binaries are statically linked (`CGO_ENABLED=0`) with stripped debug symbols.

---

## Requirements

- PHP 8.2+ with a Laravel project
- Node.js + npm (for Vite / frontend assets)
- Linux or macOS (Windows: CLI mode only, no TUI)
- Standard Unix tools: `pgrep`, `tail`, `grep`

**For HTTPS proxy (optional):** No extra tools required. Certificate generation is built into the binary.

---

## License

MIT
