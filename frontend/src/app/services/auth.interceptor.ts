import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError } from 'rxjs/operators';
import { throwError, of } from 'rxjs';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (request, next) => {
  const auth = inject(AuthService);
  // Autoriser les endpoints d'authentification sans attacher de token
  const url: string = (request && (request.url || '')) as string;
  const isAuthEndpoint = url.includes('/auth/login') || url.includes('/auth/refresh') || url.includes('/auth/logout');
  if (isAuthEndpoint) {
    return next(request);
  }

  // Si on a un token en mémoire ou en localStorage, l'ajouter en header Authorization
  const token = auth.getAccessToken();
  if (token) {
    request = request.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }

  return next(request).pipe(
    catchError((err: any) => {
      const status = err instanceof HttpErrorResponse ? err.status : (err && err.status) || null;
      const url: string = (request && (request.url || '')) as string;
      const isAuthLogin = url.includes('/auth/login') || url.includes('/auth/refresh');
      if (status === 401 && !isAuthLogin) {
        try {
          // vider le token et forcer la redirection vers /login
          auth.logout().subscribe(() => {
            try { window.location.assign('/login'); } catch (e) {}
          });
        } catch (e) {
          // ignore
        }
      }
      return throwError(() => err);
    })
  );
};
