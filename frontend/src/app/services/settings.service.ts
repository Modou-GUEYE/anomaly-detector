import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class SettingsService {
  constructor(private http: HttpClient) {}

  // Fetch settings (may return 404 if backend not implemented)
  getSettings(): Observable<any> {
    return this.http.get('/api/v1/settings');
  }

  // Save a named section
  saveSettings(section: string, payload: any): Observable<any> {
    const body = { section, payload };
    const headers = new HttpHeaders({ 'Content-Type': 'application/json' });
    return this.http.post('/api/v1/settings', body, { headers });
  }
}
