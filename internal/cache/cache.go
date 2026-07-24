package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Pack struct {
	Dir  string
	Name string
}

func FindPremiumCache(minecraftDir string) (string, error) {
	if minecraftDir != "" {
		cache := filepath.Join(minecraftDir, "premium_cache", "skin_packs")
		if info, err := os.Stat(cache); err == nil && info.IsDir() {
			return cache, nil
		}
		return "", fmt.Errorf("not found: %s", cache)
	}

	appdata := os.Getenv("APPDATA")
	if appdata != "" {
		cache := filepath.Join(appdata, "Minecraft Bedrock", "premium_cache", "skin_packs")
		if info, err := os.Stat(cache); err == nil && info.IsDir() {
			return cache, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find premium_cache directory")
	}
	wineBase := filepath.Join(home, ".local", "share", "bedrock-on-linux", "compatdata", "pfx", "drive_c", "users")
	matches, _ := filepath.Glob(filepath.Join(wineBase, "*"))
	for _, userDir := range matches {
		cache := filepath.Join(userDir, "AppData", "Roaming", "Minecraft Bedrock", "premium_cache", "skin_packs")
		if info, err := os.Stat(cache); err == nil && info.IsDir() {
			return cache, nil
		}
	}

	return "", fmt.Errorf("could not find premium_cache directory")
}

func getPackName(packDir string) string {
	fallback := filepath.Base(packDir)

	textsDir := filepath.Join(packDir, "texts")
	textEntries, err := os.ReadDir(textsDir)
	if err == nil {
		for _, te := range textEntries {
			if te.IsDir() || !strings.HasSuffix(te.Name(), ".lang") {
				continue
			}
			langData, err := os.ReadFile(filepath.Join(textsDir, te.Name()))
			if err != nil {
				continue
			}
			content := strings.TrimPrefix(string(langData), "\xEF\xBB\xBF")
			for _, line := range strings.Split(content, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "skinpack.") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}

	manifestPath := filepath.Join(packDir, "manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest struct {
			Header struct {
				Name string `json:"name"`
			} `json:"header"`
		}
		if json.Unmarshal(data, &manifest) == nil && manifest.Header.Name != "" {
			return manifest.Header.Name
		}
	}

	return fallback
}

func ListPacks(cacheDir string) ([]Pack, error) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	var packs []Pack
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".backup") {
			continue
		}
		packDir := filepath.Join(cacheDir, entry.Name())
		if _, err := os.Stat(filepath.Join(packDir, "manifest.json")); os.IsNotExist(err) {
			continue
		}

		packs = append(packs, Pack{Dir: packDir, Name: getPackName(packDir)})
	}

	sort.Slice(packs, func(i, j int) bool {
		return strings.ToLower(packs[i].Name) < strings.ToLower(packs[j].Name)
	})
	return packs, nil
}
