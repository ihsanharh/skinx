package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ihsanharh/skinx/internal/bedrock"
	"github.com/ihsanharh/skinx/internal/cache"
	"github.com/ihsanharh/skinx/internal/pack"
)

const version = "2.0.2"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
)

var stdin = bufio.NewReader(os.Stdin)

var alternateScreen bool

func enterAlternateScreen() {
	// Disabled to allow terminal scrollback
	if !alternateScreen {
		// fmt.Print("\033[?1049h")
		alternateScreen = true
	}
}

func leaveAlternateScreen() {
	if alternateScreen {
		// fmt.Print("\033[?1049l")
		alternateScreen = false
	}
}

func clearScreen() {
	enterAlternateScreen()
	// Clear screen and scrollback using standard Linux clear sequence
	fmt.Print("\033[H\033[2J\033[3J")
}

func getInput(prompt, defaultVal string) (string, bool) {
	for {
		if defaultVal != "" {
			fmt.Printf("%s%s [%s]%s: ", colorYellow, prompt, defaultVal, colorReset)
		} else {
			fmt.Printf("%s%s%s: ", colorYellow, prompt, colorReset)
		}
		line, err := stdin.ReadString('\n')
		if err != nil {
			return "", true
		}
		line = strings.TrimSpace(line)
		if strings.ToLower(line) == "b" || strings.ToLower(line) == "back" || strings.ToLower(line) == "menu" {
			return "", true
		}
		if line != "" {
			return line, false
		}
		if defaultVal != "" {
			return defaultVal, false
		}
		fail("Input required")
	}
}

