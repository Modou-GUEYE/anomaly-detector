import {
  HttpInterceptorFn
} from '@angular/common/http';

export const authInterceptor: HttpInterceptorFn = (request, next) => {
  // Récupère le token du localStorage
  const token = typeof window !== 'undefined' && window.localStorage ? localStorage.getItem('token') : null;

  // Si le token existe, ajoute-le au header Authorization
  if (token) {
    request = request.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  return next(request);
};
