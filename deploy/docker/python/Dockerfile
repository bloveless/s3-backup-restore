FROM python:3.8-rc-alpine

# groff is added so you can use aws cli help
RUN adduser -D app \
    && apk add --no-cache groff busybox-suid \
    && pip install awscli \
    && touch /var/log/run.log \
    && chown app:app /var/log/run.log

COPY ./s3-backup-restore/entrypoint.sh /home/app/entrypoint.sh
COPY ./s3-backup-restore/run.sh /home/app/run.sh

WORKDIR "/home/app"

ENTRYPOINT ["/home/app/entrypoint.sh"]

CMD ["./run.sh"]
