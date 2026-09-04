"""
Suite de tests de charge Locust pour le Module LAB (Moteur de Détection d'Anomalies)
Documentation et exécution : voir README.md et GUIDE_COMPLET_TESTS_DE_CHARGE.md
"""

import os
import json
import random
from locust import HttpUser, task, between, events
from data_generator import generate_transaction_payload, DEFAULT_PARTNER_BANK_CODE

# Configuration de base
DEFAULT_HOST = os.getenv("TARGET_HOST", "http://localhost:8080")
PARTNER_BANK_CODE = os.getenv("PARTNER_BANK_CODE", DEFAULT_PARTNER_BANK_CODE)

ADMIN_EMAIL = os.getenv("TEST_ADMIN_EMAIL", "admin@gsie.sn")
ADMIN_PASSWORD = os.getenv("TEST_ADMIN_PASSWORD", "admin123")

ANALYST_EMAIL = os.getenv("TEST_ANALYST_EMAIL", "analyste@gsie.sn")
ANALYST_PASSWORD = os.getenv("TEST_ANALYST_PASSWORD", "analyste123")


class BaseAuthenticatedUser(HttpUser):
    """Classe de base gérant l'authentification JWT pour les utilisateurs de test"""
    abstract = True
    host = DEFAULT_HOST
    token = None
    email = ADMIN_EMAIL
    password = ADMIN_PASSWORD

    def on_start(self):
        """Authentification au démarrage de l'utilisateur virtuel"""
        self.authenticate()

    def authenticate(self):
        """Récupère le token JWT d'accès"""
        payload = {
            "email": self.email,
            "password": self.password
        }
        with self.client.post(
            "/api/v1/auth/login",
            json=payload,
            name="[Auth] POST /api/v1/auth/login",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    self.token = data.get("access_token")
                except Exception as e:
                    response.failure(f"Impossible de parser le token: {e}")  # type: ignore
            else:
                response.failure(f"Échec authentification (code {response.status_code}): {response.text}")  # type: ignore

    @property
    def auth_headers(self):
        """En-têtes HTTP avec token Bearer"""
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        return headers


class CoreBankingLoadUser(BaseAuthenticatedUser):
    """
    Simule le trafic émis par le Core Banking / Système partenaire (SPI)
    - Poids élevé (90% du trafic global)
    - Envoi massif de transactions financières en temps réel
    - Analyse par le moteur de règles Go + écriture en base PostgreSQL
    """
    weight = 9
    wait_time = between(0.05, 0.3)  # Intervalle court pour simuler un fort débit
    email = ADMIN_EMAIL
    password = ADMIN_PASSWORD

    @task(7)
    def analyze_standard_transaction(self):
        """Analyse de transaction financière standard"""
        tx = generate_transaction_payload(partner_bank_code=PARTNER_BANK_CODE)
        with self.client.post(
            "/api/v1/anomalies/analyze",
            json=tx,
            headers=self.auth_headers,
            name="[Core] POST /api/v1/anomalies/analyze (Standard)",
            catch_response=True
        ) as response:
            # 200 (pas d'anomalie), 201 (alerte générée) ou 403 (transaction refusée) sont des statuts fonctionnels valides
            if response.status_code in (200, 201, 403):
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore

    @task(2)
    def analyze_high_amount_transaction(self):
        """Analyse de transaction à montant très élevé (déclenche seuil montant)"""
        tx = generate_transaction_payload(
            partner_bank_code=PARTNER_BANK_CODE,
            force_anomaly_type="high_amount"
        )
        with self.client.post(
            "/api/v1/anomalies/analyze",
            json=tx,
            headers=self.auth_headers,
            name="[Core] POST /api/v1/anomalies/analyze (High Amount)",
            catch_response=True
        ) as response:
            if response.status_code in (200, 201, 403):
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore

    @task(1)
    def analyze_risk_country_transaction(self):
        """Analyse de transaction impliquant un pays à haut risque"""
        tx = generate_transaction_payload(
            partner_bank_code=PARTNER_BANK_CODE,
            force_anomaly_type="risk_country"
        )
        with self.client.post(
            "/api/v1/anomalies/analyze",
            json=tx,
            headers=self.auth_headers,
            name="[Core] POST /api/v1/anomalies/analyze (Risk Country)",
            catch_response=True
        ) as response:
            if response.status_code in (200, 201, 403):
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore


class AnalystDashboardUser(BaseAuthenticatedUser):
    """
    Simule les analystes et superviseurs consultant le back-office / dashboard
    - Poids plus faible (10% du trafic)
    - Requêtes de lecture, pagination et calculs de métriques d'alertes
    """
    weight = 1
    wait_time = between(1.0, 3.0)  # Temps de réflexion humaine
    email = ANALYST_EMAIL
    password = ANALYST_PASSWORD

    @task(4)
    def get_anomalies_list(self):
        """Consultation de la liste paginée des alertes"""
        page = random.randint(1, 5)
        with self.client.get(
            f"/api/v1/anomalies?page={page}&limit=20",
            headers=self.auth_headers,
            name="[Dashboard] GET /api/v1/anomalies (List)",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore

    @task(3)
    def get_alert_metrics(self):
        """Consultation des métriques d'alertes temps réel"""
        with self.client.get(
            "/api/v1/anomalies/metrics",
            headers=self.auth_headers,
            name="[Dashboard] GET /api/v1/anomalies/metrics",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore

    @task(2)
    def check_entity_status(self):
        """Vérification rapide du statut de surveillance d'un compte"""
        sample_account = f"{PARTNER_BANK_CODE}{random.randint(10000000, 99999999):08d}01"
        with self.client.get(
            f"/api/v1/entities/{sample_account}/status",
            headers=self.auth_headers,
            name="[Entity] GET /api/v1/entities/:identifier/status",
            catch_response=True
        ) as response:
            if response.status_code in (200, 404):
                response.success()  # type: ignore
            else:
                response.failure(f"Erreur HTTP {response.status_code}: {response.text[:200]}")  # type: ignore
