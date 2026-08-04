package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"json2proxy/internal/config"
	"json2proxy/internal/flow"
	"json2proxy/internal/logging"
	"json2proxy/internal/obfs"
	"json2proxy/internal/protocol"
)

type client struct {
	cfg              config.Client
	root             []byte
	connectTimeout   time.Duration
	handshakeTimeout time.Duration
	idleTimeout      time.Duration
	log              *logging.Logger
	provider         *obfs.Provider
}

type frameWriter struct {
	wire    *obfs.Conn
	session *protocol.Session
	timeout time.Duration
	mu      sync.Mutex
}

func (w *frameWriter) envelope(env protocol.Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.wire.SetWriteDeadline(time.Now().Add(w.timeout))
	err := w.wire.WriteEnvelope(env)
	_ = w.wire.SetWriteDeadline(time.Time{})
	return err
}

func (w *frameWriter) frame(frame protocol.Frame) error {
	env, err := w.session.Seal(frame)
	if err != nil {
		return err
	}
	return w.envelope(env)
}

func main() {
	configPath := flag.String("config", ".private/configs/client.json", "client config")
	flag.Parse()

	var cfg config.Client
	must(config.Load(*configPath, &cfg))
	cfg.Defaults()
	must(cfg.Validate())
	root, err := config.Secret(cfg.Secret)
	must(err)
	provider, err := obfs.Load(cfg.Obfs)
	must(err)
	logger, err := logging.New(cfg.LogFile, cfg.LogSize)
	must(err)
	defer logger.Close()

	c := &client{cfg: cfg, root: root, log: logger, provider: provider}
	c.connectTimeout, _ = config.Duration(cfg.ConnectTimeout)
	c.handshakeTimeout, _ = config.Duration(cfg.HandshakeTimeout)
	c.idleTimeout, _ = config.Duration(cfg.IdleTimeout)

	listener, err := net.Listen("tcp", cfg.SocksListen)
	must(err)
	defer listener.Close()
	logger.Printf("client starting product=JSON2Proxy protocol=v%d socks_listen=%s server=%s log_size=%s obfs=%s obfs_version=%s participation=%.3f obfs_hash=%s",
		protocol.Version, cfg.SocksListen, cfg.Server, cfg.LogSize, provider.ID(),
		provider.Descriptor.Version, provider.Participation, provider.Hash())
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			logger.Printf("socks accept failed error=%q", err)
			continue
		}
		go c.serveSOCKS(conn)
	}
}

func (c *client) serveSOCKS(local net.Conn) {
	defer local.Close()
	started := time.Now()
	source := local.RemoteAddr().String()
	reader := bufio.NewReader(local)
	if err := socksHello(reader, local); err != nil {
		c.log.Printf("socks closed source=%s reason=handshake_failed", source)
		return
	}
	host, port, err := socksRequest(reader)
	if err != nil {
		socksReply(local, 1)
		c.log.Printf("socks closed source=%s reason=request_failed", source)
		return
	}
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))
	up, down, sessionID, err := c.tunnel(local, reader, host, port)
	if err != nil {
		c.log.Printf("socks closed source=%s target=%s session=%s bytes_up=%d bytes_down=%d duration=%s reason=tunnel_failed error=%q",
			source, target, sessionID, up, down, time.Since(started).Round(time.Millisecond), err)
		return
	}
	c.log.Printf("socks closed source=%s target=%s session=%s bytes_up=%d bytes_down=%d duration=%s reason=completed",
		source, target, sessionID, up, down, time.Since(started).Round(time.Millisecond))
}

func socksHello(reader *bufio.Reader, writer io.Writer) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	if header[0] != 5 || header[1] == 0 {
		return errors.New("invalid SOCKS5 greeting")
	}
	methods := make([]byte, int(header[1]))
	if _, err := io.ReadFull(reader, methods); err != nil {
		return err
	}
	for _, method := range methods {
		if method == 0 {
			_, err := writer.Write([]byte{5, 0})
			return err
		}
	}
	_, _ = writer.Write([]byte{5, 0xff})
	return errors.New("SOCKS5 client did not offer no-auth")
}