func confirmInput(prompt string) (bool, bool) {
	fmt.Printf("%s (y/N): ", prompt)
	line, err := stdin.ReadString('\n')
	if err != nil {
		return false, true
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "y" || line == "yes" {
		return true, false
	}
	if line == "n" || line == "no" || line == "" {
		return false, false
	}
	if line == "b" || line == "back" || line == "menu" {
		return false, true
	}
	return false, false
}

func pickPack(packs []cache.Pack) (int, string, string, bool) {
	fmt.Println()
	for i, p := range packs {
		fmt.Printf("  [%d] %s\n", i, p.Name)
	}
	idxStr, back := getInput("Select pack (or 'b' to go back)", "")
	if back {
		return -1, "", "", true
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(packs) {
		fail("Invalid index")
		return -1, "", "", false
	}
	return idx, packs[idx].Dir, packs[idx].Name, false
}

func askOutput(defaultName, defaultDir string) (string, string, bool) {
	if defaultDir == "" {
		defaultDir, _ = os.Getwd()
	}
	name, back := getInput("Output name", defaultName)
	if back {
		return "", "", true
	}
	outDir, back := getInput("Output directory", defaultDir)
	if back {
		return "", "", true
	}
	return name, outDir, false
}

func success(msg string, args ...any) {
	fmt.Printf("%s[+] %s%s\n", colorGreen, fmt.Sprintf(msg, args...), colorReset)
}

func fail(msg string, args ...any) {
	fmt.Printf("%s[-] %s%s\n", colorRed, fmt.Sprintf(msg, args...), colorReset)
}

func waitForEnter() {
	fmt.Printf("%s\nPress Enter to return to main menu...%s", colorYellow, colorReset)
	stdin.ReadString('\n')
}

func cmdImport(minecraftDir string) {
	for {
		clearScreen()
		fmt.Printf("%s=== Replace Skins ===%s\n", colorCyan, colorReset)
		fmt.Println("Replace skins in an owned pack with your custom skins.")
		fmt.Println("You must own the target pack first.")
		fmt.Println("[B]ack to menu")
		fmt.Println()

		source, back := getInput("Skin pack folder path", "")
		if back {
			return
		}

		sourcePath, err := filepath.Abs(source)
		if err != nil {
			fail("Invalid path: %s", source)
			waitForEnter()
			continue
		}

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			fail("Path not found: %s", source)
			waitForEnter()
			continue
		}

		if !pack.ValidateSkinpack(sourcePath) {
			fail("Not a valid skin pack (needs manifest.json, skins.json, and .png files)")
			waitForEnter()
			continue
		}

		cacheDir, err := cache.FindPremiumCache(minecraftDir)
		if err != nil {
			fail("%s", err)
			waitForEnter()
			continue
		}

		packs, err := cache.ListPacks(cacheDir)
		if err != nil || len(packs) == 0 {
			fail("No skin packs found in cache")
			waitForEnter()
			return
		}

		fmt.Println("\nAvailable premium packs:")
		idx, packDir, _, back := pickPack(packs)
		if back {
			continue
		}
		if idx < 0 {
			continue
		}

		if err := pack.ImportPack(sourcePath, packDir); err != nil {
			fail("Import failed: %s", err)
		} else {
			success("Imported skins into %s", filepath.Base(packDir))
		}
		waitForEnter()
		return
	}
}

func cmdRemove(minecraftDir string) {
	clearScreen()
	fmt.Printf("%s=== Remove Skin Pack ===%s\n", colorCyan, colorReset)
	fmt.Println("Delete a pack from premium_cache.")
	fmt.Println("You must own the pack first.")
	fmt.Println("[B]ack to menu")
	fmt.Println()

	cacheDir, err := cache.FindPremiumCache(minecraftDir)
	if err != nil {
		fail("%s", err)
		waitForEnter()
		return
	}

	packs, err := cache.ListPacks(cacheDir)
	if err != nil || len(packs) == 0 {
		fail("No skin packs found")
		waitForEnter()
		return
	}

	idx, packDir, packName, back := pickPack(packs)
	if back || idx < 0 {
		return
	}

	confirm, back := confirmInput(fmt.Sprintf("Remove '%s'?", packName))
	if back {
		return
	}
	if confirm {
		os.RemoveAll(packDir)
		success("Removed: %s", packName)
	} else {
		fail("Cancelled")
	}
	waitForEnter()
}

func cmdListPacks(minecraftDir string) {
	clearScreen()
	fmt.Printf("%s=== List Skin Packs ===%s\n", colorCyan, colorReset)
	fmt.Println("Packs in premium_cache (must own them).")
	fmt.Println("[B]ack to menu")
	fmt.Println()

	cacheDir, err := cache.FindPremiumCache(minecraftDir)
	if err != nil {
		fail("%s", err)
		waitForEnter()
		return
	}

	packs, err := cache.ListPacks(cacheDir)
	if err != nil || len(packs) == 0 {
		fail("No skin packs found in premium_cache")
		waitForEnter()
		return
	}

	fmt.Printf("Found %d pack(s) in:\n  %s\n\n", len(packs), cacheDir)

	type row struct {
		display string
		uuid    string
		dir     string
	}
	var rows []row

	for _, p := range packs {
		manifest, err := pack.ReadManifest(filepath.Join(p.Dir, "manifest.json"))
		uuid := "unknown"
		display := p.Name
		if err == nil {
			uuid = manifest.Header.UUID
			verStr := ""
			for j, v := range manifest.Header.Version {
				if j > 0 {
					verStr += "."
				}
				verStr += strconv.FormatFloat(v, 'f', 0, 64)
			}
			if verStr != "" {
				display = fmt.Sprintf("%s v%s", p.Name, verStr)
			}
		}
		rows = append(rows, row{display: display, uuid: uuid, dir: filepath.Base(p.Dir)})
	}

	maxName, maxUUID, maxDir := 4, 4, 3
	for _, r := range rows {
		if len(r.display) > maxName {
			maxName = len(r.display)
		}
		if len(r.uuid) > maxUUID {
			maxUUID = len(r.uuid)
		}
		if len(r.dir) > maxDir {
			maxDir = len(r.dir)
		}
	}

	prefixWidth := len(fmt.Sprintf("[%d]", len(rows)-1)) + 2

	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s", prefixWidth, "#", maxName, "Name", maxUUID, "UUID", maxDir, "Dir")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))

	for i, r := range rows {
		fmt.Printf("%-*s  %-*s  %-*s  %-*s\n", prefixWidth, fmt.Sprintf("[%d]", i), maxName, r.display, maxUUID, r.uuid, maxDir, r.dir)
	}

	waitForEnter()
}

func cmdExtract(minecraftDir string) {
	clearScreen()
	fmt.Printf("%s=== Extract Skin Pack ===%s\n", colorCyan, colorReset)
	fmt.Println("Decrypt an owned marketplace pack to a folder.")
	fmt.Println("You must own the pack first.")
	fmt.Println("[B]ack to menu")
	fmt.Println()

	cacheDir, err := cache.FindPremiumCache(minecraftDir)
	if err != nil {
		fail("%s", err)
		waitForEnter()
		return
	}

	packs, err := cache.ListPacks(cacheDir)
	if err != nil || len(packs) == 0 {
		fail("No skin packs found")
		waitForEnter()
		return
	}

	idx, packDir, packName, back := pickPack(packs)
	if back || idx < 0 {
		return
	}

	name, outDir, back := askOutput(packName, "")
	if back {
		return
	}

	if err := pack.DecryptPack(packDir, filepath.Join(outDir, name)); err != nil {
		fail("Extract failed: %s", err)
	} else {
		success("Extracted to %s", filepath.Join(outDir, name))
	}
	waitForEnter()
}

func cmdConvert() {
	for {
		clearScreen()
		fmt.Printf("%s=== Convert Skin Pack ===%s\n", colorCyan, colorReset)
		fmt.Println("Encrypt, decrypt, or convert any skin pack.")
		fmt.Println("[B]ack to menu")
		fmt.Println()

		source, back := getInput("Skin pack path (folder or .mcpack)", "")
		if back {
			return
		}

		sourcePath, err := filepath.Abs(source)
		if err != nil {
			fail("Invalid path: %s", source)
			waitForEnter()
			continue
		}

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			fail("Path not found: %s", source)
			waitForEnter()
			continue
		}

		pathType := pack.DetectPathType(sourcePath)

		switch pathType {
		case "mcpack":
			fmt.Println("  Detected: .mcpack file")
			name, outDir, back := askOutput(filepath.Base(sourcePath), "")
			if back {
				return
			}
			if err := unpackMcpack(sourcePath, filepath.Join(outDir, name)); err != nil {
				fail("Extract failed: %s", err)
			} else {
				success("Extracted to %s", filepath.Join(outDir, name))
			}

		case "encrypted_folder":
			fmt.Println("  Detected: encrypted folder")
			fmt.Println("\nAction:")
			fmt.Println("  [1] Decrypt")
			fmt.Println("  [2] Convert to .mcpack")
			action, _ := getInput("Select", "1")
			if action == "BACK" {
				return
			}

			name, outDir, back := askOutput(filepath.Base(sourcePath), "")
			if back {
				return
			}

			if action == "1" {
				if err := pack.DecryptPack(sourcePath, filepath.Join(outDir, name)); err != nil {
					fail("Decrypt failed: %s", err)
				} else {
					success("Decrypted to %s", filepath.Join(outDir, name))
				}
			} else {
				if err := pack.PackToMcpackRaw(sourcePath, outDir, name); err != nil {
					fail("Convert failed: %s", err)
				} else {
					success("Converted to %s.mcpack", filepath.Join(outDir, name))
				}
			}

		case "unencrypted_folder":
			fmt.Println("  Detected: unencrypted folder")
			fmt.Println("\nAction:")
			fmt.Println("  [1] Encrypt")
			fmt.Println("  [2] Convert to .mcpack")
			action, _ := getInput("Select", "1")
			if action == "BACK" {
				return
			}

			name, outDir, back := askOutput(filepath.Base(sourcePath), "")
			if back {
				return
			}

			if action == "1" {
				if err := pack.EncryptToFolder(sourcePath, filepath.Join(outDir, name)); err != nil {
					fail("Encrypt failed: %s", err)
				} else {
					success("Encrypted to %s", filepath.Join(outDir, name))
				}
			} else {
				if err := pack.PackToMcpackRaw(sourcePath, outDir, name); err != nil {
					fail("Convert failed: %s", err)
				} else {
					success("Converted to %s.mcpack", filepath.Join(outDir, name))
				}
			}

		default:
			fail("Not a skin pack (no manifest.json)")
			waitForEnter()
			continue
		}

		waitForEnter()
		return
	}
}

