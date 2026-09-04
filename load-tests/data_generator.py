"""
Générateur de données de transactions financières pour tests de charge Locust
Module LAB - Moteur de Détection d'Anomalies
"""

import random
from datetime import datetime, timezone
from typing import Optional
from faker import Faker

fake = Faker("fr_FR")

# Profils clients conformes à la base de données
CLIENT_TYPES = ["Particulier", "Entreprise", "Commerçant", "Business"]
ACCOUNT_TYPES = ["Courant", "Epargne", "Professionnel", "Mobile Money"]
CATEGORIES = ["Standard", "VIP", "Premium", "Institutionnel", "PME"]
SCOPES = ["Nationale", "UEMOA", "Internationale"]

# Pays usuels (UEMOA + International) et pays à haut risque
COUNTRIES_STANDARD = ["SN", "CI", "ML", "BF", "TG", "BJ", "NE", "GW", "FR", "US"]
COUNTRIES_HIGH_RISK = ["IR", "KP", "SY", "YE", "MM"]

# Banques partenaires et tiers
DEFAULT_PARTNER_BANK_CODE = "SN079"
OTHER_BANK_CODES = ["SN012", "SN045", "CI022", "ML088", "BF019"]

def generate_account_number(bank_code: str = DEFAULT_PARTNER_BANK_CODE) -> str:
    """Génère un numéro de compte conforme (ex: SN079-0123456789-01)"""
    account_seq = f"{random.randint(10000000, 99999999):08d}"
    rib_key = f"{random.randint(10, 99):02d}"
    return f"{bank_code}{account_seq}{rib_key}"

def generate_transaction_payload(
    partner_bank_code: str = DEFAULT_PARTNER_BANK_CODE,
    force_anomaly_type: Optional[str] = None
) -> dict:
    """
    Génère un payload de transaction complet et valide pour /api/v1/anomalies/analyze.
    
    force_anomaly_type:
      - None: transaction standard aléatoire
      - "high_amount": montant très élevé (> seuil standard)
      - "risk_country": pays payeur ou payé à haut risque
      - "refused": compte ciblé pour interdiction
    """
    # Déterminer si le partenaire est payeur ou payé
    partner_is_payer = random.choice([True, False])
    
    if partner_is_payer:
        account_payer = generate_account_number(partner_bank_code)
        other_bank = random.choice(OTHER_BANK_CODES)
        account_payee = generate_account_number(other_bank)
    else:
        other_bank = random.choice(OTHER_BANK_CODES)
        account_payer = generate_account_number(other_bank)
        account_payee = generate_account_number(partner_bank_code)

    # Pays
    country_payer = "SN" if partner_is_payer else random.choice(COUNTRIES_STANDARD)
    country_payee = random.choice(COUNTRIES_STANDARD) if partner_is_payer else "SN"

    # Montant (en FCFA / XOF)
    if force_anomaly_type == "high_amount":
        amount = float(random.randint(50_000_000, 200_000_000))
    else:
        # Montant standard entre 10 000 et 5 000 000 FCFA
        amount = float(random.choice([
            random.randint(5_000, 50_000),
            random.randint(50_000, 500_000),
            random.randint(500_000, 2_500_000),
            random.randint(2_500_000, 10_000_000),
        ]))

    # Cas pays à risque
    if force_anomaly_type == "risk_country":
        if partner_is_payer:
            country_payee = random.choice(COUNTRIES_HIGH_RISK)
        else:
            country_payer = random.choice(COUNTRIES_HIGH_RISK)

    # Date ISO 8601 UTC
    now_iso = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    scope = "Nationale" if country_payer == country_payee else "Internationale"

    return {
        "date_operation": now_iso,
        "montant": amount,
        "portee": scope,
        # Données Payeur
        "pays_payeur": country_payer,
        "ville_payeur": "Dakar" if country_payer == "SN" else fake.city(),
        "adresse_payeur": fake.street_address(),
        "type_compte_payeur": random.choice(ACCOUNT_TYPES),
        "type_client_payeur": random.choice(CLIENT_TYPES),
        "categorie_payeur": random.choice(CATEGORIES),
        "numero_compte_payeur": account_payer,
        "nom_payeur": fake.company() if "Entreprise" in CLIENT_TYPES else fake.name(),
        # Données Bénéficiaire / Payé
        "pays_paye": country_payee,
        "ville_paye": "Dakar" if country_payee == "SN" else fake.city(),
        "adresse_paye": fake.street_address(),
        "type_compte_paye": random.choice(ACCOUNT_TYPES),
        "type_client_paye": random.choice(CLIENT_TYPES),
        "categorie_paye": random.choice(CATEGORIES),
        "numero_compte_paye": account_payee,
        "nom_paye": fake.company() if "Entreprise" in CLIENT_TYPES else fake.name(),
        "raw_data": {
            "channel": random.choice(["WEB_BANKING", "MOBILE_APP", "AGENCY", "ATM", "SPI_SYSTEM"]),
            "reference": f"TX-{random.randint(100000000, 999999999)}",
            "currency": "XOF"
        }
    }
