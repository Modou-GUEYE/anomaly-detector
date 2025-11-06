import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Compte {
  id: string;
  nom_client: string;
  numero_compte: string;
  type_compte: string;
  type_client: string;
  categorie: string;
  pays: string;
  ville: string;
  adresse: string;
  profil_client_id: string;
  date_creation: string;
}

@Injectable({ providedIn: 'root' })
export class CompteService {
  constructor(private http: HttpClient) {}

  /**
   * Récupère la liste de tous les comptes clients
   */
  getAllComptes(): Observable<Compte[]> {
    return this.http.get<Compte[]>('/api/v1/accounts');
  }

  /**
   * Récupère les comptes surveillés
   */
  getMonitoredComptes(): Observable<{ count: number; comptes: string[] }> {
    return this.http.get<{ count: number; comptes: string[] }>('/api/v1/accounts/monitored');
  }

  /**
   * Crée un nouveau compte
   */
  createCompte(compte: Omit<Compte, 'id' | 'date_creation'>): Observable<Compte> {
    return this.http.post<Compte>('/api/v1/accounts', compte);
  }

  /**
   * Récupère les règles d'un compte
   */
  getAccountRules(numeroCompte: string): Observable<any[]> {
    return this.http.get<any[]>(`/api/v1/accounts/${numeroCompte}/rules`);
  }
}
