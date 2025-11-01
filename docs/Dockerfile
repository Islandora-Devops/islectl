FROM python:3.14-alpine@sha256:8373231e1e906ddfb457748bfc032c4c06ada8c759b7b62d9c73ec2a3c56e710

RUN mkdir -p /docs

WORKDIR /docs

COPY requirements.txt .
RUN apk add --no-cache cairo \
  && pip install -r requirements.txt 

COPY . .
RUN mkdocs build

# serve the docs via nginx
FROM nginx:1.29@sha256:f547e3d0d5d02f7009737b284abc87d808e4252b42dceea361811e9fc606287f
COPY --from=0 /docs/site /usr/share/nginx/html
