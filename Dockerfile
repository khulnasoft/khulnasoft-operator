# Build the manager binary
FROM golang:1.19 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
#FROM gcr.io/distroless/static:nonroot
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/khulnasoft-operator \
    USER_UID=1001 \
    USER_NAME=khulnasoft-operator

LABEL name="Khulnasoft Operator" \
      vendor="Khulnasoft Security Software Production" \
      maintainer="KhulnaSoft Ltd." \
      version="v1.0.2" \
      release="1" \
      summary="Khulnasoft Security Operator." \
      description="The Khulnasoft Security Operator runs within a Openshift (or Kubernetes) cluster, and provides a means to deploy and manage the Khulnasoft Security cluster and components"

WORKDIR /

COPY licenses /licenses
COPY --from=builder /workspace/manager .

USER ${USER_UID}
ENTRYPOINT ["/manager"]


