import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Transaction {
  id: string;
  date_operation: string;
  montant: number;
  portee: string;
  pays_payeur: string;
  ville_payeur: string;
  adresse_payeur: string;
  type_compte_payeur: string;
  type_client_payeur: string;
  categorie_payeur: string;
  numero_compte_payeur: string;
  nom_payeur: string;
  pays_paye: string;
  ville_paye: string;
  adresse_paye: string;
  type_compte_paye: string;
  type_client_paye: string;
  categorie_paye: string;
  numero_compte_paye: string;
  nom_paye: string;
}

@Injectable({ providedIn: 'root' })
export class TransactionService {
  private analyzeUrl = '/api/v1/anomalies/analyze';

  constructor(private http: HttpClient) {}

  /**
   * Analyser une transaction
   * Endpoint: POST /api/v1/anomalies/analyze
   */
  analyzeTransaction(transaction: Partial<Transaction>): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.post<any>(this.analyzeUrl, transaction, { headers });
  }

  /**
   * Helper: Récupérer les headers d'authentification
   */
  private getAuthHeaders(): HttpHeaders {
    let headers = new HttpHeaders();
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        headers = headers.set('Authorization', `Bearer ${token}`);
      }
    }
    return headers;
  }
}
