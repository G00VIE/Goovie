<p align="center">
  <img src="internal/assets/logos/goovie_logo.gif" alt="Goovie Logo" width="600">
</p>

# Goovie

Goovie is a cross-platform CLI tool built in Go that lets you search and stream movies, shows, and anime directly from your terminal. It bridges your local Prowlarr instance with `webtorrent` and `mpv` to deliver seamless, instant streaming without waiting for downloads to finish.

> [!TIP]
> **🌸 Anime lovers rejoice!** Anime streaming works completely out of the box with **zero setup required**. You don't even need Prowlarr for anime! Just open the app, search, and start watching.

## 🛠️ Prerequisites

Before running Goovie, ensure you have the following installed on your system. Each tool plays a specific role in making the magic happen!

1. **[Prowlarr](https://prowlarr.com/)**: An indexer proxy/manager. Goovie uses this to search across all your configured torrent indexers at once.
2. **[WebTorrent CLI](https://webtorrent.io/)**: The streaming torrent client. This is what makes instant playback possible without waiting for full downloads.
3. **[MPV](https://mpv.io/)**: A highly capable, robust media player. Goovie pipes the stream directly into mpv for viewing.
4. **[Flaresolverr](https://github.com/FlareSolverr/FlareSolverr) (Optional but Recommended)**: A proxy server to bypass Cloudflare protection. Useful if your indexers in Prowlarr are protected.

---

## 📦 Installation Guide

Here is a quick breakdown of how to install the prerequisites on each device!

### 🪟 Windows
You can use [Winget](https://learn.microsoft.com/en-us/windows/package-manager/winget/) (comes pre-installed on modern Windows) or Node.js.
- **Node.js (for WebTorrent)**: Download from [nodejs.org](https://nodejs.org/).
- **WebTorrent**: Open command prompt and run `npm install -g webtorrent-cli`
- **MPV**: `winget install mpv`
- **Prowlarr / Flaresolverr**: Best installed via Docker (see Docker section below) or via their Windows installers.

### 🍎 macOS
You will need [Homebrew](https://brew.sh/), the missing package manager for macOS. If you don't have it, run `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"` in your terminal.
- **WebTorrent**: `npm install -g webtorrent-cli` (Requires Node.js: `brew install node`)
- **MPV**: `brew install mpv`
- **Prowlarr / Flaresolverr**: Run via Docker or `brew install --cask prowlarr`

### 🐧 Linux (Debian/Ubuntu)
Use your built-in package manager `apt` and Node.js.
- **WebTorrent**: `npm install -g webtorrent-cli` (Requires Node.js: `sudo apt install nodejs npm`)
- **MPV**: `sudo apt install mpv`
- **Prowlarr / Flaresolverr**: Run via Docker.

### 🐳 Docker (Prowlarr & Flaresolverr)
The cleanest way to run Prowlarr and Flaresolverr is via Docker. You can use tools like [Docker Desktop](https://www.docker.com/products/docker-desktop/) or standard Docker Engine.
Just grab their official images:
- `lscr.io/linuxserver/prowlarr:latest`
- `ghcr.io/flaresolverr/flaresolverr:latest`

---

## 🚀 App Installation & Setup

### Option 1: Download Pre-compiled Binaries
You can find pre-compiled binaries on the [GitHub Releases](https://github.com/G00VIE/Goovie/releases) page.
- Windows: `goovie-windows-amd64.exe`
- Linux: `goovie-linux-amd64`
- macOS (Intel): `goovie-darwin-amd64`
- macOS (Apple Silicon): `goovie-darwin-arm64`

**Note for Linux/macOS users:** After downloading, you will need to make the binary executable before you can run it:
```bash
chmod +x goovie-*
```

### Option 2: Build from Source
If you have Go installed (1.20+), you can easily build it yourself:
```bash
git clone https://github.com/G00VIE/goovie.git
cd goovie
go build -o goovie ./cmd/goovie/
```

---

## ⚙️ Configuration & API Keys

Goovie needs to talk to Prowlarr to search for movies/shows. Here is the priority list for how Goovie connects:

1. **Environment Variables**: Top priority. For advanced users, you can bypass everything by setting these variables permanently in your shell profiles (e.g., `~/.zshrc` or `~/.bashrc`):
   - `PROWLARR_URL` (defaults to `http://localhost:9696`)
   - `PROWLARR_API_KEY`
2. **Goovie Config (`~/.goovie/config.json`)**: Second priority. If you run Prowlarr via Docker or remotely, Goovie won't be able to auto-detect the key. Instead, it will gracefully prompt you in the terminal to paste your **Prowlarr API Key** and save it in this local file for future use.
3. **Auto-Detection (Magic! ✨)**: Fallback. If you installed Prowlarr directly on your PC (Windows/Mac/Linux), Goovie will automatically find your Prowlarr `config.xml` file, extract the API key, and connect instantly!

---

## 🔍 Recommended Indexers (Prowlarr)

When setting up Prowlarr, we recommend adding the following popular public indexers to get the best search results for movies and shows:
- **1337x**
- **The Pirate Bay**
- **YTS**
- **LimeTorrents**
- **EZTV**
- **ShowRSS**
- **MagnetDL**

> [!WARNING]
> **Why do I need FlareSolverr?**
> Some of these indexers (like 1337x or The Pirate Bay) might be blocked by your ISP/DNS, or they might have aggressive Cloudflare "Checking your browser" protections. **FlareSolverr** acts as a proxy that automatically bypasses these Cloudflare checks so Prowlarr can successfully search them. If an indexer fails to connect in Prowlarr, route it through FlareSolverr!

---

## How it works (The Workflow)
1. **Search**: Enter the name of the movie or show into the beautiful terminal UI.
2. **Scrape**: Goovie concurrently queries all your active Prowlarr indexers using the API.
3. **Select**: Goovie aggregates the results and presents a clean list sorted by the number of seeders.
4. **Resolve**: Once you pick a torrent, Goovie resolves the proxy link into a raw `magnet:` URI or a `.torrent` file payload.
5. **Watch**: WebTorrent immediately buffers the media sequentially and pipes it directly into the MPV player.

## Architecture Details
For a deeper dive into the codebase and project structure, see [ARCHITECTURE.md](ARCHITECTURE.md).

## Credits
Goovie was built with and inspired by the following incredible open-source projects:

- [ani-cli](https://github.com/pystardust/ani-cli) - The original inspiration that sparked the idea to build this tool.
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The powerful, elegant TUI framework used to build the interface.
- [WebTorrent](https://github.com/webtorrent/webtorrent-cli) - The streaming torrent client that makes instant playback possible.
- [MPV](https://github.com/mpv-player/mpv) - The robust command-line video player used for rendering the media.
- [Prowlarr](https://github.com/Prowlarr/Prowlarr) - The supreme indexer manager/proxy.
