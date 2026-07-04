# Goovie Architecture

This document describes the architectural layout and data flow of the Goovie CLI application.

## Overview
Goovie is a cross-platform CLI tool built in Go that searches for torrents using Prowlarr and streams them instantly using `webtorrent` and `mpv`. It features a rich, responsive terminal user interface (TUI) powered by the `charmbracelet/bubbletea` framework.

## Project Structure (Standard Go Layout)

```
goovie/
├── cmd/goovie/main.go      # Application entry point
├── internal/
│   ├── config/             # Configuration management (env vars)
│   ├── prowlarr/           # API interactions and data types for Prowlarr
│   ├── tui/                # Bubble Tea UI components and state machine
│   └── player/             # Streaming logic (Webtorrent & MPV execution)
```

## Module Breakdown

### 1. Entry Point (`cmd/goovie/main.go`)
- **Responsibility:** Wires the components together.
- **Flow:** 
  1. Invokes the Bubble Tea TUI model via `tui.NewModel()`.
  2. The TUI initializes the configuration via `config.InitConfig()`.
  3. Runs the TUI loop.
  4. Upon TUI exit, handles the selected stream based on the selected mode (Movie/Show/Anime).

### 2. Configuration (`internal/config`)
- **Responsibility:** Setup, persistence, and priority-based configuration ingestion.
- **Flow (`InitConfig` priority list):** 
  1. **Environment Variables:** Checks for `PROWLARR_URL` and `PROWLARR_API_KEY`.
  2. **Saved Config (`LoadConfig`):** Reads the API key from `~/.goovie/config.json`.
  3. **Auto-Detect (`AutoDetectAPIKey`):** Parses the local Prowlarr `config.xml` for an API key.

### 3. TUI State Machine (`internal/tui`)
Uses the Elm architecture via Bubble Tea (Model, Update, View).
- **States:**
  - `StateSearchInput`: Initial state. User inputs the query.
  - `StateLoading`: Waiting for indexer discovery and search completion. Shows a spinner.
  - `StateList`: Displays results sorted by seeders.
- **Concurrency:** TUI commands send asynchronous HTTP requests to the Prowlarr API using Bubble Tea's `tea.Cmd`.

### 4. Search and Metadata Client (`internal/prowlarr`)
- **Responsibility:** Interacts with Prowlarr, TVMaze, Cinemeta, and Jikan APIs for metadata and torrents.
- **Flow:**
  1. Fetches metadata depending on media type (`FetchCinemetaMovies`, `FetchTVShows`, `FetchAnime`).
  2. `FetchIndexers`: Retrieves active indexers using `config.ProwlarrURL` and `config.ProwlarrAPIKey`.
  3. `SearchSingleIndexer`: Searches for torrents with specific indexers.
  4. `ResolveProxyLink`: Parses `InfoHash`, `magnetUrl` or resolves `downloadUrl` into a valid magnet URI.

### 5. Player logic (`internal/player`)
- **Responsibility:** Handles streaming from torrents, direct MP4/HLS streams (Anime/Asian Media), and launches external players.
- **Anime / Asian Media Scrapers:** Custom logic scraping domains (e.g. `anikototv.to`, `kisskh.do`) for direct streaming links, using local HLS proxy `VibeProxy` if needed.
- **Execution:** Spawns child processes depending on the OS (e.g., `LaunchPlayer` with `mpv` or `webtorrent`).
