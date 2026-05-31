FROM node:22-alpine AS build

WORKDIR /workspace
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /workspace/dist /usr/share/nginx/html
RUN sed 's#<head>#<head>\n    <script>window.PLAYER_STATS_PUBLIC_ONLY = true;</script>#' \
    /usr/share/nginx/html/index.html > /usr/share/nginx/html/share-index.html

EXPOSE 80 8081
