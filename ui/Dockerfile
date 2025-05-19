FROM node:18-alpine AS build-stage

WORKDIR /app

COPY package.json .

RUN npm install

COPY . .

RUN npm run build


# Deploy the application binary into a lean image
FROM nginx AS build-release-stage

COPY --from=build-stage /app/dist /usr/share/nginx/html

