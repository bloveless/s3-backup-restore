FROM golang:1.14.1-buster AS builder

WORKDIR "/app"

COPY . .

RUN go build -o /bin/s3-backup-restore cmd/s3-backup-restore/main.go

FROM debian:buster-slim

COPY --from=builder /bin/s3-backup-restore /bin/s3-backup-restore

RUN apt-get update \
 && apt-get install -y ca-certificates \
 && rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/bin/s3-backup-restore"]

CMD ["cron"]
