import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

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

  // Récupérer tous les pays à risque
  getAllPaysARisque(): Observable<string[]> {
    return this.http.get<string[]>(this.paysRisqueUrl);
  }

  // Ajouter un pays à risque
  createPaysARisque(nomPays: string): Observable<any> {
    return this.http.post(this.paysRisqueUrl, { nom_pays: nomPays });
  }

  // Supprimer un pays à risque
  deletePaysARisque(nomPays: string): Observable<any> {
    return this.http.delete(`${this.paysRisqueUrl}/${nomPays}`);
  }

  // Mettre à jour un pays à risque
  updatePaysARisque(ancienNom: string, nouveauNom: string): Observable<any> {
    return this.http.put(`${this.paysRisqueUrl}/${ancienNom}`, { nom_pays: nouveauNom });
  }

  // Importer des pays à partir d'un fichier
  importPaysARisque(file: File): Observable<any> {
    const formData = new FormData();
    formData.append('file', file);
    return this.http.post(`${this.paysRisqueUrl}/import`, formData);
  }
}

