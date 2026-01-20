import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { tap, map, catchError } from 'rxjs/operators';
import { jwtDecode } from 'jwt-decode';
import { ConfigService } from './config.service';

interface LoginResp {
  access_token: string;
  expires_in: number;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private loginUrl = '/api/v1/auth/login';

  private accessToken: string | null = null;

  constructor(private http: HttpClient, private config: ConfigService) {
    // Charger le token depuis localStorage au démarrage (si présent).
    this.loadTokenFromStorage();
  }

  private loadTokenFromStorage() {
    try {
      if (typeof window !== 'undefined' && window.sessionStorage) {
        const t = sessionStorage.getItem('token');
        if (t) this.accessToken = t;
      }
    } catch (e) {
      // ignore storage errors (e.g. private mode)
    }
  }

  // Se connecter avec email/mot de passe auprès du backend
  login(email: string, password: string, role?: string): Observable<boolean> {
    const body: any = { email, password };
    if (role) body.role = role;
    return this.http.post<LoginResp>(this.loginUrl, body, { withCredentials: true }).pipe(
      tap((res) => {
        // Sauvegarder le token en mémoire et en sessionStorage pour les rechargements de page
        this.accessToken = res.access_token;
        try {
          if (typeof window !== 'undefined' && window.sessionStorage) {
            sessionStorage.setItem('token', res.access_token);
          }
        } catch (e) {
          // ignore storage errors
        }
      }),
      map(() => true),
      catchError(() => of(false))
    );
  }

  // Déconnexion : suppression du token côté client
  logout(): Observable<boolean> {
    this.accessToken = null;
    try {
      sessionStorage.removeItem('token');
    } catch (e) {
      // ignore storage errors
    }
    return of(true);
  }

  getAccessToken(): string | null {
    return this.accessToken;
  }

  isLoggedIn(): boolean {
    const token = this.getAccessToken();
    if (!token) return false;
    return this.isTokenValid(token);
  }


  isTokenValid(token?: string | null): boolean {
    const t = token || this.getAccessToken();
    if (!t) return false;
    try {
      const decoded: any = jwtDecode(t);
      if (decoded && decoded.exp) {
        const now = Math.floor(Date.now() / 1000);
        return decoded.exp > now;
      }
      // If no exp claim, treat as valid (legacy tokens)
      return true;
    } catch (e) {
      return false;
    }
  }

  // Decode user role from token if present
  // Retourne la liste des rôles présents dans le token (normalisé en minuscules).
  getUserRoles(): string[] {
    const token = this.getAccessToken();
    if (!token) return [];
    try {
      const decoded: any = jwtDecode(token);
      if (!decoded) return [];
      // Prefer 'roles' claim (array), fallback to singular 'role'
      if (Array.isArray(decoded.roles)) {
        return decoded.roles.map((r: any) => String(r).toLowerCase());
      }
      if (decoded.role) {
        // some tokens use `role` as string
        return [String(decoded.role).toLowerCase()];
      }
      // Some tokens might encode roles under different keys (e.g., 'groups') —
      // keep it minimal here and return empty if nothing found.
      return [];
    } catch (e) {
      return [];
    }
  }

 // returne le premier rôle (ou null)
  getUserRole(): string | null {
    const roles = this.getUserRoles();
    return roles.length > 0 ? roles[0] : null;
  }

  isAdmin(): boolean {
    return this.hasRole('admin');
  }

  isAnalyst(): boolean {
    return this.hasRole('analyst');
  }

  // Utilitaire pour vérifier la présence d'un rôle (normalisé en lowercase)
  hasRole(role: string): boolean {
    if (!role) return false;
    const roles = this.getUserRoles();
    // If ConfigService has a mapping for this logical role, prefer that name
    try {
      const expected = this.config ? this.config.getRoleName(role) : String(role).toLowerCase();
      return roles.includes(expected.toLowerCase());
    } catch (e) {
      return roles.includes(String(role).toLowerCase());
    }
  }
}
