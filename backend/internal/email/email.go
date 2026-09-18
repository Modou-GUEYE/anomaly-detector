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

// SendUserProvisioningEmail envoie un email de bienvenue après création du compte par un administrateur
func (s *Service) SendUserProvisioningEmail(toEmail, username, password, role string) error {
	subject := "Création de votre compte — Moteur de Détection d'Anomalies"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Création de votre compte</title>
</head>
<body style="margin: 0; padding: 24px 0; background-color: #f1f5f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; color: #334155; line-height: 1.6;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
        <tr>
            <td align="center" style="padding: 0 16px;">
                <table role="presentation" width="100%%" style="max-width: 580px; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden;" cellspacing="0" cellpadding="0" border="0">
                    <!-- En-tête institutionnel -->
                    <tr>
                        <td style="padding: 24px 32px 20px; border-bottom: 1px solid #edf2f7; background-color: #ffffff;">
                            <div style="font-size: 13px; font-weight: 700; color: #1e3a8a; letter-spacing: 0.5px; text-transform: uppercase;">
                                Moteur de Détection d'Anomalies
                            </div>
                            <div style="font-size: 11px; color: #64748b; margin-top: 2px;">
                                Plateforme de surveillance et conformité
                            </div>
                        </td>
                    </tr>
                    <!-- Corps du message -->
                    <tr>
                        <td style="padding: 32px 32px 28px;">
                            <h1 style="margin: 0 0 16px; font-size: 18px; font-weight: 600; color: #0f172a;">
                                Bienvenue, %s
                            </h1>
                            <p style="margin: 0 0 24px; font-size: 14px; color: #475569;">
                                Votre compte utilisateur a été créé sur la plateforme <strong>Moteur de Détection d'Anomalies</strong>. Vous trouverez ci-dessous vos paramètres d'accès provisoires :
                            </p>

                            <!-- Tableau récapitulatif des identifiants -->
                            <table role="presentation" width="100%%" style="background-color: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; margin: 0 0 24px; font-size: 13px;" cellspacing="0" cellpadding="10" border="0">
                                <tr>
                                    <td width="35%%" style="padding: 10px 14px; color: #64748b; border-bottom: 1px solid #e2e8f0; font-weight: 500;">Identifiant (Email)</td>
                                    <td style="padding: 10px 14px; color: #0f172a; border-bottom: 1px solid #e2e8f0; font-weight: 600;">%s</td>
                                </tr>
                                <tr>
                                    <td style="padding: 10px 14px; color: #64748b; border-bottom: 1px solid #e2e8f0; font-weight: 500;">Mot de passe temporaire</td>
                                    <td style="padding: 10px 14px; color: #0f172a; border-bottom: 1px solid #e2e8f0; font-family: Consolas, Monaco, 'Courier New', monospace; font-size: 13px; font-weight: 600;">%s</td>
                                </tr>
                                <tr>
                                    <td style="padding: 10px 14px; color: #64748b; border-bottom: 1px solid #e2e8f0; font-weight: 500;">Profil attribué</td>
                                    <td style="padding: 10px 14px; color: #0f172a; border-bottom: 1px solid #e2e8f0;">%s</td>
                                </tr>
                                <tr>
                                    <td style="padding: 10px 14px; color: #64748b; font-weight: 500;">État du compte</td>
                                    <td style="padding: 10px 14px; color: #0f172a;">En attente de validation administrateur</td>
                                </tr>
                            </table>

                            <p style="margin: 0 0 16px; font-size: 13px; color: #64748b;">
                                Un administrateur procédera à l'activation de vos accès. Dès validation, un email de confirmation vous sera transmis afin de vous permettre de vous connecter.
                            </p>
                            <p style="margin: 0; font-size: 12px; color: #64748b;">
                                Par mesure de sécurité, nous vous recommandons de modifier ce mot de passe temporaire dès votre première connexion.
                            </p>
                        </td>
                    </tr>
                    <!-- Pied de page standard -->
                    <tr>
                        <td style="padding: 20px 32px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
                            Ce message est généré automatiquement par la plateforme. Merci de ne pas y répondre.
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`, username, toEmail, password, role)

	return s.SendEmail(toEmail, subject, body)
}

// SendAccountActivatedEmail envoie un email de confirmation d'activation
func (s *Service) SendAccountActivatedEmail(toEmail, username string) error {
	subject := "Activation de votre compte — Moteur de Détection d'Anomalies"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Compte activé</title>
</head>
<body style="margin: 0; padding: 24px 0; background-color: #f1f5f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; color: #334155; line-height: 1.6;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
        <tr>
            <td align="center" style="padding: 0 16px;">
                <table role="presentation" width="100%%" style="max-width: 580px; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden;" cellspacing="0" cellpadding="0" border="0">
                    <!-- En-tête institutionnel -->
                    <tr>
                        <td style="padding: 24px 32px 20px; border-bottom: 1px solid #edf2f7; background-color: #ffffff;">
                            <div style="font-size: 13px; font-weight: 700; color: #1e3a8a; letter-spacing: 0.5px; text-transform: uppercase;">
                                Moteur de Détection d'Anomalies
                            </div>
                            <div style="font-size: 11px; color: #64748b; margin-top: 2px;">
                                Plateforme de surveillance et conformité
                            </div>
                        </td>
                    </tr>
                    <!-- Corps du message -->
                    <tr>
                        <td style="padding: 32px 32px 28px;">
                            <h1 style="margin: 0 0 16px; font-size: 18px; font-weight: 600; color: #0f172a;">
                                Votre compte est désormais actif
                            </h1>
                            <p style="margin: 0 0 16px; font-size: 14px; color: #475569;">
                                Bonjour %s,
                            </p>
                            <p style="margin: 0 0 20px; font-size: 14px; color: #475569;">
                                Votre accès à la plateforme <strong>Moteur de Détection d'Anomalies</strong> a été validé et activé par un administrateur.
                            </p>
                            <p style="margin: 0 0 8px; font-size: 14px; color: #475569;">
                                Vous pouvez à présent vous connecter à l'application à l'aide de vos identifiants de connexion pour accéder à votre espace de travail.
                            </p>
                        </td>
                    </tr>
                    <!-- Pied de page standard -->
                    <tr>
                        <td style="padding: 20px 32px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
                            Ce message est généré automatiquement par la plateforme. Merci de ne pas y répondre.
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`, username)

	return s.SendEmail(toEmail, subject, body)
}

