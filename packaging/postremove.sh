#!/bin/sh
# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

if command -V systemctl >/dev/null 2>&1; then
	echo "Stop and disable ciy-scheduler service"
	systemctl stop ciy-scheduler >/dev/null 2>&1 || true
	systemctl disable ciy-scheduler >/dev/null 2>&1 || true
	echo "Running daemon-reload"
	systemctl daemon-reload || true
fi

