"""Interactive CLI for skinx."""

import os
import shutil
import subprocess
import zipfile
from pathlib import Path

from .cache import find_premium_cache, list_packs
from .pack import (
    decrypt_pack,
    detect_path_type,
    encrypt_to_folder,
    import_pack,
    pack_to_mcpack_raw,
    read_manifest,
    validate_skinpack,
)


def clear_screen():
    subprocess.run("cls" if os.name == "nt" else "clear", shell=True, check=False)


def get_input(prompt, default=None, allow_back=True):
    while True:
        if default:
            user_input = input(f"{prompt} [{default}]: ").strip()
        else:
            user_input = input(f"{prompt}: ").strip()
        if allow_back and user_input.lower() in ("b", "back", "menu"):
            return "BACK"
        if user_input:
            return user_input
        if default:
            return default
        print("[-] Input required")


def confirm_input(prompt, default=False):
    while True:
        user_input = input(f"{prompt} (y/N): ").strip().lower()
        if user_input in ("y", "yes"):
            return True
        if user_input in ("n", "no", ""):
            return default
        if user_input.lower() in ("b", "back", "menu"):
            return "BACK"


def pick_pack(packs, prompt="Select pack"):
    """Show pack list and return (index, pack_dir, pack_name) or None."""
    print()
    for i, (_path, name) in enumerate(packs):
        print(f"  [{i}] {name}")
    idx = get_input(prompt + " (or 'b' to go back)", allow_back=True)
    if idx == "BACK":
        return None
    try:
        idx = int(idx)
        if 0 <= idx < len(packs):
            return idx, packs[idx][0], packs[idx][1]
    except ValueError:
        pass
    print("[-] Invalid index")
    return None


def ask_output(default_name, default_dir=None):
    """Ask for output name and directory. Returns (name, dir) or 'BACK'."""
    if not default_dir:
        default_dir = str(Path.cwd())
    name = get_input("Output name", default_name)
    if name == "BACK":
        return "BACK"
    out_dir = get_input("Output directory", default_dir)
    if out_dir == "BACK":
        return "BACK"
    return name, Path(out_dir)


# ── commands ─────────────────────────────────────────────────────────


def cmd_import(minecraft_dir=None):
    while True:
        clear_screen()
        print("=== Import Custom Skins ===")
        print("Replace skins in an owned marketplace pack.")
        print("You must own the target pack first.")
        print("[B]ack to menu\n")

        source = get_input("Skin pack folder path", allow_back=True)
        if source == "BACK":
            return

        source_path = Path(source).resolve()
        if not source_path.exists():
            print(f"[-] Path not found: {source}")
            input("\nPress Enter to continue...")
            continue

        if not validate_skinpack(source_path):
            input("\nPress Enter to continue...")
            continue

        cache = find_premium_cache(minecraft_dir)
        packs = list_packs(cache)

        if not packs:
            print("[-] No skin packs found in cache")
            input("\nPress Enter to continue...")
            return

        print("\nAvailable premium packs:")
        result = pick_pack(packs, "Replace which pack?")
        if result is None:
            continue
        idx, _pack_dir, _pack_name = result

        import_pack(source_path, idx, minecraft_dir)
        input("\nPress Enter to continue...")
        return


def cmd_remove(minecraft_dir=None):
    clear_screen()
    print("=== Remove Owned Pack ===")
    print("Delete a pack from premium_cache.")
    print("You must own the pack first.")
    print("[B]ack to menu\n")

    cache = find_premium_cache(minecraft_dir)
    packs = list_packs(cache)

    if not packs:
        print("[-] No skin packs found")
        input("\nPress Enter to continue...")
        return

    result = pick_pack(packs, "Remove which pack?")
    if result is None:
        return
    _idx, pack_dir, pack_name = result

    confirm = confirm_input(f"Remove '{pack_name}'?", default=False)
    if confirm == "BACK":
        return
    if confirm:
        for item in pack_dir.iterdir():
            if item.is_dir():
                shutil.rmtree(item)
            else:
                item.unlink()
        print(f"[+] Removed: {pack_name}")
    else:
        print("[-] Cancelled")
    input("\nPress Enter to continue...")


def cmd_list_packs(minecraft_dir=None):
    clear_screen()
    print("=== Owned Marketplace Packs ===")
    print("Packs in premium_cache (must own them).")
    print("[B]ack to menu\n")

    try:
        cache = find_premium_cache(minecraft_dir)
    except FileNotFoundError as e:
        print(f"[-] {e}")
        input("\nPress Enter to continue...")
        return

    packs = list_packs(cache)
    if not packs:
        print("[-] No skin packs found in premium_cache")
        input("\nPress Enter to continue...")
        return

    print(f"Found {len(packs)} pack(s) in:\n  {cache}\n")
    print(f"{'#':<4} {'Name':<30} {'UUID':<38} {'Dir'}")
    print("-" * 95)
    for i, (path, name) in enumerate(packs):
        try:
            manifest = read_manifest(path / "manifest.json")
            uuid = manifest.get("header", {}).get("uuid", "unknown")
            version = manifest.get("header", {}).get("version", "")
            ver_str = ".".join(str(v) for v in version) if version else ""
            display = f"{name} v{ver_str}" if ver_str else name
        except Exception:
            uuid = "unknown"
            display = name
        print(f"[{i}]  {display:<30} {uuid:<38} {path.name}")

    input("\nPress Enter to continue...")


