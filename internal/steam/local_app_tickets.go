package steam

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"TcNo-Acc-Switcher/internal/vdfsafe"
)

// localAppTicketOwners returns positive account -> app entitlement evidence
// already cached by the Steam client.
//
// Steam keeps per-account app tickets in userdata/<account>/config/localconfig.vdf.
// A ticket is useful here because it ties an app to one concrete Steam account
// without requiring the switcher to hold a web/API session for that account.
// The cache is intentionally only additive: absence of a ticket does not mean an
// account does not own the app, so the authenticated owned-games sweep remains
// the source for complete libraries when Steam Guard/Login Only is configured.
func localAppTicketOwners() map[string][]string {
	root, err := installRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return nil
	}
	root = accountsRoot(root)

	owners := make(map[string][]string)
	for _, account := range knownAccountsForRoot(root) {
		steamID64 := strings.TrimSpace(account.SteamID64)
		if steamID64 == "" {
			continue
		}
		formats, err := FormatsFromID64(steamID64)
		if err != nil || strings.TrimSpace(formats.ID32) == "" {
			continue
		}
		path := filepath.Join(root, "userdata", formats.ID32, "config", "localconfig.vdf")
		appIDs, err := readLocalAppTicketIDs(path)
		if err != nil {
			if !os.IsNotExist(err) {
				steamLog().Debug("steam app tickets unavailable", "steamId", tailSteamID(steamID64), "error", err)
			}
			continue
		}
		for _, appID := range appIDs {
			owners[appID] = append(owners[appID], steamID64)
		}
	}
	return owners
}

func readLocalAppTicketIDs(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseLocalAppTicketIDs(raw)
}

func parseLocalAppTicketIDs(raw []byte) ([]string, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	kv, err := vdfsafe.ReadBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("parse localconfig.vdf: %w", err)
	}
	root := kv
	if root.Key == "" && len(root.Children) > 0 {
		root = root.Children[0]
	}
	tickets := findChildDeep(&root, "apptickets")
	if tickets == nil {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(tickets.Children))
	out := make([]string, 0, len(tickets.Children))
	for _, child := range tickets.Children {
		appID, ok := normalizeAppID(child.Key)
		if !ok || strings.TrimSpace(child.Value) == "" {
			continue
		}
		if _, exists := seen[appID]; exists {
			continue
		}
		seen[appID] = struct{}{}
		out = append(out, appID)
	}
	sort.Strings(out)
	return out, nil
}
