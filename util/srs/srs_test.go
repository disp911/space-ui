package srs

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sagernet/sing/common/domain"
)

// read decodes what write wrote: the version, and the one rule's item type and
// reader positioned at its value, or no rule.
func read(t *testing.T, data []byte) (version byte, item byte, r *bufio.Reader) {
	t.Helper()
	if !bytes.HasPrefix(data, []byte("SRS")) || len(data) < 4 {
		t.Fatalf("no SRS header: %x", data[:min(len(data), 8)])
	}
	z, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		t.Fatal(err)
	}
	r = bufio.NewReader(z)
	n, err := binary.ReadUvarint(r)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		return data[3], 0, nil
	}
	if n != 1 {
		t.Fatalf("%d rules", n)
	}
	if kind, _ := r.ReadByte(); kind != ruleDefault {
		t.Fatalf("rule type %d", kind)
	}
	item, _ = r.ReadByte()
	return data[3], item, r
}

func TestWriteDomains(t *testing.T) {
	var buf bytes.Buffer
	in := []string{"youtube.com", "googlevideo.com", "xn--p1ai"}
	if err := WriteDomains(&buf, in); err != nil {
		t.Fatal(err)
	}
	version, item, r := read(t, buf.Bytes())
	if version != Version || item != itemDomain {
		t.Fatalf("version %d item %d", version, item)
	}
	m, err := domain.ReadMatcher(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"youtube.com", "www.youtube.com", "r1.googlevideo.com", "xn--80aswg.xn--p1ai"} {
		if !m.Match(name) {
			t.Errorf("%s does not match", name)
		}
	}
	for _, name := range []string{"notyoutube.com", "youtube.co", "example.org"} {
		if m.Match(name) {
			t.Errorf("%s matches", name)
		}
	}
	if end, _ := r.ReadByte(); end != itemFinal {
		t.Fatalf("no end of items: %d", end)
	}
	if invert, _ := r.ReadByte(); invert != 0 {
		t.Fatal("inverted")
	}
}

func TestWritePrefixes(t *testing.T) {
	var buf bytes.Buffer
	in := []netip.Prefix{
		netip.MustParsePrefix("91.108.4.0/22"),
		netip.MustParsePrefix("91.108.8.0/22"), // joins the one before into 91.108.4.0–91.108.11.255
		netip.MustParsePrefix("2001:b28:f23d::/48"),
		netip.MustParsePrefix("149.154.167.51/32"),
	}
	if err := WritePrefixes(&buf, in); err != nil {
		t.Fatal(err)
	}
	_, item, r := read(t, buf.Bytes())
	if item != itemIPCIDR {
		t.Fatalf("item %d", item)
	}
	if v, _ := r.ReadByte(); v != 1 {
		t.Fatalf("set version %d", v)
	}
	var n uint64
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		t.Fatal(err)
	}
	var got []string
	for range n {
		var ends [2]netip.Addr
		for i := range ends {
			size, _ := binary.ReadUvarint(r)
			raw := make([]byte, size)
			if _, err := r.Read(raw); err != nil {
				t.Fatal(err)
			}
			ends[i], _ = netip.AddrFromSlice(raw)
		}
		got = append(got, ends[0].String()+"-"+ends[1].String())
	}
	want := []string{
		"91.108.4.0-91.108.11.255",
		"149.154.167.51-149.154.167.51",
		"2001:b28:f23d::-2001:b28:f23d:ffff:ffff:ffff:ffff:ffff",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ranges\n got %v\nwant %v", got, want)
	}
}

