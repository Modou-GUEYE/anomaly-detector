package email

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeSMTP est un serveur SMTP minimal : STARTTLS puis AUTH avec les mécanismes annoncés.
// Il répond 504 5.7.4 à un mécanisme non annoncé, comme Microsoft 365.
type fakeSMTP struct {
	t         *testing.T
	ln        net.Listener
	mechs     string // ex. "LOGIN XOAUTH2" (Microsoft 365) ou "PLAIN LOGIN" (AWS SES)
	tlsConfig *tls.Config
	silent    atomic.Bool // n'envoie jamais le message d'accueil

	mu       sync.Mutex
	authUsed string
	authData string
	message  string
}

func newFakeSMTP(t *testing.T, mechs string) (*fakeSMTP, *tls.Config) {
	t.Helper()
	serverTLS, clientTLS := testTLSConfigs(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{t: t, ln: ln, mechs: mechs, tlsConfig: serverTLS}
	t.Cleanup(func() { ln.Close() })
	go f.serve()
	return f, clientTLS
}

func (f *fakeSMTP) service(clientTLS *tls.Config) *Service {
	host, port, _ := net.SplitHostPort(f.ln.Addr().String())
	return &Service{
		SMTPHost: host, SMTPPort: port,
		SMTPUser: "lab@partenaire.sn", SMTPPassword: "secret",
		FromEmail: "lab@partenaire.sn", FromName: "Moteur de détection d'anomalies - LAB",
		Timeout:   2 * time.Second,
		tlsConfig: clientTLS,
	}
}

func (f *fakeSMTP) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeSMTP) handle(conn net.Conn) {
	defer conn.Close()
	if f.silent.Load() {
		time.Sleep(5 * time.Second)
		return
	}
	tp := textproto.NewConn(conn)
	reply := func(format string, args ...interface{}) { _ = tp.PrintfLine(format, args...) }
	readLine := func() string { l, _ := tp.ReadLine(); return l }
	decode := func(s string) string { b, _ := base64.StdEncoding.DecodeString(s); return string(b) }

	reply("220 fake.smtp ESMTP")
	secure := false
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.Fields(line + " ")[0])
		switch cmd {
		case "EHLO":
			if !secure {
				reply("250-fake.smtp")
				reply("250 STARTTLS")
			} else {
				reply("250-fake.smtp")
				reply("250 AUTH %s", f.mechs)
			}
		case "STARTTLS":
			reply("220 2.0.0 Ready to start TLS")
			tlsConn := tls.Server(conn, f.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			tp = textproto.NewConn(tlsConn)
			secure = true
		case "AUTH":
			parts := strings.Fields(line)
			mech := strings.ToUpper(parts[1])
			if !strings.Contains(" "+f.mechs+" ", " "+mech+" ") {
				reply("504 5.7.4 Unrecognized authentication type")
				continue
			}
			var data string
			switch mech {
			case "PLAIN", "XOAUTH2":
				data = decode(parts[2])
			case "LOGIN":
				reply("334 %s", base64.StdEncoding.EncodeToString([]byte("Username:")))
				user := decode(readLine())
				reply("334 %s", base64.StdEncoding.EncodeToString([]byte("Password:")))
				data = user + ":" + decode(readLine())
			}
			f.mu.Lock()
			f.authUsed, f.authData = mech, data
			f.mu.Unlock()
			reply("235 2.7.0 Authentication successful")
		case "MAIL", "RCPT":
			reply("250 OK")
		case "DATA":
			reply("354 Start mail input")
			lines, _ := tp.ReadDotLines()
			f.mu.Lock()
			f.message = strings.Join(lines, "\n")
			f.mu.Unlock()
			reply("250 Queued")
		case "QUIT":
			reply("221 Bye")
			return
		default:
			reply("502 Command not implemented")
		}
	}
}

func (f *fakeSMTP) result() (string, string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authUsed, f.authData, f.message
}

// testTLSConfigs génère un certificat autosigné pour 127.0.0.1.
func testTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fake.smtp"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	server := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	client := &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
	return server, client
}

func TestSendEmailMicrosoft365UsesLogin(t *testing.T) {
	srv, clientTLS := newFakeSMTP(t, "LOGIN XOAUTH2")
	if err := srv.service(clientTLS).SendEmail("user@bis-bank.sn", "Création de votre compte", "<p>Bonjour é</p>"); err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	mech, data, msg := srv.result()
	if mech != "LOGIN" || data != "lab@partenaire.sn:secret" {
		t.Fatalf("auth = %s %q, attendu LOGIN avec les identifiants", mech, data)
	}
	for _, want := range []string{
		"Subject: =?utf-8?q?Cr=C3=A9ation_de_votre_compte?=",
		"From: =?utf-8?b?TW90ZXVyIGRlIGTDqXRlY3Rpb24gZCdhbm9tYWxpZXMgLSBMQUI=?= <lab@partenaire.sn>",
		"Message-ID: <", "Date: ", "Content-Transfer-Encoding: quoted-printable",
		"Bonjour =C3=A9",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message sans %q:\n%s", want, msg)
		}
	}
}

func TestSendEmailSESUsesPlain(t *testing.T) {
	srv, clientTLS := newFakeSMTP(t, "PLAIN LOGIN")
	if err := srv.service(clientTLS).SendEmail("user@gsie.sn", "Test", "<p>ok</p>"); err != nil {
		t.Fatalf("SendEmail: %v", err)
	}
	if mech, data, _ := srv.result(); mech != "PLAIN" || data != "\x00lab@partenaire.sn\x00secret" {
		t.Fatalf("auth = %s %q, attendu PLAIN", mech, data)
	}
}

