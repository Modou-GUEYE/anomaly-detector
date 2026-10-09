package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// ===================== TRANSPORT SMTP =====================

// SendEmail envoie un email HTML via SMTP (appel synchrone, borné par Timeout).
func (s *Service) SendEmail(to, subject, htmlBody string) error {
	if s.SMTPHost == "" || s.SMTPPort == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}
	msg, err := s.buildMessage(to, subject, htmlBody)
	if err != nil {
		return err
	}

	client, _, err := s.openSession()
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer client.Close()

	if err := client.Mail(s.FromEmail); err != nil {
		return fmt.Errorf("failed to send email: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to send email: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send email: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return client.Quit()
}

// ProbeResult décrit la négociation avec le serveur SMTP (utilisé par cmd/smtpcheck).
type ProbeResult struct {
	Address    string
	TLS        string
	Extensions map[string]string
	AuthMethod string
}

// Probe se connecte, négocie TLS et s'authentifie sans envoyer d'email.
func (s *Service) Probe() (*ProbeResult, error) {
	client, result, err := s.openSession()
	if err != nil {
		return result, err
	}
	defer client.Close()
	return result, client.Quit()
}

// openSession ouvre une session SMTP prête à envoyer : connexion, EHLO, TLS et authentification.
func (s *Service) openSession() (*smtp.Client, *ProbeResult, error) {
	addr := net.JoinHostPort(s.SMTPHost, s.SMTPPort)
	result := &ProbeResult{Address: addr, Extensions: map[string]string{}}
	deadline := time.Now().Add(s.timeout())

	tlsMode := s.tlsMode()
	dialer := &net.Dialer{Deadline: deadline}
	var conn net.Conn
	var err error
	if tlsMode == "implicit" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, s.clientTLSConfig())
	} else {
		conn, err = dialer.DialContext(context.Background(), "tcp", addr)
	}
	if err != nil {
		return nil, result, fmt.Errorf("connexion à %s impossible: %w", addr, err)
	}
	// La deadline couvre toute la session : un serveur muet ne bloque plus l'appelant.
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, result, err
	}

	client, err := smtp.NewClient(conn, s.SMTPHost)
	if err != nil {
		conn.Close()
		return nil, result, fmt.Errorf("accueil SMTP: %w", err)
	}
	fail := func(err error) (*smtp.Client, *ProbeResult, error) {
		client.Close()
		return nil, result, err
	}

	if err := client.Hello(s.heloName()); err != nil {
		return fail(fmt.Errorf("EHLO: %w", err))
	}

	switch tlsMode {
	case "implicit":
		result.TLS = "TLS direct"
	case "starttls":
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fail(errors.New("le serveur n'annonce pas STARTTLS (définir SMTP_TLS_MODE=none pour un relais interne non chiffré)"))
		}
		if err := client.StartTLS(s.clientTLSConfig()); err != nil {
			return fail(fmt.Errorf("STARTTLS: %w", err))
		}
		result.TLS = "STARTTLS"
	default:
		result.TLS = "aucun"
	}

	for _, ext := range []string{"AUTH", "SIZE", "8BITMIME", "SMTPUTF8"} {
		if ok, param := client.Extension(ext); ok {
			result.Extensions[ext] = param
		}
	}

	auth, name, err := s.chooseAuth(result.Extensions["AUTH"])
	if err != nil {
		return fail(err)
	}
	result.AuthMethod = name
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fail(fmt.Errorf("authentification %s refusée: %w", name, err))
		}
	}
	return client, result, nil
}