func cmdInfo() {
	clearScreen()
	fmt.Println()
	fmt.Printf("%s%s%s\n", colorDim, strings.Repeat("=", 50), colorReset)
	fmt.Printf("%sskinx%s - Minecraft Bedrock Skin Pack Tool\n", colorCyan, colorReset)
	fmt.Printf("Version: %s%s%s\n", colorGreen, version, colorReset)
	fmt.Println("Author: ihsanharh.com")
	fmt.Println("Encryption format reverse-engineered by")
	fmt.Println("  BedrockReverse/McTools")
	fmt.Printf("%s%s%s\n", colorDim, strings.Repeat("=", 50), colorReset)
	fmt.Println("\nManage Minecraft Bedrock GDK skin packs:")
	fmt.Println()
	fmt.Println("  Replace skins in owned marketplace packs")
	fmt.Println("    with custom skins (supports custom geometry)")
	fmt.Println()
	fmt.Println("  Extract owned packs to edit their contents")
	fmt.Println()
	fmt.Println("  Encrypt, decrypt, or convert packs")
	fmt.Println("    to .mcpack or folder format")
	fmt.Println()
	fmt.Println("  Requires: Minecraft Bedrock GDK version")
	fmt.Println("  Cache: premium_cache/skin_packs")
	waitForEnter()
}

