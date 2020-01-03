ARG ALPINE_VERSION=3.11
ARG GO_VERSION=1.13.5
ARG BASE_IMAGE=scratch
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} as builder
ENV CGO_ENABLED=0
RUN mkdir /build /empty
WORKDIR /build
ADD go.mod go.sum /build/
RUN go mod download
ADD . /build/
RUN go build -o /api-exam

FROM ${BASE_IMAGE}
COPY --from=builder /api-exam /api-exam
COPY --from=builder /empty /input
COPY --from=builder /empty /output
COPY --from=builder /empty /errors
ENTRYPOINT ["/api-exam"]
