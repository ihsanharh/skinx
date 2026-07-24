"""Pack-level operations: encrypt, decrypt, import, and helpers."""

import json
import os
import secrets
import shutil
import uuid as uuid_lib
import zipfile
from pathlib import Path

from .cache import find_premium_cache, list_packs
from .crypto import (
    MAGIC,
    SKINPACK_IV,
    SKINPACK_KEY,
    cfb8_decrypt,
    cfb8_decrypt_file,
    cfb8_encrypt,
    read_encrypted_header,
    write_encrypted_header,
    write_encrypted_header_bytes,
)

DONT_ENCRYPT = {"manifest.json", "contents.json", "texts", "pack_icon.png"}

# ── helpers ──────────────────────────────────────────────────────────


def generate_key() -> str:
    """Random 32-char alphanumeric key for per-file encryption."""
    chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    return "".join(secrets.choice(chars) for _ in range(32))


def is_dont_encrypt(rel_path: str) -> bool:
    """True if the file should be copied as plaintext."""
    return rel_path in DONT_ENCRYPT or any(
        p in DONT_ENCRYPT for p in Path(rel_path).parts
    )


def collect_entries(source_dir: Path):
    """Walk *source_dir* and return sorted ``(rel_path, src_path, is_dir)`` tuples.

    Directories have a trailing ``/`` in *rel_path*.
    """
    entries = []
    for root, dirs, files in os.walk(source_dir):
        for fname in files:
            src = Path(root) / fname
            rel = src.relative_to(source_dir)
            entries.append((rel.as_posix(), src, False))
        for dname in sorted(dirs):
            dir_path = Path(root) / dname
            rel = dir_path.relative_to(source_dir)
            entries.append((rel.as_posix() + "/", dir_path, True))
    entries.sort(key=lambda e: e[0])
    return entries


def detect_encrypted(pack_dir: Path) -> bool:
    """Check if a folder is encrypted by looking for the magic header in contents.json."""
    contents_path = pack_dir / "contents.json"
    if not contents_path.exists():
        return False
    try:
        data = contents_path.read_bytes()
        return len(data) >= 8 and int.from_bytes(data[4:8], "little") == MAGIC
    except Exception:
        return False


def detect_path_type(path: Path) -> str:
    """Detect input type: ``'mcpack'``, ``'encrypted_folder'``, or ``'unencrypted_folder'``."""
    if path.suffix == ".mcpack":
        return "mcpack"
    if path.is_dir() and (path / "manifest.json").exists():
        return "encrypted_folder" if detect_encrypted(path) else "unencrypted_folder"
    return "unknown"


def read_manifest(path: Path) -> dict:
    with open(path) as f:
        return json.load(f)


def get_uuids(manifest: dict):
    """Return ``(header_uuid, module_uuid)``."""
    return manifest["header"]["uuid"], manifest["modules"][0]["uuid"]


def write_contents_json(filepath: Path, header_uuid: str, contents: dict):
    """Encrypt and write ``contents.json`` with the 256-byte header."""
    plaintext = json.dumps(contents, separators=(",", ":")).encode("utf-8")
    write_encrypted_header(filepath, header_uuid, plaintext, SKINPACK_KEY)


def write_manifest(filepath: Path, manifest: dict):
    """Write ``manifest.json`` in compact format."""
    filepath.write_text(json.dumps(manifest, separators=(",", ":")))


def encrypt_entries(
    entries,
    dest_dir: Path,
    contents: dict,
    skip_files: set = None,
    skip_paths: set = None,
    zip_obj=None,
):
    """Core encrypt/copy loop shared by ``encrypt_pack``, ``encrypt_to_folder``
    and ``import_pack``.

    When *zip_obj* is a :class:`zipfile.ZipFile`, encrypted files are written
    directly into the archive instead of to *dest_dir* on disk.
    """
    skip_files = skip_files or set()
    skip_paths = skip_paths or set()

    for rel_str, src_path, is_dir in entries:
        if rel_str in skip_files:
            continue

        if is_dir:
            if zip_obj is None:
                dst = dest_dir / rel_str.rstrip("/")
                dst.mkdir(parents=True, exist_ok=True)
            contents["content"].append({"path": rel_str})
            continue

        if rel_str in skip_paths:
            continue

        data = src_path.read_bytes()

        if is_dont_encrypt(rel_str):
            if zip_obj is not None:
                zip_obj.writestr(rel_str, data)
            else:
                dst = dest_dir / rel_str
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_bytes(data)
            contents["content"].append({"path": rel_str})
        else:
            key_str = generate_key()
            key = key_str.encode("utf-8")
            ciphertext = cfb8_encrypt(key, key[:16], data)

            if zip_obj is not None:
                zip_obj.writestr(rel_str, ciphertext)
            else:
                dst = dest_dir / rel_str
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_bytes(ciphertext)

            contents["content"].append({"path": rel_str, "key": key_str})


