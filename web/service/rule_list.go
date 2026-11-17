package service

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/util/common"
	"github.com/disp911/space-ui/v3/util/random"
	"github.com/disp911/space-ui/v3/util/srs"

	"golang.org/x/net/idna"
)

// Lists of domains or subnets for a router running sing-box: kept in the
// panel and served as sing-box rule-sets at <listsPath><listsSecret>/<name>.srs.
// Routers tell the format by the address's extension, so the secret goes in
// the path and the address ends in ".srs".

const (
	RuleListDomains = "domains"
	RuleListSubnets = "subnets"

	// MaxRuleListEntries bounds one list.
	MaxRuleListEntries = 200000
)

// A list's name is its file's: routers name the rule-set after it.
var ruleListName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

// A domain label once in ASCII; "_" is let through, as service names use it.
var domainLabel = regexp.MustCompile(`^[a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?$`)

// ListsSecretPattern is what a lists secret may be.
var ListsSecretPattern = regexp.MustCompile(`^[A-Za-z0-9]{16,64}$`)

// RuleListLineError is a list line that is not a domain or a subnet.
type RuleListLineError struct {
	Line int
	Text string
}

func (e *RuleListLineError) Error() string {
	return fmt.Sprintf("line %d is not valid: %s", e.Line, e.Text)
}

// NormalizeDomain returns the name as a list keeps it: lowercase ASCII
// (punycode), without a leading "*." or "." and a trailing dot. A name
// matches itself and every name under it.
func NormalizeDomain(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "*.")
	s = strings.TrimPrefix(s, ".")
	s = strings.TrimSuffix(s, ".")
	if s == "" || len(s) > 253 {
		return "", false
	}
	ascii, err := idna.Lookup.ToASCII(s)
	if err != nil {
		// Lookup refuses "_"; a name with one is checked as it is.
		if strings.Contains(s, "_") && isASCII(s) {
			ascii = s
		} else {
			return "", false
		}
	}
	if len(ascii) > 253 {
		return "", false
	}
	for _, label := range strings.Split(ascii, ".") {
		if !domainLabel.MatchString(label) {
			return "", false
		}
	}
	return ascii, true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// NormalizeSubnet returns the prefix as a list keeps it: masked, with a
// lone address as a /32 or /128. IPv4-mapped IPv6 is refused: it is written
// as IPv4.
func NormalizeSubnet(s string) (netip.Prefix, bool) {
	s = strings.TrimSpace(s)
	var p netip.Prefix
	if strings.Contains(s, "/") {
		var err error
		if p, err = netip.ParsePrefix(s); err != nil {
			return netip.Prefix{}, false
		}
	} else {
		a, err := netip.ParseAddr(s)
		if err != nil || a.Zone() != "" {
			return netip.Prefix{}, false
		}
		p = netip.PrefixFrom(a, a.BitLen())
	}
	if p.Addr().Is4In6() {
		return netip.Prefix{}, false
	}
	return p.Masked(), true
}

// ParseRuleList reads a list's text: one entry a line, empty lines and lines
// starting with "#" skipped, repeats dropped. It returns the entries as the
// list keeps them, or the first line that is wrong.
func ParseRuleList(kind, text string) ([]string, error) {
	if kind != RuleListDomains && kind != RuleListSubnets {
		return nil, common.NewError("unknown list kind:", kind)
	}
	seen := map[string]bool{}
	var out []string
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var entry string
		if kind == RuleListDomains {
			d, ok := NormalizeDomain(line)
			if !ok {
				return nil, &RuleListLineError{Line: i + 1, Text: line}
			}
			entry = d
		} else {
			p, ok := NormalizeSubnet(line)
			if !ok {
				return nil, &RuleListLineError{Line: i + 1, Text: line}
			}
			entry = p.String()
		}
		if seen[entry] {
			continue
		}
		seen[entry] = true
		out = append(out, entry)
		if len(out) > MaxRuleListEntries {
			return nil, common.NewErrorf("a list holds at most %d entries", MaxRuleListEntries)
		}
	}
	return out, nil
}

