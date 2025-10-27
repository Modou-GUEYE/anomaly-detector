# 🔑 Guide du Token Système (system_id/role)

## 📌 Qu'est-ce que c'est ?

Le endpoint `POST /api/v1/auth/token` génère un **token JWT pour les systèmes externes** (partenaires, applications, services).

C'est **DIFFÉRENT** de :
- `POST /api/v1/auth/login` → Pour les **utilisateurs** (email/password)
- `POST /api/v1/auth/token` → Pour les **systèmes** (system_id/role)

---

## 🎯 Cas d'Usage

| Scenario | Endpoint | Paramètres | Exemple |
|----------|----------|-----------|---------|
| **Utilisateur se connecte** | POST /auth/login | email, password | User Analyste |
| **Système externe** | POST /auth/token | system_id, role | App mobile, Service tiers |

---

## 🚀 Comment l'utiliser ?

### 1️⃣ Via Postman

| Propriété | Valeur |
|-----------|--------|
| **Méthode** | POST |
| **URL** | http://localhost:8080/api/v1/auth/token |

**Headers** :
```
Content-Type: application/json
```

**Body (raw JSON)** :
```json
{
  "system_id": "my-external-app",
  "role": "analyst"
}
```

**Puis utilise le token** :
```
Authorization: Bearer <token_reçu>
```

---

### 2️⃣  Dans du Code (JavaScript)

```javascript
async function getSystemToken(systemId, role) {
  const response = await fetch('http://localhost:8080/api/v1/auth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      system_id: systemId,
      role: role
    })
  });
  
  const data = await response.json();
  return data.token;
}

// Utilisation
const token = await getSystemToken('my-app', 'analyst');
console.log('Token:', token);
```

---

## 🎪 Valeurs Possibles

### system_id
C'est l'identifiant **unique** de ton système externe :
- `"anomaly-detector"` 
- `"my-external-app"`
- `"mobile-app-v1"`
- `"partner-api"`
- Tout ce que tu veux !

### role
Doit être **UN des rôles autorisés** :

| Role | Permissions |
|------|------------|
| `"admin"` | Accès complet + configuration |
| `"analyst"` | Accès aux anomalies + analyse |
| `"system"` | Accès système (API interne) |
| `"reviewer"` | Accès lecture seule |

---

## 📊 Flux Complet d'Authentification

```
┌─────────────────────────────────────────────────────────┐
│                  UTILISATEUR (email/password)           │
└──────────────────┬──────────────────────────────────────┘
                   │
         POST /api/v1/auth/login
         {"email": "analyste@gsie.sn", "password": "analyste123"}
                   │
                   ▼
          ✅ Retourne JWT User Token
          (valide 24h, lié à l'utilisateur)
         
┌─────────────────────────────────────────────────────────┐
│              SYSTÈME EXTERNE (system_id/role)           │
└──────────────────┬──────────────────────────────────────┘
                   │
         POST /api/v1/auth/token
         {"system_id": "my-app", "role": "analyst"}
                   │
                   ▼
          ✅ Retourne JWT System Token
          (valide, lié au système)
```

---

## ✅ Exemple Complet - Postman

### Step 1 : Générer le Token Système
```bash
POST http://localhost:8080/api/v1/auth/token

Body:
{
  "system_id": "test-app",
  "role": "analyst"
}
```

Copie le `token` retourné.

### Step 2 : Utiliser le Token pour Analyser une Transaction
```bash
POST http://localhost:8080/api/v1/anomalies/analyze

Authorization: Bearer <token_copié>

Body:
{
  "montant": 500000.0,
  "numero_compte_payeur": "SN01100040099887766554433",
  "nom_payeur": "Test Transaction"
}
```

### Step 3 : Récupérer les Alertes
```bash
GET http://localhost:8080/api/v1/anomalies

Authorization: Bearer <token_copié>
```

---

## 🔒 Sécurité

- ⚠️ **Ne partage PAS tes tokens** (ils contiennent le role et system_id)
- ⏱️ Les tokens sont valides jusqu'à la fin de la session
- 🔐 Change la `jwtKey` dans `server.go` pour ton système

---

## ❓ FAQ

**Q: Quelle est la différence avec `/auth/login` ?**
A: `/auth/login` = Authentification utilisateur humain  
`/auth/token` = Authentification système/app automatisée

**Q: Je peux utiliser n'importe quel system_id ?**
A: Oui, c'est un identifiant libre. À toi de le gérer.

**Q: Le token expire ?**
A: Non, dans la version actuelle.

**Q: Je peux avoir plusieurs tokens ?**
A: Oui, chaque appel génère un nouveau token.

