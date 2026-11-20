// Package srs writes sing-box rule-sets in their binary form (.srs), as
// "sing-box rule-set compile" does, for lists of domain suffixes or of IP
// prefixes. Only what such lists need is written: one rule with a domain item,
// or rules with an ip_cidr item, each kept to a network and to ports if asked.
//
// The format is sing-box's (common/srs in sing-box 1.12): "SRS", a version
// byte, then zlib holding the rules. The domain item is sing's succinct
// domain matcher, the same library sing-box builds it with.
package srs

import (
	"bufio"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"net/netip"
	"strconv"

	"github.com/sagernet/sing/common/domain"
	"go4.org/netipx"
)

// Version is the rule-set version written: 3, read by sing-box 1.11 and
// later (routers run 1.12 and later). Versions 2 and 3 store domains and prefixes
// alike; version 1 matched domain suffixes the old way.
const Version = 3

var magic = [3]byte{'S', 'R', 'S'}

const (
	ruleDefault uint8 = 0

	itemNetwork   uint8 = 1
	itemDomain    uint8 = 2
	itemIPCIDR    uint8 = 6
	itemPort      uint8 = 9
	itemPortRange uint8 = 10
	itemFinal     uint8 = 0xFF
)

// PrefixRule is a rule matching the addresses in its prefixes; with a network
// ("tcp" or "udp") only that network's connections, and with ports or port
// ranges only connections to one of them.
type PrefixRule struct {
	Prefixes []netip.Prefix
	Network  string
	Ports    []uint16
	Ranges   []PortRange
}

// PortRange is the ports From to To, both included.
type PortRange struct {
	From, To uint16
}

// WriteDomains writes a rule-set whose rule matches each name and every name
// under it (domain_suffix). No names make a rule-set with no rules.
func WriteDomains(w io.Writer, suffixes []string) error {
	for _, s := range suffixes {
		if s == "" {
			return errors.New("srs: empty domain")
		}
	}
	if len(suffixes) == 0 {
		return write(w, nil)
	}
	return write(w, []func(*bufio.Writer) error{func(b *bufio.Writer) error {
		if err := b.WriteByte(itemDomain); err != nil {
			return err
		}
		return domain.NewMatcher(nil, suffixes, false).Write(b)
	}})
}

// WritePrefixes writes a rule-set whose rule matches the addresses in the
// prefixes (ip_cidr). No prefixes make a rule-set with no rules.
func WritePrefixes(w io.Writer, prefixes []netip.Prefix) error {
	if len(prefixes) == 0 {
		return WritePrefixRules(w, nil)
	}
	return WritePrefixRules(w, []PrefixRule{{Prefixes: prefixes}})
}

// WritePrefixRules writes a rule-set of the rules in their order, each with
// its network, ip_cidr, port and port_range items as sing-box writes them. A
// connection that any rule matches is in the rule-set. Every rule has
// prefixes: one with only a network or ports would match any address.
func WritePrefixRules(w io.Writer, rules []PrefixRule) error {
	items := make([]func(*bufio.Writer) error, 0, len(rules))
	for _, r := range rules {
		if len(r.Prefixes) == 0 {
			return errors.New("srs: a rule without prefixes")
		}
		if r.Network != "" && r.Network != "tcp" && r.Network != "udp" {
			return errors.New("srs: unknown network " + r.Network)
		}
		for _, p := range r.Ports {
			if p == 0 {
				return errors.New("srs: port 0")
			}
		}
		for _, pr := range r.Ranges {
			if pr.From == 0 || pr.From > pr.To {
				return errors.New("srs: invalid port range")
			}
		}
		var builder netipx.IPSetBuilder
		for _, p := range r.Prefixes {
			if !p.IsValid() {
				return errors.New("srs: invalid prefix")
			}
			builder.AddPrefix(p)
		}
		set, err := builder.IPSet()
		if err != nil {
			return err
		}
		items = append(items, func(b *bufio.Writer) error {
			if r.Network != "" {
				if err := writeStrings(b, itemNetwork, []string{r.Network}); err != nil {
					return err
				}
			}
			if err := b.WriteByte(itemIPCIDR); err != nil {
				return err
			}
			if err := writeIPSet(b, set); err != nil {
				return err
			}
			if len(r.Ports) > 0 {
				if err := b.WriteByte(itemPort); err != nil {
					return err
				}
				if err := writeUvarint(b, uint64(len(r.Ports))); err != nil {
					return err
				}
				if err := binary.Write(b, binary.BigEndian, r.Ports); err != nil {
					return err
				}
			}
			if len(r.Ranges) > 0 {
				ranges := make([]string, len(r.Ranges))
				for i, pr := range r.Ranges {
					ranges[i] = strconv.Itoa(int(pr.From)) + ":" + strconv.Itoa(int(pr.To))
				}
				if err := writeStrings(b, itemPortRange, ranges); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return write(w, items)
}

// write writes the header and the rules, each holding its items.
func write(w io.Writer, rules []func(*bufio.Writer) error) error {
	if _, err := w.Write(magic[:]); err != nil {
		return err
	}
	if _, err := w.Write([]byte{Version}); err != nil {
		return err
	}
	z, err := zlib.NewWriterLevel(w, zlib.BestCompression)
	if err != nil {
		return err
	}
	b := bufio.NewWriter(z)
	if err := writeUvarint(b, uint64(len(rules))); err != nil {
		return err
	}
	for _, items := range rules {
		if err := b.WriteByte(ruleDefault); err != nil {
			return err
		}
		if err := items(b); err != nil {
			return err
		}
		// The end of the rule's items, then its invert flag.
		if _, err := b.Write([]byte{itemFinal, 0}); err != nil {
			return err
		}
	}
	if err := b.Flush(); err != nil {
		return err
	}
	return z.Close()
}

// writeIPSet writes the set as sing-box does: a version byte, the number of
// ranges as a big-endian uint64, and each range's first and last address,
// each as a uvarint length and its bytes.
func writeIPSet(b *bufio.Writer, set *netipx.IPSet) error {
	if err := b.WriteByte(1); err != nil {
		return err
	}
	ranges := set.Ranges()
	if err := binary.Write(b, binary.BigEndian, uint64(len(ranges))); err != nil {
		return err
	}
	for _, r := range ranges {
		for _, a := range []netip.Addr{r.From(), r.To()} {
			raw := a.AsSlice()
			if err := writeUvarint(b, uint64(len(raw))); err != nil {
				return err
			}
			if _, err := b.Write(raw); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeStrings writes an item of strings: its type, their number, and each as
// a uvarint length and its bytes.
func writeStrings(b *bufio.Writer, item uint8, values []string) error {
	if err := b.WriteByte(item); err != nil {
		return err
	}
	if err := writeUvarint(b, uint64(len(values))); err != nil {
		return err
	}
	for _, v := range values {
		if err := writeUvarint(b, uint64(len(v))); err != nil {
			return err
		}
		if _, err := b.WriteString(v); err != nil {
			return err
		}
	}
	return nil
}

func writeUvarint(b *bufio.Writer, v uint64) error {
	var buf [binary.MaxVarintLen64]byte
	_, err := b.Write(buf[:binary.PutUvarint(buf[:], v)])
	return err
}
