package email

import (
	"fmt"
	"net/smtp"
	"os"
)

// Service gère l'envoi d'emails
type Service struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// NewService crée une nouvelle instance du service email à partir des variables d'environnement
func NewService() *Service {
	return &Service{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("SMTP_FROM_EMAIL"),
		FromName:     os.Getenv("SMTP_FROM_NAME"),
	}
}

// SendSignupEmail envoie un email de bienvenue après inscription
func (s *Service) SendSignupEmail(toEmail, username, password, role string) error {
	subject := "Bienvenue sur Le Moteur de Détection d'Anomalies !"
	loginURL := os.Getenv("FRONTEND_LOGIN_URL")
	if loginURL == "" {
		loginURL = "http://localhost:4200/login"
	}
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2463DE;">Bienvenue, %s !</h2>
        <p>Votre compte a été créé avec succès sur <strong>Le Moteur de Détection d'Anomalies</strong>.</p>
        
        <div style="background: #f0f4ff; border-left: 4px solid #2463DE; padding: 15px; margin: 20px 0; border-radius: 4px;">
            <h3 style="margin-top: 0; color: #2463DE;">✉️ Vos identifiants de connexion</h3>
            <p><strong>Email :</strong> <code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
            <p><strong>Mot de passe :</strong> <code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
            <p><strong>Rôle :</strong> <code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
            <p><strong>Statut :</strong> En attente d'activation par un administrateur</p>
        </div>

        <div style="background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; border-radius: 4px;">
            <p style="margin: 0; color: #856404;">
                <strong>⚠️ Attention :</strong> Conservez vos identifiants en lieu sûr. Pour des raisons de sécurité, vous pouvez modifier votre mot de passe après activation.
            </p>
        </div>

        <p style="color: #666; margin: 20px 0;">
            <strong>⏳ Un administrateur activera votre compte dans les 24 heures.</strong><br>
            Une fois activé, vous recevrez un email de confirmation et pourrez accéder à votre espace.
        </p>

        <p style="margin-top: 30px;">
			<a href="%s" style="background-color: #2463DE; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">
                Accéder à la connexion
            </a>
        </p>

        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 30px 0;">
        <p style="color: #999; font-size: 12px;">
            Cet email a été envoyé automatiquement. Merci de ne pas répondre à ce message.
        </p>
    </div>
</body>
</html>
`, username, toEmail, password, role, loginURL)

	return s.SendEmail(toEmail, subject, body)
}

// SendAccountActivatedEmail envoie un email de confirmation d'activation
func (s *Service) SendAccountActivatedEmail(toEmail, username string) error {
	subject := "Votre compte a été activé ✓"
	loginURL := os.Getenv("FRONTEND_LOGIN_URL")
	if loginURL == "" {
		loginURL = "http://localhost:4200/login"
	}
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #10b981;">✓ Compte activé avec succès !</h2>
        <p>Bonjour %s,</p>
        
        <p>Votre compte a été <strong>activé par un administrateur</strong>. Vous pouvez maintenant accéder au <strong>Moteur de Détection d'Anomalies</strong>.</p>

        <div style="background: #f0fdf4; border-left: 4px solid #10b981; padding: 15px; margin: 20px 0; border-radius: 4px;">
            <h3 style="margin-top: 0; color: #10b981;">🚀 Prêt à commencer ?</h3>
            <p>Connectez-vous avec vos identifiants pour accéder à votre dashboard.</p>
        </div>

        <p style="margin-top: 30px;">
			<a href="%s" style="background-color: #10b981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">
                Accéder à ma session
            </a>
        </p>

        <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 30px 0;">
        <p style="color: #999; font-size: 12px;">
            Cet email a été envoyé automatiquement. Merci de ne pas répondre à ce message.
        </p>
    </div>
</body>
</html>
`, username, loginURL)

	return s.SendEmail(toEmail, subject, body)
}

// SendEmail envoie un email via SMTP
func (s *Service) SendEmail(to, subject, htmlBody string) error {
	// Vérifier la configuration
	if s.SMTPHost == "" || s.SMTPPort == "" {
		return fmt.Errorf("SMTP configuration is missing")
	}

	// Construire le message avec l'en-tête Content-Type pour HTML
	from := s.FromEmail
	if s.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.FromName, s.FromEmail)
	}

	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		from, to, subject, htmlBody))

	// Authentification SMTP
	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPassword, s.SMTPHost)

	// Envoyer l'email
	addr := fmt.Sprintf("%s:%s", s.SMTPHost, s.SMTPPort)
	err := smtp.SendMail(addr, auth, s.FromEmail, []string{to}, message)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
