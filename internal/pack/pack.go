package pack

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ihsanharh/skinx/internal/bedrock"
)

var DontEncrypt = map[string]bool{
	"manifest.json": true,
	"contents.json": true,
	"pack_icon.png": true,
}

type Entry struct {
	RelPath string
	SrcPath string
	IsDir   bool
}

type Manifest struct {
	FormatVersion float64 `json:"format_version"`
	Header        Header  `json:"header"`
	Modules       []Module `json:"modules"`
}

type Header struct {
	Name             string    `json:"name"`
	UUID             string    `json:"uuid"`
	Version          []float64 `json:"version"`
	MinEngineVersion []float64 `json:"min_engine_version"`
}

type Module struct {
	Type    string    `json:"type"`
	UUID    string    `json:"uuid"`
	Version []float64 `json:"version"`
}

type Contents struct {
	Version float64           `json:"version"`
	Content []ContentsEntry   `json:"content"`
}

type ContentsEntry struct {
	Path string `json:"path"`
	Key  string `json:"key,omitempty"`
}

type PackInfo struct {
	Dir  string
	Name string
}

func zipWrite(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func generateKey() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	key := make([]byte, 32)
	for i := range key {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		key[i] = chars[n.Int64()]
	}
	return string(key)
}

func isDontEncrypt(relPath string) bool {
	if DontEncrypt[relPath] {
		return true
	}
	parts := strings.Split(relPath, "/")
	for _, p := range parts {
		if DontEncrypt[p] {
			return true
		}
	}
	if strings.HasPrefix(relPath, "texts/") {
		return true
	}
	return false
}

func CollectEntries(sourceDir string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			entries = append(entries, Entry{RelPath: rel + "/", SrcPath: path, IsDir: true})
		} else {
			entries = append(entries, Entry{RelPath: rel, SrcPath: path, IsDir: false})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelPath < entries[j].RelPath
	})
	return entries, nil
}

func DetectEncrypted(packDir string) bool {
	contentsPath := filepath.Join(packDir, "contents.json")
	data, err := os.ReadFile(contentsPath)
	if err != nil {
		return false
	}
	if len(data) < 8 {
		return false
	}
	return uint32(data[4])|uint32(data[5])<<8|uint32(data[6])<<16|uint32(data[7])<<24 == bedrock.Magic
}

func DetectPathType(path string) string {
	if strings.HasSuffix(path, ".mcpack") {
		return "mcpack"
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "unknown"
	}
	if _, err := os.Stat(filepath.Join(path, "manifest.json")); err == nil {
		if DetectEncrypted(path) {
			return "encrypted_folder"
		}
		return "unencrypted_folder"
	}
	return "unknown"
}

func ReadManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var m Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func getUUIDs(m *Manifest) (string, string) {
	return m.Header.UUID, m.Modules[0].UUID
}

