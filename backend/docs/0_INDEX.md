# 📚 Index de documentation du Backend

Cette page centralise toute la documentation du backend organisée par sujet.

## 📖 Documents principaux

### 1. **API_ROUTES.md**
> Documentation complète des endpoints API

**Contient :**
- ✅ Matrice des rôles et permissions
- ✅ 79+ endpoints documentés
- ✅ Groupes de routes (8 domaines)
- ✅ Exemples cURL détaillés
- ✅ Codes HTTP et erreurs
- ✅ Notes sur Keycloak

**À consulter quand :**
- Vous avez besoin du spec d'un endpoint
- Vous testez une nouvelle intégration
- Vous documentez un client API

---

### 2. **SERVER_STRUCTURE.md**
> Guide complet de la structure interne du serveur.go

**Contient :**
- ✅ 9 sections du fichier expliquées
- ✅ Navigation rapide par ligne
- ✅ Détail des middlewares
- ✅ Organisation des handlers
- ✅ Principes de design
- ✅ Guide d'ajout d'handlers

**À consulter quand :**
- Vous modifiez server.go
- Vous ajoutez un nouveau handler
- Vous comprenez l'architecture interne

---

### 3. **SERVER_ARCHITECTURE_DIAGRAM.md**
> Vue d'ensemble visuelle de l'architecture

**Contient :**
- ✅ Diagramme ASCII en 8 layers
- ✅ Flux de requête typique
- ✅ Matrice des rôles et permissions
- ✅ Statistiques du code
- ✅ Architecture détaillée

**À consulter quand :**
- Vous présentez l'architecture
- Vous comprenez le flux de sécurité
- Vous apprenez le système

---

### 4. **REFACTORING_SUMMARY.md**
> Résumé complet de la refactorisation effectuée

**Contient :**
- ✅ Changements phase par phase
- ✅ Avant/Après comparaison
- ✅ Statistiques détaillées
- ✅ Architecture finale
- ✅ Points forts & prochaines étapes

**À consulter quand :**
- Vous comprenez l'historique des changements
- Vous validez les changements
- Vous présentez à l'équipe

---

### 5. **docs.go** (Swagger)
> Configuration Swagger/OpenAPI

**Contient :**
- ✅ Métadonnées API
- ✅ Authentification Bearer JWT
- ✅ Schémas des réponses
- ✅ Génération de la doc

**À consulter quand :**
- Documentation automatique est nécessaire
- Génération OpenAPI

---

## 🗺️ Autres documentation existante

### Structure API
- `structure_alertes.md` - Structure des alertes
- `PLAN_TEST.md` - Plan de test

### Base de données
- Voir `backend/internal/models/models.go` pour les schémas

### Configuration
- `backend/go.mod` - Dépendances Go
- `backend/cmd/api/main.go` - Point d'entrée

---

## 🎯 Guide rapide par besoin

### "Je veux tester un endpoint"
1. Ouvrez **API_ROUTES.md**
2. Cherchez l'endpoint par domaine
3. Copiez la commande
4. Adaptez les paramètres
5. Exécutez

### "Je veux ajouter un nouvel endpoint"
1. Créez le handler dans **server.go**
2. Ajoutez les commentaires Swagger
3. Enregistrez la route dans `NewServer()`
4. Documentez dans **API_ROUTES.md**
5. Testez

### "Je dois comprendre le flux de sécurité"
1. Lisez **SERVER_ARCHITECTURE_DIAGRAM.md** (Flux section)
2. Consultez les middlewares dans **SERVER_STRUCTURE.md**
3. Tracez un exemple de requête

### "Je dois ajouter une permission fine"
1. Consultez les rôles dans **SERVER_ARCHITECTURE_DIAGRAM.md**
2. Utilisez `checkRole()` pour granularité
3. Testez avec différents rôles

### "Je dois documenter pour un client"
1. Envoyez **API_ROUTES.md** (spec complète)
2. Ajoutez **SERVER_ARCHITECTURE_DIAGRAM.md** (vue globale)
3. Fourni des exemples cURL spécifiques

---

## 📊 Statistiques documentelles

| Document | Lignes | Type | Audience |
|----------|--------|------|----------|
| API_ROUTES.md | 331 | Spec | Développeurs |
| SERVER_STRUCTURE.md | ~200 | Guide | Développeurs |
| SERVER_ARCHITECTURE_DIAGRAM.md | ~350 | Visuel | Tous |
| REFACTORING_SUMMARY.md | ~250 | Résumé | Managers/Leads |
| docs.go | ~50 | Code | CI/CD |
| **Total** | ~1181 | Mixte | Tous |

---

## 🔄 Comment maintenir la documentation

### Quand modifier API_ROUTES.md :
- ✏️ Vous ajoutez un nouvel endpoint
- ✏️ Vous changez les permissions
- ✏️ Vous modifiez le format de réponse

### Quand modifier SERVER_STRUCTURE.md :
- ✏️ Vous réorganisez server.go
- ✏️ Vous ajoutez un middleware
- ✏️ Vous changez une section

### Quand créer un nouveau document :
- ✏️ Nouvelle architecture/module
- ✏️ Nouveau domaine d'expertise
- ✏️ Décisions architecturales importantes

---

## 🚀 Utilisation en DEVELOPPEMENT

### Documentation pour les développeurs
```bash
# API spec
cat docs/API_ROUTES.md

# Architecture
cat docs/SERVER_ARCHITECTURE_DIAGRAM.md

# Implémentation
cat docs/SERVER_STRUCTURE.md
```

### Documentation pour les clients API
```bash
# Envoyez-leur API_ROUTES.md
# + un aperçu du flux de sécurité
```

---

## 🆘 Troubleshooting

| Problème | Solution |
|----------|----------|
| Endpoint non trouvé | Vérifiez API_ROUTES.md, la route est peut-être sous un autre groupe |
| Permission refusée (403) | Consultez les rôles dans SERVER_ARCHITECTURE_DIAGRAM.md |
| Token invalide | Vérifiez le format Authorization dans API_ROUTES.md |
| Modification confuse | Lisez SERVER_STRUCTURE.md pour comprendre le flux |

---

## 📝 Dernière mise à jour

- **Date** : 22 octobre 2025
- **Auteur** : Modou GUEYE
- **Version** : 2.0
- **Status** : ✅ Documentation complète

---

*Cette documentation est générée et maintenue à jour.*  
*
