package shortcuts

import "testing"

func TestSteamGameAccountShortcutArgs(t *testing.T) {
	steamID64, appID, err := steamGameAccountShortcutArgs("76561198000000000", "1172470")
	if err != nil {
		t.Fatal(err)
	}
	if steamID64 != "76561198000000000" || appID != "1172470" {
		t.Fatalf("got steamID64=%q appID=%q", steamID64, appID)
	}
}

func TestSteamGameAccountShortcutArgsRejectsInjectedOrNonCanonicalValues(t *testing.T) {
	tests := []struct {
		steamID64 string
		appID     string
	}{
		{"76561198000000000 --run-appid=730", "1172470"},
		{"76561198000000000", "1172470 --foo"},
		{"076561198000000000", "1172470"},
		{"76561198000000000", "01172470"},
		{"76561198000000000", "4294967296"},
	}
	for _, tc := range tests {
		if _, _, err := steamGameAccountShortcutArgs(tc.steamID64, tc.appID); err == nil {
			t.Fatalf("accepted steamID64=%q appID=%q", tc.steamID64, tc.appID)
		}
	}
}
