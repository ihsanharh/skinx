import json
import shutil
import zipfile
from pathlib import Path

import pytest

from skinx import (
    decrypt_pack,
    detect_encrypted,
    encrypt_pack,
    encrypt_to_folder,
    extract_mcpack,
    pack_to_mcpack_raw,
    read_manifest,
)
from skinx.crypto import MAGIC, read_encrypted_header

FIXTURES = Path(__file__).parent / "fixtures" / "sample_pack"


@pytest.fixture
def sample_pack(tmp_path):
    dest = tmp_path / "sample_pack"
    shutil.copytree(FIXTURES, dest)
    return dest


@pytest.fixture
def decrypted_pack(tmp_path, sample_pack):
    dest = tmp_path / "decrypted"
    shutil.copytree(sample_pack, dest)
    return dest


@pytest.fixture
def encrypted_folder(tmp_path, decrypted_pack):
    dest = tmp_path / "encrypted"
    encrypt_to_folder(decrypted_pack, dest)
    return dest


class TestDetectEncrypted:
    def test_unencrypted(self, decrypted_pack):
        assert detect_encrypted(decrypted_pack) is False

    def test_encrypted(self, encrypted_folder):
        assert detect_encrypted(encrypted_folder) is True

    def test_nonexistent(self, tmp_path):
        assert detect_encrypted(tmp_path / "nope") is False


class TestEncryptToFolder:
    def test_contents_json_has_header(self, encrypted_folder):
        data = (encrypted_folder / "contents.json").read_bytes()
        assert int.from_bytes(data[4:8], "little") == MAGIC

    def test_contents_json_not_self_referencing(self, encrypted_folder):
        _, ct = read_encrypted_header(encrypted_folder / "contents.json")
        from skinx.crypto import cfb8_decrypt, SKINPACK_KEY, SKINPACK_IV
        contents = json.loads(cfb8_decrypt(SKINPACK_KEY, SKINPACK_IV, ct).decode())
        assert "contents.json" not in [e["path"] for e in contents["content"]]

    def test_files_encrypted(self, encrypted_folder):
        assert not (encrypted_folder / "skin.png").read_bytes().startswith(b"\x89PNG")

    def test_manifest_not_encrypted(self, encrypted_folder):
        assert b'"Sample Skin Pack"' in (encrypted_folder / "manifest.json").read_bytes()

    def test_uuids_preserved(self, decrypted_pack, encrypted_folder):
        orig = read_manifest(decrypted_pack / "manifest.json")
        enc = read_manifest(encrypted_folder / "manifest.json")
        assert orig["header"]["uuid"] == enc["header"]["uuid"]
        assert orig["modules"][0]["uuid"] == enc["modules"][0]["uuid"]


class TestEncryptPack:
    def test_creates_encrypted_mcpack(self, tmp_path, decrypted_pack):
        out = tmp_path / "output"
        out.mkdir()
        result = encrypt_pack(decrypted_pack, out)
        assert result.exists()
        assert result.suffix == ".mcpack"
        with zipfile.ZipFile(result) as zf:
            data = zf.read("contents.json")
            assert int.from_bytes(data[4:8], "little") == MAGIC


class TestDecryptPack:
    def test_roundtrip(self, tmp_path, decrypted_pack, encrypted_folder):
        out = tmp_path / "decrypted"
        assert decrypt_pack(encrypted_folder, out) is True
        original = (decrypted_pack / "skin.png").read_bytes()
        assert (out / "skin.png").read_bytes() == original

    def test_missing_contents_returns_false(self, tmp_path):
        empty = tmp_path / "empty"
        empty.mkdir()
        (empty / "manifest.json").write_text("{}")
        assert decrypt_pack(empty, tmp_path / "out") is False


class TestPackToMcpackRaw:
    def test_creates_open_mcpack(self, tmp_path, decrypted_pack):
        out = tmp_path / "output"
        out.mkdir()
        result = pack_to_mcpack_raw(decrypted_pack, out, "test")
        assert result.name == "test.mcpack"
        with zipfile.ZipFile(result) as zf:
            assert zf.read("skin.png").startswith(b"\x89PNG")


class TestExtractMcpack:
    def test_roundtrip(self, tmp_path, decrypted_pack):
        mcpack_dir = tmp_path / "mcpack"
        mcpack_dir.mkdir()
        mcpack = pack_to_mcpack_raw(decrypted_pack, mcpack_dir, "test")

        out = tmp_path / "extracted"
        extract_mcpack(mcpack, out)
        assert (out / "skin.png").read_bytes() == (decrypted_pack / "skin.png").read_bytes()
