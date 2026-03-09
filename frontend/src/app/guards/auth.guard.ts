import { Injectable } from '@angular/core';
import { CanActivate, Router } from '@angular/router';
import { AuthStateService } from '../services/auth-state.service';
import { jwtDecode } from 'jwt-decode';

@Injectable({ providedIn: 'root' })
export class AuthGuard implements CanActivate {
  constructor(
    private readonly router: Router,
    private readonly authStateService: AuthStateService
  ) {}

  async canActivate(): Promise<boolean> {
    if (!this.isBrowser()) {
      return true;
    }

    await this.waitForInitialization();

    if (this.authStateService.isAuthenticated()) {
      return true;
    }

    const token = this.getSessionToken();
    if (token && this.hasValidToken(token)) {
      return true;
    }

    this.router.navigate(['/login']);
    return false;
  }

  private isBrowser(): boolean {
    return globalThis.window !== undefined;
  }

  private async waitForInitialization(): Promise<void> {
    if (this.authStateService.isInitialized()) {
      return;
    }

    await new Promise<void>((resolve) => {
      const interval = setInterval(() => {
        if (this.authStateService.isInitialized()) {
          clearInterval(interval);
          resolve();
        }
      }, 50);
    });
  }

  private getSessionToken(): string | null {
    return globalThis.window?.sessionStorage?.getItem('token') ?? null;
  }

  private clearSessionToken(): void {
    if (globalThis.window?.sessionStorage) {
      globalThis.window.sessionStorage.removeItem('token');
    }
  }

  private hasValidToken(token: string): boolean {
    try {
      const decoded: { exp?: number } = jwtDecode(token);
      if (decoded?.exp) {
        const now = Math.floor(Date.now() / 1000);
        if (decoded.exp > now) {
          return true;
        }
        this.clearSessionToken();
        return false;
      }
      return true;
    } catch {
      this.clearSessionToken();
      return false;
    }
  }
}

