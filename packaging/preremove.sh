#!/bin/sh
# nfpm preremove — stop + disable service before uninstall
set -e

systemctl stop golify 2>/dev/null || true
systemctl disable golify 2>/dev/null || true
exit 0
