package service

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/util/srs"

	"github.com/op/go-logging"
)

func TestNormalizeDomain(t *testing.T) {
	for in, want := range map[string]string{
		"YouTube.com":      "youtube.com",
		" *.ytimg.com ":    "ytimg.com",
		".googlevideo.com": "googlevideo.com",
		"example.org.":     "example.org",
		"ru":               "ru",
		"сайт.рф":          "xn--80aswg.xn--p1ai",
		"_dmarc.example":   "_dmarc.example",
		"a-b.c":            "a-b.c",
	} {
		got, ok := NormalizeDomain(in)
		if !ok || got != want {
			t.Errorf("%q: %q %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", ".", "exa mple.com", "-bad.com", "bad-.com", "a..b", "http://x.com", "x.com/path", "1.2.3.4/24"} {
		if got, ok := NormalizeDomain(in); ok {
			t.Errorf("%q passes as %q", in, got)
		}
	}
}

func TestNormalizeSubnet(t *testing.T) {
	for in, want := range map[string]string{
		"91.108.4.0/22":      "91.108.4.0/22",
		"91.108.5.7/22":      "91.108.4.0/22",
		"149.154.167.51":     "149.154.167.51/32",
		"2001:b28:f23d::/48": "2001:b28:f23d::/48",
		"2001:db8::1":        "2001:db8::1/128",
	} {
		got, ok := NormalizeSubnet(in)
		if !ok || got.String() != want {
			t.Errorf("%q: %v %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "1.2.3", "1.2.3.4/33", "::ffff:1.2.3.4", "::ffff:1.2.3.0/120", "fe80::1%eth0", "example.com"} {
		if got, ok := NormalizeSubnet(in); ok {
			t.Errorf("%q passes as %v", in, got)
		}
	}
}

func TestParseRuleList(t *testing.T) {
	got, err := ParseRuleList(RuleListDomains, "# video\r\nyoutube.com\n\n  www.YOUTUBE.com\nyoutube.com\n*.ytimg.com\n")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"youtube.com", "www.youtube.com", "ytimg.com"}; !slices.Equal(got, want) {
		t.Fatalf("%v, want %v", got, want)
	}
	_, err = ParseRuleList(RuleListSubnets, "1.2.3.0/24\n\nnot-a-subnet\n")
	var line *RuleListLineError
	if !errors.As(err, &line) || line.Line != 3 || line.Text != "not-a-subnet" {
		t.Fatalf("error %v", err)
	}
	if _, err := ParseRuleList("ports", "80"); err == nil {
		t.Fatal("an unknown kind passes")
	}
}

func TestBuildRuleListTakesTheEnabledSections(t *testing.T) {
	sections, entries, err := BuildRuleList(RuleListDomains, []model.RuleListSection{
		{Name: " YouTube ", Enabled: true, Entries: "youtube.com\nWWW.youtube.com\nytimg.com"},
		{Name: "Discord", Enabled: false, Entries: "discord.com"},
		{Name: "Google", Enabled: true, Entries: "# shared\nytimg.com\ngoogle.com"},
		{Name: "Empty", Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"youtube.com", "www.youtube.com", "ytimg.com", "google.com"}; !slices.Equal(entries, want) {
		t.Fatalf("entries %v, want %v", entries, want)
	}
	if sections[0].Name != "YouTube" || sections[0].Count != 3 || sections[0].Entries != "youtube.com\nwww.youtube.com\nytimg.com" {
		t.Fatalf("first section kept as %+v", sections[0])
	}
	// A section switched off keeps its entries, only out of the rule-set.
	if sections[1].Enabled || sections[1].Count != 1 || sections[1].Entries != "discord.com" {
		t.Fatalf("switched off section kept as %+v", sections[1])
	}

	for name, in := range map[string][]model.RuleListSection{
		"no name":   {{Name: " ", Enabled: true}},
		"same name": {{Name: "Telegram"}, {Name: "telegram"}},
		"long name": {{Name: string(make([]rune, MaxRuleListSectionName+1))}},
		"two lines": {{Name: "a\nb"}},
	} {
		if _, _, err := BuildRuleList(RuleListDomains, in); err == nil {
			t.Errorf("%s passes", name)
		}
	}

	_, _, err = BuildRuleList(RuleListSubnets, []model.RuleListSection{{Name: "Telegram", Entries: "91.108.4.0/22"}, {Name: "Bad", Entries: "1.2.3.0/24\nnot-a-subnet"}})
	var sec *RuleListSectionError
	var line *RuleListLineError
	if !errors.As(err, &sec) || sec.Section != 2 || sec.Name != "Bad" || !errors.As(err, &line) || line.Line != 2 {
		t.Fatalf("a wrong line in the second section: %v", err)
	}
	if _, _, err := BuildRuleList(RuleListDomains, make([]model.RuleListSection, MaxRuleListSections+1)); err == nil {
		t.Fatal("too many sections pass")
	}
}

func TestParseRuleListPorts(t *testing.T) {
	kept, ports, ranges, err := ParseRuleListPorts([]string{" 3478 ", "19000-20000", "19000 - 20000", "443-443", "50000-65535"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"3478", "19000-20000", "443", "50000-65535"}; !slices.Equal(kept, want) {
		t.Fatalf("kept %v, want %v", kept, want)
	}
	if !slices.Equal(ports, []uint16{3478, 443}) || len(ranges) != 2 || ranges[0] != (srs.PortRange{From: 19000, To: 20000}) || ranges[1] != (srs.PortRange{From: 50000, To: 65535}) {
		t.Fatalf("ports %v ranges %v", ports, ranges)
	}
	for _, bad := range []string{"", "0", "65536", "20000-19000", "80-", "-80", "a", "1.5", "+80", "080", "80:90"} {
		if _, _, _, err := ParseRuleListPorts([]string{bad}); err == nil {
			t.Errorf("%q passes", bad)
		}
	}
	if _, _, _, err := ParseRuleListPorts(make([]string, MaxRuleListPorts+1)); err == nil {
		t.Error("too many ports pass")
	}
}

// TestRuleListSectionsKeptToPorts checks a subnets section's protocol and
// ports as they are kept, refused, and written: the sections for every
// connection in the first rule, those kept to the same ports in one more.
func TestRuleListSectionsKeptToPorts(t *testing.T) {
	sections, _, err := BuildRuleList(RuleListSubnets, []model.RuleListSection{
		{Name: "Discord", Enabled: true, Entries: "104.29.128.0/18", Network: " UDP ", Ports: []string{"19000-20000", "50000-65535"}},
		{Name: "Telegram", Enabled: true, Entries: "91.108.4.0/22\n149.154.160.0/20"},
		{Name: "Voice", Enabled: true, Entries: "66.22.192.0/18", Network: "udp", Ports: []string{"19000-20000", "50000-65535"}},
		{Name: "Off", Enabled: false, Entries: "1.2.3.0/24", Network: "tcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s := sections[0]; s.Network != "udp" || !slices.Equal(s.Ports, []string{"19000-20000", "50000-65535"}) {
		t.Fatalf("kept as %+v", s)
	}
	if s := sections[1]; s.Network != "" || s.Ports != nil {
		t.Fatalf("a section for every connection kept as %+v", s)
	}

	for name, in := range map[string]model.RuleListSection{
		"protocol": {Name: "A", Entries: "1.2.3.0/24", Network: "icmp"},
		"port":     {Name: "A", Entries: "1.2.3.0/24", Ports: []string{"70000"}},
	} {
		if _, _, err := BuildRuleList(RuleListSubnets, []model.RuleListSection{in}); err == nil {
			t.Errorf("a wrong %s passes", name)
		}
	}
	for _, in := range []model.RuleListSection{{Name: "A", Entries: "example.com", Network: "udp"}, {Name: "A", Entries: "example.com", Ports: []string{"443"}}} {
		if _, _, err := BuildRuleList(RuleListDomains, []model.RuleListSection{in}); err == nil {
			t.Errorf("a domains section kept to %+v passes", in)
		}
	}

	data, err := CompileRuleList(RuleListSubnets, sections)
	if err != nil {
		t.Fatal(err)
	}
	z, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(z)
	if n, _ := binary.ReadUvarint(r); n != 2 {
		t.Fatalf("%d rules, want 2", n)
	}
	// The first rule is the one for every connection: its first item is the addresses (6), not a network (1).
	if kind, item := must(r.ReadByte()), must(r.ReadByte()); kind != 0 || item != 6 {
		t.Fatalf("first rule starts %d %d", kind, item)
	}

	empty, err := CompileRuleList(RuleListSubnets, []model.RuleListSection{{Name: "Off", Entries: "1.2.3.0/24", Network: "udp", Ports: []string{"443"}}})
	if err != nil {
		t.Fatal(err)
	}
	z, _ = zlib.NewReader(bytes.NewReader(empty[4:]))
	if n, _ := binary.ReadUvarint(bufio.NewReader(z)); n != 0 {
		t.Fatalf("a list with its sections off has %d rules", n)
	}
}

// TestRuleListLoneSubnetRefused checks that no rule of a subnets list is one
// subnet, its addresses together, whichever sections it is made of.
func TestRuleListLoneSubnetRefused(t *testing.T) {
	udp := func(name, entries string) model.RuleListSection {
		return model.RuleListSection{Name: name, Enabled: true, Entries: entries, Network: "udp", Ports: []string{"27015-27050"}}
	}
	every := func(name, entries string) model.RuleListSection {
		return model.RuleListSection{Name: name, Enabled: true, Entries: entries}
	}
	many := every("Telegram", "91.108.4.0/22\n149.154.160.0/20")
	for name, c := range map[string]struct {
		sections []model.RuleListSection
		lone     []string
		filtered bool
	}{
		"filtered":        {[]model.RuleListSection{many, udp("Faceit", "121.127.33.0/25")}, []string{"Faceit"}, true},
		"for every":       {[]model.RuleListSection{every("Telegram", "91.108.4.0/22"), udp("Voice", "66.22.192.0/18\n104.29.128.0/18")}, []string{"Telegram"}, false},
		"halves":          {[]model.RuleListSection{many, udp("Faceit", "10.0.0.0/25\n10.0.0.128/25")}, []string{"Faceit"}, true},
		"inside":          {[]model.RuleListSection{many, udp("A", "10.0.0.0/8"), udp("B", "10.1.0.0/16")}, []string{"A", "B"}, true},
		"other off":       {[]model.RuleListSection{many, udp("Faceit", "121.127.33.0/25"), {Name: "Off", Entries: "1.2.3.0/24", Network: "udp", Ports: []string{"27015-27050"}}}, []string{"Faceit"}, true},
		"two":             {[]model.RuleListSection{many, udp("Faceit", "121.127.33.0/25\n121.127.46.0/24")}, nil, false},
		"one each, apart": {[]model.RuleListSection{many, udp("A", "10.0.0.0/8"), udp("B", "11.0.0.0/16")}, nil, false},
		"v4 and v6":       {[]model.RuleListSection{udp("A", "10.0.0.0/8\n2001:db8::/32")}, nil, false},
		"all off":         {[]model.RuleListSection{{Name: "Off", Entries: "1.2.3.0/24"}}, nil, false},
		"none":            {nil, nil, false},
	} {
		built, _, err := BuildRuleList(RuleListSubnets, c.sections)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		_, err = CompileRuleList(RuleListSubnets, built)
		var lone *RuleListLoneSubnetError
		if c.lone == nil {
			if err != nil {
				t.Errorf("%s: refused: %v", name, err)
			}
			continue
		}
		if !errors.As(err, &lone) || !slices.Equal(lone.Names, c.lone) || lone.Filtered != c.filtered {
			t.Errorf("%s: %v, want sections %v filtered %v", name, err, c.lone, c.filtered)
		}
	}
}

func must[T any](v T, _ error) T { return v }

func listsTestDB(t *testing.T) {
	t.Helper()
	logger.InitLogger(logging.CRITICAL)
	if err := database.InitDB(filepath.Join(t.TempDir(), "space-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.CloseDB() })
}

func TestRuleListsAreServedBySecretAndName(t *testing.T) {
	listsTestDB(t)
	s := RuleListService{}
	list, err := s.Save(&model.RuleList{Name: "youtube", Kind: RuleListDomains, Sections: []model.RuleListSection{
		{Name: "YouTube", Enabled: true, Entries: "youtube.com\nytimg.com"},
		{Name: "Off", Entries: "example.com"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if list.Count != 2 || len(list.Sections) != 2 || list.Sections[1].Entries != "example.com" {
		t.Fatalf("saved %+v", list)
	}
	var stored model.RuleList
	if err := database.GetDB().First(&stored, list.Id).Error; err != nil || len(stored.Sections) != 2 || stored.Sections[0].Name != "YouTube" {
		t.Fatalf("stored %+v %v", stored, err)
	}
	subnets := []model.RuleListSection{{Name: "a", Entries: "1.2.3.4"}}
	if _, err := s.Save(&model.RuleList{Name: "youtube", Kind: RuleListSubnets, Sections: subnets}); err == nil {
		t.Fatal("a second list takes the same name")
	}
	if _, err := s.Save(&model.RuleList{Id: list.Id, Name: "youtube", Kind: RuleListSubnets, Sections: subnets}); err == nil {
		t.Fatal("a list changes its kind")
	}
	if _, err := s.Save(&model.RuleList{Name: "Bad Name", Kind: RuleListDomains}); err == nil {
		t.Fatal("a name with a space passes")
	}

	secret, err := s.GetListsSecret()
	if err != nil || !ListsSecretPattern.MatchString(secret) {
		t.Fatalf("secret %q %v", secret, err)
	}
	if again, _ := s.GetListsSecret(); again != secret {
		t.Fatal("the secret changes between reads")
	}
	file, ok := s.ServedRuleList(secret + "/youtube.srs")
	if !ok || len(file.Data) < 4 || string(file.Data[:3]) != "SRS" || file.ETag == "" {
		t.Fatalf("served %v %v", file, ok)
	}
	for _, rest := range []string{"wrong/youtube.srs", secret + "/youtube.json", secret + "/missing.srs", secret + "/a/youtube.srs", "youtube.srs", ""} {
		if _, ok := s.ServedRuleList(rest); ok {
			t.Errorf("%q is served", rest)
		}
	}

	lists, err := s.All()
	if err != nil || len(lists) != 1 || lists[0].Srs != nil {
		t.Fatalf("all %v %v", lists, err)
	}
	if err := s.Delete(list.Id); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.ServedRuleList(secret + "/youtube.srs"); ok {
		t.Fatal("a deleted list is served")
	}
}
