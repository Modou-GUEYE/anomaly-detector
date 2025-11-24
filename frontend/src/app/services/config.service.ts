import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

export interface AppRuntimeConfig {
  // legacy roles mapping (optional)
  roles?: Record<string, string>;
  // system role name (string)
  system?: string;
  // jwt ttl in minutes (string or number) - empty or "0" means no exp
  jwt_ttl_min?: string | number;
  [key: string]: any;
}

@Injectable({ providedIn: 'root' })
export class ConfigService {
  private config: AppRuntimeConfig = {
    roles: {
      admin: 'admin',
      analyst: 'analyst',
      system: 'system'
    },
    system: 'system',
    jwt_ttl_min: ''
  };
  private loaded = false;

  constructor(private http: HttpClient) {}

  // Charge la config runtime depuis le backend. Doit être appelée au démarrage
  // via APP_INITIALIZER pour que le reste de l'app puisse accéder aux noms de rôle.
  async load(): Promise<void> {
    if (this.loaded) return;
    try {
      const resp = await firstValueFrom(this.http.get<AppRuntimeConfig>('/api/v1/config'));
      if (resp) {
        // merge roles si présent (backward compat)
        if (resp.roles) this.config.roles = { ...this.config.roles, ...resp.roles };
        // charger system role si fourni
        if ((resp as any).system) this.config.system = String((resp as any).system);
        // charger jwt ttl si fourni
        if ((resp as any).jwt_ttl_min !== undefined) this.config.jwt_ttl_min = (resp as any).jwt_ttl_min;
        // merge autres clefs si nécessaire
        Object.keys(resp).forEach((k) => {
          if (k !== 'roles' && k !== 'system' && k !== 'jwt_ttl_min') (this.config as any)[k] = (resp as any)[k];
        });
      }
    } catch (e) {
      // en dev, tolérer l'échec et utiliser les valeurs par défaut
      console.warn('ConfigService: failed to load runtime config, using defaults');
    }
    this.loaded = true;
  }

  getRoles(): Record<string, string> {
    return this.config.roles || {};
  }

  // Retourne le nom de rôle configuré pour une clé logique (ex: 'admin').
  // Si absent, renvoie la clé elle-même (fallback safe).
  getRoleName(key: string): string {
    if (!key) return '';
    const r = this.getRoles();
    return (r[key] || key).toLowerCase();
  }

  // Retourne le nom du rôle système (runtime) — fallback 'system'
  getSystemRoleName(): string {
    const viaSystem = (this.config as any)['system'];
    const viaRoles = this.config.roles ? this.config.roles['system'] : undefined;
    return (viaSystem || viaRoles || 'system').toString().toLowerCase();
  }

  // Retourne la TTL des tokens d'accès (minutes) telle que fournie par le backend
  // Peut être une chaine vide or "0" signifiant pas d'exp
  getJwtTtlMin(): string | number {
    return this.config.jwt_ttl_min || '';
  }

  isLoaded(): boolean {
    return this.loaded;
  }
}
