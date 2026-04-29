# STAGE 1: build
FROM node:20-alpine AS builder

WORKDIR /app
# copy only package.json, rest is cached
COPY frontend/package*.json ./
RUN npm install

# copy rest of code
COPY frontend/ .

# copy env
COPY .env-frontend .env

RUN npm run build

# STAGE 2: serve
FROM nginx:alpine

RUN rm -rf /usr/share/nginx/html*

COPY --from=builder /app/dist /usr/share/nginx/html

# copy nginx config
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
