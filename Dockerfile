FROM --platform=${BUILDPLATFORM} golang:1.19 as builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILDPLATFORM

ARG BIN_NAME=azure-openai-proxy
ARG LDFLAGS="-s -w"


COPY . /builder
WORKDIR /builder

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "${LDFLAGS}" -o bin/${BIN_NAME} ./cmd

FROM alpine:3

WORKDIR /app

EXPOSE 8080
COPY --from=builder /builder/bin .

ENTRYPOINT ["/app/azure-openai-proxy"]