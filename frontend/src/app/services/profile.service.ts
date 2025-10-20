import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class ProfileService {
  private apiUrl = '/api/v1/profile';

  constructor(private http: HttpClient) {}

  getProfile(): Observable<any> {
    let headers = {};
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        headers = { Authorization: `Bearer ${token}` };
      }
    }
    return this.http.get(this.apiUrl, { headers });
  }
}
