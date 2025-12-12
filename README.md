<<<<<<< HEAD
# xdl – X (Twitter) Media Downloader & Scraper (CLI)  

`xdl` is a simple, fast, **local** tool that downloads **images and videos** from any public X (Twitter) profile. Everything runs directly on your machine.

---

## ⭐ Key Features

- Download **media** (images + videos) from public profiles  
- Works with **the same endpoints used by the X web client**  
- Also works on private profiles **you follow**  
- 100% **local**  
- Cross-platform: Windows, Linux, macOS  
- Fast CLI workflow with minimal configuration
=======
# xdl – X (Twitter) Media Downloader & Scraper (CLI)

Keywords: twitter media downloader, x scraper, twitter image downloader, twitter video downloader, cli, golang.

[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://go.dev/)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-success.svg)](#license)

`xdl` is a **local-first** CLI tool for downloading **images and videos** from X (Twitter) profiles that your logged-in session can see.  
No hosted API, no accounts, no telemetry — everything runs on your machine.

> 🛈 xdl **intentionally trades raw speed for quality**.  
> It always prefers **HQ (high quality) media variants** and uses a more careful request pattern, so downloads may feel slower by design.

---

## ⭐ Highlights

- **High-quality media first** – always aims for the best available image/video variants.
- **Local-only** – runs entirely on your machine, no remote processing.
- **Uses your existing login** – if your browser session can see it, `xdl` can too.
- **Cross-platform** – Windows, Linux, macOS.
- **Simple CLI flow** – one binary, one command, minimal setup.
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)

---

## 🚀 Quick Start

### 1. Install Go

Requires **Go 1.21+**  
Download from: https://go.dev/dl/

<<<<<<< HEAD
1. Open Cookie-Editor  
2. Click **Export → Export as JSON**  
3. Save the result to:

```text
config/cookies.json
```

---

## Installation

Requires **Go 1.21+**
=======
### 2. Clone & build
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)

```bash
# Clone one of the repositories:
git clone https://github.com/ghostlawless/xdl.git  # GitHub (primary)
# or
git clone https://gitlab.com/medusax/xdl           # GitLab (mirror)

# Enter the project directory:
cd xdl

# Build
<<<<<<< HEAD
go build ./cmd/xdl       # Linux / macOS
go build ./cmd/xdl   # Windows
=======
go build -o xdl ./cmd/xdl       # Linux / macOS
go build -o xdl.exe ./cmd/xdl   # Windows
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)
```

### 3. Export your cookies

`xdl` uses your existing X login from the browser.

1. Install the **Cookie-Editor** extension.
2. Log into `https://x.com` in your browser.
3. Open Cookie-Editor.
4. Click **Export → Export as JSON**.
5. Save the file as:

```text
config/cookies.json
```

The file is read locally by `xdl` and is not sent anywhere else.

### 4. Run

```bash or powershell (using .exe)
xdl USERNAME
```

Example:

```bash
xdl lawlessmedusax
xdl.exe google
```

---

## 📁 Output Layout

<<<<<<< HEAD
```text
debug
logs
  /run_id
*xDownloads*
  /username_run
    /images
    /videos
```

---

## Project Structure

```text
cmd/xdl          → CLI entrypoint  
config/          → essentials  
internal/  
  scraper/       → media discovery  
  downloader/    → file downloading  
  runtime/       → timing & behavior  
  httpx/         → HTTP helpers  
  app/           → orchestration  
  utils/         → small helpers  
LICENSE  
README.md  
```
=======
By default, `xdl` saves files like this:

```text
exports/
  USERNAME/
    images/
    videos/
logs/
debug/
debug_raw/
```

- `exports/USERNAME/images/` – downloaded images  
- `exports/USERNAME/videos/` – downloaded videos  
- `logs/` and `debug*/` – extra information that can help with troubleshooting
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)

---

## 🐢 About speed & HQ mode

`xdl` is not trying to be the fastest possible downloader.  
It is designed around a few priorities:

- **Best available quality over “good enough”**  
- **Stable behavior over short bursts of speed**  
- **Friendlier request patterns over aggressive scraping**

In practice, this means:

- It may take more time per profile compared to brute-force tools.
- It is more deliberate when fetching and saving media.
- The default behavior is tuned around quality, not benchmarks.

If downloads feel slower than expected, that’s usually a **conscious trade-off**, not a performance bug.

---

## 📉 Limits imposed by X

`xdl` can only download media that X itself exposes to a logged-in user:

- If the **Media** tab on the site stops loading older posts, `xdl` will also stop seeing new media.
- Some profiles will only expose a portion of their full historical content through the normal web interface.

In other words:

> If your browser cannot see more media when you scroll to the bottom, `xdl` will not magically find more either.

This is a limitation of the platform, not of the tool.

---

## 🔐 Privacy

- No telemetry  
- No analytics  
- No external services  

Network traffic is only between **your machine and X**, using your cookies.  
Everything else happens locally.

---

## ⚖️ Legal

This project is intended for **educational and personal use**.

You are responsible for:

- Respecting X’s Terms of Service  
- Respecting copyrights and local laws  
- Only downloading content you are allowed to access and store  

The authors and contributors are **not** responsible for misuse.

---

## 🤝 Contributing

Suggestions, issues, and pull requests are welcome.

When reporting a problem, it helps to include:

<<<<<<< HEAD
**Note on HQ mode:** `xdl` now always runs in HQ (high quality) mode, prioritizing the best available media variants over raw speed. As a result, downloads may feel slower, since the tool performs extra checks and uses more cautious, human-like request pacing and batching to stay friendly to the underlying platform.

`xdl` mirrors this exact behavior:

- It fetches **every media item** delivered by X’s `UserMedia` timeline  
- When X stops supplying new pages, `xdl` reaches the **end of the visible media history**  
- No hidden or older content exists for the tool to retrieve via the normal web interface

This is **not** a bug in `xdl` — it’s a structural limitation of the X web client API.

If X’s UI does not load more media when you scroll to the bottom of the **Media** tab,  
`xdl` will not receive more media either.
=======
- OS (Windows / Linux / macOS)
- Go version
- Command you ran (`xdl ...`)
- A short description of what happened
- Relevant snippets from `logs/` (you can redact usernames/paths)
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)

---

## 📜 License

**AGPL-3.0**

You can:

- Fork  
- Study  
- Modify  
- Contribute  

as long as you follow the terms of the AGPL-3.0 license.

---

<<<<<<< HEAD
### xdl — practical, searchable, local-first media downloader
=======
### xdl — local-first, quality-focused media downloader for X.
>>>>>>> d6e1a42 (fix bug, improve readme, new behavior)
