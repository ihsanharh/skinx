# skinx

Minecraft Bedrock (GDK) skin pack tool. Encrypt, decrypt, convert and manage skin packs in `premium_cache`.

Supports Windows and Linux ([BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux)).

> Only supports the GDK version (Minecraft 1.21.120 and above).

## Features

- **Import** custom skins into owned marketplace packs (supports custom geometry, models, animations)
- **Extract** (decrypt) owned packs to edit their contents
- **Convert** any skin pack — encrypt, decrypt, or convert between folder and `.mcpack` formats

> **Note:** You must own the marketplace packs first. This tool is for personal use with packs you have purchased.

> **Disclaimer:** This tool modifies Minecraft game files and may violate Mojang's Terms of Service. Use at your own risk.

## Install

### Download

Download the latest binary from [Releases](https://github.com/ihsanharh/skinx/releases):

- `skinx` — Linux
- `skinx.exe` — Windows

### Build from source

Requires [Go 1.22+](https://go.dev/doc/install).

```bash
git clone https://github.com/ihsanharh/skinx.git
cd skinx
go build -o skinx ./cmd/skinx/
```

## Usage

```bash
./skinx              # interactive CLI
./skinx -m <path>    # custom Minecraft Bedrock directory
```

### Windows

Double-click `skinx.exe` to launch. The terminal window closes automatically when you exit.

## Cache locations

The tool auto-discovers `premium_cache/skin_packs` from:

| Environment | Path |
|---|---|
| **Windows** | `%APPDATA%\Minecraft Bedrock\premium_cache\skin_packs` |
| **Linux (Wine)** via [BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux) | `~/.local/share/bedrock-on-linux/compatdata/pfx/drive_c/users/<user>/AppData/Roaming/Minecraft Bedrock/premium_cache/skin_packs` |

If auto-discovery doesn't work, pass your Minecraft Bedrock directory directly:

```bash
./skinx -m ~/path/to/Minecraft Bedrock
./skinx --minecraft-dir ~/path/to/Minecraft Bedrock
```

> The `bedrock-on-linux` path is from the [BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux) launcher which runs Minecraft Bedrock via Wine on Linux.

## Development

```bash
# Run tests
go test ./... -v

# Build for current platform
go build -o skinx ./cmd/skinx/

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o skinx.exe ./cmd/skinx/

# Build optimized binaries
go build -ldflags="-s -w" -o skinx ./cmd/skinx/
```

## Credits

Encryption format reverse-engineered by [BedrockReverse/McTools](https://silica.codes/BedrockReverse/McTools).

## License

[MIT](LICENSE)
