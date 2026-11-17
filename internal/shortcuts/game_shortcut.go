package shortcuts

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"TcNo-Acc-Switcher/internal/cli"
	"TcNo-Acc-Switcher/internal/paths"
	"TcNo-Acc-Switcher/internal/profileimage"
	"TcNo-Acc-Switcher/internal/winutil"
)

var steamRungameIDRE = regexp.MustCompile(`(?i)steam://rungameid/(\d+)`)

func normalizeSteamShortcutDecimal(raw string, bitSize int) (string, bool) {
	id := strings.TrimSpace(raw)
	if id == "" || (len(id) > 1 && id[0] == '0') {
		return "", false
	}
	n, err := strconv.ParseUint(id, 10, bitSize)
	if err != nil || n == 0 {
		return "", false
	}
	return id, true
}

func steamGameAccountShortcutArgs(uniqueID, appID string) (string, string, error) {
	steamID64, ok := normalizeSteamShortcutDecimal(uniqueID, 64)
	if !ok {
		return "", "", fmt.Errorf("invalid Steam account id")
	}
	appID, ok = normalizeSteamShortcutDecimal(appID, 32)
	if !ok {
		return "", "", fmt.Errorf("invalid Steam app id")
	}
	return steamID64, appID, nil
}

// CreateGameAccountShortcut: steam://rungameid .url tiles use --run-appid; otherwise --run-shortcut.
func CreateGameAccountShortcut(platformKey, uniqueID, accountDisplayName, accountLogin, gameFileName string) (ShortcutResult, error) {
	platformKey = strings.TrimSpace(platformKey)
	uniqueID = strings.TrimSpace(uniqueID)
	gameFileName = filepath.Base(strings.TrimSpace(gameFileName))
	if platformKey == "" || uniqueID == "" {
		return ShortcutResult{}, fmt.Errorf("missing platform or account id")
	}
	if gameFileName == "" || !isShortcutFile(gameFileName) {
		return ShortcutResult{}, fmt.Errorf("invalid game shortcut name")
	}

	root, err := paths.LoginCacheDir(platformKey)
	if err != nil {
		return ShortcutResult{}, err
	}
	cacheFull := filepath.Join(root, "Shortcuts", gameFileName)
	if st, err := os.Stat(cacheFull); err != nil || st.IsDir() {
		return ShortcutResult{}, fmt.Errorf("game shortcut not found in cache")
	}

	self, err := os.Executable()
	if err != nil {
		return ShortcutResult{}, err
	}
	self = filepath.Clean(self)

	desktop, err := winutil.DesktopWriteDir()
	if err != nil {
		return ShortcutResult{}, err
	}

	title := gameAccountShortcutDesktopBaseName(platformKey, uniqueID, accountDisplayName, accountLogin, gameFileName)
	outPath := filepath.Join(desktop, title+".lnk")

	var argv string
	if strings.EqualFold(platformKey, "Steam") {
		argv = "+s:" + uniqueID
		if appID, ok := steamRungameIDFromURLFile(cacheFull); ok {
			argv += " --run-appid=" + appID
		} else {
			argv += " --run-shortcut=" + url.QueryEscape(gameFileName)
		}
	} else {
		idx, err := cli.LoadPlatformIndex()
		if err != nil {
			return ShortcutResult{}, err
		}
		short := cli.ShortTokenForPlatform(idx, platformKey)
		if short == "" {
			return ShortcutResult{}, fmt.Errorf("platform %q has no CLI short token", platformKey)
		}
		argv = "+" + short + ":" + uniqueID + " --run-shortcut=" + url.QueryEscape(gameFileName)
	}

	icon := buildGameAccountShortcutIcon(platformKey, uniqueID, gameFileName)

	return writeGameAccountShortcut(outPath, self, platformKey, uniqueID, title, argv, icon, gameFileName)
}

