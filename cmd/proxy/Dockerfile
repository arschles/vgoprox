# Links to compare against to ensure we have all VCS's setup in this build
# https://github.com/docker-library/buildpack-deps/blob/1845b3f918f69b4c97912b0d4d68a5658458e84f/stretch/scm/Dockerfile
# https://github.com/golang/go/blob/f082dbfd4f23b0c95ee1de5c2b091dad2ff6d930/src/cmd/go/internal/get/vcs.go#L90
#
# You can override the Go version used to build the image.
# See project Makefile if using make.
# See docker --build-arg if building directly.
ARG GOLANG_VERSION=1.14
ARG ALPINE_VERSION=3.11.5

FROM golang:${GOLANG_VERSION}-alpine AS builder

WORKDIR $GOPATH/src/github.com/gomods/athens

COPY . .

ARG VERSION="unset"

RUN DATE="$(date -u +%Y-%m-%d-%H:%M:%S-%Z)" && GO111MODULE=on CGO_ENABLED=0 GOPROXY="https://proxy.golang.org" go build -ldflags "-X github.com/gomods/athens/pkg/build.version=$VERSION -X github.com/gomods/athens/pkg/build.buildDate=$DATE" -o /bin/athens-proxy ./cmd/proxy

FROM alpine:${ALPINE_VERSION}

ENV GO111MODULE=on

COPY --from=builder /bin/athens-proxy /bin/athens-proxy
COPY --from=builder /go/src/github.com/gomods/athens/config.dev.toml /config/config.toml
COPY --from=builder /usr/local/go/bin/go /bin/go

RUN chmod 700 /config/config.toml

# Add tini, see https://github.com/gomods/athens/issues/1155 for details.
RUN apk add --update bzr git git-lfs mercurial openssh-client subversion procps fossil tini && \
	mkdir -p /usr/local/go

EXPOSE 3000

ENTRYPOINT [ "/sbin/tini", "--" ]

CMD ["athens-proxy", "-config_file=/config/config.toml"]
