# 📖 Guide Complet des Routes API (Corrigé)

## ✅ Changements Effectués

| Objet | Avant | Maintenant |
|-------|-------|-----------|
| **Lister comptes** | ❌ N'existait pas | ✅ `GET /api/v1/accounts` |
| **Lister transactions** | ✅ `GET /api/v1/accounts` (confus) | ✅ `GET /api/v1/transactions` |
| **Comptes surveillés** | ✅ `GET /api/v1/accounts/monitored` | ✅ `GET /api/v1/accounts/monitored` (inchangé) |

---

## 📊 Transactions & Comptes

### 1️⃣ Lister TOUS les comptes clients
```
GET /api/v1/accounts
Authorization: Bearer <token>

Response: [
  {
    "id": "uuid",
    "nom_client": "Client Test 2",
    "numero_compte": "SN07900010012345678901205",
    "type_compte": "Courant",
    "type_client": "M",
    "categorie": "Entreprise",
    "pays": "Senegal",
    "ville": "Dakar",
    "adresse": "123 Rue Test",
    "profil_client_id": "uuid",
    "date_creation": "2025-10-27T19:25:12Z"
  },
  ...
]
```

### 2️⃣ Lister TOUTES les transactions
```
GET /api/v1/transactions
Authorization: Bearer <token>

Response: [
  {
    "id": "uuid",
    "date_operation": "2025-10-27T19:25:12Z",
    "montant": 500000,
    "portee": "National",
    "pays_payeur": "Senegal",
    "nom_payeur": "Client Test 2",
    "numero_compte_payeur": "SN07900010012345678901205",
    "pays_paye": "Mali",
    "nom_paye": "Client Test",
    "numero_compte_paye": "SN07900040099887766554406",
    ...
  },
  ...
]
```

### 3️⃣ Lister comptes surveillés (monitored)
```
GET /api/v1/accounts/monitored
Authorization: Bearer <token>

Response: {
  "count": 5,
  "comptes": [
    "SN07900010012345678901205",
    "SN07900040099887766554406",
    ...
  ]
}
```

### 4️⃣ Créer un nouveau compte
```
POST /api/v1/accounts
Authorization: Bearer <token> (admin only)
Content-Type: application/json

{
  "nom_client": "Nouveau Client",
  "numero_compte": "SN07900010999999999999999",
  "type_compte": "Courant",
  "type_client": "M",
  "categorie": "Entreprise",
  "pays": "Senegal",
  "ville": "Dakar",
  "adresse": "123 Rue Test",
  "profil_client_id": "uuid"
}

Response: { "id": "uuid", ... }
```

### 5️⃣ Obtenir les règles d'un compte
```
GET /api/v1/accounts/<numero_compte>/rules
Authorization: Bearer <token>

Exemple: GET /api/v1/accounts/SN07900010012345678901205/rules

Response: [
  {
    "id": "uuid",
    "nom_regle": "Cumul 7 jours",
    "type_regle": "montant",
    "seuil": 2000000,
    ...
  },
  ...
]
```

---

## 🚨 Anomalies & Alertes

### 1️⃣ Analyser une transaction (MDA)
```
POST /api/v1/anomalies/analyze
Authorization: Bearer <token>
Content-Type: application/json

{
  "date_operation": "2025-10-28T10:30:00Z",
  "montant": 600000,
  ...
}
```

### 2️⃣ Lister toutes les alertes
```
GET /api/v1/anomalies
Authorization: Bearer <token>
```

---

## 📌 Résumé des Routes

| Ressource | Endpoint |
|-----------|----------|
| **Comptes** | `GET /api/v1/accounts` |
| **Transactions** | `GET /api/v1/transactions` |
| **Comptes surveillés** | `GET /api/v1/accounts/monitored` |
| **Alertes** | `GET /api/v1/anomalies` |
| **Entités Blacklist** | `GET /api/v1/entities` |
| **Règles** | `GET /api/v1/rules` |
| **Profils** | `GET /api/v1/profiles` |
