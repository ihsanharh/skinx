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
		manifestPath := filepath.Join(packDir, "manifest.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}

		name := entry.Name()
		data, err := os.ReadFile(manifestPath)
		if err == nil {
			var manifest struct {
				Header struct {
					Name string `json:"name"`
				} `json:"header"`
			}
			if json.Unmarshal(data, &manifest) == nil && manifest.Header.Name != "" {
				name = manifest.Header.Name
			}
		}
		packs = append(packs, Pack{Dir: packDir, Name: name})
	}

	sort.Slice(packs, func(i, j int) bool {
		return strings.ToLower(packs[i].Name) < strings.ToLower(packs[j].Name)
	})
	return packs, nil
}