# ── decrypt ──────────────────────────────────────────────────────────


def decrypt_pack(pack_dir: Path, output_dir: Path, zip_obj=None):
    """Decrypt an entire encrypted skin pack.

    If *zip_obj* is a :class:`zipfile.ZipFile`, decrypt from the in-memory
    archive instead of reading from *pack_dir* on disk.
    """
    print(f"[+] Decrypting: {pack_dir.name}")
    output_dir.mkdir(parents=True, exist_ok=True)

    pack_dir = pack_dir.resolve()
    output_dir = output_dir.resolve()
    same_path = pack_dir == output_dir

    # Read and decrypt contents.json
    if zip_obj:
        ct = zip_obj.read("contents.json")
    else:
        contents_path = pack_dir / "contents.json"
        if not contents_path.exists():
            print("  [-] No contents.json")
            return False
        try:
            _, ct = read_encrypted_header(contents_path)
        except Exception as e:
            print(f"  [-] Failed to read contents.json: {e}")
            return False

    try:
        contents_pt = cfb8_decrypt(SKINPACK_KEY, SKINPACK_IV, ct)
        contents = json.loads(contents_pt.decode("utf-8"))
        print(f"  [+] Decrypted contents.json ({len(contents.get('content', []))} files)")
    except Exception as e:
        print(f"  [-] Failed to decrypt contents.json: {e}")
        return False

    file_keys = {}
    for item in contents.get("content", []):
        key_str = item.get("key")
        if key_str:
            file_keys[item["path"]] = key_str.encode("utf-8")

    # Collect (rel_str, data) pairs
    if zip_obj:
        entries = []
        for info in zip_obj.infolist():
            if info.is_dir() or info.filename == "contents.json":
                continue
            entries.append((info.filename, zip_obj.read(info.filename)))
    elif same_path:
        entries = []
        for root, _dirs, files in os.walk(pack_dir):
            for fname in files:
                src = Path(root) / fname
                rel = src.relative_to(pack_dir).as_posix()
                if rel == "contents.json":
                    continue
                entries.append((rel, src.read_bytes()))
    else:
        entries = []
        for root, _dirs, files in os.walk(pack_dir):
            for fname in files:
                src = Path(root) / fname
                rel = src.relative_to(pack_dir).as_posix()
                if rel == "contents.json":
                    continue
                entries.append((rel, None))  # None = read from disk later

    # Decrypt and write
    for rel_str, data in entries:
        dst = output_dir / rel_str

        if is_dont_encrypt(rel_str):
            dst.parent.mkdir(parents=True, exist_ok=True)
            if data is not None:
                dst.write_bytes(data)
            else:
                shutil.copy2(pack_dir / rel_str, dst)
            continue

        if rel_str not in file_keys:
            print(f"  [-] No key for {rel_str}, copying as-is")
            dst.parent.mkdir(parents=True, exist_ok=True)
            if data is not None:
                dst.write_bytes(data)
            else:
                shutil.copy2(pack_dir / rel_str, dst)
            continue

        try:
            if data is None:
                data = (pack_dir / rel_str).read_bytes()
            plaintext = cfb8_decrypt_file(rel_str, file_keys[rel_str], data=data)
            dst.parent.mkdir(parents=True, exist_ok=True)
            dst.write_bytes(plaintext)
            print(f"  [+] Decrypted: {rel_str}")
        except Exception as e:
            print(f"  [-] Failed to decrypt {rel_str}: {e}")
            if data is not None:
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_bytes(data)

    (output_dir / "contents.json").write_text(json.dumps(contents, indent=2))
    print(f"  [+] Done: {output_dir}")
    return True


# ── encrypt ──────────────────────────────────────────────────────────


def encrypt_pack(source_dir: Path, output_dir: Path, pack_uuid: str = None):
    """Encrypt a plaintext skin pack into a ``.mcpack`` file."""
    if not pack_uuid:
        pack_uuid = str(uuid_lib.uuid4())

    manifest = read_manifest(source_dir / "manifest.json")
    header_uuid, _ = get_uuids(manifest)

    entries = collect_entries(source_dir)
    contents = {"version": 1, "content": []}

    output_path = output_dir / f"{pack_uuid}.mcpack"
    with zipfile.ZipFile(output_path, "w", zipfile.ZIP_DEFLATED) as zf:
        encrypt_entries(entries, output_dir, contents, skip_files={"contents.json", "manifest.json"}, zip_obj=zf)

        plaintext = json.dumps(contents, separators=(",", ":")).encode("utf-8")
        zf.writestr("contents.json", write_encrypted_header_bytes(header_uuid, plaintext, SKINPACK_KEY))

        zf.writestr("manifest.json", json.dumps(manifest, separators=(",", ":")))

    print(f"[+] Created {output_path}")
    return output_path


