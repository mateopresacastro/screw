FROM node:22-alpine AS frontend
WORKDIR /build/frontend
COPY frontend/package*.json .
RUN npm ci
COPY frontend/ .
RUN npm run build

FROM nginx:alpine
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY --from=frontend /build/frontend/out /usr/share/nginx/html
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]