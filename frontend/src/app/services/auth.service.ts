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
  private readonly loginUrl = '/api/v1/auth/login';

  private accessToken: string | null = null;

  constructor(private readonly http: HttpClient, private readonly config: ConfigService) {
    // Charger le token depuis localStorage au démarrage (si présent).
    this.loadTokenFromStorage();
  }

  private loadTokenFromStorage() {
    if (globalThis.window?.sessionStorage) {
      const t = globalThis.window.sessionStorage.getItem('token');
      if (t) this.accessToken = t;
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
        if (globalThis.window?.sessionStorage) {
          globalThis.window.sessionStorage.setItem('token', res.access_token);
        }
      }),
      map(() => true),
      catchError(() => of(false))
    );
  }

  // Déconnexion : suppression du token côté client
  logout(): Observable<boolean> {
    this.accessToken = null;
    if (globalThis.window?.sessionStorage) {
      globalThis.window.sessionStorage.removeItem('token');
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
      if (decoded?.exp) {
        const now = Math.floor(Date.now() / 1000);
        return decoded.exp > now;
      }
      // If no exp claim, treat as valid (legacy tokens)
      return true;
    } catch {
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
    } catch {
      return [];
    }
  }

 // returne le premier rôle (ou null)
  getUserRole(): string | null {
    const roles = this.getUserRoles();
    return roles.length > 0 ? roles[0] : null;
  }

  // Vérifie si l'utilisateur doit changer son mot de passe
  mustChangePassword(): boolean {
    const token = this.getAccessToken();
    if (!token) return false;
    try {
      const decoded: any = jwtDecode(token);
      return decoded?.must_change_password === true;
    } catch {
      return false;
    }
  }

  // Changer le mot de passe lors de la première connexion
  changeFirstPassword(oldPassword: string, newPassword: string): Observable<boolean> {
    const url = '/api/v1/auth/first-password-change';
    const body = { oldPassword, newPassword };
    return this.http.post<{ access_token: string; expires_in: number; message: string }>(url, body).pipe(
      tap((res) => {
        // Mettre à jour le token avec le nouveau
        this.accessToken = res.access_token;
        if (globalThis.window?.sessionStorage) {
          globalThis.window.sessionStorage.setItem('token', res.access_token);
        }
      }),
      map(() => true),
      catchError(() => of(false))
    );
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
    const expected = this.config.getRoleName(role);
    return roles.includes(expected.toLowerCase());
  }
}
