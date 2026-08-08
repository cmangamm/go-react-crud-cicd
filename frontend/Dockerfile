# ============= Build Stage =============
FROM node:20-alpine AS build

# Set working directory
WORKDIR /app

# Copy dependency files first
COPY package*.json ./

# Download & Install exact dependencies from package-lock.json
RUN npm ci

# Copy source code
COPY . .

# Build React application
RUN npm run build

# ============= Runtime Stage ============
FROM nginx:alpine

# Copy react build only from Build Stage
COPY --from=build /app/build /usr/share/nginx/html

# Copy nginx.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Nginx listens on port 80
EXPOSE 80

# Run Nginx Application
CMD ["nginx", "-g", "daemon off;"]
