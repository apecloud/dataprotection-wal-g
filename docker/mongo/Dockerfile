# Build the manager binary
ARG DIST_IMG=mongo:5.0.14

ARG GO_VERSION=1.21

FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} as builder

## docker buildx build injected build-args:
#BUILDPLATFORM — matches the current machine. (e.g. linux/amd64)
#BUILDOS — os component of BUILDPLATFORM, e.g. linux
#BUILDARCH — e.g. amd64, arm64, riscv64
#BUILDVARIANT — used to set ARM variant, e.g. v7
#TARGETPLATFORM — The value set with --platform flag on build
#TARGETOS - OS component from --platform, e.g. linux
#TARGETARCH - Architecture from --platform, e.g. arm64
#TARGETVARIANT

ARG TARGETOS
ARG TARGETARCH

ARG GIT_COMMIT_ID
ARG GIT_TAG_VERSION

ARG GOPROXY=https://goproxy.cn
ARG BUILD_DATE
ARG BUILD_TAGS

ARG LD_FLAGS="-s -w \
  -X github.com/apecloud/dataprotection-wal-g/cmd/mongo.buildDate=${BUILD_DATE} \
  -X github.com/apecloud/dataprotection-wal-g/cmd/mongo.gitRevision=${GIT_COMMIT_ID} \
  -X github.com/apecloud/dataprotection-wal-g/cmd/mongo.walgVersion=${GIT_TAG_VERSION}"

ENV GONOPROXY=github.com/apecloud
ENV GONOSUMDB=github.com/apecloud
ENV GOPRIVATE=github.com/apecloud
ENV GOPROXY=${GOPROXY}

WORKDIR /src

# RUN go mod tidy -compat=1.21 && go mod verify &&  go mod vendor
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    go env && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOFLAGS="-mod=mod" go build -mod vendor -tags "${BUILD_TAGS}"  -ldflags="${LD_FLAGS}" -o /wal-g ./main/mongo/main.go

FROM ${DIST_IMG} as dist

COPY --from=builder /wal-g /bin

USER root:root

ENTRYPOINT ["wal-g"]