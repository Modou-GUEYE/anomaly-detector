import { Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';

@Injectable({ providedIn: 'root' })
export class AuthStateService {
  isInitialized = signal(false);
  isAuthenticated = signal(false);

  constructor(private router: Router) {}

  async initializeAuth(): Promise<void> {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        const token = localStorage.getItem('token');
        
        // Si tu as un token, tu es authentifié
        if (token) {
          this.isAuthenticated.set(true);
        } else {
          this.isAuthenticated.set(false);
        }
      }
      
      this.isInitialized.set(true);
    } catch (error) {
      this.isInitialized.set(true);
    }
  }

  logout(): void {
    localStorage.removeItem('token');
    this.isAuthenticated.set(false);
    this.router.navigate(['/login']);
  }
}
