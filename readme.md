# d4r

A terminal UI for Docker. Manage containers, volumes, networks, and images without leaving your keyboard.

![containers](./img/d4r-containers.png)

## Features

**Containers**
- Lists all containers by default (running and stopped) — press `a` to narrow to running-only
- View inspect details
- Start / stop / delete (with confirmation)
- Tail and follow logs
- Shell into a running container

**Volumes**
- List volumes with size and reference count
- View inspect details
- Delete (with confirmation)

**Networks**
- List networks with subnet/gateway info
- View inspect details
- Delete (with confirmation)

**Images**
- List images with size and tags
- View inspect details
- Delete (with confirmation)

**Themes**
- Five built-in themes: Charm, Dracula, Tokyo Night, Base16, Catppuccin
- Press `t` to open the theme picker — the UI previews each theme live as you navigate
- Selected theme is persisted to `~/.config/d4r/config.toml`

## Requirements

- [Go](https://go.dev/) 1.21 or later
- Docker daemon running and accessible (local socket or via `DOCKER_HOST`)

## Build

```sh
git clone <repo-url>
cd d4r
go build -o d4r .
```

Or run without installing:

```sh
go run .
```

## Install

```sh
go install .
```

This places `d4r` on your `$GOPATH/bin` (ensure it is in `$PATH`).

## Usage

```sh
./d4r
```

Respects the standard Docker environment variables — set `DOCKER_HOST` to point at a remote or rootless daemon:

```sh
DOCKER_HOST=ssh://user@host ./d4r
```

## Configuration

Config is stored at `~/.config/d4r/config.toml` and is created automatically on first theme selection.

```toml
theme = "dracula"
```

Available theme values: `charm`, `dracula`, `tokyo-night`, `base16`, `catppuccin`

## Key Bindings

### Global

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle between tabs |
| `1` `2` `3` `4` | Jump to Containers / Volumes / Networks / Images |
| `j` / `k` / `↑` / `↓` | Navigate list |
| `r` | Refresh |
| `t` | Open theme picker |
| `q` / `Ctrl+C` | Quit |

### Containers

| Key | Action |
|-----|--------|
| `a` | Toggle all containers / running-only |
| `Enter` / `d` | View details |
| `l` | View logs |
| `s` | Shell into container (`exit` to return) |
| `x` | Stop container (confirmation required) |
| `u` | Start / unpause container |
| `D` | Delete container (confirmation required) |

### Volumes, Networks, Images

| Key | Action |
|-----|--------|
| `Enter` / `d` | View details |
| `D` | Delete (confirmation required) |

### Detail / Log view

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Scroll |
| `PgUp` / `PgDn` | Page scroll |
| `f` | Toggle log follow (logs view only) |
| `Esc` / `q` | Back to list |

### Theme picker

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate themes (live preview) |
| `Enter` | Select and save theme |
| `Esc` | Cancel (reverts to previous theme) |

### Confirmation prompt

| Key | Action |
|-----|--------|
| `y` | Confirm action |
| `n` / `Esc` | Cancel |
