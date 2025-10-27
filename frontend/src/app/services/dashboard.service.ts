import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class DashboardService {

  constructor(private http: HttpClient) { }

  /**
   * Récupère le résumé complet du dashboard en UNE SEULE requête
   * Retourne: {
   *   transactions_count,
   *   anomalies_count,
   *   anomalies_treated,
   *   monitored_accounts,
   *   metrics,
   *   recent_transactions,
   *   recent_anomalies,
   *   anomalies_by_type
   * }
   */
  getDashboardSummary(): Observable<any> {
    const token = typeof window !== 'undefined' && window.localStorage ? localStorage.getItem('token') : null;
    const headers = token ? new HttpHeaders({ Authorization: `Bearer ${token}` }) : new HttpHeaders();
    return this.http.get<any>('/api/v1/dashboard/summary', { headers });
  }

  /**
   * Récupère la liste complète des anomalies
   */
  getAnomalies(): Observable<any> {
    const token = typeof window !== 'undefined' && window.localStorage ? localStorage.getItem('token') : null;
    const headers = token ? new HttpHeaders({ Authorization: `Bearer ${token}` }) : new HttpHeaders();
    return this.http.get<any>('/api/v1/anomalies', { headers });
  }

  /**
   * Mise à jour du statut d'une anomalie
   */
  updateAnomalyStatus(anomalyId: string, newStatus: string): Observable<any> {
    const token = typeof window !== 'undefined' && window.localStorage ? localStorage.getItem('token') : null;
    const headers = token ? new HttpHeaders({ Authorization: `Bearer ${token}` }) : new HttpHeaders();
    return this.http.put<any>(`/api/v1/anomalies/${anomalyId}/status`, { statut: newStatus }, { headers });
  }
}
