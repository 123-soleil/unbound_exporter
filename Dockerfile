FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.0-bookworm AS build

WORKDIR /go/src/app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

ENV CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -o /go/bin/unbound_exporter .

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/unbound_exporter /

ENTRYPOINT ["/unbound_exporter"]
