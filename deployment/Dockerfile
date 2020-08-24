FROM golang:1.14-stretch AS builder

WORKDIR $GOPATH/src/distributed-encoder
COPY . ./

ARG cmd
RUN cd cmd/${cmd} && CGO_ENABLED=0 \
  go build -mod=vendor -a -o /app/${cmd} .

# final step
FROM alpine:latest
RUN ls -a
ARG cmd
RUN apk add --no-cache ffmpeg
COPY --from=builder /app/${cmd} /app/main
CMD ["/app/main"]