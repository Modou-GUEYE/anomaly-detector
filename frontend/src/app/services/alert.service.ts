import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Alert {
  id: string;
  transactionId: string;
  code: string;
  type: string;
  statut: string;
  message: string;
  niveauCriticite: string;
  scoreRisque: number;
  analysteAssigneID?: string;
  dateGeneration: string;
  action: string;
  numeroCompte: string;
  paysRisque?: string;
  regleDeclencheeID?: string;
}

@Injectable({ providedIn: 'root' })
export class AlertService {
  constructor(private http: HttpClient) {}

  getAlertes(): Observable<Alert[]> {
    let headers = {};
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        headers = { Authorization: `Bearer ${token}` };
      }
    }
    return this.http.get<Alert[]>('/api/v1/alertes', { headers });
  }
}
