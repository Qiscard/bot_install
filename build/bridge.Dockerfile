# qq-resource-bridge 镜像：复用 bot-ctl 二进制，运行 `bot-ctl bridge serve`
# 构建：docker build -f build/bridge.Dockerfile -t botctl/qq-resource-bridge:latest .
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/bot-ctl ./cmd/bot-ctl

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 bridge
COPY --from=build /out/bot-ctl /usr/local/bin/bot-ctl
USER bridge
ENV BRIDGE_OUTPUT=/output \
    BRIDGE_LISTEN=:8787
VOLUME ["/output"]
EXPOSE 8787
ENTRYPOINT ["bot-ctl", "bridge", "serve"]