func unpackMcpack(mcpackPath, destDir string) error {
	r, err := zip.OpenReader(mcpackPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var minecraftDir string
	flag.StringVar(&minecraftDir, "m", "", "Path to Minecraft Bedrock directory")
	flag.StringVar(&minecraftDir, "minecraft-dir", "", "Path to Minecraft Bedrock directory")
	flag.Parse()

	defer leaveAlternateScreen()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		leaveAlternateScreen()
		os.Exit(0)
	}()

	clearScreen()

	cacheDir, err := cache.FindPremiumCache(minecraftDir)
	if err != nil {
		fail("%s", err)
		fmt.Println("  Make sure Minecraft Bedrock is installed")
		return
	}
	success("Found premium_cache: %s", cacheDir)
	time.Sleep(500 * time.Millisecond)

	for {
		clearScreen()
		fmt.Printf("%s%s%s\n", colorDim, strings.Repeat("=", 50), colorReset)
		fmt.Printf("      %sskinx%s - Minecraft Bedrock Skin Pack Tool\n", colorCyan, colorReset)
		fmt.Printf("%s%s%s\n\n", colorDim, strings.Repeat("=", 50), colorReset)
		fmt.Printf("%sMarketplace Skin Packs%s\n", colorCyan, colorReset)
		fmt.Printf("  %s[1]%s Replace skin pack    Replace skins in an owned pack with your custom skins\n", colorGreen, colorReset)
		fmt.Printf("  %s[2]%s List skin packs      Show all marketplace packs you own\n", colorGreen, colorReset)
		fmt.Printf("  %s[3]%s Remove skin pack     Delete a pack from your collection\n", colorGreen, colorReset)
		fmt.Printf("  %s[4]%s Extract skin pack    Decrypt and extract an owned pack to a folder\n", colorGreen, colorReset)
		fmt.Printf("\n%sAny Skin Pack%s\n", colorCyan, colorReset)
		fmt.Printf("  %s[5]%s Convert skin pack    Encrypt, decrypt, or convert between .mcpack and folder\n", colorGreen, colorReset)
		fmt.Printf("\n%sOther%s\n", colorCyan, colorReset)
		fmt.Printf("  %s[6]%s Info\n", colorGreen, colorReset)
		fmt.Printf("  %s[7]%s Exit\n", colorGreen, colorReset)

		choice, _ := getInput("Select option", "1")

		switch choice {
		case "1":
			cmdImport(minecraftDir)
		case "2":
			cmdListPacks(minecraftDir)
		case "3":
			cmdRemove(minecraftDir)
		case "4":
			cmdExtract(minecraftDir)
		case "5":
			cmdConvert()
		case "6":
			cmdInfo()
		case "7", "exit", "quit", "q":
			if runtime.GOOS == "windows" {
				bedrock.FreeConsole()
			}
			return
		default:
			fail("Invalid option")
		}
	}
}
