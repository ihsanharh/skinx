"""Low-level AES-256-CFB8 encryption and Minecraft Bedrock file format."""

import struct
from pathlib import Path

from cryptography.hazmat.primitives.ciphers import Cipher, algorithms
from cryptography.hazmat.decrepit.ciphers.modes import CFB8
from cryptography.hazmat.backends import default_backend

SKINPACK_KEY = b"s5s5ejuDru4uchuF2drUFuthaspAbepE"
SKINPACK_IV = SKINPACK_KEY[:16]
MAGIC = 0x9BCFB9FC  # File magic at bytes 4-7 of header
HEADER_SIZE = 0x100  # 256-byte header before ciphertext
DONT_ENCRYPT = {"manifest.json", "contents.json", "texts", "pack_icon.png"}


def cfb8_encrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
    cipher = Cipher(algorithms.AES(key), CFB8(iv), backend=default_backend())
    encryptor = cipher.encryptor()
    return encryptor.update(data) + encryptor.finalize()


def cfb8_decrypt(key: bytes, iv: bytes, data: bytes) -> bytes:
    cipher = Cipher(algorithms.AES(key), CFB8(iv), backend=default_backend())
    decryptor = cipher.decryptor()
    return decryptor.update(data) + decryptor.finalize()


def read_encrypted_header(filepath: Path):
    """Read the 256-byte header. Returns (uuid, ciphertext)."""
    data = filepath.read_bytes()
    uuid_len = data[16]
    uuid = data[17:17 + uuid_len].decode("utf-8", errors="replace")
    ciphertext = data[HEADER_SIZE:]
    return uuid, ciphertext


def write_encrypted_header(filepath: Path, uuid: str, plaintext: bytes, key: bytes):
    """Write a file with the 256-byte header format."""
    data = write_encrypted_header_bytes(uuid, plaintext, key)
    with open(filepath, "wb") as f:
        f.write(data)


def write_encrypted_header_bytes(uuid: str, plaintext: bytes, key: bytes) -> bytes:
    """Return the 256-byte header format as bytes (no file I/O)."""
    iv = key[:16]
    uuid_bytes = uuid.encode("utf-8")
    ciphertext = cfb8_encrypt(key, iv, plaintext)

    header = bytearray()
    header += struct.pack("<IIQ", 0, MAGIC, 0)
    header += bytes([len(uuid_bytes)])
    header += uuid_bytes
    if len(header) < HEADER_SIZE:
        header += b"\x00" * (HEADER_SIZE - len(header))
    header += ciphertext
    return bytes(header)


def cfb8_decrypt_file(filepath: Path, file_key: bytes = None, data: bytes = None) -> bytes:
    """Decrypt a file, auto-detecting the header format."""
    if data is None:
        data = filepath.read_bytes()
    if len(data) >= 8 and int.from_bytes(data[4:8], "little") == MAGIC:
        ciphertext = data[HEADER_SIZE:]
        key = file_key or SKINPACK_KEY
        return cfb8_decrypt(key, key[:16], ciphertext)
    if file_key:
        return cfb8_decrypt(file_key, file_key[:16], data)
    raise ValueError("No header and no key provided")
