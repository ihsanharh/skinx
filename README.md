<div align="center">

<h1>skinx</h1>

A command-line tool for Minecraft Bedrock Edition (GDK) that lets you replace marketplace skin packs with your own custom packs — including custom geometry, models, capes, and animations.

**How it works:** Minecraft Bedrock stores purchased skin packs in an encrypted `premium_cache` directory. skinx decrypts these packs, replaces the skins with your own, and re-encrypts them so the game loads your custom content as if it were the original marketplace pack.

Supports Windows and Linux ([BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux)).

Works in multiplayer world and servers that allow custom geometry, such as The Hive.

> **Note:** Only supports the GDK version (Minecraft 1.21.120 and above). You must own the marketplace packs first — this tool is for personal use with packs you have purchased.

</div>

---

## Features

| Feature | Description |
|---|---|
| **Replace skins** | Replace skins in an owned marketplace pack with your custom pack |
| **Extract pack** | Decrypt an owned pack to a folder for editing |
| **Convert pack** | Encrypt, decrypt, or convert between folder and `.mcpack` formats |

---

## Install

### Download

Download the latest binary from [Releases](https://github.com/ihsanharh/skinx/releases):

- `skinx` — Linux (run `chmod +x skinx` after downloading)
- `skinx.exe` — Windows

### Build from source

Requires [Go 1.22+](https://go.dev/doc/install).

```bash
git clone https://github.com/ihsanharh/skinx.git
cd skinx
go build -o skinx ./cmd/skinx/
```

---

## Usage

```bash
./skinx              # interactive CLI
./skinx -m <path>    # custom Minecraft Bedrock directory
```

### Windows

Double-click `skinx.exe` to launch. The terminal window closes automatically when you exit.

---

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

---

## Credits

Encryption format reverse-engineered by [BedrockReverse/McTools](https://silica.codes/BedrockReverse/McTools).

## License

[MIT](LICENSE)