func TestForcedPlainOnMicrosoft365ReproducesPartnerError(t *testing.T) {
	srv, clientTLS := newFakeSMTP(t, "LOGIN XOAUTH2")
	svc := srv.service(clientTLS)
	svc.AuthMethod = "plain" // ancien comportement (smtp.PlainAuth)
	err := svc.SendEmail("user@bis-bank.sn", "Test", "<p>ok</p>")
	if err == nil || !strings.Contains(err.Error(), "504 5.7.4 Unrecognized authentication type") {
		t.Fatalf("erreur attendue 504 5.7.4, obtenu %v", err)
	}
	if isTemporary(err) {
		t.Fatal("un refus d'authentification ne doit pas être retenté")
	}
}

func TestSendEmailXOAUTH2(t *testing.T) {
	var tokenCalls atomic.Int32
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls.Add(1)
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("scope") != defaultOAuthScope {
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, `{"access_token":"jeton-123","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	srv, clientTLS := newFakeSMTP(t, "LOGIN XOAUTH2")
	svc := srv.service(clientTLS)
	svc.OAuth = OAuthConfig{TenantID: "tenant-test-xoauth2", ClientID: "client", ClientSecret: "s", TokenURL: tokenSrv.URL}

	for i := 0; i < 2; i++ {
		if err := svc.SendEmail("user@bis-bank.sn", "Test", "<p>ok</p>"); err != nil {
			t.Fatalf("SendEmail: %v", err)
		}
	}
	mech, data, _ := srv.result()
	if mech != "XOAUTH2" || data != "user=lab@partenaire.sn\x01auth=Bearer jeton-123\x01\x01" {
		t.Fatalf("auth = %s %q, attendu XOAUTH2", mech, data)
	}
	if n := tokenCalls.Load(); n != 1 {
		t.Fatalf("jeton demandé %d fois, attendu 1 (cache)", n)
	}
}

func TestSendEmailTimeoutOnSilentServer(t *testing.T) {
	srv, clientTLS := newFakeSMTP(t, "LOGIN")
	srv.silent.Store(true)
	svc := srv.service(clientTLS)
	svc.Timeout = 300 * time.Millisecond

	start := time.Now()
	err := svc.SendEmail("user@bis-bank.sn", "Test", "<p>ok</p>")
	if err == nil {
		t.Fatal("erreur attendue sur serveur muet")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("envoi bloqué %s, le délai maximum n'est pas respecté", elapsed)
	}
	if !isTemporary(err) {
		t.Fatalf("un délai dépassé doit être retenté: %v", err)
	}
}

func TestChooseAuth(t *testing.T) {
	tests := []struct {
		name, method, advertised, user string
		oauth                          bool
		want                           string
		wantErr                        bool
	}{
		{"M365 auto", "", "LOGIN XOAUTH2", "u", false, "LOGIN", false},
		{"SES auto", "auto", "PLAIN LOGIN", "u", false, "PLAIN", false},
		{"OAuth prioritaire", "", "LOGIN XOAUTH2", "u", true, "XOAUTH2", false},
		{"relais sans identifiants", "", "", "", false, "none", false},
		{"forcé login", "login", "PLAIN", "u", false, "LOGIN", false},
		{"aucun mécanisme compatible", "", "CRAM-MD5", "u", false, "", true},
		{"aucun mécanisme annoncé", "", "", "u", false, "", true},
		{"xoauth2 sans config", "xoauth2", "XOAUTH2", "u", false, "", true},
		{"méthode inconnue", "ntlm", "LOGIN", "u", false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{AuthMethod: tt.method, SMTPUser: tt.user}
			if tt.oauth {
				// Jeton déjà en cache : aucun appel réseau.
				svc.OAuth = OAuthConfig{TenantID: "t-choose", ClientID: "c", ClientSecret: "s"}
				tokenMu.Lock()
				tokenCache["t-choose|c|"+defaultOAuthScope] = cachedToken{value: "x", expires: time.Now().Add(time.Hour)}
				tokenMu.Unlock()
			}
			_, got, err := svc.chooseAuth(tt.advertised)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("chooseAuth(%q) = %q, %v ; attendu %q (erreur: %v)", tt.advertised, got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestSendAsyncRetriesTemporaryErrors(t *testing.T) {
	retryDelay = time.Millisecond
	var temp, perm atomic.Int32
	done := make(chan struct{}, 2)

	SendAsync("temporaire", func() error {
		if temp.Add(1) < maxAttempts {
			return &textproto.Error{Code: 451, Msg: "4.4.1 try again"}
		}
		done <- struct{}{}
		return nil
	})
	SendAsync("définitif", func() error {
		perm.Add(1)
		done <- struct{}{}
		return &textproto.Error{Code: 504, Msg: "5.7.4 Unrecognized authentication type"}
	})

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("envoi asynchrone non exécuté")
		}
	}
	time.Sleep(20 * time.Millisecond)
	if temp.Load() != maxAttempts {
		t.Fatalf("erreur 4xx: %d tentatives, attendu %d", temp.Load(), maxAttempts)
	}
	if perm.Load() != 1 {
		t.Fatalf("erreur 5xx: %d tentatives, attendu 1", perm.Load())
	}
	if !isTemporary(&net.OpError{Op: "dial", Err: errors.New("refused")}) {
		t.Fatal("une erreur réseau doit être retentée")
	}
}
