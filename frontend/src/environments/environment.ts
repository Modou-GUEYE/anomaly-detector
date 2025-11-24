// Configuration pour l'environnement de développement
export const environment = {
  production: false,
  // URL de base de l'API backend (modifiable via variable d'env au build ou runtime)
  apiUrl: '/api/v1',  // proxy en dev (voir proxy.conf.json)
  // Pour pointer directement vers le backend en dev sans proxy:
  // apiUrl: 'http://localhost:8080/api/v1',
};
