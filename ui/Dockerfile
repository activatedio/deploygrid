FROM node:22-alpine AS build-stage

WORKDIR /app

COPY package.json package-lock.json ./

RUN npm ci

COPY . .

RUN npm run build


# nginx serves the UI and proxies /api to the server container in the same
# pod, so the browser talks to one origin and the server needs no CORS.
FROM nginxinc/nginx-unprivileged:1.27-alpine

# envsubst renders the template at start; only API_PORT is substituted so
# nginx variables such as $host and $uri survive.
ENV API_PORT=8080 NGINX_ENVSUBST_FILTER=^API_PORT$
COPY default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build-stage /app/dist /usr/share/nginx/html