// CreateSteamGameAccountShortcut creates a Desktop shortcut directly from a
// Games-view AppID and account. It does not depend on a cached .url/.lnk entry:
// launching the shortcut switches to uniqueID and then opens steam://rungameid.
func CreateSteamGameAccountShortcut(uniqueID, accountDisplayName, accountLogin, appID, gameName string) (ShortcutResult, error) {
	uniqueID, appID, err := steamGameAccountShortcutArgs(uniqueID, appID)
	if err != nil {
		return ShortcutResult{}, err
	}
	gameName = strings.TrimSpace(gameName)
	if gameName == "" {
		gameName = "Steam " + appID
	}

	self, err := os.Executable()
	if err != nil {
		return ShortcutResult{}, err
	}
	self = filepath.Clean(self)
	desktop, err := winutil.DesktopWriteDir()
	if err != nil {
		return ShortcutResult{}, err
	}

	title := gameAccountShortcutDesktopBaseName("Steam", uniqueID, accountDisplayName, accountLogin, gameName)
	outPath := filepath.Join(desktop, title+".lnk")
	argv := "+s:" + uniqueID + " --run-appid=" + appID
	icon := buildSteamAppAccountShortcutIcon(uniqueID, appID)
	return writeGameAccountShortcut(outPath, self, "Steam", uniqueID, title, argv, icon, "app-"+appID)
}

func writeGameAccountShortcut(outPath, self, platformKey, uniqueID, title, argv, icon, identity string) (ShortcutResult, error) {
	workDir := filepath.Dir(self)
	desc := fmt.Sprintf("TcNo Account Switcher - %s - %s", platformKey, title)
	appUserModelID := winutil.ShortcutAppUserModelID(platformKey, uniqueID, identity)
	if shortcutAlreadyMatches(outPath, self, argv, icon) {
		return ShortcutResult{Path: outPath, AlreadyExisted: true}, nil
	}
	if err := winutil.WriteShortcutLnk(outPath, self, argv, workDir, desc, icon, appUserModelID); err != nil {
		return ShortcutResult{}, err
	}
	return ShortcutResult{Path: outPath}, nil
}

func steamRungameIDFromURLFile(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	m := steamRungameIDRE.FindSubmatch(b)
	if len(m) < 2 {
		return "", false
	}
	return string(m[1]), true
}

func gameAccountShortcutDesktopBaseName(platformKey, uniqueID, displayName, accountLogin, gameFileName string) string {
	gameStem := removeShortcutExt(gameFileName)
	sanGame := sanitizeShortcutFileName(gameStem)
	if isDegenerateShortcutBasename(sanGame) {
		sanGame = "Game"
	}

	sanAcc := resolvedAccountShortcutStem(platformKey, uniqueID, displayName, accountLogin)
	return sanitizeShortcutFileName(fmt.Sprintf("%s (%s)", sanGame, sanAcc))
}

func buildGameAccountShortcutIcon(platformKey, uniqueID, gameFileName string) string {
	p, ok := cachedShortcutAvatar(platformKey, uniqueID)
	if !ok || p == "" {
		return buildAccountShortcutIcon(platformKey, uniqueID)
	}
	root, err := paths.DataRoot()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(root, "IconCache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return ""
	}

	gameIconPath, err := iconDiskPath(platformKey, gameFileName)
	if err == nil && gameIconPath != "" {
		if _, err := os.Stat(gameIconPath); err == nil {
			icoName := fmt.Sprintf("%s_game_%s_%s.ico",
				profileimage.PlatformFolder(platformKey),
				sanitizeShortcutFileName(removeShortcutExt(gameFileName)),
				sanitizeShortcutFileName(uniqueID))
			icoPath := filepath.Join(cacheDir, icoName)
			if err := winutil.BuildCombinedGameIcon(gameIconPath, p, icoPath); err == nil {
				return icoPath + ",0"
			}
		}
	}

	return buildAccountShortcutIcon(platformKey, uniqueID)
}

func buildSteamAppAccountShortcutIcon(uniqueID, appID string) string {
	p, ok := cachedShortcutAvatar("Steam", uniqueID)
	if ok && p != "" {
		root, rootErr := paths.DataRoot()
		www, wwwErr := paths.WwwrootDir()
		if rootErr == nil && wwwErr == nil {
			gameIconPath := filepath.Join(www, "img", "gameicons", "square", appID+".jpg")
			if st, statErr := os.Stat(gameIconPath); statErr == nil && !st.IsDir() {
				cacheDir := filepath.Join(root, "IconCache")
				if mkErr := os.MkdirAll(cacheDir, 0o755); mkErr == nil {
					icoName := fmt.Sprintf("steam_game_%s_%s.ico", appID, sanitizeShortcutFileName(uniqueID))
					icoPath := filepath.Join(cacheDir, icoName)
					if buildErr := winutil.BuildCombinedGameIcon(gameIconPath, p, icoPath); buildErr == nil {
						return icoPath + ",0"
					}
				}
			}
		}
	}
	return buildGameAccountShortcutIcon("Steam", uniqueID, appID+".url")
}
