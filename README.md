<p align="center">
  <img src="internal/assets/logos/goovie_logo.gif" alt="Goovie Logo" width="600">
</p>

# 🎬 Goovie

> *"Because why wait 45 minutes for a torrent to download when you can just stream it immediately in terminal glory?"* 🍿⚡

**Goovie** is an ultra-fast, aesthetic terminal streaming suite written in 100% pure **Go**. Whether you want to channel your inner **Deadpool** for blockbuster movies, cook up some prestige television with **Heisenberg**, curse-spirit binge anime with **Itadori**, or cry your eyes out to your favorite **K-Drama**, Goovie hooks directly into your terminal and pipes live media into `mpv`.

Zero Node.js. Zero WebTorrent CLI bloat. Pure compiled Go power. 🏎️💨

---

## ✨ Features That Slap

- 🚀 **Pure Go BitTorrent Engine**: Built-in streaming torrent engine (`anacrolix/torrent`) compiled directly into the binary. Instant sequential piece streaming, multi-tracker announcements, and aggressive unchoking for max swarm speed.
- 🩺 **Startup Health Check & 1-Click Auto-Installer**: Fresh PC? No problem. Goovie checks your setup on boot. Press **`[ 1 ]`** to auto-install MPV, Prowlarr, bypass browser login wizards, and auto-inject top indexers automatically!
- 🌸 **Zero-Setup Anime**: Powered by pure Go scrapers (AniList, Kitsu, and Anikoto). Works right out of the box with **zero** external indexers or torrent clients needed.
- 🫰 **Asian & K-Drama Streaming**: Embedded headless Rod engine (runs right on your built-in Edge or Chrome) to scrape and stream drama episodes seamlessly.
- 🧹 **Zero Open Ports & Permanent Purge**: When you exit or finish an episode, all streaming servers shut down tight, and all temp torrent data is permanently unlinked from your disk (bypassing the Recycle Bin completely).
- 🎨 **Aesthetic Bubble Tea TUI**: Retro ASCII banners, live spinners, dynamic camera views, and custom art for every genre.

---

## 🚦 System Requirements & Cheat Sheet

| Media Category | What Powers It? | What Do You Need? | Out-of-the-Box Status |
| :--- | :--- | :--- | :--- |
| **🌸 Anime** | Pure Go + Anikoto + AniList | `mpv` | 🟢 **Ready immediately** |
| **🫰 K-Drama** | Rod Headless Browser | `mpv` + Edge/Chrome | 🟢 **Ready on Windows (Edge pre-installed)** |
| **🍿 Western Movies & Shows** | Prowlarr + Pure Go BitTorrent | `mpv` + Prowlarr | 🟡 **Press `[ 1 ]` to auto-install!** |

> [!NOTE]
> **No Prowlarr?** Western media is a **NO GO** without indexers, but Anime and K-Drama will work instantly! If you want full access to Hollywood movies and TV shows, simply tap **`[ 1 ]`** on the setup screen to let Goovie handle the heavy lifting.

---

## ⚡ The 1-Click Quickstart (Windows)

1. Grab the latest [goovie.exe](https://github.com/G00VIE/Goovie/releases) release.
2. Double-click or run in your favorite terminal (`wt`, PowerShell, Command Prompt):
   ```powershell
   .\goovie.exe
   ```
3. **If it's your first run**, the **Goovie Setup & Health** dashboard will appear:
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
   - Silently downloads and installs `mpv` and `Prowlarr` via Windows Package Manager (`winget`).
   - Pre-seeds Prowlarr configuration (`<AuthenticationMethod>None</AuthenticationMethod>`) so you **never have to deal with browser login wizards**.
   - Auto-injects top public indexers (**YTS**, **The Pirate Bay**, **LimeTorrents**, **EZTV**, **TorrentGalaxy**).
   - Done! You're ready to stream anything in existence. 🥂

---

## 🕹️ Controls & Navigation

| Keybinding | Action |
| :--- | :--- |
| `←` / `→` | Switch Genres (**Movies**, **TV Shows**, **Anime**) |
| `↑` / `↓` | Navigate menus and search results |
| `[Enter]` | Confirm selection / Play stream |
| `[s]` | Open **System Setup & Health** anytime from menus |
| `0` - `9` | Instant jump (type `12` in season menu to jump straight to Episode 12) |
| `[Backspace]` | Go back to the previous screen |
| `[Esc]` / `[q]` | Exit cleanly (auto-purges all temp dumps and frees ports) |

---

## 🛠️ Manual Installation (For Power Users & Other OSes)

### 1. The Video Player (`mpv`)
- **Windows**: `winget install shinchiro.mpv` or `choco install mpv`
- **macOS**: `brew install mpv`
- **Linux**: `sudo apt install mpv` / `sudo pacman -S mpv`

### 2. The Western Media Indexer (`Prowlarr`)
- **Windows**: Run `winget install TeamProwlarr.Prowlarr`, or let Goovie install it for you!
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

### 3. FlareSolverr (Optional Proxy for Cloudflare)
If your ISP blocks indexers or Cloudflare throws captcha roadblocks at Prowlarr, spin up FlareSolverr:
```bash
docker run -d \
  --name=flaresolverr \
  -p 8191:8191 \
  -e LOG_LEVEL=info \
  --restart unless-stopped \
  ghcr.io/flaresolverr/flaresolverr:latest
```

---

## 🏗️ Build From Source

Make sure you have [Go 1.22+](https://golang.org/dl/) installed:

```bash
# Clone the repository
git clone https://github.com/G00VIE/Goovie.git
cd Goovie

# Build the executable
go build -o goovie.exe ./cmd/goovie

# Run it!
./goovie.exe
```

---

## 🧠 How It Works Under The Hood

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

1. **Resolution & Search**: Queries Cinemeta, TVMaze, and AniList for metadata, then dispatches lightning searches across Prowlarr or our scrapers.
2. **Sequential Piece Prioritization**: Instead of waiting for full torrent downloads, chunks are prioritized sequentially so playback starts in seconds.
3. **Local HTTP Range Server**: Streams content directly into MPV over `127.0.0.1:0` with HTTP byte-range support for instant seeking forward and backward.
4. **Clean Exit**: When MPV exits, the engine unlinks all torrent cache files and closes every network listener. Zero residue.

---

## 🤝 Credits & Shoutouts

Goovie stands on the shoulders of giants:
- [ani-cli](https://github.com/pystardust/ani-cli) - The spark that inspired terminal streaming.
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Beautiful, expressive terminal aesthetics.
- [anacrolix/torrent](https://github.com/anacrolix/torrent) - Full-featured pure Go BitTorrent client implementation.
- [go-rod/rod](https://github.com/go-rod/rod) - DevTools-driven browser automation for zero-hassle drama scraping.
- [MPV](https://mpv.io/) - The undisputed king of media players.
- [Prowlarr](https://prowlarr.com/) - The ultimate indexer proxy and aggregator.

---

<p align="center">
  <i>Crafted with 💜 and Go. Grab your popcorn and enjoy the show!</i>
</p>
