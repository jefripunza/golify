#!/bin/sh
# nfpm postinstall — install systemd unit + start service
set -e

cat > /etc/systemd/system/golify.service <<'UNIT'
[Unit]
Description=Golify — application deployment platform
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/var/lib/golify
ExecStart=/usr/local/bin/golify
Restart=always
RestartSec=5
# Bind to all interfaces; TLS/ACME handled internally on 80/443.
Environment=GOLIFY_PORT=3000

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable golify
systemctl restart golify
exit 0