// SendPasswordResetEmail envoie un lien de réinitialisation de mot de passe.
func (s *Service) SendPasswordResetEmail(toEmail, username, resetLink string) error {
	subject := "Réinitialisation de votre mot de passe — Moteur de Détection d'Anomalies"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Réinitialisation du mot de passe</title>
</head>
<body style="margin: 0; padding: 24px 0; background-color: #f1f5f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; color: #334155; line-height: 1.6;">
    <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
        <tr>
            <td align="center" style="padding: 0 16px;">
                <table role="presentation" width="100%%" style="max-width: 580px; background-color: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden;" cellspacing="0" cellpadding="0" border="0">
                    <!-- En-tête institutionnel -->
                    <tr>
                        <td style="padding: 24px 32px 20px; border-bottom: 1px solid #edf2f7; background-color: #ffffff;">
                            <div style="font-size: 13px; font-weight: 700; color: #1e3a8a; letter-spacing: 0.5px; text-transform: uppercase;">
                                Moteur de Détection d'Anomalies
                            </div>
                            <div style="font-size: 11px; color: #64748b; margin-top: 2px;">
                                Plateforme de surveillance et conformité
                            </div>
                        </td>
                    </tr>
                    <!-- Corps du message -->
                    <tr>
                        <td style="padding: 32px 32px 28px;">
                            <h1 style="margin: 0 0 16px; font-size: 18px; font-weight: 600; color: #0f172a;">
                                Réinitialisation de mot de passe
                            </h1>
                            <p style="margin: 0 0 12px; font-size: 14px; color: #475569;">
                                Bonjour %s,
                            </p>
                            <p style="margin: 0 0 24px; font-size: 14px; color: #475569;">
                                Une demande de réinitialisation de mot de passe a été initiée pour votre compte. Cliquez sur le bouton ci-dessous pour choisir un nouveau mot de passe :
                            </p>

                            <!-- Bouton Call to Action sobre -->
                            <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin: 0 0 24px;">
                                <tr>
                                    <td align="left" style="border-radius: 6px; background-color: #1e3a8a;">
                                        <a href="%s" target="_blank" style="display: inline-block; padding: 11px 24px; font-size: 13px; font-weight: 600; color: #ffffff; text-decoration: none; border-radius: 6px; letter-spacing: 0.2px;">
                                            Définir un nouveau mot de passe
                                        </a>
                                    </td>
                                </tr>
                            </table>

                            <p style="margin: 0 0 12px; font-size: 12px; color: #64748b;">
                                Ce lien est à usage unique et reste valable pendant 15 minutes.
                            </p>
                            <p style="margin: 0 0 16px; font-size: 12px; color: #64748b;">
                                Si le bouton ne fonctionne pas, vous pouvez copier et coller l'adresse suivante dans votre navigateur :<br>
                                <a href="%s" style="color: #1e3a8a; word-break: break-all; text-decoration: underline; font-size: 11px;">%s</a>
                            </p>
                            <p style="margin: 0; font-size: 12px; color: #94a3b8;">
                                Si vous n'êtes pas à l'origine de cette demande, vous pouvez ignorer cet email en toute sécurité ; votre mot de passe actuel demeure inchangé.
                            </p>
                        </td>
                    </tr>
                    <!-- Pied de page standard -->
                    <tr>
                        <td style="padding: 20px 32px; background-color: #f8fafc; border-top: 1px solid #e2e8f0; font-size: 11px; color: #94a3b8; text-align: center;">
                            Ce message est généré automatiquement par la plateforme. Merci de ne pas y répondre.
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`, username, resetLink, resetLink, resetLink)

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

	message := fmt.Appendf(nil,
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		from, to, subject, htmlBody)

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
