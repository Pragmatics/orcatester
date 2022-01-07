FROM golang:1.17.3@sha256:6556ce40115451e40d6afbc12658567906c9250b0fda250302dffbee9d529987 AS builder

WORKDIR /tmp/source

# Test dependencies
COPY Makefile Makefile
RUN bash -c "make get-dependencies"

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY hack/ hack/
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY config/ config/

# Tests
RUN OUTPUT=$(go fmt ./...); if [ ! -z ${OUTPUT} ] ; then echo "ERROR: file(s) failed golang formatting rules"; echo ${OUTPUT}; exit 1; fi
RUN go vet ./...
RUN make test

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

FROM gcr.io/distroless/static:nonroot@sha256:bca3c203cdb36f5914ab8568e4c25165643ea9b711b41a8a58b42c80a51ed609
WORKDIR /
COPY --from=builder /tmp/source/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
