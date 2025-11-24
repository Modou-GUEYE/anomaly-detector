# DELIVERABLES — Remise à l'équipe infra

Ce fichier liste de façon minimale et explicite ce qu'il faut transmettre à l'équipe infra — adapté au fait que l'infra peut pull depuis le Registry.

1) Images 

- backend : registry.gitlab.com/PISFN/LAB-FT/anomaly-api:1.0.1
- frontend : registry.gitlab.com/PISFN/LAB-FT/anomaly-frontend:1.0.1

2) Fichier d'environnement 

- `backend/anomaly.env`  — contient la liste des variables attendues.

3) Orchestration minimale (optionnel)

- `docker-compose.registry.yml` — compose d'exemple utilisant les tags du registry (db/backend/frontend).

4) Migrations / initialisation BD

- Méthodes supportées :
  - AutoMigrate intégré (DB_AUTO_MIGRATE=true) — l'application créera les tables au démarrage si activé.
  - SQL dump (`db/init.sql`) — fourni sur demande.
