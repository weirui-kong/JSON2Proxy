package dns

import (
	"net/netip"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const cacheLimit = 4096

type cacheKey struct {
	name  string
	qtype dnsmessage.Type
}

type cacheEntry struct {
	addrs   []netip.Addr
	rcode   dnsmessage.RCode
	expires time.Time
}

type cache struct {
	mu      sync.Mutex
	entries map[cacheKey]cacheEntry
	now     func() time.Time
}

func newCache() *cache {
	return &cache{entries: make(map[cacheKey]cacheEntry), now: time.Now}
}

func (c *cache) get(key cacheKey) (cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return cacheEntry{}, false
	}
	if !c.now().Before(e.expires) {
		delete(c.entries, key)
		return cacheEntry{}, false
	}
	return e, true
}

func (c *cache) put(key cacheKey, a answer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if len(c.entries) >= cacheLimit {
		for k, e := range c.entries {
			if !now.Before(e.expires) {
				delete(c.entries, k)
			}
		}
		// Still full of live entries: drop an arbitrary one.
		for k := range c.entries {
			if len(c.entries) < cacheLimit {
				break
			}
			delete(c.entries, k)
		}
	}
	c.entries[key] = cacheEntry{addrs: a.addrs, rcode: a.rcode, expires: now.Add(a.ttl)}
}

func (c *cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.entries)
}
