<p align="center">
  <img src="internal/assets/logos/goovie_logo.gif" alt="Goovie Logo" width="600">
</p>

# Goovie

> Stream torrents and web media directly in your terminal with zero waiting time.

**Goovie** is a lightweight, high-performance terminal streaming suite written in pure **Go**. It integrates sequential BitTorrent piece streaming, web scrapers, and local HLS proxying to stream movies, television shows, anime, and drama directly into `mpv`.

Built without Node.js or external torrent client runtimes—just a single compiled binary.

---

## Key Features

- **Embedded BitTorrent Streaming Engine**: Built-in sequential torrent streaming powered by `anacrolix/torrent`. Starts playback within seconds using sequential piece prioritization, aggressive peer unchoking, and multi-tracker announcements.
- **Netflix-Style Skip Intro / Outro**: On-screen prompts (`[S] Skip Intro`) or automated skipping powered by multi-tier detection: AniSkip API timestamps, MKV/MP4 embedded chapters, unnamed chapter heuristics, and an +85s smart jump fallback with instant undo (`[U]` / `[Backspace]`).
- **Configurable Download Cache Buffer**: Choose your buffer cushion in settings: 150 MB (~9 min), 256 MB (~16 min default), 512 MB (~32 min), or 1024 MB (full episode). BitTorrent readahead and MPV demuxer cache are fully synchronized.
- **Slow-Connection Anti-Stutter Suite**: Designed for unreliable or low-bandwidth networks. MPV automatically pauses on buffer underruns with a 15-second recharge window (`--cache-pause-wait=15`) and displays a live buffering HUD instead of audio-glitching micro-freezes.
- **Persistent Torrent Cache across Reboots**: Downloaded torrent chunks are safely retained in `~/.goovie/torrent_cache/` across application restarts, system shutdowns, and crashes. Playback resumes without re-downloading. Progressive LRU pruning automatically manages a 10 GB disk quota, and cache can be cleared on demand (`[x]`).
- **Continue Watching (Pick Up Where You Left Off)**: Tracks playback timestamps in real-time (every 5 seconds, upon pause, and on shutdown). The home screen displays your active session with a 1-key instant resume shortcut (`[c]`) to jump straight to the exact second.
- **Watch History & Binge Progression**: Displays watched checkmarks (`[✓]`) in episode navigation and automatically advances the selection cursor to the next episode upon completion.
- **Startup Health Check & Autonomous Setup**: Automatically validates system dependencies on launch. One-key auto-installation (`[1]`) configures MPV and Prowlarr, bypasses local authentication prompts, and injects top public indexers.
- **Zero-Setup Anime**: Built-in scrapers for AniList, Kitsu, and Anikoto requiring zero external indexers or torrent clients.
- **Asian Drama & K-Drama Streaming**: Headless browser automation (via Rod using local Edge or Chrome) to scrape and stream drama episodes seamlessly.
- **Clean Lifecycle & Resource Management**: Cleanly closes network listeners, drops active swarms, and unlinks temporary scratch data on shutdown.
- **Responsive Terminal UI**: Built with Bubble Tea and Lip Gloss, featuring retro ASCII typography, dynamic camera navigation, and genre artwork.

---

## System Requirements

| Media Category | Backend Engine | Requirements | Status |
| :--- | :--- | :--- | :--- |
| **Anime** | Pure Go + Anikoto + AniList | `mpv` | Ready out of the box |
| **K-Drama / Asian Media** | Rod Headless Browser | `mpv` + Edge / Chrome | Ready on Windows (Edge pre-installed) |
| **Western Movies & Shows** | Prowlarr + Pure Go BitTorrent | `mpv` + Prowlarr | Press `[ 1 ]` in setup to auto-configure |

> [!NOTE]
> Prowlarr is required to index Western movies and TV shows. Anime and Asian drama streams function immediately without Prowlarr. If Prowlarr is not installed, press `[ 1 ]` on the initial setup screen to let Goovie configure it automatically.

---

## Quickstart (Windows)

