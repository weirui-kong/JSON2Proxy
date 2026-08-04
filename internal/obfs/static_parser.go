package obfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	maxProviderBytes = 256 << 10
	maxASTNodes      = 8192
	maxASTDepth      = 16
)

type staticParser struct {
	source   []byte
	offset   int
	nodes    int
	deadline time.Time
}

func parseStaticProvider(source []byte) (map[string]any, error) {
	if len(source) > maxProviderBytes {
		return nil, errors.New("obfs provider exceeds 256 KiB")
	}
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	p := &staticParser{source: source, deadline: time.Now().Add(50 * time.Millisecond)}
	p.skipSpace()
	name, err := p.identifier()
	if err != nil || name != "defineObfsProvider" {
		return nil, errors.New("provider must contain exactly defineObfsProvider({...})")
	}
	if err = p.expect('('); err != nil {
		return nil, err
	}
	value, err := p.value(0)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("defineObfsProvider argument must be an object literal")
	}
	if err = p.expect(')'); err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.peek() == ';' {
		p.offset++
	}
	p.skipSpace()
	if p.offset != len(p.source) {
		return nil, errors.New("provider contains extra statements or executable syntax")
	}
	return object, nil
}

func (p *staticParser) value(depth int) (any, error) {
	if depth > maxASTDepth {
		return nil, errors.New("provider AST is too deep")
	}
	p.nodes++
	if p.nodes > maxASTNodes || time.Now().After(p.deadline) {
		return nil, errors.New("provider AST resource limit exceeded")
	}
	p.skipSpace()
	switch p.peek() {
	case '{':
		return p.object(depth + 1)
	case '[':
		return p.array(depth + 1)
	case '"':
		return p.string()
	case '-':
		p.offset++
		n, err := p.number()
		return -n, err
	default:
		if isDigit(p.peek()) {
			return p.number()
		}
		id, err := p.identifier()
		if err != nil {
			return nil, errors.New("provider values must be static literals")
		}
		switch id {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null":
			return nil, nil
		default:
			return nil, fmt.Errorf("identifier %q is not allowed as a value", id)
		}
	}
}

func (p *staticParser) object(depth int) (map[string]any, error) {
	if err := p.expect('{'); err != nil {
		return nil, err
	}
	result := map[string]any{}
	p.skipSpace()
	if p.peek() == '}' {
		p.offset++
		return result, nil
	}
	for {
		p.skipSpace()
		var key string
		var err error
		if p.peek() == '"' {
			key, err = p.string()
		} else {
			key, err = p.identifier()
		}
		if err != nil {
			return nil, errors.New("object key must be a static identifier or string")
		}
		if key == "__proto__" || key == "prototype" || key == "constructor" {
			return nil, fmt.Errorf("dangerous property %q is forbidden", key)
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate provider property %q", key)
		}
		if err = p.expect(':'); err != nil {
			return nil, err
		}
		result[key], err = p.value(depth)
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.offset++
			p.skipSpace()
			if p.peek() == '}' {
				return nil, errors.New("trailing commas are not allowed")
			}
		case '}':
			p.offset++
			return result, nil
		default:
			return nil, errors.New("expected ',' or '}' in provider object")
		}
	}
}

func (p *staticParser) array(depth int) ([]any, error) {
	if err := p.expect('['); err != nil {
		return nil, err
	}
	var result []any
	p.skipSpace()
	if p.peek() == ']' {
		p.offset++
		return result, nil
	}
	for {
		value, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.offset++
			p.skipSpace()
			if p.peek() == ']' {
				return nil, errors.New("trailing commas are not allowed")
			}
		case ']':
			p.offset++
			return result, nil
		default:
			return nil, errors.New("expected ',' or ']' in provider array")
		}
	}
}

func (p *staticParser) string() (string, error) {
	start := p.offset
	if p.peek() != '"' {
		return "", errors.New("expected string")
	}
	p.offset++
	for p.offset < len(p.source) {
		switch p.source[p.offset] {
		case '\\':
			p.offset += 2
		case '"':
			p.offset++
			var value string
			if err := json.Unmarshal(p.source[start:p.offset], &value); err != nil {
				return "", err
			}
			if len(value) > 4096 {
				return "", errors.New("provider string exceeds 4096 bytes")
			}
			return value, nil
		default:
			if p.source[p.offset] < 0x20 {
				return "", errors.New("control character in provider string")
			}
			p.offset++
		}
	}
	return "", errors.New("unterminated provider string")
}

func (p *staticParser) number() (float64, error) {
	start := p.offset
	for p.offset < len(p.source) && strings.ContainsRune("0123456789.eE+-", rune(p.source[p.offset])) {
		p.offset++
	}
	if start == p.offset {
		return 0, errors.New("expected finite number literal")
	}
	value, err := strconv.ParseFloat(string(p.source[start:p.offset]), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("invalid finite number literal")
	}
	return value, nil
}

func (p *staticParser) identifier() (string, error) {
	p.skipSpace()
	if p.offset >= len(p.source) || !(unicode.IsLetter(rune(p.source[p.offset])) || p.source[p.offset] == '_' || p.source[p.offset] == '$') {
		return "", errors.New("expected identifier")
	}
	start := p.offset
	p.offset++
	for p.offset < len(p.source) {
		r := rune(p.source[p.offset])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$' && r != '-' {
			break
		}
		p.offset++
	}
	return string(p.source[start:p.offset]), nil
}

func (p *staticParser) expect(want byte) error {
	p.skipSpace()
	if p.peek() != want {
		return fmt.Errorf("expected %q at provider byte %d", want, p.offset)
	}
	p.offset++
	return nil
}

func (p *staticParser) peek() byte {
	if p.offset >= len(p.source) {
		return 0
	}
	return p.source[p.offset]
}

func (p *staticParser) skipSpace() {
	for p.offset < len(p.source) {
		if strings.ContainsRune(" \t\r\n", rune(p.source[p.offset])) {
			p.offset++
			continue
		}
		if p.offset+1 < len(p.source) && p.source[p.offset] == '/' && p.source[p.offset+1] == '/' {
			p.offset += 2
			for p.offset < len(p.source) && p.source[p.offset] != '\n' {
				p.offset++
			}
			continue
		}
		if p.offset+1 < len(p.source) && p.source[p.offset] == '/' && p.source[p.offset+1] == '*' {
			end := bytes.Index(p.source[p.offset+2:], []byte("*/"))
			if end < 0 {
				p.offset = len(p.source)
				return
			}
			p.offset += end + 4
			continue
		}
		return
	}
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
