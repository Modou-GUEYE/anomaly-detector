# Multi-stage build: build Angular app with Node, serve with nginx
FROM node:20-alpine AS builder
WORKDIR /app

# Install dependencies and build
COPY package.json package-lock.json* ./
COPY .npmrc* ./
RUN npm ci --legacy-peer-deps || npm install --legacy-peer-deps
COPY . .
RUN npm run build --silent

# Stage: nginx to serve built files and proxy /api to backend service
FROM nginx:1.25-alpine
COPY --from=builder /app/dist /usr/share/nginx/html

# Default nginx config included in repo at frontend/nginx.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