1. Download the latest `goovie.exe` from [Releases](https://github.com/G00VIE/Goovie/releases).
2. Run executable in your terminal (Windows Terminal, PowerShell, or Command Prompt):
   ```powershell
   .\goovie.exe
   ```
3. On first launch, the **Goovie System Setup & Health** dashboard will assess your environment:
   ```text
   ═══════════════════ GOOVIE SYSTEM SETUP & HEALTH ═══════════════════

     [✓ INSTALLED]  MPV Video Player
     [✗ NOT FOUND]  Prowlarr Torrent Indexer
         ↳ Western Media is a NO GO (Movies & TV shows disabled)
     [✓ INSTALLED]  Browser Engine (Rod / Edge / Chrome for K-Drama)
     [✓ BUILT-IN ]  Pure Go Anime Scraper (AniList / Kitsu / Anikoto)

   ─────────────────────────────────────────────────────────────────────
     Current Limitations:
     • Western Media is a NO GO (Prowlarr missing)
   ─────────────────────────────────────────────────────────────────────

     [ 1 ] Auto-install everything (MPV + Prowlarr + Top Indexers)
     [ 2 ] Continue to Goovie (Watch Anime + K-Drama)
     [ q ] Quit
   ```
4. Press **`1`**:
   - Downloads and installs `mpv` and `Prowlarr` via Windows Package Manager (`winget`).
   - Configures Prowlarr authentication mode to `None` to bypass initial browser setup wizards.
   - Automatically injects top public indexers (YTS, The Pirate Bay, LimeTorrents, EZTV, TorrentGalaxy).
   - Returns to the main menu ready for streaming.

---

## Controls and Keybindings

### Terminal UI Controls

| Keybinding | Action |
| :--- | :--- |
| `←` / `→` | Switch categories (Movies, TV Shows, Anime) |
| `↑` / `↓` | Navigate list items and search results |
| `[Enter]` | Confirm selection / Start playback |
| `[c]` / `[C]` | **Resume Playback / Continue Watching** (Home screen & Mode Select) • Cycle download cache buffer (in Setup & Health) |
| `[s]` | Open **System Setup & Health** dashboard |
| `[k]` / `[K]` | Toggle **Skip Intro Mode** (`Netflix Prompt` ➔ `Auto-Skip` ➔ `Off`) in Setup & Health |
| `[r]` / `[R]` | Toggle **Auto-Resume** in Setup & Health |
| `[x]` / `[X]` | **Clear Persistent Torrent Cache** in Setup & Health |
| `0` - `9` | Instant episode jump (type `12` in season menu to jump directly to Episode 12) |
| `[Backspace]` | Return to previous view |
| `[Esc]` / `[q]` | Exit application cleanly |

### In-Player (MPV) Video Controls

| Keybinding | Action |
| :--- | :--- |
| `[S]` / `[s]` | **Skip Intro / Outro** (triggers +85s smart jump if metadata is absent) |
| `[U]` / `[Backspace]` | **Instant Undo** (reverts jump back to exact previous timestamp) |
| `[I]` / `[i]` | Toggle **Live Stream Info & Cache HUD** (buffered megabytes, readahead time, positions) |
| `[Space]` | Pause / Resume playback (triggers live buffering overlay during cache underruns) |
| `[q]` | Quit player (saves playback timestamp for continuation) |

---

## Manual Installation

### 1. Video Player (`mpv`)
- **Windows**: `winget install shinchiro.mpv` or `choco install mpv`
- **macOS**: `brew install mpv`
- **Linux**: `sudo apt install mpv` or `sudo pacman -S mpv`

### 2. Torrent Indexer (`Prowlarr`)
- **Windows**: Run `winget install TeamProwlarr.Prowlarr`, or use the built-in installer (`[ 1 ]`).
- **macOS**: `brew install --cask prowlarr`
- **Docker**:
  ```bash
  docker run -d \
    --name=prowlarr \
    -p 9696:9696 \
    -v /path/to/data:/config \
    --restart unless-stopped \
    lscr.io/linuxserver/prowlarr:latest
  ```

### 3. FlareSolverr (Optional Cloudflare Bypass)
If indexers require Cloudflare challenge resolution:
```bash
docker run -d \
  --name=flaresolverr \
  -p 8191:8191 \
  -e LOG_LEVEL=info \
  --restart unless-stopped \
  ghcr.io/flaresolverr/flaresolverr:latest
```

---

## Building from Source

Prerequisites: [Go 1.22+](https://golang.org/dl/)

```bash
# Clone the repository
git clone https://github.com/G00VIE/Goovie.git
cd Goovie

# Build executable
go build -o goovie ./cmd/goovie

# Run
./goovie
```

---

## Architecture and Data Flow

```text
┌────────────────────────────────────────────────────────┐
│                      GOOVIE CLI                        │
└─────┬───────────────────┬───────────────────┬──────────┘
      │ (Anime)           │ (K-Drama)         │ (Western Movies & TV)
      ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────────────────┐
│ AniList/Kitsu │   │  Rod Browser  │   │ Prowlarr REST API (9696)  │
│  + Anikoto    │   │  (Edge/Chrome)│   └─────────────┬─────────────┘
└───────┬───────┘   └───────┬───────┘                 ▼
        │                   │           ┌───────────────────────────┐
        │                   │           │ Pure Go BitTorrent Engine │
        │                   │           │ • Multi-tracker scrape    │
        │                   │           │ • Sequential buffer       │
        │                   │           │ • High-peer swarm unchoke │
        │                   │           └─────────────┬─────────────┘
        ▼                   ▼                         ▼
┌───────────────────────────────────────────────────────────────────┐
│                       MPV Video Player                            │
└───────────────────────────────────────────────────────────────────┘
```

1. **Resolution & Search**: Queries Cinemeta, TVMaze, and AniList for metadata, then dispatches concurrent searches across Prowlarr or direct scrapers.
2. **Sequential Piece Prioritization**: Pieces are prioritized sequentially to allow immediate streaming without waiting for full torrent downloads.
3. **Local HTTP Range Server**: Streams content directly into MPV over `127.0.0.1:0` with HTTP byte-range support for seeking.
4. **Cache & Resource Management**: Downloaded chunks are preserved in persistent cache (`~/.goovie/torrent_cache/`) for instant re-watching or continuation, while network sockets and ephemeral streams are cleanly closed.

---

## Acknowledgements

- [ani-cli](https://github.com/pystardust/ani-cli) - Terminal streaming inspiration.
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal UI framework and styling.
- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Full-featured pure Go BitTorrent client implementation.
- [go-rod/rod](https://github.com/go-rod/rod) - DevTools-driven browser automation for media scraping.
- [MPV](https://mpv.io/) - Media player engine.
- [Prowlarr](https://prowlarr.com/) - Indexer aggregation proxy.