def cmd_extract(minecraft_dir=None):
    clear_screen()
    print("=== Extract Pack ===")
    print("Decrypt an owned marketplace pack to a folder.")
    print("You must own the pack first.")
    print("[B]ack to menu\n")

    try:
        cache = find_premium_cache(minecraft_dir)
    except FileNotFoundError as e:
        print(f"[-] {e}")
        input("\nPress Enter to continue...")
        return

    packs = list_packs(cache)
    if not packs:
        print("[-] No skin packs found")
        input("\nPress Enter to continue...")
        return

    result = pick_pack(packs, "Extract which pack?")
    if result is None:
        return
    _idx, pack_dir, _pack_name = result

    out = ask_output(pack_dir.name)
    if out == "BACK":
        return
    name, out_dir = out

    decrypt_pack(pack_dir, out_dir / name)
    input("\nPress Enter to continue...")


def cmd_convert():
    while True:
        clear_screen()
        print("=== Convert ===")
        print("Encrypt, decrypt, or convert any skin pack.")
        print("[B]ack to menu\n")

        source = get_input("Skin pack path (folder or .mcpack)", allow_back=True)
        if source == "BACK":
            return

        source_path = Path(source).resolve()
        if not source_path.exists():
            print(f"[-] Path not found: {source}")
            input("\nPress Enter to continue...")
            continue

        path_type = detect_path_type(source_path)

        if path_type == "mcpack":
            print("  Detected: .mcpack file")

            out = ask_output(source_path.stem)
            if out == "BACK":
                return
            name, out_dir = out

            with zipfile.ZipFile(source_path, "r") as zf:
                zf.extractall(out_dir / name)
            print(f"[+] Extracted to {out_dir / name}")

        elif path_type == "encrypted_folder":
            print("  Detected: encrypted folder")
            print("\nAction:")
            print("  [1] Decrypt")
            print("  [2] Convert to .mcpack")
            action = get_input("Select", "1")
            if action == "BACK":
                return

            out = ask_output(source_path.name)
            if out == "BACK":
                return
            name, out_dir = out

            if action == "1":
                decrypt_pack(source_path, out_dir / name)
                print(f"[+] Decrypted to {out_dir / name}")
            else:
                pack_to_mcpack_raw(source_path, out_dir, name)
                print(f"[+] Converted to {out_dir / name}.mcpack")

        elif path_type == "unencrypted_folder":
            print("  Detected: unencrypted folder")
            print("\nAction:")
            print("  [1] Encrypt")
            print("  [2] Convert to .mcpack")
            action = get_input("Select", "1")
            if action == "BACK":
                return

            out = ask_output(source_path.name)
            if out == "BACK":
                return
            name, out_dir = out

            if action == "1":
                encrypt_to_folder(source_path, out_dir / name)
                print(f"[+] Encrypted to {out_dir / name}")
            else:
                pack_to_mcpack_raw(source_path, out_dir, name)
                print(f"[+] Converted to {out_dir / name}.mcpack")

        else:
            print("[-] Not a skin pack (no manifest.json)")
            input("\nPress Enter to continue...")
            continue

        input("\nPress Enter to continue...")
        return


def cmd_info():
    clear_screen()
    print("\n" + "=" * 50)
    print("skinx - Minecraft Bedrock Skin Pack Tool")
    print("Version: 1.0")
    print("Author: ihsanharh.com")
    print("Encryption format reverse-engineered by")
    print("  BedrockReverse/McTools")
    print("=" * 50)
    print("\nManage Minecraft Bedrock GDK skin packs:")
    print()
    print("  Replace skins in owned marketplace packs")
    print("    with custom skins (supports custom geometry)")
    print()
    print("  Extract owned packs to edit their contents")
    print()
    print("  Encrypt, decrypt, or convert packs")
    print("    to .mcpack or folder format")
    print()
    print("  Requires: Minecraft Bedrock GDK version")
    print("  Cache: premium_cache/skin_packs")
    input("\nPress Enter to return to menu...")


# ── main loop ────────────────────────────────────────────────────────


def main(minecraft_dir=None):
    clear_screen()

    try:
        cache = find_premium_cache(minecraft_dir)
        print(f"[+] Found premium_cache: {cache}")
    except FileNotFoundError as e:
        print(f"[-] {e}")
        print("  Make sure Minecraft Bedrock is installed")
        return

    while True:
        clear_screen()
        print("=" * 50)
        print("      skinx - Minecraft Bedrock Skin Pack Tool")
        print("=" * 50)
        print("\nMarketplace")
        print("  [1] Import custom skins    Replace skins in an owned marketplace pack")
        print("  [2] List owned packs       Show packs in premium_cache")
        print("  [3] Remove owned pack      Delete a pack from premium_cache")
        print("\nExtract")
        print("  [4] Extract pack           Decrypt an owned pack to a folder")
        print("\nConvert")
        print("  [5] Convert pack           Encrypt, decrypt, or convert any skin pack")
        print("\n  [6] Info")
        print("  [7] Exit")

        choice = get_input("Select option", "1")

        if choice == "1":
            cmd_import(minecraft_dir)
        elif choice == "2":
            cmd_list_packs(minecraft_dir)
        elif choice == "3":
            cmd_remove(minecraft_dir)
        elif choice == "4":
            cmd_extract(minecraft_dir)
        elif choice == "5":
            cmd_convert()
        elif choice == "6":
            cmd_info()
        elif choice == "7" or choice.lower() in ("exit", "quit", "q"):
            clear_screen()
            print("Goodbye!")
            break
        else:
            print("[-] Invalid option")
