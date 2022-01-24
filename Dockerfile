FROM golang:1.17

WORKDIR /src
COPY . /src

RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o shiba github.com/moycat/shiba/cmd

FROM debian:11-slim

RUN apt update && apt install -y iptables && rm -rf /var/lib/apt/lists/* /var/log/dpkg.log /var/log/apt/*

COPY --from=0 /src/shiba /usr/bin/shiba
CMD ["/usr/bin/shiba"]
