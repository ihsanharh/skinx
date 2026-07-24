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

```bash
git clone https://github.com/ihsanharh/skinx.git
cd skinx
pip install -e .
```

Requires Python 3.10+ and `cryptography>=49.0`.

## Usage

```bash
skinx           # interactive CLI
python -m skinx # same thing
```

### As a library

```python
from skinx import (
    encrypt_to_folder, encrypt_pack, decrypt_pack,
    import_pack, detect_encrypted, detect_path_type, pack_to_mcpack_raw,
)

# Detect what a path is
detect_path_type(path)  # "mcpack" | "encrypted_folder" | "unencrypted_folder" | "unknown"

# Decrypt a pack from premium_cache
decrypt_pack(cache_pack_path, Path("output_decrypted"))

# Encrypt a pack folder (creates .mcpack)
encrypt_pack(decrypted_path, Path("output"))

# Pack folder as .mcpack without encryption (for sharing)
pack_to_mcpack_raw(source_folder, Path("output"), "my_pack")

# Check if a folder is encrypted
detect_encrypted(pack_folder)  # True/False

# Import into Minecraft premium_cache
import_pack(source_folder, pack_index)
```

## Cache locations

The tool auto-discovers `premium_cache/skin_packs` from:

| Environment | Path |
|---|---|
| **Windows** | `%APPDATA%\Minecraft Bedrock\premium_cache\skin_packs` |
| **Linux (Wine)** via [BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux) | `~/.local/share/bedrock-on-linux/compatdata/pfx/drive_c/users/<user>/AppData/Roaming/Minecraft Bedrock/premium_cache/skin_packs` |

If auto-discovery doesn't work, pass your Minecraft Bedrock directory directly:

```bash
skinx -m ~/path/to/Minecraft Bedrock
skinx --minecraft-dir ~/path/to/Minecraft Bedrock
```

> The `bedrock-on-linux` path is from the [BedrockOnLinux](https://github.com/Wyze3306/BedrockOnLinux) launcher which runs Minecraft Bedrock via Wine on Linux.

## Credits

Encryption format reverse-engineered by [BedrockReverse/McTools](https://silica.codes/BedrockReverse/McTools).

## License

[MIT](LICENSE)
