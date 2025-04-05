FROM golang:1.18

WORKDIR /src
COPY . /src

RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o output/shiba github.com/moycat/shiba/cmd

FROM debian:12-slim

RUN apt update && apt install -y iptables && apt clean && rm -rf /var/lib/apt/lists/* /var/log/dpkg.log /var/log/apt/*

COPY --from=0 /src/output/shiba /usr/bin/shiba
CMD ["/usr/bin/shiba"]