func writeContentsJSON(path, headerUUID string, contents *Contents) error {
	plaintext, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	data, err := bedrock.WriteEncryptedHeaderBytes(headerUUID, plaintext, bedrock.SkinpackKey)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeManifest(path string, m *Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func encryptEntries(entries []Entry, destDir string, contents *Contents, skipFiles, skipPaths map[string]bool, zw *zip.Writer) error {
	for _, entry := range entries {
		if skipFiles[entry.RelPath] {
			continue
		}

		if entry.IsDir {
			if zw == nil {
				dir := filepath.Join(destDir, strings.TrimSuffix(entry.RelPath, "/"))
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}
			contents.Content = append(contents.Content, ContentsEntry{Path: entry.RelPath})
			continue
		}

		if skipPaths[entry.RelPath] {
			continue
		}

		data, err := os.ReadFile(entry.SrcPath)
		if err != nil {
			return err
		}

		if isDontEncrypt(entry.RelPath) {
			if zw != nil {
				if err := zipWrite(zw, entry.RelPath, data); err != nil {
					return err
				}
			} else {
				dst := filepath.Join(destDir, entry.RelPath)
				if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(dst, data, 0644); err != nil {
					return err
				}
			}
			contents.Content = append(contents.Content, ContentsEntry{Path: entry.RelPath})
		} else {
			keyStr := generateKey()
			key := []byte(keyStr)
			ciphertext, err := bedrock.CFB8Encrypt(key, key[:16], data)
			if err != nil {
				return err
			}

			if zw != nil {
				if err := zipWrite(zw, entry.RelPath, ciphertext); err != nil {
					return err
				}
			} else {
				dst := filepath.Join(destDir, entry.RelPath)
				if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(dst, ciphertext, 0644); err != nil {
					return err
				}
			}
			contents.Content = append(contents.Content, ContentsEntry{Path: entry.RelPath, Key: keyStr})
		}
	}
	return nil
}

func EncryptPack(sourceDir, outputDir string) (string, error) {
	manifest, err := ReadManifest(filepath.Join(sourceDir, "manifest.json"))
	if err != nil {
		return "", err
	}
	headerUUID, _ := getUUIDs(manifest)

	entries, err := CollectEntries(sourceDir)
	if err != nil {
		return "", err
	}
	contents := &Contents{Version: 1}

	packUUID, err := generateUUID()
	if err != nil {
		return "", err
	}
	outputPath := filepath.Join(outputDir, packUUID+".mcpack")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	skipFiles := map[string]bool{"contents.json": true, "manifest.json": true}
	if err := encryptEntries(entries, outputDir, contents, skipFiles, nil, zw); err != nil {
		return "", err
	}

	plaintext, err := json.Marshal(contents)
	if err != nil {
		return "", err
	}
	headerBytes, err := bedrock.WriteEncryptedHeaderBytes(headerUUID, plaintext, bedrock.SkinpackKey)
	if err != nil {
		return "", err
	}
	if err := zipWrite(zw, "contents.json", headerBytes); err != nil {
		return "", err
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	if err := zipWrite(zw, "manifest.json", manifestBytes); err != nil {
		return "", err
	}

	if err := zw.Close(); err != nil {
		return "", err
	}
	return outputPath, nil
}

func EncryptToFolder(sourceDir, outputDir string) error {
	manifest, err := ReadManifest(filepath.Join(sourceDir, "manifest.json"))
	if err != nil {
		return err
	}
	headerUUID, _ := getUUIDs(manifest)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	entries, err := CollectEntries(sourceDir)
	if err != nil {
		return err
	}
	contents := &Contents{Version: 1}

	skipFiles := map[string]bool{"contents.json": true}
	if err := encryptEntries(entries, outputDir, contents, skipFiles, nil, nil); err != nil {
		return err
	}

	if err := writeContentsJSON(filepath.Join(outputDir, "contents.json"), headerUUID, contents); err != nil {
		return err
	}
	return writeManifest(filepath.Join(outputDir, "manifest.json"), manifest)
}

func PackToMcpackRaw(sourceDir, outputDir, name string) error {
	if name == "" {
		name = filepath.Base(sourceDir)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(outputDir, name+".mcpack")

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return zipWrite(zw, rel, data)
	})
	if err != nil {
		return err
	}
	return zw.Close()
}

func DecryptPack(packDir, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	packDir, err := filepath.Abs(packDir)
	if err != nil {
		return err
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return err
	}

	contentsPath := filepath.Join(packDir, "contents.json")
	if _, err := os.Stat(contentsPath); os.IsNotExist(err) {
		return fmt.Errorf("no contents.json")
	}

	_, ct, err := bedrock.ReadEncryptedHeader(contentsPath)
	if err != nil {
		return fmt.Errorf("failed to read contents.json: %w", err)
	}

	contentsPT, err := bedrock.CFB8Decrypt(bedrock.SkinpackKey, bedrock.SkinpackIV, ct)
	if err != nil {
		return fmt.Errorf("failed to decrypt contents.json: %w", err)
	}

	var contents Contents
	if err := json.Unmarshal(contentsPT, &contents); err != nil {
		return fmt.Errorf("failed to parse contents.json: %w", err)
	}

	fileKeys := make(map[string][]byte)
	for _, item := range contents.Content {
		if item.Key != "" {
			fileKeys[item.Path] = []byte(item.Key)
		}
	}

	type fileEntry struct {
		relPath string
		data    []byte
	}
	var entries []fileEntry

	err = filepath.Walk(packDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(packDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "contents.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, fileEntry{relPath: rel, data: data})
		return nil
	})
	if err != nil {
		return err
	}

	for _, entry := range entries {
		dst := filepath.Join(outputDir, entry.relPath)

		if isDontEncrypt(entry.relPath) {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, entry.data, 0644); err != nil {
				return err
			}
			continue
		}

		key, ok := fileKeys[entry.relPath]
		if !ok {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, entry.data, 0644); err != nil {
				return err
			}
			continue
		}

		plaintext, err := bedrock.CFB8DecryptData(entry.data, key)
		if err != nil {
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			os.WriteFile(dst, entry.data, 0644)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, plaintext, 0644); err != nil {
			return err
		}
	}

	contentsJSON, err := json.MarshalIndent(contents, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "contents.json"), contentsJSON, 0644)
}

