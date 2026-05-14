# Build stage
FROM node:22-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# Build-time injection of the backend's X-API-Key so the SPA can hit
# /api/* endpoints without asking the end-user to paste a server-side
# secret. Operator passes DEAL_SENSE_API_KEY in `.env` (or `docker
# compose build --build-arg`), this gets baked into the bundle as
# import.meta.env.VITE_DEAL_SENSE_API_KEY.
ARG VITE_DEAL_SENSE_API_KEY=""
ENV VITE_DEAL_SENSE_API_KEY=${VITE_DEAL_SENSE_API_KEY}

COPY . .
RUN npm run build

# Runtime stage
FROM nginx:alpine

COPY --from=build /app/dist /usr/share/nginx/html

# SPA fallback
RUN echo 'server { \
    listen 80; \
    root /usr/share/nginx/html; \
    index index.html; \
    location / { \
        try_files $uri $uri/ /index.html; \
    } \
}' > /etc/nginx/conf.d/default.conf

EXPOSE 80
