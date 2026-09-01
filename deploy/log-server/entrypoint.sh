#!/bin/sh
set -e

mkdir -p /var/log/vpn

# Run logrotate once on start, then schedule it via cron-like loop in background.
(
  while true; do
    /usr/sbin/logrotate -f /etc/logrotate.d/vpn 2>/dev/null || true
    sleep 86400
  done
) &

exec /usr/sbin/rsyslogd -n
