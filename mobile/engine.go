package mobile

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"json2proxy/internal/config"
	"json2proxy/internal/engine"
	"json2proxy/internal/router"
	"json2proxy/internal/tun"
)

// Notice kinds sent by the running engine, in addition to the ones above.
const (
	NoticeConnected        = engine.NoticeConnected
	NoticeConnectionFailed = engine.NoticeConnectionFailed
	NoticeProxyUnavailable = engine.NoticeProxyUnavailable
)

// ModeCustom routes with the rules passed in the engine configuration.
const ModeCustom = "custom"

// engineConfig is the JSON accepted by StartEngine:
//
//	{"client": {...client.json...}, "mode": "global", "rules": "",
//	 "doh": "", "bootstrap": "", "mtu": 1500}
//
// Empty fields take their defaults; "rules" is only used with mode "custom".
type engineConfig struct {
	Client    config.Client `json:"client"`
	Mode      string        `json:"mode"`
	Rules     string        `json:"rules"`
	DoH       string        `json:"doh"`
	Bootstrap string        `json:"bootstrap"`
	MTU       int           `json:"mtu"`
}

var (
	engineMu sync.Mutex
	running  *engine.Engine
)

// StartEngine starts the network engine on a TUN file descriptor, which it
// takes ownership of. Only one engine runs at a time.
func StartEngine(tunFd int, configJSON string) error {
	cfg, err := parseEngineConfig(configJSON)
	if err != nil {
		return err
	}
	return startEngine(tun.NewFDDevice(tunFd), cfg)
}

func startEngine(dev tun.Device, cfg engine.Config) error {
	engineMu.Lock()
	defer engineMu.Unlock()
	if running != nil {
		_ = dev.Close()
		return errors.New("engine already running")
	}
	cfg.Dial = dialProtected
	cfg.Log = logf
	cfg.Notice = func(kind, message string) {
		if p := current(); p != nil {
			p.Notice(kind, message)
		}
	}
	e, err := engine.Start(dev, cfg)
	if err != nil {
		return err
	}
	running = e
	return nil
}

// StopEngine stops the running engine, if any.
func StopEngine() {
	engineMu.Lock()
	e := running
	running = nil
	engineMu.Unlock()
	if e != nil {
		_ = e.Close()
	}
}

// EngineRunning reports whether an engine is running.
func EngineRunning() bool {
	engineMu.Lock()
	defer engineMu.Unlock()
	return running != nil
}

// NetworkChanged tells the engine the device switched networks.
func NetworkChanged() {
	engineMu.Lock()
	e := running
	engineMu.Unlock()
	if e != nil {
		e.NetworkChanged()
	}
}

func parseEngineConfig(configJSON string) (engine.Config, error) {
	var raw engineConfig
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return engine.Config{}, fmt.Errorf("parse engine config: %w", err)
	}
	if raw.Mode == "" {
		raw.Mode = ModeGlobal
	}
	var rules []router.Rule
	var err error
	if raw.Mode == ModeCustom {
		rules, err = router.ParseRules(raw.Rules)
	} else {
		rules, err = router.ModeRules(router.Mode(raw.Mode))
	}
	if err != nil {
		return engine.Config{}, err
	}
	if raw.DoH == "" {
		raw.DoH = DefaultDoH(raw.Mode)
	}
	if raw.Bootstrap == "" {
		raw.Bootstrap = DefaultBootstrap
	}
	return engine.Config{
		Client: raw.Client, Rules: rules, DoHURL: raw.DoH, Bootstrap: raw.Bootstrap, MTU: raw.MTU,
	}, nil
}
