package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strconv"
	"time"

	"json2proxy/internal/obfs"
)

type User struct {
	ClientID       string `json:"client_id"`
	Secret         string `json:"secret"`
	Status         string `json:"status"`
	MaxConnections int    `json:"max_connections"`
	AllowPrivate   bool   `json:"allow_private"`
}

type Server struct {
	Listen           string      `json:"listen"`
	HandshakeTimeout string      `json:"handshake_timeout"`
	DialTimeout      string      `json:"dial_timeout"`
	IdleTimeout      string      `json:"idle_timeout"`
	LogSize          string      `json:"log_size"`
	LogFile          string      `json:"log_file"`
	Obfs             obfs.Config `json:"obfs"`
	Outbound         Outbound    `json:"outbound,omitempty"`
	Users            []User      `json:"users"`
}

type Outbound struct {
	Type     string `json:"type"`
	Address  string `json:"address"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	DNS      string `json:"dns,omitempty"`
}

func (o Outbound) Enabled() bool {
	return o.Type != "" || o.Address != "" || o.Username != "" || o.Password != "" || o.DNS != ""
}

type Client struct {
	SocksListen      string      `json:"socks_listen"`
	Server           string      `json:"server"`
	ConnectTimeout   string      `json:"connect_timeout"`
	HandshakeTimeout string      `json:"handshake_timeout"`
	IdleTimeout      string      `json:"idle_timeout"`
	ClientID         string      `json:"client_id"`
	Secret           string      `json:"secret"`
	LogSize          string      `json:"log_size"`
	LogFile          string      `json:"log_file"`
	Obfs             obfs.Config `json:"obfs"`
}

func Load(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(v); err != nil {
		return fmt.Errorf("decode config %s: %w", path, err)
	}
	var extra any
	if err = dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode config %s: trailing data", path)
	}
	return nil
}

func (s *Server) Defaults() {
	if s.Listen == "" {
		s.Listen = "0.0.0.0:8443"
	}
	if s.HandshakeTimeout == "" {
		s.HandshakeTimeout = "10s"
	}
	if s.DialTimeout == "" {
		s.DialTimeout = "10s"
	}
	if s.IdleTimeout == "" {
		s.IdleTimeout = "5m"
	}
	if s.LogSize == "" {
		s.LogSize = "0M"
	}
	if s.LogFile == "" {
		s.LogFile = "logs/server.log"
	}
	if s.Obfs.Provider == "" {
		s.Obfs.Provider = "none"
	}
	if s.Outbound.Enabled() && s.Outbound.DNS == "" {
		s.Outbound.DNS = "local"
	}
	for i := range s.Users {
		if s.Users[i].MaxConnections <= 0 {
			s.Users[i].MaxConnections = 4096
		}
	}
}

func (c *Client) Defaults() {
	if c.SocksListen == "" {
		c.SocksListen = "127.0.0.1:8444"
	}
	if c.ConnectTimeout == "" {
		c.ConnectTimeout = "10s"
	}
	if c.HandshakeTimeout == "" {
		c.HandshakeTimeout = "10s"
	}
	if c.IdleTimeout == "" {
		c.IdleTimeout = "5m"
	}
	if c.LogSize == "" {
		c.LogSize = "0M"
	}
	if c.LogFile == "" {
		c.LogFile = "logs/client.log"
	}
	if c.Obfs.Provider == "" {
		c.Obfs.Provider = "none"
	}
}

func (s Server) Validate() error {
	if s.Listen == "" {
		return errors.New("listen is required")
	}
	if _, err := Duration(s.HandshakeTimeout); err != nil {
		return fmt.Errorf("handshake_timeout: %w", err)
	}
	if _, err := Duration(s.DialTimeout); err != nil {
		return fmt.Errorf("dial_timeout: %w", err)
	}
	if _, err := Duration(s.IdleTimeout); err != nil {
		return fmt.Errorf("idle_timeout: %w", err)
	}
	if err := validateObfs(s.Obfs); err != nil {
		return err
	}
	if err := validateOutbound(s.Outbound); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(s.Users))
	for _, user := range s.Users {
		if user.ClientID == "" {
			return errors.New("users[].client_id is required")
		}
		if _, ok := seen[user.ClientID]; ok {
			return fmt.Errorf("duplicate client_id %q", user.ClientID)
		}
		seen[user.ClientID] = struct{}{}
		if _, err := Secret(user.Secret); err != nil {
			return fmt.Errorf("user %q: %w", user.ClientID, err)
		}
	}
	return nil
}

func validateOutbound(value Outbound) error {
	if !value.Enabled() {
		return nil
	}
	if value.Type != "socks5" {
		return errors.New("outbound.type must be socks5")
	}
	host, port, err := net.SplitHostPort(value.Address)
	if err != nil || host == "" {
		return errors.New("outbound.address must be a valid host:port")
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return errors.New("outbound.address must contain a numeric port from 1 to 65535")
	}
	if value.DNS != "local" {
		return errors.New("outbound.dns must be local")
	}
	if value.Password != "" && value.Username == "" {
		return errors.New("outbound.username is required when password is configured")
	}
	if value.Username != "" && value.Password == "" {
		return errors.New("outbound.password is required when username is configured")
	}
	if len(value.Username) > 255 || len(value.Password) > 255 {
		return errors.New("outbound SOCKS5 username and password must not exceed 255 bytes")
	}
	return nil
}

func (c Client) Validate() error {
	if c.Server == "" {
		return errors.New("server is required")
	}
	if c.ClientID == "" {
		return errors.New("client_id is required")
	}
	if _, err := Secret(c.Secret); err != nil {
		return err
	}
	if err := validateObfs(c.Obfs); err != nil {
		return err
	}
	for name, value := range map[string]string{
		"connect_timeout": c.ConnectTimeout, "handshake_timeout": c.HandshakeTimeout,
		"idle_timeout": c.IdleTimeout,
	} {
		if _, err := Duration(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func validateObfs(value obfs.Config) error {
	if math.IsNaN(value.Participation) || math.IsInf(value.Participation, 0) ||
		value.Participation < 0 || value.Participation > 1 {
		return errors.New("obfs.participation must be a finite number between 0 and 1")
	}
	return nil
}

func Secret(s string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("secret must be unpadded base64url")
	}
	if len(b) < 32 {
		return nil, errors.New("secret must be at least 32 bytes")
	}
	return b, nil
}

func Duration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid positive duration %q", s)
	}
	return d, nil
}