// TestWritePrefixRules reads the rules back: one for every port, then one kept
// to UDP and to a port and a range, with sing-box's item types and encoding.
func TestWritePrefixRules(t *testing.T) {
	var buf bytes.Buffer
	rules := []PrefixRule{
		{Prefixes: []netip.Prefix{netip.MustParsePrefix("91.108.4.0/22")}},
		{
			Prefixes: []netip.Prefix{netip.MustParsePrefix("104.29.128.0/18")},
			Network:  "udp",
			Ports:    []uint16{3478},
			Ranges:   []PortRange{{19000, 20000}, {50000, 65535}},
		},
	}
	if err := WritePrefixRules(&buf, rules); err != nil {
		t.Fatal(err)
	}
	z, err := zlib.NewReader(bytes.NewReader(buf.Bytes()[4:]))
	if err != nil {
		t.Fatal(err)
	}
	r := bufio.NewReader(z)
	if n, _ := binary.ReadUvarint(r); n != 2 {
		t.Fatalf("%d rules", n)
	}
	str := func() string {
		size, _ := binary.ReadUvarint(r)
		raw := make([]byte, size)
		if _, err := io.ReadFull(r, raw); err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	skipSet := func() {
		if v, _ := r.ReadByte(); v != 1 {
			t.Fatalf("set version %d", v)
		}
		var n uint64
		if err := binary.Read(r, binary.BigEndian, &n); err != nil {
			t.Fatal(err)
		}
		for range 2 * n {
			str()
		}
	}
	expect := func(want ...byte) {
		t.Helper()
		for _, w := range want {
			if b, _ := r.ReadByte(); b != w {
				t.Fatalf("byte %d, want %d", b, w)
			}
		}
	}
	// The first rule: its addresses only.
	expect(ruleDefault, itemIPCIDR)
	skipSet()
	expect(itemFinal, 0)
	// The second: network, addresses, ports, then port ranges.
	expect(ruleDefault, itemNetwork, 1)
	if s := str(); s != "udp" {
		t.Fatalf("network %q", s)
	}
	expect(itemIPCIDR)
	skipSet()
	expect(itemPort, 1)
	var port uint16
	if err := binary.Read(r, binary.BigEndian, &port); err != nil || port != 3478 {
		t.Fatalf("port %d %v", port, err)
	}
	expect(itemPortRange, 2)
	if a, b := str(), str(); a != "19000:20000" || b != "50000:65535" {
		t.Fatalf("ranges %q %q", a, b)
	}
	expect(itemFinal, 0)

	for _, bad := range []PrefixRule{
		{Network: "udp", Ports: []uint16{443}},
		{Prefixes: rules[0].Prefixes, Network: "icmp"},
		{Prefixes: rules[0].Prefixes, Ports: []uint16{0}},
		{Prefixes: rules[0].Prefixes, Ranges: []PortRange{{20000, 19000}}},
	} {
		if err := WritePrefixRules(&buf, []PrefixRule{bad}); err == nil {
			t.Errorf("%+v is written", bad)
		}
	}
}

func TestWriteEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteDomains(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if _, item, r := read(t, buf.Bytes()); item != 0 || r != nil {
		t.Fatal("an empty list has a rule")
	}
	if err := WriteDomains(&buf, []string{""}); err == nil {
		t.Fatal("an empty domain is written")
	}
}

// TestRealSingBox decompiles the rule-sets with sing-box itself, the oldest
// release routers run, when SINGBOX_BIN names it (as CI does).
func TestRealSingBox(t *testing.T) {
	bin := os.Getenv("SINGBOX_BIN")
	if bin == "" {
		t.Skip("SINGBOX_BIN not set")
	}
	dir := t.TempDir()
	decompile := func(name string, write func(*bytes.Buffer) error) []map[string]any {
		var buf bytes.Buffer
		if err := write(&buf); err != nil {
			t.Fatal(err)
		}
		src := filepath.Join(dir, name+".srs")
		out := filepath.Join(dir, name+".json")
		if err := os.WriteFile(src, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if b, err := exec.Command(bin, "rule-set", "decompile", src, "-o", out).CombinedOutput(); err != nil {
			t.Fatalf("decompile %s: %v\n%s", name, err, b)
		}
		raw, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		var rs struct {
			Version int              `json:"version"`
			Rules   []map[string]any `json:"rules"`
		}
		if err := json.Unmarshal(raw, &rs); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, raw)
		}
		if rs.Version != Version || len(rs.Rules) == 0 {
			t.Fatalf("%s: %s", name, raw)
		}
		return rs.Rules
	}
	strs := func(v any) []string {
		var out []string
		switch v := v.(type) {
		case string:
			out = []string{v}
		case []any:
			for _, s := range v {
				out = append(out, s.(string))
			}
		}
		slices.Sort(out)
		return out
	}

	domains := []string{"youtube.com", "googlevideo.com", "xn--p1ai"}
	rule := decompile("domains", func(b *bytes.Buffer) error { return WriteDomains(b, domains) })[0]
	want := slices.Clone(domains)
	slices.Sort(want)
	if got := strs(rule["domain_suffix"]); !slices.Equal(got, want) {
		t.Fatalf("domain_suffix %v, want %v (rule %v)", got, want, rule)
	}

	prefixes := []netip.Prefix{netip.MustParsePrefix("91.108.4.0/22"), netip.MustParsePrefix("2001:b28:f23d::/48"), netip.MustParsePrefix("149.154.167.51/32")}
	rule = decompile("subnets", func(b *bytes.Buffer) error { return WritePrefixes(b, prefixes) })[0]
	if got := strs(rule["ip_cidr"]); !slices.Equal(got, []string{"149.154.167.51/32", "2001:b28:f23d::/48", "91.108.4.0/22"}) {
		t.Fatalf("ip_cidr %v (rule %v)", got, rule)
	}

	// A rule for every port, then one kept to UDP and to a port and ranges.
	rules := decompile("filtered", func(b *bytes.Buffer) error {
		return WritePrefixRules(b, []PrefixRule{
			{Prefixes: prefixes[:1]},
			{Prefixes: []netip.Prefix{netip.MustParsePrefix("104.29.128.0/18")}, Network: "udp", Ports: []uint16{3478}, Ranges: []PortRange{{19000, 20000}, {50000, 65535}}},
		})
	})
	if len(rules) != 2 {
		t.Fatalf("rules %v", rules)
	}
	if got := strs(rules[0]["ip_cidr"]); !slices.Equal(got, []string{"91.108.4.0/22"}) || len(rules[0]) != 1 {
		t.Fatalf("first rule %v", rules[0])
	}
	second := rules[1]
	if got := strs(second["ip_cidr"]); !slices.Equal(got, []string{"104.29.128.0/18"}) {
		t.Fatalf("second ip_cidr %v (rule %v)", got, second)
	}
	if got := strs(second["network"]); !slices.Equal(got, []string{"udp"}) {
		t.Fatalf("network %v (rule %v)", got, second)
	}
	if got := strs(second["port_range"]); !slices.Equal(got, []string{"19000:20000", "50000:65535"}) {
		t.Fatalf("port_range %v (rule %v)", got, second)
	}
	if fmt.Sprint(second["port"]) != "3478" && fmt.Sprint(second["port"]) != "[3478]" {
		t.Fatalf("port %v (rule %v)", second["port"], second)
	}
}
