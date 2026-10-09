package email

import (
	"errors"
	"log"
	"net"
	"net/textproto"
	"sync"
	"time"
)

// ===================== ENVOI EN ARRIÈRE-PLAN =====================
//
// Les emails sont envoyés hors de la requête HTTP : un serveur SMTP lent ou injoignable ne
// bloque plus la création d'utilisateur ou la demande de mot de passe oublié.
// La file est en mémoire : un email en attente est perdu si le backend redémarre
// (un « mot de passe oublié » permet alors de renvoyer un lien).

const (
	asyncQueueSize = 100
	asyncWorkers   = 2
	maxAttempts    = 3
)

type asyncJob struct {
	description string
	send        func() error
}

var (
	asyncOnce  sync.Once
	asyncQueue chan asyncJob
	retryDelay = 5 * time.Second
)

// SendAsync planifie l'envoi d'un email. description sert uniquement aux logs : elle ne doit
// contenir aucune donnée sensible (mot de passe, lien de réinitialisation).
func SendAsync(description string, send func() error) {
	asyncOnce.Do(startWorkers)
	select {
	case asyncQueue <- asyncJob{description: description, send: send}:
	default:
		log.Printf("Email non envoyé (file d'attente pleine): %s", description)
	}
}

func startWorkers() {
	asyncQueue = make(chan asyncJob, asyncQueueSize)
	for i := 0; i < asyncWorkers; i++ {
		go func() {
			for job := range asyncQueue {
				runJob(job)
			}
		}()
	}
}

func runJob(job asyncJob) {
	for attempt := 1; ; attempt++ {
		err := job.send()
		if err == nil {
			return
		}
		if attempt >= maxAttempts || !isTemporary(err) {
			log.Printf("Email non envoyé (%s) après %d tentative(s): %v", job.description, attempt, err)
			return
		}
		time.Sleep(time.Duration(attempt) * retryDelay)
	}
}

// isTemporary indique si une nouvelle tentative a des chances d'aboutir : erreur réseau ou
// code SMTP 4xx. Un refus définitif (5xx, ex. authentification) n'est pas retenté.
func isTemporary(err error) bool {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		return smtpErr.Code >= 400 && smtpErr.Code < 500
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
