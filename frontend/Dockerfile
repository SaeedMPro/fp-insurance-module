# syntax=docker/dockerfile:1

# ---- build stage ----
FROM node:22-alpine AS build
WORKDIR /src

# Cache deps: copy package manifests first so `npm ci` only re-runs when
# dependencies actually change.
COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

# ---- final stage ----
FROM nginx:1.27-alpine AS final

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh
COPY --from=build /src/dist /usr/share/nginx/html

EXPOSE 80

# Writes /config.js from API_BASE_URL at start, then runs nginx.
ENTRYPOINT ["/docker-entrypoint.sh"]
