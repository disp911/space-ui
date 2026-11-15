package service

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"

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
	list, err := s.Save(&model.RuleList{Name: "youtube", Kind: RuleListDomains, Entries: "youtube.com\nytimg.com"})
	if err != nil {
		t.Fatal(err)
	}
	if list.Count != 2 || list.Entries != "youtube.com\nytimg.com" {
		t.Fatalf("saved %+v", list)
	}
	if _, err := s.Save(&model.RuleList{Name: "youtube", Kind: RuleListSubnets, Entries: "1.2.3.4"}); err == nil {
		t.Fatal("a second list takes the same name")
	}
	if _, err := s.Save(&model.RuleList{Id: list.Id, Name: "youtube", Kind: RuleListSubnets, Entries: "1.2.3.4"}); err == nil {
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
