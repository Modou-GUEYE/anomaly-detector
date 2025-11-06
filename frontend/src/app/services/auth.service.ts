import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { tap } from 'rxjs/operators';
import { jwtDecode } from 'jwt-decode';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private apiUrl = '/api/v1/auth/login'; 

  constructor(private http: HttpClient) {}

  login(email: string, password: string): Observable<any> {
    return this.http.post(this.apiUrl, { email, password }).pipe(
      tap((response: any) => {
        if (response.token) {
          if (typeof window !== 'undefined' && window.localStorage) {
            localStorage.setItem('token', response.token);
          }
        }
      })
    );
  }

  logout(): void {
    if (typeof window !== 'undefined' && window.localStorage) {
      localStorage.removeItem('token');
    }
  }

  getToken(): string | null {
    if (typeof window !== 'undefined' && window.localStorage) {
      return localStorage.getItem('token');
    }
    return null;
  }

  isLoggedIn(): boolean {
    return this.getToken() !== null;
  }

  /**
   * Récupère le rôle de l'utilisateur depuis le token JWT
   */
  getUserRole(): string | null {
    const token = this.getToken();
    if (!token) return null;
    
    try {
      const decoded: any = jwtDecode(token);
      return decoded.role || null;
    } catch (error) {
      console.error('Erreur décodage JWT:', error);
      return null;
    }
  }

  /**
   * Vérifie si l'utilisateur est admin
   */
  isAdmin(): boolean {
    const role = this.getUserRole();
    return role?.toLowerCase() === 'admin';
  }

  /**
   * Vérifie si l'utilisateur est analyste
   */
  isAnalyst(): boolean {
    const role = this.getUserRole();
    return role?.toLowerCase() === 'analyst';
  }
}
