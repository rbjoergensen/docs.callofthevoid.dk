# Traefik and Docker
## Introduction
Traefik is a lot of things, for me it is a reverse proxy that can do https redirects and handle certificates for my home server.
If you want to read more about all the things Traefik can do go check it out at [https://traefik.io](https://traefik.io/).
## Configuration
In this section i will go through the configuration of the docker-compose file I use to spin up traefik.<br/>
### Network
First up i will define a Docker network that all my containers will reside on.
``` yaml
networks:
  traefik-net:
    name: traefik-net
    external: false
```
### Service basics
I will then start defining my services starting with Traefik.<br/><br/>
Things might happen that will make the container crash so I want to tell it to always restart. I want to pin my service to a specific image version to avoid breaking changes crashing my site and I want my container to not have an auto generated name. I also set up the two ports for web traffic, in my network configuration I can then route my web traffic to these.
``` yaml
services:
  traefik:
    image: "traefik:v2.8"
    restart: unless-stopped
    container_name: "traefik"
    networks:
      - traefik-net
    ports:
      - "80:80"
      - "443:443"
    command:
```
### Commands
Under commands i want to have some different flags defined which i will explain.<br/><br/>
First up is the log level which you can set to whatever you want, mine is at `WARN` since `INFO` and `DEBUG` can get quite noisy.
``` yaml
--log.level=WARN
```
I want to enable the dashboard for fun so I can have an overview of all the routes.
``` yaml
--api.dashboard=true
```
Since I are running the application in Docker I need to add these two commands.
``` yaml
--providers.docker=true
--providers.docker.exposedbydefault=false
```
I need to configure the entrypoints that I are going to access traefik on and give them some names that I can use later when adding routes. Here I are creating on for port 80 called web and one for port 443 called websecure.
``` yaml
--entrypoints.web.address=:80
--entrypoints.websecure.address=:443
```
I then want to configure all traffic coming in on web to redirect to websecure globally.
``` yaml
--entrypoints.web.http.redirections.entryPoint.to=websecure
--entrypoints.web.http.redirections.entryPoint.scheme=https
```
Now I just need to configure my certificate resolver which in my case is CloudFlare since they are my DNS provider.
To get an overview of all the available providers you can go to this part of the documentation. [traefik(ACME)](https://doc.traefik.io/traefik/https/acme/).<br/>
I also need to define the secret access token but I will add that later as an environment variable called `CF_DNS_API_TOKEN`.
``` yaml
--certificatesresolvers.cloudflare.acme.dnschallenge=true
--certificatesresolvers.cloudflare.acme.dnschallenge.provider=cloudflare
--certificatesresolvers.cloudflare.acme.email=rasmus@cotv.dk
--certificatesresolvers.cloudflare.acme.storage=/letsencrypt/acme.json
```
### Volumes
Here I mount the letsencrypt directory to my host so the acme.json file that contains my certificates doesn't get deleted on every reboot as well as the Docker socket so Trafic can hook into Docker.
``` yaml
volumes:
  - "/letsencrypt:/letsencrypt"
  - "/traefik/usersfile:/traefikusers"
  - "/var/run/docker.sock:/var/run/docker.sock:ro"
```
The usersfile contains a list of `name:hashed-password`. You can use a tool htpasswd or openssl for this.
```
admin:$apr1$i4cUyBZl$GzyVeKlwjB5UOSw2scq420
user1:$apr1$CJ9ugIPG$yKSDt4ZkuNuz8NIyChsQP0
```
### Environment
For my environment variables I want to create a secret file can i can specify when deploying. I can call it `traefik.env`.
``` yaml
env_file: traefik.env
```
It will then have the following content. The Pilot token is if I want to add traefik Pilot which is monitoring and alerting. [traefik Pilot](https://traefik.io/traefik-pilot/).
``` yaml
CF_DNS_API_TOKEN: <secret>
TRAEFIK_PILOT_TOKEN: <secret>
```
### Labels
Labels will be added to all the services that I want traefik to act as a reverse proxy for. In this case I want the traefik dashboard available at [https://traefik.callofthevoid.dk](https://traefik.callofthevoid.dk) and [https://traefik.cotv.dk](https://traefik.cotv.dk) so I have to add the following labels.
``` yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.traefik.rule=Host(`traefik.cotv.dk`) || Host(`traefik.callofthevoid.dk`)"
  - "traefik.http.routers.traefik.entrypoints=websecure"
  - "traefik.http.routers.traefik.tls.certresolver=cloudflare"
  - "traefik.http.routers.traefik.service=api@internal"
```
I also want to have some basic authentication in front of the dashboard so I also add these labels.
``` yaml
  - "traefik.http.routers.traefik.middlewares=traefik-auth"
  - "traefik.http.middlewares.test-auth.basicauth.usersfile=/traefikusers"
```
## Full config
This is how a docker-compose would look like with an added extra nginx container as an example.
``` yaml
version: '3.5'

networks:
  traefik-net:
    name: traefik-net
    external: false

services:
  traefik:
    image: "traefik:v2.8"
    restart: unless-stopped
    container_name: "traefik"
    networks:
      - traefik-net
    ports:
      - "80:80"
      - "443:443"
    command:
      --log.level=WARN
      --api.dashboard=true
      --providers.docker=true
      --providers.docker.exposedbydefault=false
      --entrypoints.web.address=:80
      --entrypoints.websecure.address=:443
      --entrypoints.web.http.redirections.entryPoint.to=websecure
      --entrypoints.web.http.redirections.entryPoint.scheme=https
      --certificatesresolvers.cloudflare.acme.dnschallenge=true
      --certificatesresolvers.cloudflare.acme.dnschallenge.provider=cloudflare
      --certificatesresolvers.cloudflare.acme.email=rasmus@cotv.dk
      --certificatesresolvers.cloudflare.acme.storage=/letsencrypt/acme.json
    volumes:
      - "/letsencrypt:/letsencrypt"
      - "/traefik/usersfile:/traefikusers"
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
    env_file: traefik.env
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.cotv.dk`) || Host(`traefik.callofthevoid.dk`)"
      - "traefik.http.routers.traefik.entrypoints=websecure"
      - "traefik.http.routers.traefik.tls.certresolver=cloudflare"
      - "traefik.http.routers.traefik.service=api@internal"
      - "traefik.http.routers.traefik.middlewares=traefik-auth"
      - "traefik.http.middlewares.test-auth.basicauth.usersfile=/traefikusers"

  docs-callofthevoid:
    image: ghcr.io/rbjoergensen/docs.callofthevoid.dk/docs.callofthevoid.dk:latest
    restart: unless-stopped
    container_name: docs-callofthevoid
    networks:
      - traefik-net
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.docs-callofthevoid.rule=Host(`docs.cotv.dk`) || Host(`docs.callofthevoid.dk`)"
      - "traefik.http.routers.docs-callofthevoid.entrypoints=websecure"
      - "traefik.http.routers.docs-callofthevoid.tls.certresolver=cloudflare"
```
## Examples
Lets say i was running another service using ports that are not 80 or 443. I need to define this and create a rule for each.
In this example i have MinIO running which has an API on port 9000 and a web interface on port 9002.
``` yaml
version: '3.5'

services:
  minio:
    image: minio/minio:RELEASE.2022-02-01T18-00-14Z
    restart: unless-stopped 
    container_name: minio
    networks:
      - traefik-net
    env_file: minio.env
    command: server /data --console-address :9002
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.minio.rule=Host(`minio.cotv.dk`) || Host(`minio.callofthevoid.dk`)"
      - "traefik.http.routers.minio.service=minio"
      - "traefik.http.routers.minio.entrypoints=websecure"
      - "traefik.http.routers.minio.tls.certresolver=cloudflare"
      - "traefik.http.services.minio.loadbalancer.server.port=9002"
      - "traefik.http.routers.minio_api.rule=Host(`api.minio.cotv.dk`) || Host(`api.minio.callofthevoid.dk`)"
      - "traefik.http.routers.minio_api.service=minio_api"
      - "traefik.http.routers.minio_api.entrypoints=websecure"
      - "traefik.http.routers.minio_api.tls.certresolver=cloudflare"
      - "traefik.http.services.minio_api.loadbalancer.server.port=9000"
```