"""
skinx - Minecraft Bedrock Skin Pack Tool

Made by ihsanharh.com
Encryption format reverse-engineered by BedrockReverse/McTools
"""

from importlib.metadata import version as _get_version

from .cache import find_premium_cache, list_packs
from .crypto import (
    SKINPACK_KEY,
    MAGIC,
    cfb8_encrypt,
    cfb8_decrypt,
    cfb8_decrypt_data,
    read_encrypted_header,
    write_encrypted_header,
)
from .pack import (
    DONT_ENCRYPT,
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

__version__ = _get_version("skinx")

__all__ = [
    "find_premium_cache",
    "list_packs",
    "SKINPACK_KEY",
    "MAGIC",
    "DONT_ENCRYPT",
    "cfb8_encrypt",
    "cfb8_decrypt",
    "cfb8_decrypt_data",
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
