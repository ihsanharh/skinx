"""
skinx - Minecraft Bedrock Skin Pack Tool

Made by ihsanharh.com
Encryption format reverse-engineered by BedrockReverse/McTools
"""

from .cache import find_premium_cache, list_packs
from .crypto import (
    SKINPACK_KEY,
    MAGIC,
    DONT_ENCRYPT,
    cfb8_encrypt,
    cfb8_decrypt,
    cfb8_decrypt_file,
    read_encrypted_header,
    write_encrypted_header,
)
from .pack import (
    decrypt_pack,
    detect_encrypted,
    detect_path_type,
    encrypt_pack,
    encrypt_to_folder,
    import_pack,
    pack_to_mcpack_raw,
    read_manifest,
    validate_skinpack,
)

__version__ = "1.0.0"

__all__ = [
    "find_premium_cache",
    "list_packs",
    "SKINPACK_KEY",
    "MAGIC",
    "DONT_ENCRYPT",
    "cfb8_encrypt",
    "cfb8_decrypt",
    "cfb8_decrypt_file",
    "read_encrypted_header",
    "write_encrypted_header",
    "decrypt_pack",
    "detect_encrypted",
    "detect_path_type",
    "encrypt_pack",
    "encrypt_to_folder",
    "import_pack",
    "pack_to_mcpack_raw",
    "read_manifest",
    "validate_skinpack",
]