// chooseAuth sélectionne le mécanisme d'authentification à partir de la configuration et de
// la liste annoncée par le serveur (paramètre de l'extension AUTH, ex. "LOGIN XOAUTH2").
func (s *Service) chooseAuth(advertised string) (smtp.Auth, string, error) {
	offered := map[string]bool{}
	for _, m := range strings.Fields(strings.ToUpper(advertised)) {
		offered[m] = true
	}

	method := s.AuthMethod
	if method == "" || method == "auto" {
		switch {
		case s.OAuth.Enabled() && offered["XOAUTH2"]:
			method = "xoauth2"
		case s.SMTPUser == "" && !s.OAuth.Enabled():
			method = "none"
		case offered["PLAIN"]:
			method = "plain"
		case offered["LOGIN"]:
			method = "login"
		case len(offered) == 0:
			return nil, "", errors.New("le serveur n'annonce aucune méthode d'authentification (relais par IP ? définir SMTP_AUTH_METHOD=none)")
		default:
			return nil, "", fmt.Errorf("aucune méthode d'authentification compatible (annoncées par le serveur: %s)", advertised)
		}
	}

	switch method {
	case "none":
		return nil, "none", nil
	case "plain":
		return smtp.PlainAuth("", s.SMTPUser, s.SMTPPassword, s.SMTPHost), "PLAIN", nil
	case "login":
		return &loginAuth{username: s.SMTPUser, password: s.SMTPPassword}, "LOGIN", nil
	case "xoauth2":
		if !s.OAuth.Enabled() {
			return nil, "", errors.New("SMTP_AUTH_METHOD=xoauth2 mais SMTP_OAUTH_TENANT_ID / CLIENT_ID / CLIENT_SECRET ne sont pas renseignés")
		}
		token, err := s.OAuth.accessToken(s.timeout())
		if err != nil {
			return nil, "", fmt.Errorf("jeton OAuth: %w", err)
		}
		return &xoauth2Auth{username: s.oauthUser(), token: token}, "XOAUTH2", nil
	default:
		return nil, "", fmt.Errorf("SMTP_AUTH_METHOD inconnu: %q (auto | plain | login | xoauth2 | none)", method)
	}
}

// loginAuth implémente le mécanisme AUTH LOGIN, absent de net/smtp mais seul mécanisme par
// mot de passe accepté par Microsoft 365.
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		return "", nil, errors.New("AUTH LOGIN refusé sur une connexion non chiffrée")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch prompt := strings.ToLower(strings.TrimSpace(string(fromServer))); {
	case strings.HasPrefix(prompt, "username"):
		return []byte(a.username), nil
	case strings.HasPrefix(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("invite AUTH LOGIN inattendue: %q", fromServer)
	}
}

// xoauth2Auth implémente le mécanisme XOAUTH2 (jeton OAuth 2.0 Bearer).
type xoauth2Auth struct {
	username, token string
}

func (a *xoauth2Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		return "", nil, errors.New("AUTH XOAUTH2 refusé sur une connexion non chiffrée")
	}
	return "XOAUTH2", []byte("user=" + a.username + "\x01auth=Bearer " + a.token + "\x01\x01"), nil
}

func (a *xoauth2Auth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		// Le serveur renvoie le détail de l'erreur en challenge : une réponse vide termine
		// l'échange et fait remonter le code d'erreur final.
		return []byte{}, nil
	}
	return nil, nil
}

// buildMessage construit un message MIME conforme : en-têtes encodés (accents), Date,
// Message-ID et corps HTML en quoted-printable (lignes < 998 caractères, sans 8BITMIME).
func (s *Service) buildMessage(to, subject, htmlBody string) ([]byte, error) {
	from := (&mail.Address{Name: s.FromName, Address: s.FromEmail}).String()

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", s.messageID())
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}

func (s *Service) messageID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	domain := s.SMTPHost
	if at := strings.LastIndex(s.FromEmail, "@"); at >= 0 {
		domain = s.FromEmail[at+1:]
	}
	return "<" + hex.EncodeToString(b) + "@" + domain + ">"
}

func (s *Service) timeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return 15 * time.Second
}

func (s *Service) tlsMode() string {
	switch s.TLSMode {
	case "starttls", "implicit", "none":
		return s.TLSMode
	}
	if s.SMTPPort == "465" {
		return "implicit"
	}
	return "starttls"
}

func (s *Service) clientTLSConfig() *tls.Config {
	if s.tlsConfig != nil {
		return s.tlsConfig
	}
	return &tls.Config{ServerName: s.SMTPHost, MinVersion: tls.VersionTLS12}
}

func (s *Service) heloName() string {
	if s.HeloName != "" {
		return s.HeloName
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

func (s *Service) oauthUser() string {
	if s.SMTPUser != "" {
		return s.SMTPUser
	}
	return s.FromEmail
}
