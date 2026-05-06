package ping

import (
	"context"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

const (
	addrAncestra = "51.89.155.163"
)

func New() (*Pinger, error) {
	ret := &Pinger{
		wait: make([]bool, 20),
	}

	pinger := probing.New(addrAncestra)
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
	pinger *probing.Pinger
	ok     bool
	rtt    time.Duration
	nFails int
	wait   []bool
	waitIt int
	mu     sync.Mutex
}

type Stats struct {
	OK               bool
	RTT              time.Duration
	PacketLoss       float64
	PacketLossWindow time.Duration
}

func (p *Pinger) Run(ctx context.Context) error {
	return p.pinger.RunWithContext(ctx)
}

func (p *Pinger) Stats() Stats {
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

func (p *Pinger) registerRecv(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ok = true
	p.rtt = d
	p.wait[p.waitIt] = false
}

func (p *Pinger) registerSend() {
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
