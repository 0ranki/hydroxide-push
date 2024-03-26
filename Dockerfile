FROM golang:1.22 as builder

WORKDIR /src
COPY . /src
RUN CGO_ENABLED=0 go build -o /hydroxide-push ./cmd/hydroxide-push/

FROM alpine:latest
VOLUME /data
COPY --from=builder /hydroxide-push /
ENV HOME=/data
WORKDIR /data
ENTRYPOINT ["/hydroxide-push"]
CMD ["notify"]