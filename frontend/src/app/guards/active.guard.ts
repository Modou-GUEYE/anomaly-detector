import { Injectable } from '@angular/core';
import { CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
import { ProfileService } from '../services/profile.service';

@Injectable({ providedIn: 'root' })
export class ActiveGuard implements CanActivate {
  constructor(private profileService: ProfileService, private router: Router) {}

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> {
    return this.profileService.getProfile().pipe(
      map((profile: any) => {
        // If profile exists and account is explicitly inactive, block navigation
        if (profile && profile.actif === false) {
          // Allow access to dashboard even if account is deactivated
          const isDashboard = state.url === '/dashboard' || state.url.startsWith('/dashboard');
          if (isDashboard) {
            return true;
          }
          // Redirect to dashboard
          try {
            this.router.navigate(['/dashboard']);
          } catch (e) {}
          return false;
        }
        return true;
      }),
      catchError(() => {
        // On error retrieving profile, deny access and redirect to login
        try {
          this.router.navigate(['/login']);
        } catch (e) {}
        return of(false);
      })
    );
  }
  
  canActivateChild(childRoute: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> {
    return this.profileService.getProfile().pipe(
      map((profile: any) => {
        // Allow access to dashboard even if account is deactivated
        const isDashboard = state.url === '/dashboard' || state.url.startsWith('/dashboard');
        if (profile && profile.actif === false) {
          if (isDashboard) {
            return true;
          }
          // Redirect to dashboard
          try {
            this.router.navigate(['/dashboard']);
          } catch (e) {}
          return false;
        }
        return true;
      }),
      catchError(() => {
        try { this.router.navigate(['/login']); } catch (e) {}
        return of(false);
      })
    );
  }
}
