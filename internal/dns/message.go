package dns

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	minTTL         = 30 * time.Second
	maxTTL         = time.Hour
	maxMessageSize = 65535
)

// NoAddressError reports a lookup that completed but returned no usable
// address, for example NXDOMAIN. It is not a transport failure.
type NoAddressError struct {
	Name  string
	RCode dnsmessage.RCode
}

func (e *NoAddressError) Error() string {
	if e.RCode == dnsmessage.RCodeSuccess {
		return fmt.Sprintf("no address for %s", e.Name)
	}
	return fmt.Sprintf("no address for %s: %s", e.Name, e.RCode)
}

type answer struct {
	addrs     []netip.Addr
	ttl       time.Duration
	rcode     dnsmessage.RCode
	truncated bool
}

// Normalize lowercases a host name and removes a trailing dot.
func Normalize(host string) string {
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

func newQuery(id uint16, name string, qtype dnsmessage.Type) ([]byte, error) {
	n, err := dnsmessage.NewName(Normalize(name) + ".")
	if err != nil {
		return nil, fmt.Errorf("invalid DNS name %q: %w", name, err)
	}
	msg := dnsmessage.Message{
		Header:    dnsmessage.Header{ID: id, RecursionDesired: true},
		Questions: []dnsmessage.Question{{Name: n, Type: qtype, Class: dnsmessage.ClassINET}},
	}
	return msg.Pack()
}

func parseAnswer(b []byte, id uint16, qtype dnsmessage.Type) (answer, error) {
	var p dnsmessage.Parser
	h, err := p.Start(b)
	if err != nil {
		return answer{}, fmt.Errorf("parse DNS response: %w", err)
	}
	if !h.Response || h.ID != id {
		return answer{}, errors.New("unexpected DNS response")
	}
	if err := p.SkipAllQuestions(); err != nil {
		return answer{}, fmt.Errorf("parse DNS response: %w", err)
	}
	a := answer{ttl: maxTTL, rcode: h.RCode, truncated: h.Truncated}
	for {
		rh, err := p.AnswerHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}
		if err != nil {
			return answer{}, fmt.Errorf("parse DNS answer: %w", err)
		}
		if ttl := time.Duration(rh.TTL) * time.Second; ttl < a.ttl {
			a.ttl = ttl
		}
		switch {
		case rh.Type == dnsmessage.TypeA && qtype == dnsmessage.TypeA:
			r, err := p.AResource()
			if err != nil {
				return answer{}, err
			}
			a.addrs = append(a.addrs, netip.AddrFrom4(r.A))
		case rh.Type == dnsmessage.TypeAAAA && qtype == dnsmessage.TypeAAAA:
			r, err := p.AAAAResource()
			if err != nil {
				return answer{}, err
			}
			a.addrs = append(a.addrs, netip.AddrFrom16(r.AAAA))
		default:
			if err := p.SkipAnswer(); err != nil {
				return answer{}, err
			}
		}
	}
	if len(a.addrs) == 0 {
		a.ttl = minTTL
	}
	a.ttl = clampTTL(a.ttl)
	return a, nil
}

func clampTTL(ttl time.Duration) time.Duration {
	return min(max(ttl, minTTL), maxTTL)
}
