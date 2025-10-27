# Architecture du serveur.go - Vue d'ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                       server.go - 1060 lignes                   │
└─────────────────────────────────────────────────────────────────┘

┌─ LAYER 1: FONDATIONS ──────────────────────────────────────────┐
│                                                                  │
│  1. PACKAGE & IMPORTS (Lignes 1-21)                            │
│     └─ package api                                              │
│     └─ imports (core, models, store, gin, jwt, bcrypt)         │
│                                                                  │
│  2. CONSTANTES (Lignes 23-25)                                  │
│     └─ var jwtKey []byte                                        │
│                                                                  │
│  3. STRUCTURES (Lignes 27-31)                                  │
│     └─ type Server struct { store, router }                    │
│                                                                  │
└────────────────────────────────────────────────────────────────┘

┌─ LAYER 2: SÉCURITÉ & AUTHENTIFICATION ─────────────────────────┐
│                                                                  │
│  4. MIDDLEWARES (Lignes 33-125)                                │
│                                                                  │
│     ┌─ jwtAuthMiddleware() ─────────────────────────┐          │
│     │ • Valide le header Authorization              │          │
│     │ • Parse et valide le JWT                      │          │
│     │ • Extrait userID et role dans le contexte     │          │
│     └────────────────────────────────────────────────┘          │
│                                                                  │
│     ┌─ roleMiddleware(allowedRoles...) ─────────────┐          │
│     │ • Contrôle d'accès par rôle                   │          │
│     │ • Usage: roleMiddleware("admin", "analyst")   │          │
│     └────────────────────────────────────────────────┘          │
│                                                                  │
│     ┌─ checkRole(requiredRole) ──────────────────────┐          │
│     │ • Vérification granulaire d'un seul rôle      │          │
│     │ • Usage: checkRole("admin")                   │          │
│     └────────────────────────────────────────────────┘          │
│                                                                  │
└────────────────────────────────────────────────────────────────┘

┌─ LAYER 3: HANDLERS - AUTHENTIFICATION ──────────────────────────┐
│                                                                  │
│  5. HANDLERS PUBLICS (Lignes 127-184)                          │
│                                                                  │
│     ┌─ handleLogin() ────────────────────────────────┐          │
│     │ POST /api/v1/auth/login                        │          │
│     │ Email + Password → JWT Token                   │          │
│     └────────────────────────────────────────────────┘          │
│                                                                  │
│     ┌─ handleGetSystemToken() ──────────────────────┐          │
│     │ POST /api/v1/auth/token                        │          │
│     │ system_id + role → JWT Token (NEUTRE)          │          │
│     └────────────────────────────────────────────────┘          │
│                                                                  │
└────────────────────────────────────────────────────────────────┘

┌─ LAYER 4: HANDLERS - DOMAINES MÉTIER ──────────────────────────┐
│                                                                  │
│  Organisés par RESSOURCE, pas par HTTP verb                    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ ANOMALIES/TRANSACTIONS (4 handlers)    [Lignes 186-420]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleAnalyzeTransaction() - Analyser transaction    │    │
│  │ • handleGetAlertMetrics()    - Métriques             │    │
│  │ • handleGetAlertes()         - Lister alertes        │    │
│  │ • handleUpdateAlerteStatus() - Changer statut        │    │
│  │                                                        │    │
│  │ Rôles: admin, analyst, system                        │    │
│  │ Auth: JWT + roleMiddleware                           │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ ENTITÉS/BLACKLIST (5 handlers)         [Lignes 422-530]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetBlacklistEntries()    - Lister             │    │
│  │ • handleCreateBlacklistEntry()   - Créer              │    │
│  │ • handleUpdateBlacklistEntry()   - Modifier           │    │
│  │ • handleDeleteBlacklistEntry()   - Supprimer          │    │
│  │ • handleCheckBlacklist()         - Vérifier statut    │    │
│  │                                                        │    │
│  │ Rôles: admin, analyst (+ vérif publique)             │    │
│  │ Auth: JWT + roleMiddleware                           │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ RÈGLES DE CONTRÔLE (6 handlers)        [Lignes 532-638]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetAllRegles()           - Lister toutes      │    │
│  │ • handleCreateRegleControle()    - Créer (admin)      │    │
│  │ • handleUpdateRegleControle()    - Modifier (admin)   │    │
│  │ • handleDeleteRegleControle()    - Supprimer (admin)  │    │
│  │ • handleGetReglesByProfil()      - Par profil        │    │
│  │ • handleGetReglesForCompte()     - Par compte        │    │
│  │                                                        │    │
│  │ Rôles: admin (écriture), analyst (lecture)           │    │
│  │ Auth: JWT + roleMiddleware + checkRole("admin")      │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ PROFILS CLIENTS (4 handlers)           [Lignes 640-720]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetProfilsClient()       - Lister             │    │
│  │ • handleCreateProfilClient()     - Créer (admin)      │    │
│  │ • handleUpdateProfilClient()     - Modifier (admin)   │    │
│  │ • handleDeleteProfilClient()     - Supprimer (admin)  │    │
│  │                                                        │    │
│  │ Rôles: admin (écriture), analyst (lecture)           │    │
│  │ Auth: JWT + roleMiddleware + checkRole("admin")      │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ COMPTES CLIENTS (3 handlers)           [Lignes 722-780]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetTransactions()        - Lister             │    │
│  │ • handleCreateCompte()           - Créer (admin)      │    │
│  │ • handleGetComptesSurveilles()   - Surveillés         │    │
│  │                                                        │    │
│  │ Rôles: admin, analyst                                 │    │
│  │ Auth: JWT + roleMiddleware + checkRole("admin")      │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ CONFIGURATION (4 handlers)             [Lignes 782-860]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetPaysRisque()          - Lister             │    │
│  │ • handleAddPaysRisque()          - Ajouter            │    │
│  │ • handleUpdatePaysRisque()       - Modifier           │    │
│  │ • handleRemovePaysRisque()       - Retirer            │    │
│  │                                                        │    │
│  │ Rôles: admin uniquement                               │    │
│  │ Auth: JWT + roleMiddleware("admin")                  │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │ PROFIL UTILISATEUR (1 handler)         [Lignes 862-878]   │
│  ├────────────────────────────────────────────────────────┤    │
│  │ • handleGetProfile()             - Profil courant    │    │
│  │                                                        │    │
│  │ Rôles: tous authentifiés                              │    │
│  │ Auth: JWT + jwtAuthMiddleware()                       │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
└────────────────────────────────────────────────────────────────┘

