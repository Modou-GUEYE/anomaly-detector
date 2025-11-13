import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface Account {
  id: string;
  numeroCompte: string;
  nomClient: string;
  typeClient: string;
  typeCompte: string;
  categorie: string;
  pays: string;
  ville: string;
  adresse: string;
  profilClientId?: string;
  dateCreation: string;
}

export interface SuiviBlacklist {
  id: string;
  transaction_id: string;
  numero_compte: string;
  statut_suivi: string;
  raison: string; // Source du surveillance (PEP, Blacklist, Montant, etc.)
  date_detection: string;
}

export interface PaysRisque {
  id: string;
  code: string; // Code ISO 3166-1 alpha-2 (ex: FR, ML, SN) - OBLIGATOIRE - Compatible SPI
  nomPays: string;
  active: boolean;
  regleControleId?: string;
  createdAt?: string;
  updatedAt?: string;
}

@Injectable({ providedIn: 'root' })
export class ProfileService {
  private apiUrl = '/api/v1/accounts';
  private profileUrl = '/api/v1/profile';
  private suiviUrl = '/api/v1/suivi-blacklist';
  private paysRisqueUrl = '/api/v1/pays-risque';

  constructor(private http: HttpClient) {}

  getProfile(): Observable<any> {
    return this.http.get(this.profileUrl);
  }

  // Récupérer tous les comptes
  getAccounts(): Observable<Account[]> {
    return this.http.get<Account[]>(this.apiUrl);
  }

  // Récupérer le suivi blacklist (surveillance des comptes)
  getSuiviBlacklist(): Observable<SuiviBlacklist[]> {
    return this.http.get<SuiviBlacklist[]>(this.suiviUrl);
  }

  // ===================== PAYS À RISQUE =====================

  // Récupérer tous les pays à risque (retourne les objets complets)
  getAllPaysARisque(): Observable<PaysRisque[]> {
    return this.http.get<PaysRisque[]>(this.paysRisqueUrl);
  }

  // Ajouter un pays à risque (code OBLIGATOIRE)
  createPaysARisque(nomPays: string, code: string): Observable<any> {
    if (!code || code.trim().length !== 2) {
      throw new Error('Le code pays ISO alpha-2 est obligatoire (2 caractères)');
    }
    const payload = {
      nom_pays: nomPays,
      code: code.toUpperCase()
    };
    return this.http.post(this.paysRisqueUrl, payload);
  }

  // Supprimer un pays à risque
  deletePaysARisque(nomPays: string): Observable<any> {
    return this.http.delete(`${this.paysRisqueUrl}/${nomPays}`);
  }

  // Mettre à jour un pays à risque (code OBLIGATOIRE)
  updatePaysARisque(ancienNom: string, nouveauNom: string, code: string): Observable<any> {
    if (!code || code.trim().length !== 2) {
      throw new Error('Le code pays ISO alpha-2 est obligatoire (2 caractères)');
    }
    const payload = {
      nom_pays: nouveauNom,
      code: code.toUpperCase()
    };
    return this.http.put(`${this.paysRisqueUrl}/${ancienNom}`, payload);
  }

  // Importer des pays à partir d'un fichier
  importPaysARisque(file: File): Observable<any> {
    const formData = new FormData();
    formData.append('file', file);
    return this.http.post(`${this.paysRisqueUrl}/import`, formData);
  }
}

