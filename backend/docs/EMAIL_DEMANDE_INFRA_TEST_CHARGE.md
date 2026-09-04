# Demande de mise à disposition d'environnement de test de charge (Module LAB)

**Destinataire(s) :** Équipe Infrastructure / Touré (`infra@gsietechnology.com` / `toure@...`)  
**Copie (CC) :** Cheikh Sidya CAMARA, Omar THIAM, Modou GUEYE  
**Date :** 28 août 2026  
**Objet :** [URGENT - LAB] Demande d'environnement isolé pour tests de charge (Campagne Locust - 2 à 3 jours)  

---

Bonjour Touré, Bonjour l'équipe Infra,

À la suite du point de cadrage avec la direction et les équipes techniques concernant la mise en production du module **LAB (Moteur de Détection d'Anomalies)**, nous devons impérativement réaliser une campagne de **tests de charge préalables** afin d'évaluer la stabilité des temps de réponse et de valider nos indicateurs de niveau de service (SLI) face à un volume cible de **500 000 transactions**.

Afin de **ne pas impacter ni fausser les paramètres de l'environnement de test partagé de l'équipe produit**, nous sollicitons votre intervention pour la mise à disposition d'un **environnement dédié et isolé** pour une durée estimée de **2 à 3 jours**.

Voici les spécifications et besoins techniques détaillés :

---

### 1. Spécifications de l'environnement demandé

| Composant | Spécification recommandée | Remarques |
| :--- | :--- | :--- |
| **Serveur / Instance** | VM dédiée (ex: 4 vCPU, 8 Go / 16 Go RAM) | Permettra de mesurer la consommation CPU/RAM et calibrer le dimensionnement cible. |
| **Base de données** | Instance PostgreSQL 15+ dédiée et isolée | Schéma vierge (`anomaly_loadtest_db`) pour exécuter les migrations sans risque pour les autres bases. |
| **Durée de réservation** | **2 à 3 jours ouvrés** | Suppression ou archivage à l'issue de la campagne de tests. |
| **Outils de test** | Support Python 3.9+ / Docker | Pour l'exécution des scénarios de tir de charge avec [Locust](https://locust.io/). |

---

### 2. Configuration Réseau & Flux

* **Port Backend API LAB :** `8080` (HTTP/REST)
* **Port Base de données :** `5432` (PostgreSQL)
* **Port Interface Locust (optionnel) :** `8089` (Web UI pour visualisation temps réel)
* **Accès SSH / Métriques :** Accès nécessaire pour l'équipe technique pour le déploiement et la consultation des métriques système (CPU, RAM, Disk I/O, I/O wait, connexions DB).

---

### 3. Déroulement & Rôles

1. **Équipe Infra :**
   - Provisionnement de la VM / de l'instance isolée avec PostgreSQL.
   - Transmission des accès (IP, identifiants SSH, chaîne de connexion PostgreSQL).
   - Activation ou mise à disposition d'un outil de monitoring de base (Prometheus/Grafana, `htop`, `pg_stat_activity` ou logs système).

2. **Équipe Technique LAB (Modou GUEYE / Équipe Dev) :**
   - Déploiement du binaire Go du module LAB et application des migrations de base de données.
   - Injection des données de référence (règles de détection, profils, pays à risque).
   - Exécution des tirs de charge automatisés (paliers de montée en charge et pic de 500k transactions).
   - Rédaction et partage du rapport de performance final (RPS, temps de réponse p50/p95/p99, calibrage recommandé).

---

Nous restons à votre disposition immédiate pour ajuster les modalités ou pour un point technique rapide de 10 minutes si nécessaire.

Merci d'avance pour votre réactivité sur ce sujet prioritaire pour le passage en pré-production.

Bien cordialement,  
**L'équipe Projet LAB**  
*Modou GUEYE / Cheikh Sidya CAMARA / Omar THIAM*