┌─ LAYER 5: INITIALISATION & DÉMARRAGE ──────────────────────────┐
│                                                                  │
│  6. INITIALISATION SERVEUR (Lignes 880-980)                   │
│     └─ func NewServer(store) *Server                           │
│        • Crée routeur Gin                                      │
│        • Configure 8 groupes de routes                         │
│        • Enregistre tous les middlewares                       │
│        • Retourne instance Server                              │
│                                                                  │
│  7. DEMARRAGE SERVEUR (Lignes 982-986)                        │
│     └─ func (server) Start(address) error                      │
│        • Lance le serveur HTTP                                 │
│        • Écoute sur l'adresse spécifiée                        │
│                                                                  │
│  8. UTILS (Lignes 988-1003)                                   │
│     ├─ func HashPassword(password) string - Hachage bcrypt   │
│     └─ func checkPasswordHash(pwd, hash) bool - Vérification │
│                                                                  │
└────────────────────────────────────────────────────────────────┘


═════════════════════════════════════════════════════════════════
                    FLUX DE REQUÊTE TYPIQUE
═════════════════════════════════════════════════════════════════

1. REQUEST ARRIVE
   │
   ├─→ Router reçoit la requête
   │
   ├─→ Middlewares JWT (si protégé)
   │   ├─ Valide Authorization header
   │   ├─ Parse JWT
   │   └─ Injecte userID et role dans ctx
   │
   ├─→ Middleware role (si protégé)
   │   ├─ Vérifie que le rôle est autorisé
   │   └─ Rejette si non autorisé (403)
   │
   ├─→ Handler spécifique
   │   ├─ Récupère les données de la requête
   │   ├─ Appelle les méthodes du store
   │   ├─ Traite la logique métier
   │   └─ Retourne la réponse JSON
   │
   └─→ RESPONSE ENVOYÉE AU CLIENT


═════════════════════════════════════════════════════════════════
                    RÔLES ET PERMISSIONS
═════════════════════════════════════════════════════════════════

admin (Administrateur)
  │
  ├─ ✅ Accès complet à tous les endpoints
  ├─ ✅ Gestion des utilisateurs
  ├─ ✅ Création/modification/suppression de règles
  ├─ ✅ Gestion de la configuration
  ├─ ✅ Gestion des profils clients
  └─ ✅ Gestion des comptes clients

analyst (Analyste)
  │
  ├─ ✅ Lecture/écriture des anomalies
  ├─ ✅ Gestion des statuts d'alertes
  ├─ ✅ Vérification de la blacklist
  ├─ ✅ Lecture des règles et profils
  ├─ ✅ Lecture des comptes
  └─ ❌ Pas d'écriture (sauf anomalies)

system (Système externe/API)
  │
  ├─ ✅ Analyse de transactions
  ├─ ✅ Vérification publique du statut entités
  └─ ❌ Pas d'accès à l'administration

reviewer (Auditeur)
  │
  ├─ ✅ Lecture de toutes les données
  └─ ❌ Pas d'écriture (à implémenter)


═════════════════════════════════════════════════════════════════
```

## 📊 Statistiques

| Élément | Nombre | Lignes |
|---------|--------|--------|
| Middlewares | 3 | 93 |
| Handlers publics | 2 | 58 |
| Handlers protégés | 27 | 693 |
| Groupes de routes | 8 | 101 |
| Fonctions utils | 2 | 16 |
| **Total** | **34** | **1060** |

## 🎯 À retenir

✅ **Organisation claire** : Chaque section a un rôle bien défini  
✅ **Facile à naviguer** : Commentaires de section + structure logique  
✅ **Extensible** : Ajouter un nouveau handler est simple  
✅ **Sécurisé** : Middlewares JWT + RBAC en place  
✅ **Neutre** : Aucune référence à des plateformes spécifiques
