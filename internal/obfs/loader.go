package obfs

import (
	"embed"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
)

//go:embed providers/*.js
var builtins embed.FS

func Load(config Config) (*Provider, error) {
	name := config.Provider
	if name == "" {
		name = "none"
	}
	if math.IsNaN(config.Participation) || math.IsInf(config.Participation, 0) ||
		config.Participation < 0 || config.Participation > 1 {
		return nil, errors.New("obfs participation must be a finite number between 0 and 1")
	}
	var source []byte
	var err error
	if strings.HasPrefix(name, "file:") {
		path := strings.TrimPrefix(name, "file:")
		if path == "" {
			return nil, errors.New("custom obfs provider path is empty")
		}
		source, err = os.ReadFile(path)
	} else {
		source, err = builtins.ReadFile("providers/" + name + ".js")
	}
	if err != nil {
		return nil, fmt.Errorf("load obfs provider %q: %w", name, err)
	}
	object, err := parseStaticProvider(source)
	if err != nil {
		return nil, fmt.Errorf("parse obfs provider %q: %w", name, err)
	}
	provider, err := descriptorFromObject(object, source, config.Participation)
	if err != nil {
		return nil, fmt.Errorf("validate obfs provider %q: %w", name, err)
	}
	if !strings.HasPrefix(name, "file:") && provider.ID() != name {
		return nil, errors.New("built-in obfs provider id does not match filename")
	}
	// Only the embedded none provider preserves the protocol envelope
	// field-for-field, so it can use the allocation-light fast path.
	provider.identityWire = name == "none"
	return provider, nil
}
