package service

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/util/common"
	"github.com/disp911/space-ui/v3/util/random"
	"github.com/disp911/space-ui/v3/util/srs"

	"go4.org/netipx"
	"golang.org/x/net/idna"
)

// Lists of domains or subnets for a router running sing-box: kept in the
// panel and served as sing-box rule-sets at <listsPath><listsSecret>/<name>.srs.
// Routers tell the format by the address's extension, so the secret goes in
// the path and the address ends in ".srs".

const (
	RuleListDomains = "domains"
	RuleListSubnets = "subnets"

	// MaxRuleListEntries bounds one list, its sections' entries together.
	MaxRuleListEntries = 200000
	// MaxRuleListSections bounds a list's sections.
	MaxRuleListSections = 100
	// MaxRuleListSectionName bounds a section's name, in characters.
	MaxRuleListSectionName = 64
	// MaxRuleListPorts bounds a section's ports and port ranges.
	MaxRuleListPorts = 64
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

// RuleListSectionError is what is wrong in a section, counted from 1.
type RuleListSectionError struct {
	Section int
	Name    string
	Err     error
}

func (e *RuleListSectionError) Error() string {
	return fmt.Sprintf("section %d «%s»: %v", e.Section, e.Name, e.Err)
}

func (e *RuleListSectionError) Unwrap() error { return e.Err }

// RuleListLoneSubnetError is a rule of a subnets list whose addresses are one
// subnet. sing-box writes such a rule's subnet out of the rule-set as a
// string, not a list of them, and a router reading a list there fails to load
// the whole list. Names are the sections the rule is made of.
type RuleListLoneSubnetError struct {
	Names []string
	// Filtered is that the sections are kept to a protocol or ports.
	Filtered bool
}

func (e *RuleListLoneSubnetError) Error() string {
	names := make([]string, len(e.Names))
	for i, n := range e.Names {
		names[i] = "«" + n + "»"
	}
	advice := "add another subnet"
	if e.Filtered {
		advice += " or keep the section to no protocol and ports"
	}
	return fmt.Sprintf("section %s: a router cannot read a rule with one subnet and then loads none of the list: %s", strings.Join(names, ", "), advice)
}

// ParseRuleListPorts reads a section's ports, each a port ("3478") or a range
// of ports ("19000-20000"), and returns them as the list keeps them, without
// repeats, and as the rule-set writes them. A range of one port is that port.
func ParseRuleListPorts(in []string) ([]string, []uint16, []srs.PortRange, error) {
	if len(in) > MaxRuleListPorts {
		return nil, nil, nil, common.NewErrorf("a section holds at most %d ports and port ranges", MaxRuleListPorts)
	}
	port := func(s string) (uint16, bool) {
		n, err := strconv.Atoi(s)
		return uint16(n), err == nil && n >= 1 && n <= 65535 && s == strconv.Itoa(n)
	}
	seen := map[string]bool{}
	var kept []string
	var ports []uint16
	var ranges []srs.PortRange
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		from, to, isRange := strings.Cut(s, "-")
		a, okA := port(strings.TrimSpace(from))
		b, okB := a, okA
		if isRange {
			b, okB = port(strings.TrimSpace(to))
		}
		if !okA || !okB || a > b {
			return nil, nil, nil, common.NewError("not a port or a port range:", s)
		}
		norm := strconv.Itoa(int(a))
		if a != b {
			norm += "-" + strconv.Itoa(int(b))
		}
		if seen[norm] {
			continue
		}
		seen[norm] = true
		kept = append(kept, norm)
		if a == b {
			ports = append(ports, a)
		} else {
			ranges = append(ranges, srs.PortRange{From: a, To: b})
		}
	}
	return kept, ports, ranges, nil
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

// BuildRuleList checks a list's sections and returns them as the list keeps
// them, each with its entries normalized and counted, and the entries of the
// enabled ones for the rule-set, each once, in the sections' order.
func BuildRuleList(kind string, sections []model.RuleListSection) ([]model.RuleListSection, []string, error) {
	if kind != RuleListDomains && kind != RuleListSubnets {
		return nil, nil, common.NewError("unknown list kind:", kind)
	}
	if len(sections) > MaxRuleListSections {
		return nil, nil, common.NewErrorf("a list holds at most %d sections", MaxRuleListSections)
	}
	out := make([]model.RuleListSection, 0, len(sections))
	names := map[string]bool{}
	seen := map[string]bool{}
	var entries []string
	total := 0
	for i, sec := range sections {
		name := strings.TrimSpace(sec.Name)
		fail := func(err error) error { return &RuleListSectionError{Section: i + 1, Name: name, Err: err} }
		switch {
		case name == "":
			return nil, nil, fail(errors.New("a section needs a name"))
		case utf8.RuneCountInString(name) > MaxRuleListSectionName || strings.ContainsAny(name, "\r\n"):
			return nil, nil, fail(common.NewErrorf("a section's name is one line of at most %d characters", MaxRuleListSectionName))
		case names[strings.ToLower(name)]:
			return nil, nil, fail(errors.New("there is a section with this name already"))
		}
		names[strings.ToLower(name)] = true
		list, err := ParseRuleList(kind, sec.Entries)
		if err != nil {
			return nil, nil, fail(err)
		}
		network := strings.ToLower(strings.TrimSpace(sec.Network))
		if network != "" && network != "tcp" && network != "udp" {
			return nil, nil, fail(common.NewError("a section's protocol is tcp or udp, not", sec.Network))
		}
		ports, _, _, err := ParseRuleListPorts(sec.Ports)
		if err != nil {
			return nil, nil, fail(err)
		}
		if kind == RuleListDomains && (network != "" || len(ports) > 0) {
			return nil, nil, fail(errors.New("only a subnets list's sections are kept to a protocol or ports"))
		}
		if total += len(list); total > MaxRuleListEntries {
			return nil, nil, common.NewErrorf("a list holds at most %d entries", MaxRuleListEntries)
		}
		out = append(out, model.RuleListSection{Name: name, Enabled: sec.Enabled, Entries: strings.Join(list, "\n"), Count: len(list), Network: network, Ports: ports})
		if !sec.Enabled {
			continue
		}
		for _, e := range list {
			if !seen[e] {
				seen[e] = true
				entries = append(entries, e)
			}
		}
	}
	return out, entries, nil
}

// CompileRuleList builds the list's rule-set from its enabled sections, as
// BuildRuleList returns them. A subnets list is a rule for the sections kept
// to no protocol or ports, matching every connection to their addresses, then
// a rule for each protocol and ports some sections are kept to, matching only
// such connections; sections kept to the same ones share a rule. A rule whose
// addresses are one subnet is refused (RuleListLoneSubnetError).
func CompileRuleList(kind string, sections []model.RuleListSection) ([]byte, error) {
	var buf bytes.Buffer
	if kind == RuleListDomains {
		var entries []string
		seen := map[string]bool{}
		for _, sec := range sections {
			if !sec.Enabled {
				continue
			}
			for _, e := range strings.Fields(sec.Entries) {
				if !seen[e] {
					seen[e] = true
					entries = append(entries, e)
				}
			}
		}
		if err := srs.WriteDomains(&buf, entries); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	type group struct {
		network string
		ports   []string
		names   []string
		seen    map[netip.Prefix]bool
		rule    srs.PrefixRule
	}
	var groups []*group
	byKey := map[string]*group{}
	for _, sec := range sections {
		if !sec.Enabled {
			continue
		}
		key := sec.Network + "|" + strings.Join(sec.Ports, ",")
		g := byKey[key]
		if g == nil {
			g = &group{network: sec.Network, ports: sec.Ports, seen: map[netip.Prefix]bool{}}
			byKey[key] = g
			groups = append(groups, g)
		}
		if sec.Entries != "" {
			g.names = append(g.names, sec.Name)
		}
		for _, e := range strings.Fields(sec.Entries) {
			p, err := netip.ParsePrefix(e)
			if err != nil {
				return nil, err
			}
			if !g.seen[p] {
				g.seen[p] = true
				g.rule.Prefixes = append(g.rule.Prefixes, p)
			}
		}
	}
	// The rule for every connection leads, wherever its sections are.
	slices.SortStableFunc(groups, func(a, b *group) int {
		return cmp.Compare(len(a.network)+len(a.ports), len(b.network)+len(b.ports))
	})
	rules := make([]srs.PrefixRule, 0, len(groups))
	for _, g := range groups {
		if len(g.rule.Prefixes) == 0 {
			continue
		}
		if oneSubnet(g.rule.Prefixes) {
			return nil, &RuleListLoneSubnetError{Names: g.names, Filtered: g.network != "" || len(g.ports) > 0}
		}
		_, ports, ranges, err := ParseRuleListPorts(g.ports)
		if err != nil {
			return nil, err
		}
		g.rule.Network, g.rule.Ports, g.rule.Ranges = g.network, ports, ranges
		rules = append(rules, g.rule)
	}
	if err := srs.WritePrefixRules(&buf, rules); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// oneSubnet is whether the prefixes' addresses together are one subnet, as
// sing-box writes them out of a rule-set: 10.0.0.0/25 and 10.0.0.128/25 are
// 10.0.0.0/24.
func oneSubnet(prefixes []netip.Prefix) bool {
	var b netipx.IPSetBuilder
	for _, p := range prefixes {
		b.AddPrefix(p)
	}
	set, err := b.IPSet()
	return err == nil && len(set.Prefixes()) == 1
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
	sections, entries, err := BuildRuleList(list.Kind, in.Sections)
	if err != nil {
		return nil, err
	}
	data, err := CompileRuleList(list.Kind, sections)
	if err != nil {
		return nil, err
	}
	list.Name = in.Name
	list.Sections = sections
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
	if err := database.GetDB().Where("name = ?", name).First(&list).Error; err != nil {
		return nil, false
	}
	sum := sha256.Sum256(list.Srs)
	return &RuleListFile{Data: list.Srs, ETag: `"` + hex.EncodeToString(sum[:8]) + `"`, UpdatedAt: list.UpdatedAt}, true
}
