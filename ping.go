package main

import (
	"context"
	"fmt"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

const count = 3

type PingProbe struct {
	pinger *probing.Pinger
	target string
	name   string
}

func NewPinger(conf PingerConfig, probe Probe) (*PingProbe, error) {
	p, err := probing.NewPinger(probe.Target)
	if err != nil {
		return nil, fmt.Errorf("could not build probe for %w", err)
	}

	p.Count = conf.Count
	p.Timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	p.SetPrivileged(conf.UsePrivileged)

	ret := &PingProbe{
		pinger: p,
		name:   probe.Name,
		target: probe.Target,
	}

	return ret, nil
}

func (p *PingProbe) Check(_ context.Context) (bool, error) {
	if err := p.pinger.Run(); err != nil {
		ProbeErrors.WithLabelValues("ping", p.name, p.target).Inc()
		return false, err
	}

	stats := p.pinger.Statistics()
	return stats.PacketsRecv > 0, nil
}