func socksRequest(reader *bufio.Reader) (string, uint16, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return "", 0, err
	}
	if header[0] != 5 || header[1] != 1 || header[2] != 0 {
		return "", 0, errors.New("unsupported SOCKS5 request")
	}
	var host string
	switch header[3] {
	case 1:
		b := make([]byte, 4)
		if _, err := io.ReadFull(reader, b); err != nil {
			return "", 0, err
		}
		host = net.IP(b).String()
	case 3:
		n, err := reader.ReadByte()
		if err != nil || n == 0 {
			return "", 0, errors.New("invalid SOCKS5 domain")
		}
		b := make([]byte, int(n))
		if _, err = io.ReadFull(reader, b); err != nil {
			return "", 0, err
		}
		host = string(b)
	case 4:
		b := make([]byte, 16)
		if _, err := io.ReadFull(reader, b); err != nil {
			return "", 0, err
		}
		host = net.IP(b).String()
	default:
		return "", 0, errors.New("unsupported SOCKS5 address type")
	}
	p := make([]byte, 2)
	if _, err := io.ReadFull(reader, p); err != nil {
		return "", 0, err
	}
	port := uint16(p[0])<<8 | uint16(p[1])
	if port == 0 {
		return "", 0, errors.New("invalid SOCKS5 port")
	}
	return host, port, nil
}

func socksReply(writer io.Writer, code byte) {
	_, _ = writer.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
}

