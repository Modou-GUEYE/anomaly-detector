#!/bin/sh
# Script d'injection des variables d'environnement au runtime (Docker)
# À placer dans le Dockerfile frontend comme ENTRYPOINT ou CMD

# Remplace les placeholders dans les fichiers JS buildés par les vraies variables d'env
# Utilisation: docker run -e API_URL=http://10.70.1.119:8080/api/v1 ...

set -e

API_URL="${API_URL:-http://10.70.1.119:8080/api/v1}"

echo "🔧 Injecting runtime environment variables..."
echo "   API_URL=${API_URL}"

# Créer un fichier env.js dynamique
cat > /usr/share/nginx/html/assets/env.js <<EOF
(function(window) {
  window.__env = window.__env || {};
  window.__env.apiUrl = '${API_URL}';
}(this));
EOF

echo "✅ Runtime config injected"

# Démarrer nginx
exec nginx -g 'daemon off;'
