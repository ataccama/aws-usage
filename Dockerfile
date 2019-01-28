FROM golang:alpine AS build

COPY . /srv
WORKDIR /srv
ENV CGO_ENABLED=0
RUN apk add -u git && go build

FROM alpine
COPY --from=build /srv/aws-usage /usr/bin/aws-usage
COPY config.toml /etc/aws-usage/config.toml
RUN apk add -u ca-certificates
ENTRYPOINT [ "/usr/bin/aws-usage" ]
