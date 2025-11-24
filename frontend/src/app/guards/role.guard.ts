import { Injectable } from '@angular/core';
import { CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { ConfigService } from '../services/config.service';

@Injectable({ providedIn: 'root' })
export class RoleGuard implements CanActivate {
  constructor(private authService: AuthService, private router: Router, private config: ConfigService) {}

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): boolean {
    const rawAllowed: string[] | undefined = (route.data && route.data['allowedRoles']) || undefined;
    const rawForbidden: string[] = (route.data && route.data['forbiddenRoles']) || [];
    // Map logical role keys to configured role names when possible
    const allowed = rawAllowed ? rawAllowed.map(r => {
      try { return this.config.getRoleName(r); } catch (e) { return String(r).toLowerCase(); }
    }).map(x => x.toLowerCase()) : undefined;
    const forbidden = rawForbidden.map(r => {
      try { return this.config.getRoleName(r); } catch (e) { return String(r).toLowerCase(); }
    }).map(x => x.toLowerCase());
    const role = this.authService.getUserRole()?.toLowerCase() || null;

    // Si la route définit explicitement des rôles autorisés, n'autoriser que ceux-ci
    if (allowed && Array.isArray(allowed)) {
      if (!role || !allowed.includes(role)) {
        this.router.navigate(['/dashboard']);
        return false;
      }
      return true;
    }

    // Sinon, vérifier la liste des rôles interdits (compatibilité ascendante)
    if (role && forbidden.includes(role)) {
      this.router.navigate(['/dashboard']);
      return false;
    }

    return true;
  }
}
