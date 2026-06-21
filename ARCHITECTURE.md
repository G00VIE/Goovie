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
  1. Loads configuration from `internal/config`.
  2. Initializes the `prowlarr.Client`.
  3. Instantiates the Bubble Tea TUI model via `tui.NewModel()`.
  4. Runs the TUI loop.
  5. Upon TUI exit, checks if a magnet link was selected and passes it to `player.Stream()`.

### 2. Configuration (`internal/config`)
- **Responsibility:** Environment variable ingestion and default fallback.
- **Variables:** Looks for `PROWLARR_URL` and `PROWLARR_API_KEY`.

### 3. TUI State Machine (`internal/tui`)
Uses the Elm architecture via Bubble Tea (Model, Update, View).
- **States:**
  - `StateSearchInput`: Initial state. User inputs the query.
  - `StateLoading`: Waiting for indexer discovery and search completion. Shows a spinner.
  - `StateList`: Displays results sorted by seeders.
- **Concurrency:** TUI commands send asynchronous HTTP requests to the Prowlarr API using Bubble Tea's `tea.Cmd`.

### 4. Prowlarr Client (`internal/prowlarr`)
- **Responsibility:** Interacts with the local or remote Prowlarr instance.
- **Flow:**
  1. `FetchIndexersCmd`: Calls `/api/v1/indexer` to find all active indexers.
  2. `FetchSingleIndexerCmd`: For each active indexer, calls `/api/v1/search` concurrently.
  3. Converts raw JSON search results into unified `TorrentResult` models.
  4. Parses `magnetUrl` or transforms `downloadUrl` into a valid proxy URL.

### 5. Player logic (`internal/player`)
- **Responsibility:** Handles the final torrent payload and launches external playback tools.
- **URL Resolution:** If the selected link is an HTTP proxy URL (from Prowlarr), it intercepts HTTP 3xx redirects to capture raw `magnet:` URIs without Go's HTTP client throwing errors. If it receives a `.torrent` file payload, it writes it to `/tmp/stream.torrent`.
- **Execution:** Spawns a child process calling `webtorrent <path> --mpv`. Standard output and errors are piped to the user's terminal.