def encrypt_to_folder(source_dir: Path, output_dir: Path, pack_uuid: str = None):
    """Encrypt a plaintext skin pack into a folder (premium_cache format)."""
    if not pack_uuid:
        pack_uuid = str(uuid_lib.uuid4())

    manifest = read_manifest(source_dir / "manifest.json")
    header_uuid, _ = get_uuids(manifest)

    output_dir.mkdir(parents=True, exist_ok=True)

    entries = collect_entries(source_dir)
    contents = {"version": 1, "content": []}
    encrypt_entries(entries, output_dir, contents, skip_files={"contents.json"})

    write_contents_json(output_dir / "contents.json", header_uuid, contents)
    write_manifest(output_dir / "manifest.json", manifest)

    print(f"[+] Encrypted to folder: {output_dir}")
    return output_dir


def pack_to_mcpack_raw(source_dir: Path, output_dir: Path, name: str = None):
    """Zip a skin pack folder into a .mcpack without encryption."""
    if not name:
        name = source_dir.name
    output_dir.mkdir(parents=True, exist_ok=True)
    output_path = output_dir / f"{name}.mcpack"
    with zipfile.ZipFile(output_path, "w", zipfile.ZIP_DEFLATED) as zf:
        for f in source_dir.rglob("*"):
            if f.is_file():
                zf.write(f, f.relative_to(source_dir))
    print(f"[+] Created {output_path}")
    return output_path


# ── import ───────────────────────────────────────────────────────────


def validate_skinpack(source_dir: Path) -> bool:
    """Check whether *source_dir* looks like a valid skin pack."""
    if not source_dir.exists():
        print("[-] Path not found")
        return False
    if not (source_dir / "manifest.json").exists():
        print("[-] Missing manifest.json")
        return False
    if not (source_dir / "skins.json").exists():
        print("[-] Missing skins.json")
        return False
    if not any(f.endswith(".png") for _, _, files in os.walk(source_dir) for f in files):
        print("[-] No PNG textures found")
        return False
    return True


def _read_preserved_entries(contents_path: Path) -> list:
    """Extract entries that must survive an import (signatures, texts, manifest)."""
    preserved = []
    if not contents_path.exists():
        return preserved
    try:
        _, ct = read_encrypted_header(contents_path)
        old_data = json.loads(cfb8_decrypt(SKINPACK_KEY, SKINPACK_IV, ct).decode("utf-8"))
        for entry in old_data.get("content", []):
            path = entry.get("path", "").lower()
            if "signature" in path or path == "manifest.json" or path.startswith("texts/"):
                preserved.append(entry)
    except Exception as e:
        print(f"  [-] Warning: Failed to read preserved entries: {e}")
    return preserved


def _is_preserved(item_name: str) -> bool:
    name_lower = item_name.lower()
    return "signature" in name_lower or name_lower in ("manifest.json", "texts")


def import_pack(source_dir: Path, target_idx: int, minecraft_dir=None):
    """Replace the skins in an existing premium pack, preserving its identity."""
    if not validate_skinpack(source_dir):
        return False

    cache = find_premium_cache(minecraft_dir)
    packs = list_packs(cache)

    if target_idx < 0 or target_idx >= len(packs):
        print(f"[-] Invalid pack index: {target_idx}")
        return False

    target_pack, target_name = packs[target_idx]
    print(f"[+] Importing into: {target_name} ({target_pack.name})")

    target_manifest = read_manifest(target_pack / "manifest.json")
    target_header_uuid, _ = get_uuids(target_manifest)
    print(f"  Target header UUID: {target_header_uuid} (preserving)")

    preserved_entries = _read_preserved_entries(target_pack / "contents.json")

    source_has_texts = (source_dir / "texts").exists()

    preserved_paths = {"signatures.json", "manifest.json"}
    # Preserve existing texts/ unless source has its own
    for entry in preserved_entries:
        path = entry.get("path", "")
        if path.startswith("texts/") and not source_has_texts:
            preserved_paths.add(path)

    for item in target_pack.iterdir():
        if item.name.lower() == "texts" and source_has_texts:
            shutil.rmtree(item)
            continue
        if _is_preserved(item.name):
            print(f"  [+] Preserved: {item.name}")
            continue
        if item.is_dir():
            shutil.rmtree(item)
        else:
            item.unlink()

    print("  [+] Encrypting source to target...")
    entries = collect_entries(source_dir)
    contents = {"version": 1, "content": []}
    encrypt_entries(
        entries,
        target_pack,
        contents,
        skip_files={"manifest.json", "contents.json"},
        skip_paths=preserved_paths,
    )

    # Re-append preserved entries (signatures, texts, manifest) that weren't re-encrypted
    for entry in preserved_entries:
        path = entry.get("path", "")
        if path.endswith("/"):
            continue
        if source_has_texts and path.startswith("texts/"):
            continue
        contents["content"].append(entry)

    contents["content"].sort(key=lambda e: e["path"])
    write_contents_json(target_pack / "contents.json", target_header_uuid, contents)

    print("[+] Import complete! Target UUIDs preserved.")
    return True
