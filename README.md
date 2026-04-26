# Emajor

Natural language emoji picker for [Alfred](https://www.alfredapp.com/).

Search by meaning, not just name — `happy birthday` finds 🎂🥳🎉, `celebrate` finds 🎉 even though it's called "Party Popper".

## Features

- **Natural language search** — IDF-weighted scoring with stem matching across names, aliases, and keywords
- **Recently used** — opening the picker without a query shows your last 20 used emojis
- **Emoji icons** — each result shows the actual emoji as its icon, rendered natively via AppKit
- **Shortcode support** — press ⌘↵ to copy the `:shortcode:` instead of the emoji character
- **No dependencies** — single self-contained binary, no Python, no Node, no runtime

## Requirements

- macOS 12+
- Alfred 5 with a Powerpack license

## Installation

Download `emajor.alfredworkflow` from the [latest release](../../releases/latest) and double-click to install.

The default hotkey is **⌃⌥⇧⌘Space** (Hyper+Space). You can change it in Alfred Preferences → Workflows → Emajor.

## Usage

| Trigger | Behaviour |
|---|---|
| Hyper+Space | Open picker — shows recent emojis if no query |
| `em <query>` | Search all emojis by natural language |
| ↩ | Copy emoji to clipboard |
| ⌘↵ | Copy `:shortcode:` to clipboard |

## Building from source

```sh
# Local build + install into Alfred
make install

# Universal binary (arm64 + amd64) packaged as .alfredworkflow
make package
```

Requires Go 1.21+ and Xcode Command Line Tools.

## How it works

Emoji data is embedded at compile time from a merged [emoji-mart](https://github.com/missive/emoji-mart) + [emojilib](https://github.com/muan/emojilib) dataset (1849 emojis). At search time, each query token is scored against emoji names, aliases, and keywords using TF-IDF weighting with a completeness penalty for multi-token queries. Stem matching (`celebrate` → `celebration`) catches near-misses that exact prefix matching would miss.

Icons are rendered on first use via AppKit and cached at `~/.cache/emajor/icons/`. Recent emoji selections are stored at `~/.config/emajor/recent.json`.
