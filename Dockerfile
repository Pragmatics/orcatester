FROM golang:1.17.7@sha256:c2ca4729eae792e8066e471e4b71d61a69b22ad71ca31e9d524fde5addabbd7e AS builder

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
