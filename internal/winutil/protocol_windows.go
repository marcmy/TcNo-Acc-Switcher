//go:build windows

package winutil

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const protocolScheme = "tcno"

// protocolClassesRoot is where the per-user URL scheme lives. Writing through
// HKEY_CLASSES_ROOT is not safe: when the installer has already created
// HKLM\Software\Classes\tcno, Windows routes HKCR writes to HKLM, which fails
// with ERROR_ACCESS_DENIED for a non-elevated process. HKCU\Software\Classes
// needs no elevation and takes precedence over HKLM in the merged HKCR view.
var (
	protocolRootKey   = registry.CURRENT_USER
	protocolClassPath = `Software\Classes\` + protocolScheme
)

// IsProtocolRegistered reports whether HKCR\tcno has URL Protocol set (custom URL scheme).
func IsProtocolRegistered() bool {
	k, err := registry.OpenKey(registry.CLASSES_ROOT, protocolScheme, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("URL Protocol")
	return err == nil && strings.TrimSpace(v) != ""
}

// RegisterProtocol writes HKCU\Software\Classes\tcno with shell\open\command
// pointing at exe with "%1".
func RegisterProtocol(exePath string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}

	k, _, err := registry.CreateKey(protocolRootKey, protocolClassPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if err := k.SetStringValue("", "URL:TcNo Account Switcher"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	cmdKey, _, err := registry.CreateKey(protocolRootKey, protocolClassPath+`\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cmdKey.Close()

	cmd := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	return cmdKey.SetStringValue("", cmd)
}

// UnregisterProtocol removes the per-user HKCU\Software\Classes\tcno key
// (children first). A machine-wide registration made by the installer is left
// alone: removing it needs elevation and belongs to the uninstaller.
func UnregisterProtocol() error {
	_ = registry.DeleteKey(protocolRootKey, protocolClassPath+`\shell\open\command`)
	_ = registry.DeleteKey(protocolRootKey, protocolClassPath+`\shell\open`)
	_ = registry.DeleteKey(protocolRootKey, protocolClassPath+`\shell`)
	err := registry.DeleteKey(protocolRootKey, protocolClassPath)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
