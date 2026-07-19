# syntax=docker/dockerfile:1

# ---- build stage ----
FROM node:22-alpine AS build
WORKDIR /src

# Cache deps: copy package manifests first so `npm ci` only re-runs when
# dependencies actually change.
COPY package.json package-lock.json* ./
RUN npm ci

COPY . .

# Vite inlines VITE_* vars at build time, so the ARG must be turned into an
# ENV before `npm run build` executes.
ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

# ---- final stage ----
FROM nginx:1.27-alpine AS final

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
