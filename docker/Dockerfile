FROM golang:1.14.1-buster AS builder

WORKDIR "/app"

COPY . .

RUN go build -o /bin/s3-backup-restore cmd/s3-backup-restore/main.go

FROM gcr.io/distroless/base-debian10

COPY --from=builder /bin/s3-backup-restore /bin/s3-backup-restore

ENTRYPOINT ["/bin/s3-backup-restore"]

CMD ["cron"]
