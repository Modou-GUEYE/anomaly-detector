import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Guard pour forcer le changement de mot de passe lors de la première connexion.
 * Redirige vers /first-password-change si l'utilisateur doit changer son mot de passe.
 */
export const firstPasswordChangeGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (authService.mustChangePassword()) {
    // Rediriger vers la page de changement de mot de passe
    router.navigate(['/first-password-change']);
    return false;
  }

  return true;
};
