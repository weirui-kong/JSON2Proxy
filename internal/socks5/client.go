package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	version          = 5
	methodNoAuth     = 0
	methodUserPass   = 2
	methodNoAccept   = 0xff
	commandConnect   = 1
	addressIPv4      = 1
	addressDomain    = 3
	addressIPv6      = 4
	userPassVersion  = 1
	connectSucceeded = 0
)

type connectFunc func(context.Context, string, string) (net.Conn, error)

// Client establishes TCP connections through one SOCKS5 server. The returned
// connection is the original transport connection so TCP half-close remains
// available when the transport is a *net.TCPConn.
type Client struct {
	address  string
	username string
	password string
	connect  connectFunc
}

func New(address, username, password string) *Client {
	dialer := &net.Dialer{KeepAlive: 60 * time.Second}
	return &Client{
		address: address, username: username, password: password,
		connect: dialer.DialContext,
	}
}

func (c *Client) DialContext(ctx context.Context, network, target string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("SOCKS5 outbound only supports TCP")
	}
	conn, err := c.connect(ctx, "tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("connect SOCKS5 server: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	if deadline, ok := ctx.Deadline(); ok {
		if err = conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}
	watchDone := make(chan struct{})
	watchStopped := make(chan struct{})
	go func() {
		defer close(watchStopped)
		select {
		case <-ctx.Done():
			_ = conn.SetDeadline(time.Now())
		case <-watchDone:
		}
	}()
	err = c.handshake(conn, target)
	close(watchDone)
	<-watchStopped
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = conn.SetDeadline(time.Time{}); err != nil {
		return nil, err
	}
	success = true
	return conn, nil
}

func (c *Client) handshake(conn net.Conn, target string) error {
	if err := c.negotiate(conn); err != nil {
		return err
	}
	request, err := connectRequest(target)
	if err != nil {
		return err
	}
	if err = writeAll(conn, request); err != nil {
		return fmt.Errorf("write SOCKS5 CONNECT: %w", err)
	}
	return readConnectReply(conn)
}

func (c *Client) negotiate(conn net.Conn) error {
	method := byte(methodNoAuth)
	if c.username != "" {
		method = methodUserPass
	}
	if err := writeAll(conn, []byte{version, 1, method}); err != nil {
		return fmt.Errorf("write SOCKS5 greeting: %w", err)
	}
	var selection [2]byte
	if _, err := io.ReadFull(conn, selection[:]); err != nil {
		return fmt.Errorf("read SOCKS5 greeting: %w", err)
	}
	if selection[0] != version {
		return errors.New("invalid SOCKS5 greeting version")
	}
	if selection[1] == methodNoAccept {
		return errors.New("SOCKS5 server rejected all authentication methods")
	}
	if selection[1] != method {
		return errors.New("SOCKS5 server selected an unexpected authentication method")
	}
	if method == methodUserPass {
		if err := c.authenticate(conn); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) authenticate(conn net.Conn) error {
	if len(c.username) == 0 || len(c.username) > 255 || len(c.password) == 0 || len(c.password) > 255 {
		return errors.New("invalid SOCKS5 username or password length")
	}
	request := make([]byte, 0, 3+len(c.username)+len(c.password))
	request = append(request, userPassVersion, byte(len(c.username)))
	request = append(request, c.username...)
	request = append(request, byte(len(c.password)))
	request = append(request, c.password...)
	if err := writeAll(conn, request); err != nil {
		return fmt.Errorf("write SOCKS5 authentication: %w", err)
	}
	var response [2]byte
	if _, err := io.ReadFull(conn, response[:]); err != nil {
		return fmt.Errorf("read SOCKS5 authentication: %w", err)
	}
	if response[0] != userPassVersion || response[1] != 0 {
		return errors.New("SOCKS5 authentication failed")
	}
	return nil
}

func connectRequest(target string) ([]byte, error) {
	host, portText, err := net.SplitHostPort(target)
	if err != nil {
		return nil, errors.New("invalid SOCKS5 target address")
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return nil, errors.New("invalid SOCKS5 target port")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.New("SOCKS5 target must be a locally resolved IP address")
	}
	request := []byte{version, commandConnect, 0}
	if ipv4 := ip.To4(); ipv4 != nil {
		request = append(request, addressIPv4)
		request = append(request, ipv4...)
	} else {
		request = append(request, addressIPv6)
		request = append(request, ip.To16()...)
	}
	var encodedPort [2]byte
	binary.BigEndian.PutUint16(encodedPort[:], uint16(port))
	return append(request, encodedPort[:]...), nil
}

func readConnectReply(conn net.Conn) error {
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return fmt.Errorf("read SOCKS5 CONNECT reply: %w", err)
	}
	if header[0] != version || header[2] != 0 {
		return errors.New("invalid SOCKS5 CONNECT reply")
	}
	var addressLength int
	switch header[3] {
	case addressIPv4:
		addressLength = net.IPv4len
	case addressIPv6:
		addressLength = net.IPv6len
	case addressDomain:
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			return fmt.Errorf("read SOCKS5 bound address: %w", err)
		}
		addressLength = int(length[0])
	default:
		return errors.New("invalid SOCKS5 bound address type")
	}
	boundAddressAndPort := make([]byte, addressLength+2)
	if _, err := io.ReadFull(conn, boundAddressAndPort); err != nil {
		return fmt.Errorf("read SOCKS5 bound address: %w", err)
	}
	if header[1] != connectSucceeded {
		return fmt.Errorf("SOCKS5 CONNECT failed: %s", replyDescription(header[1]))
	}
	return nil
}

func replyDescription(code byte) string {
	switch code {
	case 1:
		return "general failure"
	case 2:
		return "connection not allowed"
	case 3:
		return "network unreachable"
	case 4:
		return "host unreachable"
	case 5:
		return "connection refused"
	case 6:
		return "TTL expired"
	case 7:
		return "command not supported"
	case 8:
		return "address type not supported"
	default:
		return fmt.Sprintf("reply code %d", code)
	}
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
		data = data[n:]
	}
	return nil
}
