"""
skinx - Minecraft Bedrock Skin Pack Tool

Made by ihsanharh.com
Encryption format reverse-engineered by BedrockReverse/McTools
"""

from .cache import find_premium_cache, list_packs
from .crypto import (
    SKINPACK_KEY,
    MAGIC,
    HEADER_SIZE,
    DONT_ENCRYPT,
    cfb8_encrypt,
    cfb8_decrypt,
    cfb8_decrypt_file,
    read_encrypted_header,
    write_encrypted_header,
)
from .pack import (
    collect_entries,
    decrypt_pack,
    detect_encrypted,
    encrypt_entries,
    encrypt_pack,
    encrypt_to_folder,
    extract_mcpack,
    generate_key,
    get_uuids,
    import_pack,
    is_dont_encrypt,
    pack_to_mcpack_raw,
    read_manifest,
    validate_skinpack,
    write_contents_json,
    write_manifest,
)

__version__ = "1.0.0"

__all__ = [
    "find_premium_cache",
    "list_packs",
    "SKINPACK_KEY",
    "MAGIC",
    "HEADER_SIZE",
    "DONT_ENCRYPT",
    "cfb8_encrypt",
    "cfb8_decrypt",
    "cfb8_decrypt_file",
    "read_encrypted_header",
    "write_encrypted_header",
    "collect_entries",
    "decrypt_pack",
    "detect_encrypted",
    "encrypt_entries",
    "encrypt_pack",
    "encrypt_to_folder",
    "extract_mcpack",
    "generate_key",
    "get_uuids",
    "import_pack",
    "is_dont_encrypt",
    "pack_to_mcpack_raw",
    "read_manifest",
    "validate_skinpack",
    "write_contents_json",
    "write_manifest",
]
