import { Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from './auth.service';
import { jwtDecode } from 'jwt-decode';

@Injectable({ providedIn: 'root' })
export class AuthStateService {
  isInitialized = signal(false);
  isAuthenticated = signal(false);

  constructor(private router: Router, private auth: AuthService) {}

  async initializeAuth(): Promise<void> {
    try {
      // Vérifier s'il existe un token en localStorage et initialiser l'état d'authentification
      try {
        if (typeof window !== 'undefined' && window.localStorage) {
          const token = localStorage.getItem('token');
          if (token) {
            // valider le claim exp s'il est présent
            let valide = true;
            try {
              const decoded: any = jwtDecode(token);
              if (decoded && decoded.exp) {
                const now = Math.floor(Date.now() / 1000);
                valide = decoded.exp > now;
              }
            } catch (e) {
              valide = false;
            }
            if (valide) {
              this.isAuthenticated.set(true);
            } else {
              try { localStorage.removeItem('token'); } catch (e) {}
              this.isAuthenticated.set(false);
            }
          }
        }
      } catch (e) {
        // ignorer les erreurs de localStorage
      }

      this.isInitialized.set(true);
    } catch (error) {
      console.error('Erreur initialisation auth', error);
      this.isInitialized.set(true);
    }
  }

  logout(): void {
    this.auth.logout().subscribe(() => {});
    this.isAuthenticated.set(false);
    this.router.navigate(['/login']);
  }
}