func (c *client) tunnel(local net.Conn, localReader *bufio.Reader, host string, port uint16) (uint64, uint64, string, error) {
	dialer := net.Dialer{Timeout: c.connectTimeout, KeepAlive: 60 * time.Second}
	tunnel, err := dialer.Dial("tcp", c.cfg.Server)
	if err != nil {
		socksReply(local, 1)
		return 0, 0, "", err
	}
	defer tunnel.Close()
	if tcp, ok := tunnel.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(60 * time.Second)
		_ = tcp.SetNoDelay(true)
	}
	_ = tunnel.SetDeadline(time.Now().Add(c.handshakeTimeout))
	wire, err := c.provider.NewConn(tunnel, "c2s")
	if err != nil {
		socksReply(local, 1)
		return 0, 0, "", err
	}
	defer wire.Stop()
	writer := &frameWriter{wire: wire, timeout: 30 * time.Second}

	clientHello, clientPrivate, err := protocol.NewClientHello(
		c.root, c.provider.Identity(), c.cfg.ClientID, time.Now())
	if err != nil {
		socksReply(local, 1)
		return 0, 0, "", err
	}
	if err = writer.envelope(clientHello); err != nil {
		socksReply(local, 1)
		return 0, 0, "", err
	}
	serverHello, err := wire.ReadEnvelope()
	wire.SetPhase("handshake")
	if err == nil && serverHello.Kind == "error" {
		socksReply(local, 1)
		return 0, 0, "", remoteError(serverHello)
	}
	if err != nil || protocol.ValidateServerHello(
		serverHello, clientHello, c.root, c.provider.Identity(), time.Now()) != nil {
		socksReply(local, 1)
		return 0, 0, "", errors.New("server authentication failed")
	}
	session, err := protocol.NewSession(
		c.root, c.provider.Identity(), clientHello, serverHello, clientPrivate, "c2s")
	if err != nil {
		socksReply(local, 1)
		return 0, 0, "", err
	}
	writer.session = session
	sessionID := session.ID()
	if err = writer.frame(protocol.Frame{Type: protocol.FrameKeyConfirm, Payload: session.Confirmation("client")}); err != nil {
		socksReply(local, 1)
		return 0, 0, sessionID, err
	}
	frame, err := readFrame(wire, session)
	if err != nil {
		socksReply(local, 1)
		return 0, 0, sessionID, err
	}
	if frame.Type != protocol.FrameKeyConfirm || !session.VerifyConfirmation("server", frame.Payload) {
		socksReply(local, 1)
		return 0, 0, sessionID, errors.New("server key confirmation failed")
	}
	wire.SetPhase("established")
	wire.StartScheduler()
	open, err := protocol.OpenPayload(host, port)
	if err != nil {
		socksReply(local, 8)
		return 0, 0, sessionID, err
	}
	if err = writer.frame(protocol.Frame{Type: protocol.FrameOpen, Payload: open}); err != nil {
		socksReply(local, 1)
		return 0, 0, sessionID, err
	}
	frame, err = readFrame(wire, session)
	if err != nil {
		socksReply(local, 1)
		return 0, 0, sessionID, err
	}
	if frame.Type != protocol.FrameOpenResult {
		socksReply(local, 1)
		return 0, 0, sessionID, errors.New("invalid OPEN_RESULT")
	}
	code, message, err := protocol.ParseResult(frame.Payload)
	if err != nil || code != 0 {
		socksReply(local, mapResult(code))
		return 0, 0, sessionID, fmt.Errorf("target refused: %s", message)
	}
	socksReply(local, 0)
	_ = tunnel.SetDeadline(time.Time{})

	var bytesUp, bytesDown atomic.Uint64
	if buffered := localReader.Buffered(); buffered > 0 {
		payload := make([]byte, buffered)
		if _, err = io.ReadFull(localReader, payload); err != nil {
			return 0, 0, sessionID, err
		}
		if err = writer.frame(protocol.Frame{Type: protocol.FrameData, Payload: payload}); err != nil {
			return 0, 0, sessionID, err
		}
		bytesUp.Add(uint64(len(payload)))
	}
	downDone := make(chan error, 1)
	go func() {
		serverClosed := false
		finish := func(err error) {
			if err != nil {
				_ = local.Close()
			}
			downDone <- err
		}
		for {
			_ = tunnel.SetReadDeadline(time.Now().Add(c.idleTimeout))
			frame, err := readFrame(wire, session)
			if err != nil {
				if serverClosed && flow.IsNormalClose(err) {
					err = nil
				}
				finish(err)
				return
			}
			switch frame.Type {
			case protocol.FrameData:
				if serverClosed {
					finish(errors.New("server sent data after CLOSE_WRITE"))
					return
				}
				if err = writeConn(local, frame.Payload, 30*time.Second); err != nil {
					finish(err)
					return
				}
				bytesDown.Add(uint64(len(frame.Payload)))
			case protocol.FrameCloseWrite:
				serverClosed = true
				closeWrite(local)
			case protocol.FramePing:
				if err = writer.frame(protocol.Frame{Type: protocol.FramePong, Payload: frame.Payload}); err != nil {
					finish(err)
					return
				}
			case protocol.FrameReset:
				finish(errors.New("server reset"))
				return
			default:
				finish(errors.New("invalid server frame"))
				return
			}
		}
	}()

	adaptive := flow.NewAdaptiveReader()
	for {
		payload, readErr := adaptive.ReadFrame(local)
		if len(payload) != 0 {
			started := time.Now()
			sendErr := writer.frame(protocol.Frame{Type: protocol.FrameData, Payload: payload})
			adaptive.Observe(len(payload), time.Since(started), sendErr)
			bytesUp.Add(uint64(len(payload)))
			flow.ReleaseFrame(payload)
			if sendErr != nil {
				return bytesUp.Load(), bytesDown.Load(), sessionID, sendErr
			}
		}
		if readErr != nil {
			if err = writer.frame(protocol.Frame{Type: protocol.FrameCloseWrite}); err != nil {
				return bytesUp.Load(), bytesDown.Load(), sessionID, err
			}
			closeWrite(tunnel)
			if !flow.IsNormalClose(readErr) {
				return bytesUp.Load(), bytesDown.Load(), sessionID, readErr
			}
			break
		}
	}
	err = <-downDone
	return bytesUp.Load(), bytesDown.Load(), sessionID, err
}

func readFrame(wire *obfs.Conn, session *protocol.Session) (protocol.Frame, error) {
	env, err := wire.ReadEnvelope()
	if err != nil {
		return protocol.Frame{}, err
	}
	if env.Kind == "error" {
		return protocol.Frame{}, remoteError(env)
	}
	return session.Open(env)
}

func remoteError(env protocol.Envelope) error {
	if env.Code == "" || env.Message == "" {
		return errors.New("server rejected request")
	}
	return fmt.Errorf("server rejected request: %s: %s", env.Code, env.Message)
}

func writeConn(conn net.Conn, b []byte, timeout time.Duration) error {
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	defer conn.SetWriteDeadline(time.Time{})
	for len(b) != 0 {
		n, err := conn.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func closeWrite(conn net.Conn) {
	if c, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = c.CloseWrite()
	} else {
		_ = conn.Close()
	}
}

func mapResult(code byte) byte {
	switch code {
	case 1:
		return 2
	case 2:
		return 4
	case 3:
		return 5
	default:
		return 1
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
