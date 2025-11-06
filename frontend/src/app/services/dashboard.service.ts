import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class DashboardService {

  constructor(private http: HttpClient) { }

  /**
   * Récupère le résumé complet du dashboard en UNE SEULE requête
   * L'interceptor ajoute automatiquement le token en header
   * Contient : KPIs, transactions récentes, alertes récentes, métriques
   */
  getDashboardSummary(): Observable<any> {
    return this.http.get<any>('/api/v1/dashboard/summary');
  }
}