func ValidateSkinpack(sourceDir string) bool {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "manifest.json")); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "skins.json")); os.IsNotExist(err) {
		return false
	}
	hasPNG := false
	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".png") {
			hasPNG = true
		}
		return nil
	})
	return hasPNG
}

func readPreservedEntries(contentsPath string) []ContentsEntry {
	var preserved []ContentsEntry
	if _, err := os.Stat(contentsPath); os.IsNotExist(err) {
		return preserved
	}

	_, ct, err := bedrock.ReadEncryptedHeader(contentsPath)
	if err != nil {
		return preserved
	}

	oldData, err := bedrock.CFB8Decrypt(bedrock.SkinpackKey, bedrock.SkinpackIV, ct)
	if err != nil {
		return preserved
	}

	var oldContents Contents
	if err := json.Unmarshal(oldData, &oldContents); err != nil {
		return preserved
	}

	for _, entry := range oldContents.Content {
		path := strings.ToLower(entry.Path)
		if strings.Contains(path, "signature") || path == "manifest.json" || strings.HasPrefix(path, "texts/") {
			preserved = append(preserved, entry)
		}
	}
	return preserved
}

func isPreserved(itemName string) bool {
	lower := strings.ToLower(itemName)
	return strings.Contains(lower, "signature") || lower == "manifest.json" || lower == "texts"
}

func ImportPack(sourceDir, targetPack string) error {
	if !ValidateSkinpack(sourceDir) {
		return fmt.Errorf("invalid skin pack")
	}

	targetManifest, err := ReadManifest(filepath.Join(targetPack, "manifest.json"))
	if err != nil {
		return err
	}
	targetHeaderUUID, _ := getUUIDs(targetManifest)

	preservedEntries := readPreservedEntries(filepath.Join(targetPack, "contents.json"))

	sourceHasTexts := false
	if _, err := os.Stat(filepath.Join(sourceDir, "texts")); err == nil {
		sourceHasTexts = true
	}

	preservedPaths := map[string]bool{"signatures.json": true, "manifest.json": true}
	for _, entry := range preservedEntries {
		if strings.HasPrefix(entry.Path, "texts/") && !sourceHasTexts {
			preservedPaths[entry.Path] = true
		}
	}

	items, _ := os.ReadDir(targetPack)
	for _, item := range items {
		if strings.ToLower(item.Name()) == "texts" && sourceHasTexts {
			os.RemoveAll(filepath.Join(targetPack, item.Name()))
			continue
		}
		if isPreserved(item.Name()) {
			continue
		}
		if item.IsDir() {
			os.RemoveAll(filepath.Join(targetPack, item.Name()))
		} else {
			os.Remove(filepath.Join(targetPack, item.Name()))
		}
	}

	entries, err := CollectEntries(sourceDir)
	if err != nil {
		return err
	}
	contents := &Contents{Version: 1}

	skipFiles := map[string]bool{"manifest.json": true, "contents.json": true}
	if err := encryptEntries(entries, targetPack, contents, skipFiles, preservedPaths, nil); err != nil {
		return err
	}

	for _, entry := range preservedEntries {
		if strings.HasSuffix(entry.Path, "/") {
			continue
		}
		if sourceHasTexts && strings.HasPrefix(entry.Path, "texts/") {
			continue
		}
		contents.Content = append(contents.Content, entry)
	}

	sort.Slice(contents.Content, func(i, j int) bool {
		return contents.Content[i].Path < contents.Content[j].Path
	})

	return writeContentsJSON(filepath.Join(targetPack, "contents.json"), targetHeaderUUID, contents)
}

func generateUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}
