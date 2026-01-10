# 🌌 Antigravity Dashboard TUI

**Antigravity Dashboard TUI** is a single-process terminal interface for monitoring Antigravity quotas across multiple accounts. It provides real-time visibility into Claude and Gemini API usage, reset timers, and subscription tiers, all within a keyboard-driven interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Based on [Antigravity Dashboard](https://github.com/OmerFarukOruc/antigravity-dashboard).

## ✨ Features

- **Multi-Account Monitoring**: Track quotas across multiple Google Cloud accounts.
- **Real-Time Quotas**: Progress bars for Claude and Gemini API usage.
- **Reset Countdown**: Live timers for quota resets.
- **Tier Detection**: Automatic identification of FREE vs. PRO tiers.
- **Single-Process**: Standalone Go binary with integrated services.

## 🚀 Installation

### Using Go

```bash
go install github.com/j-veylop/antigravity-dashboard-tui/cmd/antigravity-dashboard-tui@latest
```

### From Source

```bash
git clone https://github.com/j-veylop/antigravity-dashboard-tui.git
cd antigravity-tui
go build -o antigravity ./cmd/antigravity
./antigravity
```

## ⚙️ Configuration

Antigravity Dashboard TUI looks for configuration in environment variables or a `.env` file.

### Environment Variables

| Variable                 | Description                   | Default                                        |
| ------------------------ | ----------------------------- | ---------------------------------------------- |
| `GOOGLE_CLIENT_ID`       | Google OAuth Client ID        | **Required**                                   |
| `GOOGLE_CLIENT_SECRET`   | Google OAuth Client Secret    | **Required**                                   |
| `DATABASE_PATH`          | Path to SQLite usage database | `~/.config/opencode/antigravity-tui/usage.db`  |
| `ACCOUNTS_PATH`          | Path to accounts JSON file    | `~/.config/opencode/antigravity-accounts.json` |
| `QUOTA_REFRESH_INTERVAL` | How often to poll Google API  | `30s`                                          |

### `.env` File Locations

The application searches for a `.env` file in:

1. Current directory
2. `~/.config/opencode/antigravity-tui/.env`
3. `~/.config/opencode/.env`
4. `~/.antigravity/.env`

You can copy the example file to get started:

```bash
cp .env.example .env
```

Then edit `.env` and add your Google OAuth credentials.

## ⌨️ Keyboard Shortcuts

### Global Navigation

| Key             | Action                                            |
| --------------- | ------------------------------------------------- |
| `1` - `4`       | Switch Tabs (Dashboard, Accounts, Logs, Settings) |
| `Tab`           | Next Tab                                          |
| `Shift+Tab`     | Previous Tab                                      |
| `r`             | Refresh all data                                  |
| `?`             | Toggle help overlay                               |
| `q` or `Ctrl+C` | Quit                                              |

### Tab-Specific Shortcuts

#### 📊 Dashboard

| Key               | Action           |
| ----------------- | ---------------- |
| `n` or `j` or `↓` | Next Account     |
| `p` or `k` or `↑` | Previous Account |

#### 👥 Accounts

| Key        | Action                     |
| ---------- | -------------------------- |
| `Enter`    | Switch to selected account |
| `n` or `a` | Add new account            |
| `d`        | Delete selected account    |
| `Esc`      | Cancel / Close form        |

#### 📜 Lists (Logs/Accounts)

| Key             | Action                                      |
| --------------- | ------------------------------------------- |
| `j` / `k`       | Move selection down/up                      |
| `PgUp` / `PgDn` | Scroll page up/down                         |
| `g` / `G`       | Jump to top/bottom                          |
| `/`             | Filter list                                 |
| `c`             | Copy (Request ID in Logs, Path in Settings) |

## 🏗️ Architecture

- **Bubble Tea**: MVU pattern for the terminal UI.
- **Service Manager**: Manages background workers for account and quota syncing.
- **Event-Driven**: Asynchronous updates via centralized event channel.
- **Shared State**: Thread-safe container synchronized across tabs.

## 📄 License

MIT © [J-Veylop](https://github.com/j-veylop)
