# Stage 1: Build the React application
FROM node:22-alpine AS builder

WORKDIR /app

# Copy package descriptors and lockfiles
COPY package*.json ./
RUN npm ci

# Copy full frontend codebase
COPY . .

# Inject Vite environment variables (VITE_API_URL can also be customized at build time)
ENV VITE_API_URL=/api

# Build optimized static assets
RUN npm run build

# Stage 2: Serve using high-performance Nginx SPA server
FROM nginx:alpine

# Copy built HTML/JS/CSS assets to Nginx standard serving directory
COPY --from=builder /app/dist /usr/share/nginx/html

# Overwrite default Nginx server configuration with our custom SPA router
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
