FROM nginx:alpine

COPY ./site /usr/share/nginx/html/

RUN ls -la /usr/share/nginx/html/