// CompileRuleList builds the list's rule-set.
func CompileRuleList(kind string, entries []string) ([]byte, error) {
	var buf bytes.Buffer
	if kind == RuleListDomains {
		if err := srs.WriteDomains(&buf, entries); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	prefixes := make([]netip.Prefix, 0, len(entries))
	for _, e := range entries {
		p, err := netip.ParsePrefix(e)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, p)
	}
	if err := srs.WritePrefixes(&buf, prefixes); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RuleListService keeps the lists.
type RuleListService struct {
	SettingService
}

// All returns the lists without their rule-sets.
func (s *RuleListService) All() ([]model.RuleList, error) {
	var lists []model.RuleList
	err := database.GetDB().Omit("srs").Order("id").Find(&lists).Error
	return lists, err
}

// Save creates the list, or with an id changes it. A list keeps its kind.
func (s *RuleListService) Save(in *model.RuleList) (*model.RuleList, error) {
	in.Name = strings.TrimSpace(in.Name)
	if !ruleListName.MatchString(in.Name) {
		return nil, common.NewError("a list's name is lowercase letters, digits, \"-\" and \"_\":", in.Name)
	}
	db := database.GetDB()
	var list model.RuleList
	if in.Id > 0 {
		if err := db.First(&list, in.Id).Error; err != nil {
			return nil, err
		}
		if in.Kind != "" && in.Kind != list.Kind {
			return nil, errors.New("a list keeps the kind it was made with")
		}
	} else {
		list.Kind = in.Kind
	}
	var taken int64
	if err := db.Model(&model.RuleList{}).Where("name = ? AND id <> ?", in.Name, list.Id).Count(&taken).Error; err != nil {
		return nil, err
	}
	if taken > 0 {
		return nil, common.NewError("there is a list named", in.Name, "already")
	}
	entries, err := ParseRuleList(list.Kind, in.Entries)
	if err != nil {
		return nil, err
	}
	data, err := CompileRuleList(list.Kind, entries)
	if err != nil {
		return nil, err
	}
	list.Name = in.Name
	list.Entries = strings.Join(entries, "\n")
	list.Count = len(entries)
	list.Srs = data
	if err := db.Save(&list).Error; err != nil {
		return nil, err
	}
	list.Srs = nil
	return &list, nil
}

// Delete removes the list.
func (s *RuleListService) Delete(id int) error {
	return database.GetDB().Delete(&model.RuleList{}, id).Error
}

// GetListsSecret returns the secret in the lists' addresses, making one the
// first time it is asked for.
func (s *SettingService) GetListsSecret() (string, error) {
	setting, err := s.getSetting("listsSecret")
	if err == nil {
		return setting.Value, nil
	}
	if !database.IsNotFound(err) {
		return "", err
	}
	secret := random.Seq(24)
	if err := s.saveSetting("listsSecret", secret); err != nil {
		return "", err
	}
	return secret, nil
}

// GetListsPath returns the folder path the lists are served under.
func (s *SettingService) GetListsPath() (string, error) {
	return s.getString("listsPath")
}

// RuleListFile is a list's rule-set as served.
type RuleListFile struct {
	Data      []byte
	ETag      string
	UpdatedAt int64
}

// ServedRuleList returns the rule-set at rest, the path under the lists'
// folder ("<secret>/<name>.srs"), or false when there is none there.
func (s *RuleListService) ServedRuleList(rest string) (*RuleListFile, bool) {
	secret, file, ok := strings.Cut(rest, "/")
	if !ok || strings.Contains(file, "/") {
		return nil, false
	}
	name, ok := strings.CutSuffix(file, ".srs")
	if !ok || !ruleListName.MatchString(name) {
		return nil, false
	}
	want, err := s.GetListsSecret()
	if err != nil || want == "" || subtle.ConstantTimeCompare([]byte(secret), []byte(want)) != 1 {
		return nil, false
	}
	var list model.RuleList
	if err := database.GetDB().Where("name = ?", name).First(&list).Error; err != nil || list.Srs == nil {
		return nil, false
	}
	sum := sha256.Sum256(list.Srs)
	return &RuleListFile{Data: list.Srs, ETag: `"` + hex.EncodeToString(sum[:8]) + `"`, UpdatedAt: list.UpdatedAt}, true
}
