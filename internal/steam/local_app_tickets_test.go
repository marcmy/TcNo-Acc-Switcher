package steam

import (
	"reflect"
	"testing"
)

func TestParseLocalAppTicketIDsReadsOnlyAppTickets(t *testing.T) {
	raw := []byte(
		"\"UserLocalConfigStore\"\n{\n" +
			"\t\"Software\"\n\t{\n\t\t\"Valve\"\n\t\t{\n\t\t\t\"Steam\"\n\t\t\t{\n" +
			"\t\t\t\t\"apps\"\n\t\t\t\t{\n\t\t\t\t\t\"999\" { \"LastPlayed\" \"1\" }\n\t\t\t\t}\n" +
			"\t\t\t\t\"apptickets\"\n\t\t\t\t{\n" +
			"\t\t\t\t\t\"1172470\" \"deadbeef\"\n" +
			"\t\t\t\t\t\"2310\" \"cafebabe\"\n" +
			"\t\t\t\t\t\"0002310\" \"must-not-count\"\n" +
			"\t\t\t\t\t\"282440\" \"\"\n" +
			"\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}\n",
	)

	got, err := parseLocalAppTicketIDs(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1172470", "2310"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("app tickets = %v, want %v", got, want)
	}
}

func TestParseLocalAppTicketIDsWithoutTicketSectionIsEmpty(t *testing.T) {
	got, err := parseLocalAppTicketIDs([]byte("\"UserLocalConfigStore\" { \"apps\" { \"730\" { \"LastPlayed\" \"1\" } } }"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("app tickets = %v, want none", got)
	}
}

func TestParseLocalAppTicketIDsRejectsMalformedVDF(t *testing.T) {
	if _, err := parseLocalAppTicketIDs([]byte("\"UserLocalConfigStore\" { \"apptickets\"")); err == nil {
		t.Fatal("malformed VDF should fail")
	}
}
