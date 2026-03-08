import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface EntiteBlacklist {
  id?: string;
  nom: string;
  numeroCompte: string;
  typeEntite: string;
  informationsSupplementaires?: string;
  dateAjout?: string;
  expanded?: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class BlacklistService {
  private apiUrl = '/api/v1/entities';

  constructor(private http: HttpClient) {}

  private getHeaders() {
    let token = '';
    if (typeof window !== 'undefined' && window.sessionStorage) {
      token = sessionStorage.getItem('token') || '';
    }
    return new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });
  }

  getEntities(): Observable<EntiteBlacklist[]> {
    return this.http.get<EntiteBlacklist[]>(this.apiUrl, {
      headers: this.getHeaders()
    });
  }

  createEntity(entity: EntiteBlacklist): Observable<EntiteBlacklist> {
    return this.http.post<EntiteBlacklist>(this.apiUrl, entity, {
      headers: this.getHeaders()
    });
  }

  updateEntity(id: string, entity: EntiteBlacklist): Observable<EntiteBlacklist> {
    return this.http.put<EntiteBlacklist>(`${this.apiUrl}/${id}`, entity, {
      headers: this.getHeaders()
    });
  }

  deleteEntity(id: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/${id}`, {
      headers: this.getHeaders()
    });
  }
}
