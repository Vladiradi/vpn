FROM alpine:3.20

RUN apk add --no-cache rsyslog logrotate

COPY rsyslog.conf /etc/rsyslog.conf
COPY logrotate.conf /etc/logrotate.d/vpn
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && \
    mkdir -p /var/log/vpn && \
    touch /var/log/vpn/.keep

VOLUME ["/var/log/vpn"]

EXPOSE 514/udp 514/tcp

ENTRYPOINT ["/entrypoint.sh"]
