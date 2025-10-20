# Documentation des accès backend par rôle

## BIS app

**Rôle : `bis`**
- Application mobile ou serveur partenaire BIS.
- Accès strictement limité à l’analyse de transaction.
- Utilise le token BIS obtenu via `/api/v1/bis-token`.

**Endpoint principal**
- `POST /api/v1/transactions/analyze` : Analyse une transaction BIS.


**Schéma du flux BIS app**
![Schéma du flux BIS app](schema_bis_app.png)

---

## BIS web

**Rôles : `admin`, `analyste`**
- Interface web pour les équipes BIS.
- Accès complet à la gestion des utilisateurs, profils, règles, comptes, alertes et blacklist.
- Authentification via `/api/v1/login` (token JWT).


**Endpoints BIS web (admin, analyste)**

| Méthode | Endpoint                                 | Rôle         | Description                                 |
|---------|------------------------------------------|--------------|---------------------------------------------|
| POST    | /api/v1/transactions/analyze             | admin, analyste | Analyse de transaction                      |
| GET     | /api/v1/transactions                     | admin, analyste | Liste des transactions                      |
| GET     | /api/v1/alertes                          | admin, analyste | Liste des alertes                           |
| PUT     | /api/v1/alertes/:id/status               | admin, analyste | Mise à jour du statut d'une alerte          |
| GET     | /api/v1/regle/pays_risque/pays           | admin, analyste | Liste des pays à risque                     |
| GET     | /api/v1/profile                          | admin, analyste | Profil utilisateur connecté                 |
| POST    | /api/v1/blacklist                        | admin, analyste | Création d'une entrée blacklist             |
| GET     | /api/v1/blacklist                        | admin, analyste | Liste des entrées blacklist                 |
| PUT     | /api/v1/blacklist/:id                    | admin, analyste | Modification d'une entrée blacklist         |
| DELETE  | /api/v1/blacklist/:id                    | admin, analyste | Suppression d'une entrée blacklist          |
| POST    | /api/v1/profils                          | admin         | Création d'un profil client                 |
| PUT     | /api/v1/profils/:id                      | admin         | Modification d'un profil client             |
| DELETE  | /api/v1/profils/:id                      | admin         | Suppression d'un profil client              |
| POST    | /api/v1/regles                           | admin         | Création d'une règle de contrôle            |
| PUT     | /api/v1/regles/:id                       | admin         | Modification d'une règle de contrôle        |
| DELETE  | /api/v1/regles/:id                       | admin         | Suppression d'une règle de contrôle         |
| POST    | /api/v1/comptes                          | admin         | Création d'un compte client                 |
| PUT     | /api/v1/regle/pays_risque/pays           | admin         | Mise à jour de la liste des pays à risque   |
| POST    | /api/v1/regle/pays_risque/pays           | admin         | Ajout d'un pays à la liste à risque         |
| DELETE  | /api/v1/regle/pays_risque/pays           | admin         | Suppression d'un pays à la liste à risque   |
| GET     | /api/v1/profils                          | admin, analyste | Liste des profils clients                   |
| GET     | /api/v1/profils/:id/regles               | admin, analyste | Liste des règles d'un profil client         |

---

## eKYC

**Rôle : `ekyc`**
- Service externe ou module de vérification d’identité.
- Accès strictement limité à la vérification d’un compte dans la blacklist.
- Utilise le token eKYC obtenu via `/api/v1/ekyc-token`.

**Endpoint dédié**
- `POST /api/v1/blacklist/eKYCcheck`
  - Envoi : `{ "numero_compte": "SN079..." }`
  - Réception : `{ "statut": "Interdit" }` ou `"Blacklist"`, `"PEP"`, `"Surveillés"`, `"Aucun"`


**Schéma du flux eKYC**
![Schéma du flux eKYC](schema_ekyc.png)

---

## Sécurité et séparation des rôles
- Tous les accès sont protégés par JWT.
- Les rôles sont vérifiés par middleware.
- Les routes sont strictement séparées selon les droits :
  - BIS app : rôle `bis` → accès uniquement à l’analyse de transaction.
  - BIS web : rôles `admin`, `analyste` → gestion métier et administration, gestion blacklist.
  - eKYC : rôle `ekyc` → vérification blacklist uniquement.

---

## Résumé des endpoints par rôle

| Endpoint                                 | BIS app (`bis`) | BIS web (`admin`, `analyste`) | eKYC (`ekyc`) |
|-------------------------------------------|:--------------:|:----------------------------:|:-------------:|
| /api/v1/transactions/analyze              |       ✔️        |             ✔️               |      ❌        |
| /api/v1/blacklist                         |       ❌        |             ✔️               |      ❌        |
| /api/v1/blacklist/eKYCcheck               |       ❌        |             ❌               |      ✔️        |
| /api/v1/alertes                           |       ❌        |             ✔️               |      ❌        |
| /api/v1/profils, /api/v1/regles, /comptes |       ❌        |             ✔️               |      ❌        |

---

Pour toute intégration, chaque intervenant doit utiliser le bon endpoint et le bon rôle JWT pour garantir la sécurité et la conformité métier.
