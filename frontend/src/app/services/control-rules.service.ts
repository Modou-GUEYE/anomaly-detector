import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface ControlRule {
  id: string;
  nomRegle: string;
  description: string;
  typeRegle: string;
  active: boolean;
  seuil?: number;
  profilClientId?: string;
  periodeReference?: string;
  action?: string; // Action paramétrable: bloquer, surveiller, notifier
  createdAt?: string;
  updatedAt?: string;
}

export interface ProfilClient {
  id: string;
  nomProfil: string;
  description: string;
}

// Interface pour le backend
interface BackendRule {
  id: string;
  nomRegle: string;
  description: string;
  typeRegle: string;
  criticite: string;
  seuil: number;
  periodeReference: string;
  active: boolean;
  profilClientId?: string;
  listePays?: string[];
}

@Injectable({
  providedIn: 'root'
})
export class ControlRulesService {
  private apiUrl = '/api/v1/rules';
  private profilesUrl = '/api/v1/profiles';

  constructor(private http: HttpClient) {}

  /**
   * Pas de mapping - utilise directement les données backend
   */
  private mapBackendToFrontend(backendRule: any): ControlRule {
    return backendRule;
  }

  /**
   * Pas de mapping - utilise directement les données frontend
   */
  private mapFrontendToBackend(frontendRule: Partial<ControlRule>): any {
    return frontendRule;
  }

  /**
   * Récupère toutes les règles de contrôle
   * L'interceptor ajoute automatiquement le token en header
   */
  getRules(): Observable<ControlRule[]> {
    return this.http.get<any[]>(this.apiUrl).pipe(
      map(rules => rules.map(rule => this.mapBackendToFrontend(rule)))
    );
  }

  /**
   * Récupère une règle spécifique par ID
   */
  getRule(id: string): Observable<ControlRule> {
    return this.http.get<any>(`${this.apiUrl}/${id}`).pipe(
      map(rule => this.mapBackendToFrontend(rule))
    );
  }

  /**
   * Crée une nouvelle règle de contrôle
   */
  createRule(rule: Partial<ControlRule>): Observable<ControlRule> {
    const backendData = this.mapFrontendToBackend(rule);
    return this.http.post<any>(this.apiUrl, backendData).pipe(
      map(response => this.mapBackendToFrontend(response))
    );
  }

  /**
   * Met à jour une règle de contrôle existante
   */
  updateRule(id: string, rule: Partial<ControlRule>): Observable<ControlRule> {
    const backendData = this.mapFrontendToBackend(rule);
    return this.http.put<any>(`${this.apiUrl}/${id}`, backendData).pipe(
      map(response => this.mapBackendToFrontend(response))
    );
  }

  /**
   * Supprime une règle de contrôle
   */
  deleteRule(id: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/${id}`);
  }

  /**
   * Active/Désactive une règle
   */
  toggleRuleStatus(id: string, status: 'Actif' | 'Inactif'): Observable<ControlRule> {
    return this.http.patch<any>(`${this.apiUrl}/${id}/status`, { active: status === 'Actif' }).pipe(
      map(response => this.mapBackendToFrontend(response))
    );
  }

  /**
   * Récupère les profils clients disponibles
   */
  getProfilsClient(): Observable<ProfilClient[]> {
    return this.http.get<ProfilClient[]>(this.profilesUrl);
  }

  /**
   * Récupère les statistiques des règles
   */
  getStats(): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/stats`);
  }
}
