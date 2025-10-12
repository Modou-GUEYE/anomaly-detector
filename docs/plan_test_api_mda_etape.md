# Plan de Test Fonctionnel Complet pour l'API MDA

## Préalables
- Peupler la base de données :
  ```bash
  go run ./cmd/seeder
  ```
- Lancer le serveur API principal :
  ```bash
  go run ./cmd/api
  ```
- Préparer Postman : stocker deux tokens (Admin et Analyste)

---

## Scénario 1 : Authentification et Accès de Base

### Test 1.1 : Connexion Admin (Succès)
**Requête** : POST /api/v1/login
**Corps JSON** :
```json
{ "email": "admin@gsie.sn", "password": "admin123" }
```
**Résultat attendu** : 200 OK avec un token (Token Admin)

### Test 1.2 : Connexion Analyste (Succès)
**Requête** : POST /api/v1/login
**Corps JSON** :
```json
{ "email": "analyste@gsie.sn", "password": "analyste123" }
```
**Résultat attendu** : 200 OK avec un token (Token Analyste)

### Test 1.3 : Échec de connexion (mot de passe incorrect)
**Requête** : POST /api/v1/login
**Corps JSON** :
```json
{ "email": "admin@gsie.sn", "password": "mauvais_mdp" }
```
**Résultat attendu** : 401 Unauthorized

### Test 1.4 : Accès au Profil (Succès)
**Requête** : GET /api/v1/profile
**Authorization** : Bearer Token Analyste
**Résultat attendu** : 200 OK avec les infos du profil

### Test 1.5 : Échec d'accès au Profil (sans token)
**Requête** : GET /api/v1/profile
**Authorization** : Aucun
**Résultat attendu** : 401 Unauthorized

---


## Scénario 2 : Gestion des Pays à Risque (CRUD)


### Test 2.1 : Ajout d'un pays à risque
**Requête** : POST /api/v1/regle/pays_risque/pays
**Authorization** : Bearer Token Admin
**Corps JSON** :
```json
{
  "pays": "Mali"
}
```
**Résultat attendu** : 200 OK, message "Pays ajouté à la liste des pays à risque"

### Test 2.2 : Liste des pays à risque
**Requête** : GET /api/v1/regle/pays_risque/pays
**Authorization** : Bearer Token Analyste
**Résultat attendu** : 200 OK, liste JSON des pays à risque
Exemple :
```json
{
  "liste_pays": ["Mali", "Burkina Faso"]
}
```

### Test 2.3 : Modification de la liste des pays à risque
**Requête** : PUT /api/v1/regle/pays_risque/pays
**Authorization** : Bearer Token Admin
**Corps JSON** :
```json
{
  "liste_pays": ["Burkina Faso", "Niger"]
}
```
**Résultat attendu** : 200 OK, message "Liste des pays à risque mise à jour"

### Test 2.4 : Suppression d'un pays à risque
**Requête** : DELETE /api/v1/regle/pays_risque/pays
**Authorization** : Bearer Token Admin
**Corps JSON** :
```json
{
  "pays": "Mali"
}
```
**Résultat attendu** : 200 OK, message "Pays retiré de la liste des pays à risque"

### Test 2.1 : Un Admin peut créer une Règle (Succès)
**Requête** : POST /api/v1/regles
**Authorization** : Bearer Token Admin
**Corps JSON** :
```json
{
  "nomRegle": "Fréquence > 5 en 24h",
  "description": "Déclenche une alerte si plus de 5 transactions sont effectuées en 24h",
  "typeRegle": "frequence",
  "criticite": "Moyenne",
  "seuil": 5,
  "periodeReference": "24h",
  "active": true,
  "profilClientId": "ID_DU_PROFIL_PARTICULIER"
}
ou
{
  "nomRegle": "Cumul Hebdomadaire > 1 000 000 FCFA",
  "description": "Déclenche une alerte si le cumul des transactions sur 7 jours dépasse 1 000 000 FCFA",
  "typeRegle": "montant",
  "criticite": "Élevée",
  "seuil": 1000000.0,
  "periodeReference": "7d",
  "active": true,
  "profilClientId": "ID_DU_PROFIL_ENTREPRISE"
}
```
**Résultat attendu** : 201 Created

