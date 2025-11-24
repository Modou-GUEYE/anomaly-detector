// Configuration pour l'environnement de production
export const environment = {
  production: true,
  // URL de base de l'API backend (à injecter via variable d'env au build Docker)
  // Valeur par défaut (équipe infra peut override via window.__env.apiUrl dans index.html)
  apiUrl: (window as any)['__env']?.apiUrl || 'http://10.70.1.119:8080/api/v1',
  // En production finale avec domaine:
  // apiUrl: 'https://api-anomaly.banque.sn/api/v1',
};
