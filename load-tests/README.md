# Module LAB — Suite de Tests de Charge (Locust)

Suite de tests de charge et d'évaluation de performance pour le **Module LAB (Moteur de Détection d'Anomalies)**, configurable pour tests en local ou sur serveur isolé dédié.

Pour le guide méthodologique complet et réutilisable sur d'autres projets, consultez :  
👉 **[GUIDE_TESTS_DE_CHARGE_LOCUST.md](../backend/docs/GUIDE_TESTS_DE_CHARGE_LOCUST.md)**

---

## 🚀 Démarrage Rapide

### 1. Prérequis
- Python 3.9+ (`python3 --version`)
- Le backend LAB en cours d'exécution (en local sur `http://localhost:8080` ou sur serveur distant).

### 2. Lancement en un clic
Le script `run_load_test.sh` gère automatiquement l'environnement virtuel Python, l'installation des dépendances et l'export des rapports.

```bash
cd load-tests

# Menu interactif
./run_load_test.sh
```

---

## 🛠️ Options de Ligne de Commande

| Commande | Description | Durée / Volume |
| :--- | :--- | :--- |
| `./run_load_test.sh --smoke` | **Smoke test :** validation rapide du script (10 users) | 30 secondes |
| `./run_load_test.sh --rampup` | **Montée en charge :** test intermédiaire (100 users) | 3 minutes |
| `./run_load_test.sh --stress` | **Test de performance cible :** 100 utilisateurs | 5 minutes |
| `./run_load_test.sh --stress-500` | **Stress test :** test de haute intensité (500 utilisateurs) | 5 minutes |
| `./run_load_test.sh --volume-500k` | **Endurance 500k :** injection continue grand volume | ~15 minutes |
| `./run_load_test.sh --ui` | **Interface Web :** ouvre Locust sur `http://localhost:8089` | Interactif |

---

## 🌐 Cibler un Serveur Distant / Isolé

Pour lancer les tests contre l'instance isolée fournie par l'Infra (ex: `http://10.70.1.120:8080`) :

```bash
TARGET_HOST="http://10.70.1.120:8080" ./run_load_test.sh --stress
```

---

## 📊 Rapports Générés

Tous les rapports d'exécution sont horodatés et automatiquement sauvegardés dans le dossier `reports/` :
- **Rapport HTML interactif :** `reports/report_<scenario>_<date>.html` (graphiques RPS, latence, percentiles, distribution des codes HTTP).
- **Fichiers CSV :** `reports/csv_<scenario>_<date>_stats.csv` pour intégration dans Excel ou outils BI.
