#syntax=docker/dockerfile:1.2
FROM golang:rc-alpine AS build

ENV CGO_ENABLED=0
WORKDIR /workspace

# cache layers
COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,dst=/go/pkg/mod \
    go mod download

# build
COPY . .
RUN --mount=type=cache,id=gomod,dst=/go/pkg/mod \
    --mount=type=cache,id=gobuild,dst=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o app .


##### final image
FROM gcr.io/distroless/base

COPY --from=build /workspace/app /bin/image-mirror

ENTRYPOINT ["/bin/image-mirror"]
