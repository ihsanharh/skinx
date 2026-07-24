"""Entry point for ``python -m skinx``."""

import argparse

from .cli import main

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Minecraft Bedrock Skin Pack Tool")
    parser.add_argument(
        "-m", "--minecraft-dir",
        help="Path to Minecraft Bedrock directory",
    )
    args = parser.parse_args()
    try:
        main(args.minecraft_dir)
    except KeyboardInterrupt:
        print("\nGoodbye!")
