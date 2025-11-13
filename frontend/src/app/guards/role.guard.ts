import { Injectable } from '@angular/core';
import { CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { AuthService } from '../services/auth.service';

@Injectable({ providedIn: 'root' })
export class RoleGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router) {}

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): boolean {
    const forbidden: string[] = (route.data && route.data['forbiddenRoles']) || [];
    const role = this.authService.getUserRole()?.toLowerCase() || null;

    if (role && forbidden.map(r => r.toLowerCase()).includes(role)) {
      // Rediriger vers le tableau de bord si le rôle n'a pas accès
      this.router.navigate(['/dashboard']);
      return false;
    }

    return true;
  }
}
