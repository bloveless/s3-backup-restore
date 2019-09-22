FROM python:3.8-rc-alpine

# groff is added so you can use aws cli help
RUN apk add --no-cache groff busybox-suid \
    && pip install awscli \
    && touch /var/log/run.log

COPY ./s3-backup-restore/entrypoint.sh /root/entrypoint.sh
COPY ./s3-backup-restore/run.sh /root/run.sh

WORKDIR "/root"

ENTRYPOINT ["/root/entrypoint.sh"]

CMD ["./run.sh"]
