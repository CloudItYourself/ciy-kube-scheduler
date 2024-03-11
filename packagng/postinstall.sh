#!/bin/sh
# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

CIY_SCHEDULER_EXE="/usr/bin/ciy-scheduler"
BSD_HIER=""
CIY_RUN_DIR="/var/run/ciy-scheduler"

ensure_sudo() {
	if [ "$(id -u)" = "0" ]; then
		echo "Sudo permissions detected"
	else
		echo "No sudo permission detected, please run as sudo"
		exit 1
	fi
}

ensure_ciy-scheduler_path() {
	if [ ! -f "$CIY_SCHEDULER_EXE" ]; then
		echo "ciy-scheduler not in default path, exiting..."
		exit 1
	fi

	printf "Found ciy-scheduler %s\n" "$CIY_SCHEDULER_EXE"
}

create_run_dir() {
	printf "PostInstall: Creating ciy-scheduler run directory \n"
	mkdir -p "$CIY_RUN_DIR"

}

summary() {
	echo "----------------------------------------------------------------------"
	echo " ciy-scheduler package has been successfully installed."
	echo ""
	echo " Please follow the next steps to start the software:"
	echo ""
	echo "    sudo systemctl enable ciy-scheduler"
	echo "    sudo systemctl start ciy-scheduler"
	echo "    sudo systemctl status ciy-scheduler"
	echo ""
	echo " 	  Environment variables can be adjusted here:"
	echo "    /etc/ciy-scheduling/env.cfg"
	echo ""
	echo "----------------------------------------------------------------------"
}

#
# Main body of the script
#
{
	ensure_sudo
	ensure_ciy-scheduler_path
	create_run_dir
	summary
}
