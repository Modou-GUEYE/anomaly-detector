import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
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
  private anomaliesUrl = '/api/v1/anomalies';
  private metricsUrl = '/api/v1/anomalies/metrics';

  constructor(private http: HttpClient) {}

  /**
   * Récupérer toutes les anomalies/alertes
   * Endpoint: GET /api/v1/anomalies
   */
  getAnomalies(): Observable<Alert[]> {
    const headers = this.getAuthHeaders();
    return this.http.get<Alert[]>(this.anomaliesUrl, { headers });
  }

  /**
   * Mettre à jour le statut d'une alerte
   * Endpoint: PUT /api/v1/anomalies/:id/status
   */
  updateAnomalyStatus(id: string, statut: string): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.put(`${this.anomaliesUrl}/${id}/status`, { statut }, { headers });
  }

  /**
   * Récupérer les métriques des anomalies
   * Endpoint: GET /api/v1/anomalies/metrics
   */
  getMetrics(): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.get<any>(this.metricsUrl, { headers });
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
