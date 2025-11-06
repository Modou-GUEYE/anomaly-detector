import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
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
  constructor(private http: HttpClient) {}

  getRecentTransactions(): Observable<Transaction[]> {
    // L'interceptor ajoute automatiquement le token Bearer
    return this.http.get<Transaction[]>('/api/v1/transactions');
  }
}
