// Commande smtpcheck : diagnostic de la configuration SMTP d'un environnement LAB.
//
// Elle lit la même configuration que le backend (.env ou variables d'environnement), se
// connecte au serveur, affiche les méthodes d'authentification annoncées et celle retenue,
// puis envoie éventuellement un email de test.
//
//	go run ./cmd/smtpcheck                       # connexion + authentification uniquement
//	go run ./cmd/smtpcheck -to admin@partenaire.sn  # + envoi d'un email de test
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"anomaly-detector/internal/email"

	"github.com/joho/godotenv"
)

func main() {
	to := flag.String("to", "", "adresse destinataire d'un email de test (optionnel)")
	envFile := flag.String("env", "", "fichier .env à charger (par défaut backend/.env ou .env)")
	flag.Parse()

	loadEnv(*envFile)
	svc := email.NewService()

	fmt.Println("Configuration SMTP")
	fmt.Printf("  Serveur        : %s:%s\n", svc.SMTPHost, svc.SMTPPort)
	fmt.Printf("  Utilisateur    : %s\n", orDash(svc.SMTPUser))
	fmt.Printf("  Mot de passe   : %s\n", mask(svc.SMTPPassword))
	fmt.Printf("  Expéditeur     : %s <%s>\n", svc.FromName, svc.FromEmail)
	fmt.Printf("  Méthode (conf) : %s\n", orDefault(svc.AuthMethod, "auto"))
	fmt.Printf("  TLS (conf)     : %s\n", orDefault(svc.TLSMode, "auto"))
	fmt.Printf("  OAuth          : %v\n", svc.OAuth.Enabled())
	fmt.Println()

	start := time.Now()
	result, err := svc.Probe()
	if result != nil {
		fmt.Println("Négociation avec le serveur")
		fmt.Printf("  Chiffrement    : %s\n", orDash(result.TLS))
		keys := make([]string, 0, len(result.Extensions))
		for k := range result.Extensions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-14s : %s\n", k, orDash(result.Extensions[k]))
		}
		fmt.Printf("  Méthode retenue: %s\n", orDash(result.AuthMethod))
	}
	if err != nil {
		fmt.Printf("\n❌ Échec (%s): %v\n", time.Since(start).Round(time.Millisecond), err)
		os.Exit(1)
	}
	fmt.Printf("\n✅ Connexion et authentification réussies (%s)\n", time.Since(start).Round(time.Millisecond))

	if *to == "" {
		return
	}
	start = time.Now()
	body := "<p>Email de test envoyé par <strong>smtpcheck</strong> (module LAB).</p><p>Méthode : " + result.AuthMethod + "</p>"
	if err := svc.SendEmail(*to, "Test de configuration SMTP — Module LAB", body); err != nil {
		fmt.Printf("❌ Envoi vers %s échoué (%s): %v\n", *to, time.Since(start).Round(time.Millisecond), err)
		os.Exit(1)
	}
	fmt.Printf("✅ Email de test envoyé à %s (%s)\n", *to, time.Since(start).Round(time.Millisecond))
}

func loadEnv(path string) {
	if path != "" {
		if err := godotenv.Load(path); err != nil {
			fmt.Fprintf(os.Stderr, "Impossible de charger %s: %v\n", path, err)
			os.Exit(2)
		}
		return
	}
	for _, p := range []string{"backend/.env", ".env"} {
		if godotenv.Load(p) == nil {
			return
		}
	}
}

func mask(secret string) string {
	if secret == "" {
		return "-"
	}
	return fmt.Sprintf("******** (%d caractères)", len(secret))
}

func orDash(s string) string { return orDefault(s, "-") }

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
