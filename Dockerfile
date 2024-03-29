FROM golang:1.19-alpine AS build

RUN apk add git

WORKDIR /src
COPY . ./

RUN go build

ENTRYPOINT ["/src/terraform-cloud-workspace-action"]
