package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	dohContentType    = "application/dns-message"
	defaultDoHTimeout = 5 * time.Second
)

// DoHConfig configures a DNS-over-HTTPS client (RFC 8484).
type DoHConfig struct {
	URL string
	// Bootstrap resolves the DoH host when URL does not use an IP address.
	Bootstrap *Bootstrap
	// Dial opens TCP connections to the DoH server. Nil uses net.Dialer.
	Dial DialFunc
	// TLSConfig overrides the TLS settings; tests use it to trust a local CA.
	TLSConfig *tls.Config
	Timeout   time.Duration
	// OnHealthChange is called when the server switches between available
	// and unavailable. It is not called for every failed request.
	OnHealthChange func(available bool, err error)
}

// DoH resolves names through one DNS-over-HTTPS server.
type DoH struct {
	url     string
	client  *http.Client
	timeout time.Duration
	cache   *cache
	health  health
}

func NewDoH(cfg DoHConfig) (*DoH, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, fmt.Errorf("DoH URL must be an https URL: %q", cfg.URL)
	}
	if _, err := netip.ParseAddr(u.Hostname()); err != nil && cfg.Bootstrap == nil {
		return nil, fmt.Errorf("DoH host %q needs a bootstrap resolver", u.Hostname())
	}
	dial := cfg.Dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultDoHTimeout
	}
	transport := &http.Transport{
		DialContext:         bootstrapDialer(cfg.Bootstrap, dial),
		ForceAttemptHTTP2:   true,
		TLSClientConfig:     cfg.TLSConfig,
		TLSHandshakeTimeout: timeout,
		MaxIdleConns:        4,
		IdleConnTimeout:     90 * time.Second,
	}
	return &DoH{
		url:     u.String(),
		client:  &http.Client{Transport: transport},
		timeout: timeout,
		cache:   newCache(),
		health:  health{available: true, notify: cfg.OnHealthChange},
	}, nil
}

// LookupIP resolves host through DoH, returning IPv4 addresses first.
func (d *DoH) LookupIP(ctx context.Context, host string) ([]netip.Addr, error) {
	return lookupIP(ctx, d.cache, host, d.exchangeType)
}

// Exchange sends a raw DNS query and returns the raw response. The query ID
// is preserved in the response.
func (d *DoH) Exchange(ctx context.Context, query []byte) ([]byte, error) {
	if len(query) < 12 {
		return nil, errors.New("DNS query too short")
	}
	id := [2]byte{query[0], query[1]}
	// RFC 8484 recommends ID 0 so responses are cacheable.
	wire := append([]byte{0, 0}, query[2:]...)
	resp, err := d.post(ctx, wire)
	if err != nil {
		return nil, err
	}
	if len(resp) < 2 {
		return nil, errors.New("DNS response too short")
	}
	resp[0], resp[1] = id[0], id[1]
	return resp, nil
}

// ClearCache drops cached answers, for example after a network change.
func (d *DoH) ClearCache() { d.cache.clear() }

// Available reports the last known health of the server.
func (d *DoH) Available() bool { return d.health.get() }

func (d *DoH) exchangeType(ctx context.Context, name string, qtype dnsmessage.Type) (answer, error) {
	query, err := newQuery(0, name, qtype)
	if err != nil {
		return answer{}, err
	}
	resp, err := d.post(ctx, query)
	if err != nil {
		return answer{}, err
	}
	return parseAnswer(resp, 0, qtype)
}

func (d *DoH) post(ctx context.Context, query []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	resp, err := d.roundTrip(ctx, query)
	// A caller giving up is not evidence about the server's health.
	if err == nil || ctx.Err() == nil || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		d.health.report(err)
	}
	return resp, err
}

func (d *DoH) roundTrip(ctx context.Context, query []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", dohContentType)
	req.Header.Set("Accept", dohContentType)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH server returned HTTP %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != dohContentType {
		return nil, fmt.Errorf("DoH server returned content type %q", ct)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMessageSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxMessageSize {
		return nil, errors.New("DoH response too large")
	}
	var p dnsmessage.Parser
	if _, err := p.Start(body); err != nil {
		return nil, fmt.Errorf("invalid DoH response: %w", err)
	}
	return body, nil
}

// bootstrapDialer resolves the DoH host with the bootstrap resolver and tries
// each address in turn.
func bootstrapDialer(b *Bootstrap, dial DialFunc) DialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if _, err := netip.ParseAddr(host); err == nil {
			return dial(ctx, network, address)
		}
		addrs, err := b.LookupIP(ctx, host)
		if err != nil {
			return nil, err
		}
		var errs []error
		for _, addr := range addrs {
			conn, err := dial(ctx, network, net.JoinHostPort(addr.String(), port))
			if err == nil {
				return conn, nil
			}
			errs = append(errs, err)
		}
		return nil, errors.Join(errs...)
	}
}

type health struct {
	mu        sync.Mutex
	available bool
	notify    func(bool, error)
}

func (h *health) report(err error) {
	h.mu.Lock()
	available := err == nil
	changed := available != h.available
	h.available = available
	h.mu.Unlock()
	if changed && h.notify != nil {
		h.notify(available, err)
	}
}

func (h *health) get() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.available
}
