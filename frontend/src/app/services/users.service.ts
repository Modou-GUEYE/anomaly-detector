import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

export interface NewUserPayload {
  nomUtilisateur: string;
  email: string;
  password?: string;
  role: string;
  actif?: boolean;
}

@Injectable({ providedIn: 'root' })
export class UsersService {
  private usersUrl = '/api/v1/users';

  constructor(private http: HttpClient) {}

  createUser(payload: NewUserPayload): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.post(this.usersUrl, payload, { headers });
  }

  getUsers(): Observable<any[]> {
    const headers = this.getAuthHeaders();
    // Normalize date fields so Angular date pipe can format them reliably.
    return this.http.get<any[]>(this.usersUrl, { headers }).pipe(
      map((users = []) =>
        users.map((u: any) => {
          // Accept both backend shapes: date_creation (snake_case) or dateCreation (camelCase)
          const raw = (u && (u.dateCreation || u.date_creation)) as string | undefined;
          if (raw && typeof raw === 'string') {
            let d = new Date(raw);
            if (isNaN(d.getTime())) {
              try {
                let iso = raw.replace(' ', 'T');
                iso = iso.replace(/(\.\d{3})\d+/, '$1');
                d = new Date(iso);
              } catch (e) {
                d = null as any;
              }
            }
            u.dateCreation = d ? d : null;
          }
          return u;
        })
      )
    );
  }

  updateUser(id: string, updates: Partial<NewUserPayload> | any): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.put(`${this.usersUrl}/${id}`, updates, { headers });
  }

  deleteUser(id: string): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.delete(`${this.usersUrl}/${id}`, { headers });
  }

  private getAuthHeaders(): HttpHeaders {
    let headers = new HttpHeaders({ 'Content-Type': 'application/json' });
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        headers = headers.set('Authorization', `Bearer ${token}`);
      }
    }
    return headers;
  }
}
