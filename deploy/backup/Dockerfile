FROM postgres:16-alpine

RUN apk add --no-cache tzdata

WORKDIR /backup

COPY run.sh /backup/run.sh
RUN chmod +x /backup/run.sh

ENV BACKUP_DIR=/backups
ENV RETAIN_COUNT=7
ENV SCHEDULE="0 3 * * *"
ENV PGPASSWORD=""
ENV PGHOST="postgres"
ENV PGUSER="vpn"
ENV PGDATABASE="vpn"

CMD ["/backup/run.sh"]
