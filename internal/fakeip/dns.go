package fakeip

import (
	"context"

	"golang.org/x/net/dns/dnsmessage"
)

// fakeTTL keeps system caches from holding a mapping the pool may evict.
const fakeTTL = 1

// ForwardFunc sends a raw query upstream (DoH) and returns the raw response.
type ForwardFunc func(ctx context.Context, query []byte) ([]byte, error)

// HandleQuery answers a raw DNS query: A gets a fake address, AAAA gets an
// empty answer, and everything else is forwarded. It returns nil when the
// query is too malformed to answer at all.
func (p *Pool) HandleQuery(ctx context.Context, query []byte, forward ForwardFunc) []byte {
	var parser dnsmessage.Parser
	h, err := parser.Start(query)
	if err != nil {
		return nil
	}
	questions, err := parser.AllQuestions()
	if err != nil || len(questions) != 1 {
		return reply(h, nil, dnsmessage.RCodeFormatError, nil)
	}
	q := questions[0]
	if q.Class == dnsmessage.ClassINET {
		switch q.Type {
		case dnsmessage.TypeA:
			ip := p.Lookup(q.Name.String())
			return reply(h, &q, dnsmessage.RCodeSuccess, func(b *dnsmessage.Builder) error {
				rh := dnsmessage.ResourceHeader{Name: q.Name, Class: dnsmessage.ClassINET, TTL: fakeTTL}
				return b.AResource(rh, dnsmessage.AResource{A: ip.As4()})
			})
		case dnsmessage.TypeAAAA:
			// No IPv6 fake addresses yet; an empty answer makes apps use IPv4.
			return reply(h, &q, dnsmessage.RCodeSuccess, nil)
		}
	}
	resp, err := forward(ctx, query)
	if err != nil {
		return reply(h, &q, dnsmessage.RCodeServerFailure, nil)
	}
	return resp
}

func reply(h dnsmessage.Header, q *dnsmessage.Question, rcode dnsmessage.RCode, answer func(*dnsmessage.Builder) error) []byte {
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:                 h.ID,
		Response:           true,
		OpCode:             h.OpCode,
		RecursionDesired:   h.RecursionDesired,
		RecursionAvailable: true,
		RCode:              rcode,
	})
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		return nil
	}
	if q != nil {
		if err := b.Question(*q); err != nil {
			return nil
		}
	}
	if answer != nil {
		if err := b.StartAnswers(); err != nil {
			return nil
		}
		if err := answer(&b); err != nil {
			return nil
		}
	}
	msg, err := b.Finish()
	if err != nil {
		return nil
	}
	return msg
}
