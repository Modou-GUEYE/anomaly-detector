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

type UserUpdatePayload = Partial<NewUserPayload> & Record<string, unknown>;

@Injectable({ providedIn: 'root' })
export class UsersService {
  private readonly usersUrl = '/api/v1/users';

  constructor(private readonly http: HttpClient) {}

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
            let parsedDate = new Date(raw);
            if (Number.isNaN(parsedDate.getTime())) {
              let iso = raw.replace(' ', 'T');
              iso = iso.replace(/(\.\d{3})\d+/, '$1');
              parsedDate = new Date(iso);
            }
            u.dateCreation = Number.isNaN(parsedDate.getTime()) ? null : parsedDate;
          }
          return u;
        })
      )
    );
  }

  updateUser(id: string, updates: UserUpdatePayload): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.put(`${this.usersUrl}/${id}`, updates, { headers });
  }

  deleteUser(id: string): Observable<any> {
    const headers = this.getAuthHeaders();
    return this.http.delete(`${this.usersUrl}/${id}`, { headers });
  }

  private getAuthHeaders(): HttpHeaders {
    let headers = new HttpHeaders({ 'Content-Type': 'application/json' });
    if (globalThis.window?.sessionStorage) {
      const token = globalThis.window.sessionStorage.getItem('token');
      if (token) {
        headers = headers.set('Authorization', `Bearer ${token}`);
      }
    }
    return headers;
  }
}
