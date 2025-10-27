import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private loginUrl = '/api/v1/auth/login'; 
  private tokenUrl = '/api/v1/auth/token';

  constructor(private http: HttpClient) {}

  /**
   * Authentification utilisateur (email + password)
   * Endpoint: POST /api/v1/auth/login
   */
  login(email: string, password: string): Observable<any> {
    return this.http.post(this.loginUrl, { email, password });
  }

  /**
   * Authentification système externe (system_id + role)
   * Endpoint: POST /api/v1/auth/token
   */
  getSystemToken(systemId: string, role: string): Observable<any> {
    return this.http.post(this.tokenUrl, { system_id: systemId, role: role });
  }
}