### Test 2.2 : Un Analyste ne peut PAS créer une Règle (Échec)
**Requête** : POST /api/v1/regles
**Authorization** : Bearer Token Analyste
**Corps JSON** : (identique ci-dessus)
**Résultat attendu** : 403 Forbidden

---

## Scénario 3 : Flux Complet de Détection de D'anomalie

### Étape A : Test du Contrôle Blacklist
**Requête** : POST /api/v1/transactions/analyze (Token Analyste)
**Corps JSON** :
```json
{
  "montant": 50.0,
  "numero_compte_payeur": "SN01100020098765432109876",
  "nom_payeur": "Client Interdit"
}
```
**Résultat attendu** : 403 Forbidden, raison "le payeur est sur une liste d'interdiction"

### Étape B : Test de la Règle de Fréquence

**Préparation** : Récupérer le numero_compte d'un client "Particulier"
**Requête** : POST /api/v1/transactions/analyze (Token Analyste)
**Corps JSON** :
```json
{
  "montant": 100.0,
  "numero_compte_payeur": "SN01100020098765432109000"
}
```
**Action** : Envoyer la requête 5 fois → 200 OK
**Action** : 6ème requête → 201 Created avec alerte de violation de fréquence

### Étape D : Test de la Règle Pays à Risque
**Préparation** : S'assurer qu'un pays à risque est bien configuré dans la base (ex : "Mali")
**Requête** : POST /api/v1/transactions/analyze (Token Analyste)
**Corps JSON** :
```json
{
  "montant": 100.0,
  "numero_compte_payeur": "SN01100020098765432109001",
  "pays_payeur": "Mali"
}
```
**Résultat attendu** : 201 Created avec alerte pour pays à risque
**Vérification** : L'alerte doit contenir la raison "Transaction depuis un pays à haut risque: Mali" et le champ `scoreRisque` à 1.0
**Préparation** : Récupérer le knumero_compte d'un client "Particulier"
**Requête** : POST /api/v1/transactions/analyze (Token Analyste)
**Corps JSON** :
```json
{
  "montant": 100.0,
  "numero_compte_payeur": "SN01100020098765432109000"
}
```
**Action** : Envoyer la requête 5 fois → 200 OK
**Action** : 6ème requête → 201 Created avec alerte de violation de fréquence

### Étape C : Test de la Règle de Cumul
**Requête 1** : POST /api/v1/transactions/analyze (Token Analyste)
```json
{ "montant": 600000.0, "numero_compte_payeur": "SN01100040099887766554433" }
```
**Résultat** : 200 OK
**Requête 2** : POST /api/v1/transactions/analyze (Token Analyste)
```json
{ "montant": 500000.0, "numero_compte_payeur": "SN01100040099887766554433" }
```
**Résultat attendu** : 201 Created avec alerte de cumul

---

## Scénario 4 : Flux de Traitement par l'Analyste

### Étape A : Consulter les Alertes
**Requête** : GET /api/v1/alertes (Token Analyste)
**Résultat attendu** : 200 OK, liste JSON des alertes
```json
[
  {
    "id": "df8326b4-d29d-47f2-a41c-84a9e07f34db",
    "transactionId": "...",
    "raisonDetection": "Fréquence de 6 transactions dépassant le seuil de 5 sur 24h.",
    "statut": "En cours",
    "analysteAssigneId": "",
    ...
  }
]
```

### Étape B : Traiter une Alerte
**Requête** : PUT /api/v1/alertes/ID_DE_L_ALERTE_COPIÉ/status (Token Analyste)
**Body JSON** :
```json
{
  "statut": "En Investigation"
}
```
**Résultat attendu** : 200 OK, alerte mise à jour
```json
{
  "id": "df8326b4-d29d-47f2-a41c-84a9e07f34db",
  ...
  "statut": "En Investigation",
  "analysteAssigneId": "5bea9b1a-dd0a-490e-927f-a328c87176ed",
  ...
}
```

---

Ce plan de test doit être suivi étape par étape pour valider toutes les fonctionnalités principales de l'API MDA, en utilisant Postman ou tout autre outil de test d'API.