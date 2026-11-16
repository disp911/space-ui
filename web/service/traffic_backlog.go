package service

import (
	"errors"
	"fmt"
	"sync"

	"github.com/disp911/space-ui/v3/xray"
)

// ErrTrafficNotRecorded marks a failed traffic write that left nothing in the
// database, e.g. while the disk stalls and the database stays locked: the
// caller keeps the traffic in a TrafficBacklog for the next write.
var ErrTrafficNotRecorded = errors.New("traffic not recorded")

func notRecorded(err error) error {
	return fmt.Errorf("%w: %w", ErrTrafficNotRecorded, err)
}

type trafficKey struct {
	inbound, outbound bool
	tag               string
}

// TrafficBacklog holds traffic a write could not record. Xray and telemt hand
// over only what passed since they were last asked and then forget it, so
// traffic dropped on a failed write would be lost for good; kept here, the
// next write adds it. The zero value is ready to use.
type TrafficBacklog struct {
	mu       sync.Mutex
	traffics map[trafficKey]*xray.Traffic
	clients  map[string]*xray.ClientTraffic
}

// Take returns the given traffic with the backlog added and empties the
// backlog. The given values are never changed: others may read them too.
func (b *TrafficBacklog) Take(traffics []*xray.Traffic, clients []*xray.ClientTraffic) ([]*xray.Traffic, []*xray.ClientTraffic) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.traffics) == 0 && len(b.clients) == 0 {
		return traffics, clients
	}
	pendingTraffics, pendingClients := b.traffics, b.clients
	b.traffics, b.clients = nil, nil

	outTraffics := make([]*xray.Traffic, 0, len(traffics)+len(pendingTraffics))
	for _, t := range traffics {
		if t == nil {
			continue
		}
		key := trafficKey{t.IsInbound, t.IsOutbound, t.Tag}
		merged := *t
		if p, ok := pendingTraffics[key]; ok {
			merged.Up += p.Up
			merged.Down += p.Down
			delete(pendingTraffics, key)
		}
		outTraffics = append(outTraffics, &merged)
	}
	for _, p := range pendingTraffics {
		outTraffics = append(outTraffics, p)
	}

	outClients := make([]*xray.ClientTraffic, 0, len(clients)+len(pendingClients))
	for _, c := range clients {
		if c == nil {
			continue
		}
		merged := *c
		if p, ok := pendingClients[c.Email]; ok {
			merged.Up += p.Up
			merged.Down += p.Down
			delete(pendingClients, c.Email)
		}
		outClients = append(outClients, &merged)
	}
	for _, p := range pendingClients {
		outClients = append(outClients, p)
	}
	return outTraffics, outClients
}

// Keep puts traffic that failed to be recorded back into the backlog.
func (b *TrafficBacklog) Keep(traffics []*xray.Traffic, clients []*xray.ClientTraffic) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, t := range traffics {
		if t == nil || t.Up+t.Down == 0 {
			continue
		}
		if b.traffics == nil {
			b.traffics = map[trafficKey]*xray.Traffic{}
		}
		key := trafficKey{t.IsInbound, t.IsOutbound, t.Tag}
		if p, ok := b.traffics[key]; ok {
			p.Up += t.Up
			p.Down += t.Down
			continue
		}
		b.traffics[key] = &xray.Traffic{IsInbound: t.IsInbound, IsOutbound: t.IsOutbound, Tag: t.Tag, Up: t.Up, Down: t.Down}
	}
	for _, c := range clients {
		if c == nil || c.Up+c.Down == 0 {
			continue
		}
		if b.clients == nil {
			b.clients = map[string]*xray.ClientTraffic{}
		}
		if p, ok := b.clients[c.Email]; ok {
			p.Up += c.Up
			p.Down += c.Down
			continue
		}
		b.clients[c.Email] = &xray.ClientTraffic{Email: c.Email, Up: c.Up, Down: c.Down}
	}
}
