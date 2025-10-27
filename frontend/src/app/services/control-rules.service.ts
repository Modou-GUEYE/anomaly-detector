import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface ControlRule {
  id: string;
  name: string;
  description: string;
  type: string;
  status: 'Actif' | 'Inactif';
  threshold?: number;
  createdAt?: string;
  updatedAt?: string;
}

@Injectable({
  providedIn: 'root'
})
export class ControlRulesService {
  private apiUrl = '/api/v1/control-rules';

  constructor(private http: HttpClient) {}

  private getHeaders(): HttpHeaders {
    let token = '';
    if (typeof window !== 'undefined' && window.localStorage) {
      token = localStorage.getItem('token') || '';
    }
    return new HttpHeaders({
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`
    });
  }

  /**
   * Récupère toutes les règles de contrôle
   */
  getRules(): Observable<ControlRule[]> {
    return this.http.get<ControlRule[]>(this.apiUrl, {
      headers: this.getHeaders()
    });
  }

  /**
   * Récupère une règle spécifique par ID
   */
  getRule(id: string): Observable<ControlRule> {
    return this.http.get<ControlRule>(`${this.apiUrl}/${id}`, {
      headers: this.getHeaders()
    });
  }

  /**
   * Crée une nouvelle règle de contrôle
   */
  createRule(rule: Partial<ControlRule>): Observable<ControlRule> {
    return this.http.post<ControlRule>(this.apiUrl, rule, {
      headers: this.getHeaders()
    });
  }

  /**
   * Met à jour une règle de contrôle existante
   */
  updateRule(id: string, rule: Partial<ControlRule>): Observable<ControlRule> {
    return this.http.put<ControlRule>(`${this.apiUrl}/${id}`, rule, {
      headers: this.getHeaders()
    });
  }

  /**
   * Supprime une règle de contrôle
   */
  deleteRule(id: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/${id}`, {
      headers: this.getHeaders()
    });
  }

  /**
   * Active/Désactive une règle
   */
  toggleRuleStatus(id: string, status: 'Actif' | 'Inactif'): Observable<ControlRule> {
    return this.http.patch<ControlRule>(`${this.apiUrl}/${id}/status`, { status }, {
      headers: this.getHeaders()
    });
  }

  /**
   * Récupère les statistiques des règles
   */
  getStats(): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/stats`, {
      headers: this.getHeaders()
    });
  }
}
