FROM golang:1.18.0@sha256:fb5993c8c22ae9fec57f91a3e59825f9368d01b1d15a984d83080cac575f79b8 AS builder

WORKDIR /tmp/source

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY config/ config/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

FROM gcr.io/distroless/static:nonroot@sha256:bca3c203cdb36f5914ab8568e4c25165643ea9b711b41a8a58b42c80a51ed609
WORKDIR /
COPY --from=builder /tmp/source/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
