package ping

import (
	"context"
	"slices"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/s5i/tassist/settings"
	"golang.org/x/sync/errgroup"
)

func New(stStorage *settings.Storage) (*Pinger, error) {
	ret := &Pinger{}

	st := stStorage.Get()
	p, err := new(st.ServerAddr)
	if err != nil {
		return nil, err
	}
	ret.pinger = p

	for _, addr := range st.ProxyAddrs {
		p, err := new(addr)
		if err != nil {
			return nil, err
		}
		ret.proxyPingers = append(ret.proxyPingers, p)
	}

	return ret, nil
}

func new(addr string) (*pinger, error) {
	ret := &pinger{
		wait: make([]bool, 20),
	}

	pinger := probing.New(addr)
	pinger.SetPrivileged(true)
	pinger.OnRecv = func(p *probing.Packet) {
		ret.registerRecv(p.Rtt)
	}
	pinger.OnSend = func(p *probing.Packet) {
		ret.registerSend()
	}
	ret.pinger = pinger

	return ret, nil
}

type Pinger struct {
	pinger       *pinger
	proxyPingers []*pinger
}

func (m *Pinger) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return m.pinger.Run(ctx)
	})
	for _, p := range m.proxyPingers {
		eg.Go(func() error {
			return p.Run(ctx)
		})
	}
	return eg.Wait()
}

func (m *Pinger) Stats() MultiStats {
	ret := MultiStats{
		Main: m.pinger.Stats(),
	}

	if len(m.proxyPingers) > 0 {
		var proxy []Stats
		for _, p := range m.proxyPingers {
			proxy = append(proxy, p.Stats())
		}

		slices.SortFunc(proxy, func(a, b Stats) int {
			if a.PacketLoss == b.PacketLoss {
				return int(a.RTT - b.RTT)
			}
			return int(1000 * (a.PacketLoss - b.PacketLoss))
		})
		ret.Proxy = &proxy[0]
	}

	return ret
}

type pinger struct {
	pinger *probing.Pinger
	ok     bool
	rtt    time.Duration
	nFails int
	wait   []bool
	waitIt int
	mu     sync.Mutex
}

type MultiStats struct {
	Main  Stats
	Proxy *Stats
}

type Stats struct {
	OK               bool
	RTT              time.Duration
	PacketLoss       float64
	PacketLossWindow time.Duration
}

func (p *pinger) Run(ctx context.Context) error {
	return p.pinger.RunWithContext(ctx)
}

func (p *pinger) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()

	ret := Stats{
		OK:               p.ok,
		RTT:              p.rtt,
		PacketLoss:       float64(p.nFails) / float64(len(p.wait)-1),
		PacketLossWindow: (p.pinger.Interval * time.Duration(len(p.wait))).Truncate(time.Second),
	}

	return ret
}

func (p *pinger) registerRecv(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ok = true
	p.rtt = d
	p.wait[p.waitIt] = false
}

func (p *pinger) registerSend() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Last recv didn't arrive.
	if p.wait[p.waitIt] {
		p.ok = false
		p.rtt = 0
		p.nFails++
	}

	p.waitIt = (p.waitIt + 1) % len(p.wait)

	// Discount the failure from the previous loop if present.
	if p.wait[p.waitIt] {
		p.nFails--
	}

	p.wait[p.waitIt] = true
}
