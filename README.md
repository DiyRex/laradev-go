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
 PHP:8007 [ON]  Vite:5173 [ON]  Queue [ON]  Sched [--]  Reverb [--]
 ╭──────────────────────────────────────────────────────────────────╮
 │ Taskify [local]  ~/Projects/Laravel/taskify                     │
 │                                                                 │
 │ PHP 8.4.1 │ Node v22.0.0 │ DB sqlite (48K) │ Log 4.0K          │
 │                                                                 │
 │ App http://0.0.0.0:8007    Vite http://localhost:5173           │
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
- **Database operations** — Migrate, rollback, fresh, seed — all with confirmation dialogs for destructive actions
- **Code generation** — 15 `artisan make:*` generators with smart defaults (Model with `-mfscR`, Event + Listener paired, etc.)
- **Test runner** — Auto-detects Pest vs PHPUnit, run all/unit/feature/filtered
- **Log viewer** — Live-tailing with scrollable viewports, Laravel Pail support, grep search
- **Project detection** — Reads `.env`, `composer.json`, and project structure to show app name, environment, versions, DB type, starter kit, and more
- **Cache tools** — Clear individual or all caches, optimize for production
- **Configurable** — Ports, queue settings, and more via `.dev.conf` or interactive config editor
- **Zero dependencies** — Single ~5MB binary. No runtime Go, Python, or Node requirements beyond what Laravel itself needs

---

## Install

### Quick Install (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/DiyRex/laradev-go/main/install.sh | sh
```

Auto-detects your OS and architecture, downloads the latest release binary, and installs to `/usr/local/bin`.

### Download from Releases

Pre-built binaries for every platform are available on the [Releases](https://github.com/DiyRex/laradev-go/releases) page:

| Platform | Binary |
|---|---|
| Linux (x86_64) | `laradev-linux-amd64` |
| Linux (ARM64) | `laradev-linux-arm64` |
| macOS (Intel) | `laradev-darwin-amd64` |
| macOS (Apple Silicon) | `laradev-darwin-arm64` |
| Windows (x86_64) | `laradev-windows-amd64.exe` |

Download, `chmod +x`, and move to your PATH.

### Build from Source

```bash
git clone https://github.com/DiyRex/laradev-go.git
cd laradev
go mod tidy
make build
```

Requires Go 1.21+. Produces the `laradev` binary in the parent directory.

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

## Compatibility

LaraDev works with any standard Laravel project:

| Variant | Supported |
|---|---|
| Plain Laravel | Yes |
| Laravel Breeze (React / Vue / Blade) | Yes |
| Laravel Jetstream (Livewire / Inertia) | Yes |
| Filament | Yes |
| Pest / PHPUnit | Auto-detected |
| Vite / Laravel Mix | Auto-detected |
| SQLite / MySQL / PostgreSQL | Yes |
| Queue workers (database, Redis, SQS) | Yes |
| Laravel Reverb (WebSockets) | Auto-detected |

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

Default ports: PHP `0.0.0.0:8007`, Vite `localhost:5173` (configurable).

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

**Main Menu** — Title bar, live service status bar, project info box (name, env, path, versions, DB, URLs, detected tools), and section-based navigation.

**Manage Services** — Per-service control with live PID and memory display. Supports PHP Server, Vite, Queue Worker, Scheduler, and Reverb WebSocket.

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

**Config** — Edit settings interactively. Changes persist to `.dev.conf`:

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

This information is displayed in the TUI info box and the `laradev status` CLI output.

---

## Configuration

Config overrides are stored in `.dev.conf` at the project root (add to `.gitignore`):

```ini
PHP_HOST="0.0.0.0"
PHP_PORT="8007"
VITE_PORT="5173"
QUEUE_TRIES="1"
QUEUE_TIMEOUT="90"
QUEUE_SLEEP="3"
```

Edit via the TUI Config page or any text editor. Service PIDs and logs are stored in `.dev_pids/`.

---

## Architecture

- **Single binary** — Compiled Go with no runtime dependencies. ~5MB static binary.
- **Bubble Tea** — Elm-architecture TUI framework for robust terminal handling.
- **Lipgloss** — Terminal styling with the Cerise color theme.
- **Process management** — PID tracking in `.dev_pids/`, recursive child process kill via process groups and `pgrep` tree walking.
- **Alternate screen** — TUI runs in the terminal's alternate buffer; your shell stays clean.
- **Interactive handoff** — Tinker and Pail fully take over the terminal, then return to the TUI seamlessly.
- **Graceful shutdown** — Process groups ensure ports are released immediately on stop.

---

## Releases

Releases are automated via GitHub Actions. Creating a GitHub release triggers builds for all platforms:

- `laradev-linux-amd64`
- `laradev-linux-arm64`
- `laradev-darwin-amd64`
- `laradev-darwin-arm64`
- `laradev-windows-amd64.exe`

All binaries are statically linked with stripped debug symbols.

---

## Requirements

- PHP 8.2+ with a Laravel project
- Node.js + npm (for Vite / frontend assets)
- Linux or macOS
- Standard Unix tools: `pgrep`, `tail`, `grep`

---

## License

MIT
