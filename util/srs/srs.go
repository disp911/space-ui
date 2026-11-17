// Package srs writes sing-box rule-sets in their binary form (.srs), as
// "sing-box rule-set compile" does, for lists of domain suffixes or of IP
// prefixes. Only what such lists need is written: one rule with a domain or an
// ip_cidr item.
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

	itemDomain uint8 = 2
	itemIPCIDR uint8 = 6
	itemFinal  uint8 = 0xFF
)

// WriteDomains writes a rule-set whose rule matches each name and every name
// under it (domain_suffix). No names make a rule-set with no rules.
func WriteDomains(w io.Writer, suffixes []string) error {
	for _, s := range suffixes {
		if s == "" {
			return errors.New("srs: empty domain")
		}
	}
	return write(w, len(suffixes) > 0, func(b *bufio.Writer) error {
		if err := b.WriteByte(itemDomain); err != nil {
			return err
		}
		return domain.NewMatcher(nil, suffixes, false).Write(b)
	})
}

// WritePrefixes writes a rule-set whose rule matches the addresses in the
// prefixes (ip_cidr). No prefixes make a rule-set with no rules.
func WritePrefixes(w io.Writer, prefixes []netip.Prefix) error {
	var builder netipx.IPSetBuilder
	for _, p := range prefixes {
		if !p.IsValid() {
			return errors.New("srs: invalid prefix")
		}
		builder.AddPrefix(p)
	}
	set, err := builder.IPSet()
	if err != nil {
		return err
	}
	return write(w, len(prefixes) > 0, func(b *bufio.Writer) error {
		if err := b.WriteByte(itemIPCIDR); err != nil {
			return err
		}
		return writeIPSet(b, set)
	})
}

// write writes the header and, with one, a single rule holding the item.
func write(w io.Writer, one bool, item func(*bufio.Writer) error) error {
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
	rules := uint64(0)
	if one {
		rules = 1
	}
	if err := writeUvarint(b, rules); err != nil {
		return err
	}
	if one {
		if err := b.WriteByte(ruleDefault); err != nil {
			return err
		}
		if err := item(b); err != nil {
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

func writeUvarint(b *bufio.Writer, v uint64) error {
	var buf [binary.MaxVarintLen64]byte
	_, err := b.Write(buf[:binary.PutUvarint(buf[:], v)])
	return err
}
