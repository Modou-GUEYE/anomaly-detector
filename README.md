# Anomaly Detector — Détection d'anomalies et de fraudes LBC/FT

Plateforme de détection d'anomalies sur des transactions financières dans le cadre de la **lutte contre le blanchiment de capitaux et le financement du terrorisme (LBC/FT)**. Un moteur de règles analyse chaque transaction, génère des alertes et les met à disposition d'analystes via un tableau de bord web.

Le projet comprend une API REST en Go, une interface Angular, une suite de tests de charge Locust et un pipeline CI/CD de construction d'images Docker.

## Fonctionnalités

- **Analyse de transactions** : endpoint `POST /api/v1/anomalies/analyze` qui applique le moteur de règles et retourne les alertes générées.
- **Règles de contrôle configurables** : création, modification, activation/désactivation de règles (réservé aux administrateurs).
- **Blacklist** : gestion des entités sensibles et vérification d'un identifiant.
- **Pays à risque** : référentiel éditable, avec import depuis un fichier.
- **Gestion des alertes** : consultation, filtres, métriques, changement de statut.
- **Suivi de comptes** et profils de risque.
- **Tableau de bord** et rapports (efficacité des règles, violations, audit avec export Excel).
- **Authentification JWT (RSA)** avec rôles `admin`, `analyst` et `system`, endpoint JWKS, changement de mot de passe à la première connexion et réinitialisation par email.
- **Notifications email** : envoi SMTP en arrière-plan avec tentatives multiples, authentification PLAIN, LOGIN ou XOAUTH2 (Microsoft 365).
- **Documentation Swagger** de l'API, exposée sur `/swagger/index.html`.

## Stack technique

| Couche | Technologies |
| :--- | :--- |
| Backend | Go 1.25, Gin, GORM, JWT (golang-jwt), Swagger (swaggo), excelize |
| Base de données | PostgreSQL |
| Frontend | Angular 20, TypeScript, Chart.js (ng2-charts), Nginx |
| Tests de charge | Locust (Python), générateur de données Faker |
| CI/CD | GitLab CI, Docker |

## Architecture

```
anomaly-detector/
├── backend/
│   ├── cmd/
│   │   ├── api/         # point d'entrée de l'API
│   │   ├── migrate/     # initialisation du schéma (AutoMigrate)
│   │   └── smtpcheck/   # diagnostic de la configuration SMTP
│   ├── internal/
│   │   ├── api/         # routes, handlers, middlewares (auth, CORS, timing)
│   │   ├── auth/        # génération et validation des JWT
│   │   ├── core/        # moteur de règles (engine.go) et ses tests
│   │   ├── email/       # envoi SMTP asynchrone, OAuth
│   │   ├── models/      # modèles de données
│   │   └── store/       # accès aux données, caches, requêtes métier
│   └── docs/            # Swagger, scripts SQL, guide et rapport de tests de charge
├── frontend/            # application Angular (pages, services, guards)
├── load-tests/          # scénarios Locust et script de lancement
└── .gitlab-ci.yml       # construction et publication des images Docker
```

Le moteur de règles ([backend/internal/core/engine.go](backend/internal/core/engine.go)) applique trois familles de contrôles à chaque transaction : correspondance avec la blacklist, règles métier par compte (seuils par période) et pays à risque.

## Démarrage rapide

### Prérequis

- Go 1.25 ou plus
- Node.js 20 ou plus et npm
- PostgreSQL
- OpenSSL, pour générer la clé JWT

### 1. Backend

```bash
cd backend

# Clé privée RSA utilisée pour signer les JWT
openssl genrsa -out dev_jwt_private.pem 2048

# Configuration : copier le modèle puis remplacer les valeurs CHANGE_ME
cp .env.example .env
# Pour un usage local, adapter notamment :
#   KEY_JWT_PRIVATE_KEY_FILE=./dev_jwt_private.pem
#   DEV_DB_* (accès PostgreSQL)

go run ./cmd/api
```

L'API écoute sur `http://localhost:8080`. Avec `DB_AUTO_MIGRATE=true`, le schéma est créé au démarrage. La documentation Swagger est disponible sur `http://localhost:8080/swagger/index.html`.

Les fichiers `.env` et `*.pem` sont ignorés par git. Ne les publiez jamais.

### 2. Frontend

```bash
cd frontend
npm install
npm start
```

L'application est servie sur `http://localhost:4200`. Les appels `/api` sont redirigés vers le backend local par [proxy.conf.json](frontend/proxy.conf.json).

### 3. Docker

Chaque partie possède son `Dockerfile` (`backend/Dockerfile`, `frontend/Dockerfile`). Le pipeline GitLab CI construit et publie les deux images.

```bash
docker build -t anomaly-detector-backend ./backend
docker build -t anomaly-detector-frontend ./frontend
```

## Tests

```bash
# Tests unitaires du backend (moteur de règles, envoi email)
cd backend && go test ./...

# Tests du frontend
cd frontend && npm test
```

## Tests de charge

La suite Locust ([load-tests/](load-tests/README.md)) propose des scénarios allant du smoke test (10 utilisateurs) au stress test (500 utilisateurs), ainsi qu'une injection de grand volume.

```bash
cd load-tests
TARGET_HOST="http://localhost:8080" ./run_load_test.sh --smoke
```

La méthodologie est détaillée dans [backend/docs/GUIDE_TESTS_DE_CHARGE_LOCUST.md](backend/docs/GUIDE_TESTS_DE_CHARGE_LOCUST.md). Les optimisations qui en découlent (index SQL, pool de connexions, caches mémoire) sont décrites dans [backend/docs/RAPPORT_TESTS_DE_CHARGE_LAB.md](backend/docs/RAPPORT_TESTS_DE_CHARGE_LAB.md).

## Configuration

Toutes les variables sont documentées dans [backend/.env.example](backend/.env.example) : identité du partenaire, JWT, PostgreSQL et pool de connexions, caches, SMTP, URLs du frontend.

Pour diagnostiquer l'envoi d'emails :

```bash
cd backend
go run ./cmd/smtpcheck -to votre.adresse@exemple.com
```

## Auteur

**Modou Gueye** — développeur.
