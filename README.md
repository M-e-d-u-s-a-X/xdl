# xdl — X (Twitter) Media Downloader (CLI)

Keywords: x media downloader, twitter media downloader, x scraper, twitter image downloader, twitter video downloader, cli.

![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-green)

`xdl` is a **local-first** CLI that downloads images and videos from **a single X (Twitter) profile** that your logged-in session can see.

- No hosted API
- No accounts
- No telemetry
- Runs on your machine

> **Quality-first by design:** `xdl` intentionally trades raw speed for higher-quality media variants and more stable behavior.

---

## Download (no Go required)

Prebuilt binaries are published in GitHub **Releases**:

- **Windows (amd64)**: `xdl-windows-amd64.exe`
- **Linux (amd64)**: `xdl-linux-amd64`

Download the correct file for your OS here:
- [Releases](https://github.com/m-e-d-u-s-a-x/xdl/releases)

`xdl` ships with **embedded defaults** (no separate `essentials.json` required).

---

## Folder layout

Place the binary in a folder, and keep cookies at `cookies.txt` next to it:

```
xdl.exe (or xdl-linux-amd64)
cookies.json
```

---

## Quick start

### 1) Export cookies

`xdl` uses your existing X login via browser cookies.

1. Log into X in your browser
2. Export cookies as **JSON** (for example, using the “Cookie-Editor” extension)
3. Save the file as:

`cookies.json`

This file is read locally and is not uploaded anywhere by `xdl`.

### 2) Run

### Windows (PowerShell)

(Optional) rename the file to make commands shorter:
```powershell
ren .\xdl-windows-amd64.exe xdl.exe
```

Run:
```powershell
.\xdl.exe USERNAME
```

### Linux

Make it executable:
```bash
chmod +x ./xdl-linux-amd64
```

Run:
```bash
./xdl-linux-amd64 USERNAME
```

Example:
```bash
./xdl-linux-amd64 google
```

---

## What to expect

- Only content that your session can see will be downloadable.
- If X stops loading older media in the web UI, results may be limited as well.
- Slower-than-expected runs are often the intended quality/stability trade-off.

---

## Troubleshooting

### “403” / “Unauthorized” / downloads stop
- Cookies may be missing, expired, or exported incorrectly.
- Re-export cookies and confirm the file is exactly at `config/cookies.json`.

### Windows says “not a valid application”
- The wrong binary was used (e.g., Linux binary on Windows).
- Download `xdl-windows-amd64.exe` from Releases.

---

## Build from source (optional)

Only needed if you want to modify the code.

```bash
go build ./cmd/xdl
```

Cross-compile from Linux (Ubuntu):

```bash
mkdir -p dist

# Linux (amd64)
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/xdl-linux-amd64 ./cmd/xdl

# Windows (amd64)
env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/xdl-windows-amd64.exe ./cmd/xdl
```

---

## Legal

This project is intended for educational and personal use.
Users are responsible for complying with X’s Terms of Service and applicable laws.

---

## License

AGPL-3.0
