package protocol

import (
	"bufio"
	"bytes"
	"testing"
	"time"
)

func testSessions(t *testing.T) (*Session, *Session, Envelope, Envelope) {
	t.Helper()
	root := bytes.Repeat([]byte{7}, 32)
	providerIdentity := []byte("provider:test")
	now := time.Unix(1785300000, 0)
	ch, clientPrivate, err := NewClientHello(root, providerIdentity, "test-user", now)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateClientHello(ch, root, providerIdentity, now, time.Minute); err != nil {
		t.Fatal(err)
	}
	sh, serverPrivate, err := NewServerHello(root, providerIdentity, ch, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateServerHello(sh, ch, root, providerIdentity, now); err != nil {
		t.Fatal(err)
	}
	client, err := NewSession(root, providerIdentity, ch, sh, clientPrivate, "c2s")
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewSession(root, providerIdentity, ch, sh, serverPrivate, "s2c")
	if err != nil {
		t.Fatal(err)
	}
	return client, server, ch, sh
}

func TestHandshakeAndEncryptedFrameRoundTrip(t *testing.T) {
	client, server, _, _ := testSessions(t)
	if !server.VerifyConfirmation("client", client.Confirmation("client")) {
		t.Fatal("client key confirmation mismatch")
	}
	if !client.VerifyConfirmation("server", server.Confirmation("server")) {
		t.Fatal("server key confirmation mismatch")
	}
	env, err := client.Seal(Frame{Type: FrameData, Payload: []byte("secret payload")})
	if err != nil {
		t.Fatal(err)
	}
	if env.Seq == nil || *env.Seq != 0 || bytes.Contains([]byte(env.Data), []byte("secret")) {
		t.Fatal("bad encrypted envelope")
	}
	frame, err := server.Open(env)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Type != FrameData || string(frame.Payload) != "secret payload" {
		t.Fatalf("unexpected frame: %#v", frame)
	}
}

func TestSessionRejectsTamperingAndReplay(t *testing.T) {
	client, server, _, _ := testSessions(t)
	env, err := client.Seal(Frame{Type: FrameData, Payload: []byte("payload")})
	if err != nil {
		t.Fatal(err)
	}
	tampered := env
	raw, _ := UnB64(tampered.Data)
	raw[0] ^= 1
	tampered.Data = B64(raw)
	if _, err = server.Open(tampered); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err = server.Open(env); err != nil {
		t.Fatalf("valid frame rejected after failed authentication: %v", err)
	}
	if _, err = server.Open(env); err == nil {
		t.Fatal("replayed frame accepted")
	}
}

func TestProviderIdentityIsBoundToAuthentication(t *testing.T) {
	root := bytes.Repeat([]byte{9}, 32)
	now := time.Unix(1785300000, 0)
	iotIdentity := []byte("iot-cam:v1:650")
	mysqlIdentity := []byte("mysql:v1:650")
	hello, _, err := NewClientHello(root, iotIdentity, "test-user", now)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateClientHello(hello, root, mysqlIdentity, now, time.Minute); err == nil {
		t.Fatal("client hello authenticated with a different provider identity")
	}
}

func TestReadLineRejectsDuplicateKeysAndOversize(t *testing.T) {
	r := bufio.NewReader(bytes.NewBufferString(`{"v":4,"v":4,"kind":"frame"}` + "\n"))
	if _, err := ReadLine(r); err == nil {
		t.Fatal("duplicate key accepted")
	}
	tooLarge := bytes.Repeat([]byte{'x'}, MaxLine)
	if _, err := ReadLine(bufio.NewReader(bytes.NewReader(append(tooLarge, '\n')))); err == nil {
		t.Fatal("oversize line accepted")
	}
}

func TestReadLineAcceptsPresetError(t *testing.T) {
	line := `{"v":4,"kind":"error","code":"invalid_request","message":"invalid request"}` + "\n"
	env, err := ReadLine(bufio.NewReader(bytes.NewBufferString(line)))
	if err != nil {
		t.Fatal(err)
	}
	if env.Kind != "error" || env.Code != "invalid_request" || env.Message != "invalid request" {
		t.Fatalf("unexpected error envelope: %#v", env)
	}
}

func TestOpenPayloadAddressTypes(t *testing.T) {
	for _, host := range []string{"example.com", "192.0.2.1", "2001:db8::1"} {
		payload, err := OpenPayload(host, 443)
		if err != nil {
			t.Fatal(err)
		}
		gotHost, gotPort, err := ParseOpen(payload)
		if err != nil || gotHost != host || gotPort != 443 {
			t.Fatalf("%s => %s:%d (%v)", host, gotHost, gotPort, err)
		}
	}
}
