FROM golang:alpine AS build

WORKDIR /src
COPY . ./

RUN go build

FROM alpine

WORKDIR /app

COPY --from=build /src/terraform-cloud-workspace-action ./

ENTRYPOINT ["./terraform-cloud-workspace-action"]
