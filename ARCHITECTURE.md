# Goovie Architecture

This document describes the architectural layout, modules, and data flow of the Goovie terminal streaming suite.

## Overview
Goovie is a 100% pure Go cross-platform terminal streaming suite that searches for media using Prowlarr and web scrapers, streams torrents sequentially with an embedded high-throughput BitTorrent client, and plays media seamlessly through `mpv`. It features a rich, responsive terminal user interface (TUI) powered by Charm's `bubbletea` framework.

## Project Structure (Standard Go Layout)

```
goovie/
├── cmd/
│   ├── goovie/main.go          # Application entry point & lifecycle management
│   └── test_torrent/main.go    # Torrent engine diagnostic utility
├── internal/
│   ├── assets/                 # Embedded logos, ANSI fonts, and MPV Lua scripts
│   │   └── scripts/            # goovie_skip.lua (Netflix skip intro/outro & HUD)
│   ├── bittorrent/             # Pure Go BitTorrent engine (anacrolix/torrent)
│   ├── config/                 # App configuration, cache directory & resume state
│   ├── player/                 # MPV process control, AniSkip API, HLS VibeProxy
│   ├── prowlarr/               # Prowlarr, TVMaze, Cinemeta, and Jikan clients
│   ├── sysutil/                # Health checks, 1-click installer & cleanup
│   └── tui/                    # Bubble Tea TUI state machine, models & views
```

## Module Breakdown

### 1. Entry Point (`cmd/goovie/main.go`)
- **Lifecycle Management**: Initializes the local HLS `VibeProxy`, loads embedded ANSI fonts and logo art, and mounts the Bubble Tea program.
- **Graceful Shutdown**: Intercepts OS signals (`SIGINT`, `SIGTERM`) to cleanly terminate active streaming engines, shut down background proxies, enforce the 10 GB torrent cache quota (`config.PruneTorrentCache(10, "")`), and unlink scratch temp files via `sysutil.PurgeAllTempData()`.

### 2. Configuration & State Persistence (`internal/config`)
- **AppConfig**: Manages Prowlarr/FlareSolverr credentials, minimum seeders, download cache size (150 MB, 256 MB default, 512 MB, 1024 MB), skip intro modes (`prompt`, `auto`, `off`), and auto-resume toggles.
- **Persistent Torrent Cache**: Provides `TorrentCacheDir()` (`~/.goovie/torrent_cache/`), storing downloaded torrent chunks across application restarts and system reboots.
- **Progressive Cleanup**: Implements `PruneTorrentCache(maxSizeGB, keepActiveName)` to enforce LRU cache quotas without disturbing currently watched media.
- **Watch History & Resume State**: Persists watched episodes to `~/.goovie/watched.json` and active resume session data to `~/.goovie/resume.json`.

### 3. Pure Go BitTorrent Streaming Engine (`internal/bittorrent`)
- **Zero-Dependency Torrent Engine**: Built directly on `anacrolix/torrent` with DHT, PEX, and multi-tracker tier scraping.
- **Persistent Piece Completion**: Uses SQLite piece completion (`storage.NewDefaultPieceCompletionForDir`) to instantly verify existing disk chunks on restart with zero re-downloading.
- **Local HTTP Range Server**: Creates an ephemeral HTTP byte-range server (`127.0.0.1:<port>/stream`) with configurable dynamic readahead (`reader.SetReadahead`) matching the user's cache buffer settings.
- **Session Lifecycle**: On stream close, drops the torrent from the active swarm while preserving downloaded chunks in the persistent cache.

### 4. Player & Stream Integration (`internal/player`)
- **MPV Execution**: Launches `mpv` with anti-stutter flags (`--demuxer-max-bytes`, `--demuxer-max-back-bytes`, `--cache-pause=yes`, `--cache-pause-wait=15`, `--save-position-on-quit=yes`, and `--start=<seconds>`).
- **AniSkip Client (`aniskip.go`)**: Queries the AniSkip API (`https://api.aniskip.com/v2/skip-times`) for opening and ending timestamps.
- **Embedded Skip Script (`goovie_skip.lua`)**: Embedded via `embed.FS` and extracted to `~/.goovie/scripts/goovie_skip.lua`. Provides Netflix-style on-screen skip buttons (`[S]`), smart +85s fallback jump with instant undo (`[U]`), live buffering HUD, and real-time playback position sync (`~/.goovie/last_pos.json`).
- **Local HLS Proxy (`VibeProxy`)**: Intercepts, decrypts, and proxies anime and Asian drama m3u8 playlists and MPEG-TS segments on localhost.

### 5. TUI State Machine (`internal/tui`)
- **Elm Architecture**: Implements Bubble Tea's `Init`, `Update`, and `View` pattern.
- **Key States**:
  - `StateFrontPage`: Displays ASCII branding and the **▶ CONTINUE WATCHING** banner for 1-key instant resumption (`[ C ]`).
  - `StateModeSelect`: Interactive selection between Movies, TV Shows, and Anime.
  - `StateSystemHealthCheck`: 1-Click Auto-Installer and interactive toggles for download cache buffers, skip intro modes, and torrent cache clearing (`[ x ]`).
  - `StateTVFileSelect` / `StateAnikotoEpSelect`: Episode navigation featuring watched badges (`[✓]`) and automatic cursor advance.
  - `StateList`: Search results highlighting bandwidth-efficient releases (`x265`, `HEVC`, `PSA`) with `⚡` icons.

### 6. System Utilities & Health Check (`internal/sysutil`)
- **Health Check (`healthcheck.go`)**: Probes local environment for MPV, Prowlarr, FlareSolverr, and Edge/Chrome browser availability.
- **1-Click Auto-Installer (`installer.go`)**: Autonomous package installation via `winget`, pre-seeding Prowlarr authentication configurations to bypass login wizards and inject top indexers automatically.
