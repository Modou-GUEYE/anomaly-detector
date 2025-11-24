import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { AuthStateService } from '../services/auth-state.service';
import { jwtDecode } from 'jwt-decode';

@Injectable({ providedIn: 'root' })
export class AuthGuard implements CanActivate {
  constructor(
    private router: Router,
    private authStateService: AuthStateService
  ) {}

  async canActivate(): Promise<boolean> {
    // Vérifier qu'on est côté client
    if (typeof window === 'undefined') {
      return true;
    }

    // Attendre que l'initialisation soit complète
    if (!this.authStateService.isInitialized()) {
      await new Promise(resolve => {
        const interval = setInterval(() => {
          if (this.authStateService.isInitialized()) {
            clearInterval(interval);
            resolve(true);
          }
        }, 50);
      });
    }

    // Vérifier d'abord le signal
    if (this.authStateService.isAuthenticated()) {
      return true;
    }

    // Sinon, vérifier directement le localStorage
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        // validate exp claim if present
        try {
          const decoded: any = jwtDecode(token);
          if (decoded && decoded.exp) {
            const now = Math.floor(Date.now() / 1000);
            if (decoded.exp > now) {
              return true;
            }
            // expired -> remove
            try { localStorage.removeItem('token'); } catch (e) {}
            return false;
          }
          // no exp claim -> accept
          return true;
        } catch (e) {
          try { localStorage.removeItem('token'); } catch (e) {}
          return false;
        }
      }
    }

    this.router.navigate(['/login']);
    return false;
  }
}

