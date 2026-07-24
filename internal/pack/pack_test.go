package pack

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ihsanharh/skinx/internal/bedrock"
)

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0644)
	})
}

func TestDetectEncrypted_Unencrypted(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "sample_pack")
	if err := copyDir("testdata/sample_pack", packDir); err != nil {
		t.Fatal(err)
	}
	if DetectEncrypted(packDir) {
		t.Error("expected false for unencrypted pack")
	}
}

func TestDetectEncrypted_Encrypted(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}
	if !DetectEncrypted(dstDir) {
		t.Error("expected true for encrypted pack")
	}
}

func TestDetectEncrypted_Nonexistent(t *testing.T) {
	if DetectEncrypted("/nonexistent/path") {
		t.Error("expected false for nonexistent path")
	}
}

func TestEncryptToFolder_ContentsJSONHasHeader(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "contents.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 {
		t.Fatal("contents.json too small")
	}
	magic := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	if magic != bedrock.Magic {
		t.Errorf("expected magic 0x%X, got 0x%X", bedrock.Magic, magic)
	}
}

func TestEncryptToFolder_ContentsJSONNotSelfReferencing(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}

	_, ct, err := bedrock.ReadEncryptedHeader(filepath.Join(dstDir, "contents.json"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := bedrock.CFB8Decrypt(bedrock.SkinpackKey, bedrock.SkinpackIV, ct)
	if err != nil {
		t.Fatal(err)
	}
	var contents Contents
	if err := json.Unmarshal(pt, &contents); err != nil {
		t.Fatal(err)
	}
	for _, entry := range contents.Content {
		if entry.Path == "contents.json" {
			t.Error("contents.json should not reference itself")
		}
	}
}

func TestEncryptToFolder_FilesEncrypted(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "skin.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		t.Error("skin.png should be encrypted, not plaintext PNG")
	}
}

func TestEncryptToFolder_ManifestNotEncrypted(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dstDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(string(data), "Sample Skin Pack") {
		t.Error("manifest.json should contain plaintext name")
	}
}

func TestEncryptToFolder_UUIDsPreserved(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	dstDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, dstDir); err != nil {
		t.Fatal(err)
	}

	orig, err := ReadManifest(filepath.Join(srcDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := ReadManifest(filepath.Join(dstDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if orig.Header.UUID != enc.Header.UUID {
		t.Errorf("UUID mismatch: %s != %s", orig.Header.UUID, enc.Header.UUID)
	}
	if orig.Modules[0].UUID != enc.Modules[0].UUID {
		t.Errorf("Module UUID mismatch: %s != %s", orig.Modules[0].UUID, enc.Modules[0].UUID)
	}
}

func TestEncryptPack_CreatesEncryptedMcpack(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	result, err := EncryptPack(srcDir, outDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(result) != ".mcpack" {
		t.Errorf("expected .mcpack extension, got %s", filepath.Ext(result))
	}

	r, err := zip.OpenReader(result)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "contents.json" {
			rc, _ := f.Open()
			data := make([]byte, 8)
			rc.Read(data)
			rc.Close()
			magic := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
			if magic != bedrock.Magic {
				t.Errorf("contents.json should have magic header, got 0x%X", magic)
			}
		}
	}
}

func TestDecryptPack_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	encDir := filepath.Join(dir, "encrypted")
	if err := EncryptToFolder(srcDir, encDir); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "decrypted")
	if err := DecryptPack(encDir, outDir); err != nil {
		t.Fatal(err)
	}

	orig, err := os.ReadFile(filepath.Join(srcDir, "skin.png"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "skin.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != string(got) {
		t.Error("decrypted skin.png does not match original")
	}
}

func TestDecryptPack_MissingContentsReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	os.MkdirAll(emptyDir, 0755)
	os.WriteFile(filepath.Join(emptyDir, "manifest.json"), []byte("{}"), 0644)

	outDir := filepath.Join(dir, "out")
	err := DecryptPack(emptyDir, outDir)
	if err == nil {
		t.Error("expected error for missing contents.json")
	}
}

func TestPackToMcpackRaw_CreatesOpenMcpack(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "output")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := PackToMcpackRaw(srcDir, outDir, "test"); err != nil {
		t.Fatal(err)
	}

	mcpackPath := filepath.Join(outDir, "test.mcpack")
	if _, err := os.Stat(mcpackPath); os.IsNotExist(err) {
		t.Fatal("test.mcpack not created")
	}

	r, err := zip.OpenReader(mcpackPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "skin.png" {
			rc, _ := f.Open()
			data := make([]byte, 8)
			rc.Read(data)
			rc.Close()
			if data[0] != 0x89 || data[1] != 'P' || data[2] != 'N' || data[3] != 'G' {
				t.Error("skin.png should be unencrypted PNG")
			}
		}
	}
}

func TestExtractMcpack_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "source")
	if err := copyDir("testdata/sample_pack", srcDir); err != nil {
		t.Fatal(err)
	}
	mcpackDir := filepath.Join(dir, "mcpack")
	if err := os.MkdirAll(mcpackDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := PackToMcpackRaw(srcDir, mcpackDir, "test"); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "extracted")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(filepath.Join(mcpackDir, "test.mcpack"))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, _ := f.Open()
		data := make([]byte, 8)
		rc.Read(data)
		rc.Close()
	}

	orig, err := os.ReadFile(filepath.Join(srcDir, "skin.png"))
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := zip.OpenReader(filepath.Join(mcpackDir, "test.mcpack"))
	defer r2.Close()
	for _, f := range r2.File {
		if f.Name == "skin.png" {
			rc, _ := f.Open()
			got := make([]byte, len(orig))
			rc.Read(got)
			rc.Close()
			if string(orig) != string(got) {
				t.Error("extracted skin.png does not match original")
			}
		}
	}
}

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
