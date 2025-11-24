# Single-stage Dockerfile for anomaly-detector frontend
FROM node:20-alpine

# Install nginx
RUN apk add --no-cache nginx

# Set working directory
WORKDIR /app

# Copy package files and install dependencies
COPY package.json package-lock.json* ./
COPY .npmrc* ./
RUN npm ci --legacy-peer-deps || npm install --legacy-peer-deps

# Copy source code and build
COPY . .
RUN npm run build --silent

# Copy built files to nginx directory
RUN mkdir -p /usr/share/nginx/html && \
    cp -r /app/dist/* /usr/share/nginx/html/

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

# Run nginx
CMD ["nginx", "-g", "daemon off;"]
