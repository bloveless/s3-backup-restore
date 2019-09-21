FROM python:3.8-rc-alpine

RUN adduser -D appuser \
    && apk add --no-cache groff \
    && pip install awscli

COPY --chown=appuser:appuser ./s3-backup-restore/entrypoint.sh /home/appuser/entrypoint.sh
COPY --chown=appuser:appuser ./s3-backup-restore/run.sh /home/appuser/run.sh

USER appuser

WORKDIR "/home/appuser"

ENTRYPOINT ["/home/appuser/entrypoint.sh"]

CMD ["./run.sh"]
