#!/usr/bin/env bash

# ==============================================================================
# Script de lancement automatisé des tests de charge Locust (Module LAB)
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"
TIMESTAMP="$(date +'%Y%m%d_%H%M%S')"
VENV_DIR="${SCRIPT_DIR}/.venv"

TARGET_HOST="${TARGET_HOST:-http://localhost:8080}"
PARTNER_BANK_CODE="${PARTNER_BANK_CODE:-SN079}"

# Couleurs pour le terminal
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

mkdir -p "${REPORTS_DIR}"

echo -e "${BLUE}================================================================${NC}"
echo -e "${BLUE}        MODULE LAB - SUITE DE TESTS DE CHARGE LOCUST           ${NC}"
echo -e "${BLUE}================================================================${NC}"
echo -e "Cible (Host)       : ${GREEN}${TARGET_HOST}${NC}"
echo -e "Code Partenaire    : ${GREEN}${PARTNER_BANK_CODE}${NC}"
echo -e "Dossier Rapports   : ${GREEN}${REPORTS_DIR}${NC}"
echo -e "----------------------------------------------------------------"

# Vérification ou création de l'environnement virtuel Python
if [ ! -d "${VENV_DIR}" ]; then
    echo -e "${YELLOW}→ Initialisation de l'environnement virtuel Python (.venv)...${NC}"
    python3 -m venv "${VENV_DIR}"
fi

source "${VENV_DIR}/bin/activate"

# Installation / vérification des dépendances
if ! command -v locust &> /dev/null; then
    echo -e "${YELLOW}→ Installation des dépendances Locust, Requests, Faker...${NC}"
    pip install -q --upgrade pip
    pip install -q -r "${SCRIPT_DIR}/requirements.txt"
fi

show_help() {
    echo "Usage: ./run_load_test.sh [OPTION]"
    echo ""
    echo "Options:"
    echo "  --smoke       : Test court de validation (10 users, 30 secondes)"
    echo "  --rampup      : Montée en charge progressive (100 users, 3 minutes)"
    echo "  --stress      : Test de performance cible (100 users, 5 minutes)"
    echo "  --stress-500  : Test de stress poussé (500 users, 5 minutes)"
    echo "  --volume-500k : Simulation gros volume (injection intensive continue)"
    echo "  --ui          : Lancement de l'interface Web interactive Locust (port 8089)"
    echo "  --help        : Affiche cette aide"
    echo ""
    echo "Sans argument, un menu interactif est proposé."
}

run_test() {
    local users=$1
    local spawn_rate=$2
    local duration=$3
    local scenario_name=$4

    local report_html="${REPORTS_DIR}/report_${scenario_name}_${TIMESTAMP}.html"
    local report_csv_prefix="${REPORTS_DIR}/csv_${scenario_name}_${TIMESTAMP}"

    echo -e "${GREEN}→ Démarrage du scénario : ${scenario_name}${NC}"
    echo -e "  - Utilisateurs virtuels : ${users}"
    echo -e "  - Taux d'arrivée (spawn) : ${spawn_rate} users/s"
    echo -e "  - Durée du tir          : ${duration}"
    echo -e "  - Rapport HTML          : ${report_html}"
    echo ""

    locust \
        -f "${SCRIPT_DIR}/locustfile.py" \
        --headless \
        --host="${TARGET_HOST}" \
        --users="${users}" \
        --spawn-rate="${spawn_rate}" \
        --run-time="${duration}" \
        --html="${report_html}" \
        --csv="${report_csv_prefix}"

    echo ""
    echo -e "${GREEN}✓ Tir de charge terminé avec succès !${NC}"
    echo -e "Rapport HTML généré : ${BLUE}file://${report_html}${NC}"
}

run_web_ui() {
    echo -e "${GREEN}→ Lancement de l'interface Web Locust...${NC}"
    echo -e "Ouvrez votre navigateur sur : ${BLUE}http://localhost:8089${NC}"
    echo -e "Hôte ciblé par défaut      : ${TARGET_HOST}"
    echo ""
    locust -f "${SCRIPT_DIR}/locustfile.py" --host="${TARGET_HOST}"
}

# Gestion des arguments en ligne de commande
case "$1" in
    --smoke)
        run_test 10 2 "30s" "smoke_test"
        exit 0
        ;;
    --rampup)
        run_test 100 10 "3m" "rampup_100u"
        exit 0
        ;;
    --stress)
        run_test 100 10 "5m" "stress_100u"
        exit 0
        ;;
    --stress-500)
        run_test 500 25 "5m" "stress_500u"
        exit 0
        ;;
    --volume-500k)
        run_test 250 50 "15m" "volume_500k_target"
        exit 0
        ;;
    --ui)
        run_web_ui
        exit 0
        ;;
    --help|-h)
        show_help
        exit 0
        ;;
esac

# Menu interactif si aucun argument
echo "Choisissez un scénario de test :"
echo "  1) Smoke Test (10 users, 30s) - Validation rapide"
echo "  2) Montée en charge (100 users, 3 min) - Test intermédiaire"
echo "  3) Test de performance (100 users, 5 min) - Cible Cheikh"
echo "  4) Volume 500k (250 users, 15 min) - Simulation gros volume"
echo "  5) Stress Test (500 users, 5 min) - Haute intensité"
echo "  6) Interface Web interactive (Locust UI sur port 8089)"
echo "  7) Quitter"
echo ""
read -p "Votre choix (1-7) : " choice

case "$choice" in
    1) run_test 10 2 "30s" "smoke_test" ;;
    2) run_test 100 10 "3m" "rampup_100u" ;;
    3) run_test 100 10 "5m" "stress_100u" ;;
    4) run_test 250 50 "15m" "volume_500k_target" ;;
    5) run_test 500 25 "5m" "stress_500u" ;;
    6) run_web_ui ;;
    7) echo "Arrêt."; exit 0 ;;
    *) echo -e "${RED}Choix invalide.${NC}"; exit 1 ;;
esac
