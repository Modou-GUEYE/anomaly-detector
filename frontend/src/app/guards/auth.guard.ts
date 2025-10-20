import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';

@Injectable({ providedIn: 'root' })
export class AuthGuard implements CanActivate {
  constructor(private router: Router) {}

  async canActivate(): Promise<boolean> {
    if (typeof window !== 'undefined' && window.localStorage) {
      const token = localStorage.getItem('token');
      if (token) {
        // Vérification du token côté backend
        try {
          const response = await fetch('/api/v1/profile', {
            headers: { Authorization: `Bearer ${token}` }
          });
          if (response.ok) {
            return true;
          }
        } catch (e) {
          // Erreur réseau ou backend
        }
      }
    }
    this.router.navigate(['/login']);
    return false;
  }
}
