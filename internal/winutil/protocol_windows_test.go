//go:build windows

package winutil

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// useTestProtocolKey points registration at a throwaway HKCU key so the test
// never touches the real tcno: handler.
func useTestProtocolKey(t *testing.T) string {
	t.Helper()
	parent := fmt.Sprintf(`Software\TcNoAccSwitcherTest\%d`, os.Getpid())
	path := parent + `\tcno`
	prevRoot, prevPath := protocolRootKey, protocolClassPath
	protocolRootKey, protocolClassPath = registry.CURRENT_USER, path
	t.Cleanup(func() {
		_ = UnregisterProtocol()
		_ = registry.DeleteKey(registry.CURRENT_USER, parent)
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\TcNoAccSwitcherTest`)
		protocolRootKey, protocolClassPath = prevRoot, prevPath
	})
	return path
}

func TestProtocolPathIsPerUserByDefault(t *testing.T) {
	if protocolRootKey != registry.CURRENT_USER || protocolClassPath != `Software\Classes\tcno` {
		t.Fatalf("protocol registers at root=%v path=%q, want HKCU\\Software\\Classes\\tcno", protocolRootKey, protocolClassPath)
	}
}

func TestRegisterAndUnregisterProtocol(t *testing.T) {
	path := useTestProtocolKey(t)
	exe := `C:\Program Files\TcNo Account Switcher\TcNo-Acc-Switcher.exe`

	if err := RegisterProtocol(exe); err != nil {
		t.Fatalf("RegisterProtocol: %v", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := k.GetStringValue("URL Protocol"); err != nil {
		t.Fatalf("URL Protocol value missing: %v", err)
	}
	k.Close()

	cmdKey, err := registry.OpenKey(registry.CURRENT_USER, path+`\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := cmdKey.GetStringValue("")
	cmdKey.Close()
	if want := `"` + exe + `" "%1"`; err != nil || got != want {
		t.Fatalf("command = %q (%v), want %q", got, err, want)
	}

	if err := UnregisterProtocol(); err != nil {
		t.Fatalf("UnregisterProtocol: %v", err)
	}
	if _, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE); err != registry.ErrNotExist {
		t.Fatalf("key still present after unregister: %v", err)
	}
	if err := UnregisterProtocol(); err != nil {
		t.Fatalf("second UnregisterProtocol should be a no-op, got %v", err)
	}
}

func TestRegisterProtocolRejectsEmptyPath(t *testing.T) {
	useTestProtocolKey(t)
	if err := RegisterProtocol("   "); err == nil {
		t.Fatal("empty exe path should fail")
	}
}
