#!/bin/bash

if [ $# -lt 3 ]; then
	echo "param num must  three, exit. "
	exit 1
fi

SCP_FILE=$1
DST_HOST=$2
DST_PATH=$3
CMD_KILL=$4
WIN_HOST="192.168.1.102"
WIN_PORT="9000"
ETH_NAME="ens33"
SRC_HOST=$(ifconfig ${ETH_NAME} | grep -w inet | head -n 1 | awk '{print $2}')

if [ -z "${SRC_HOST}" ]; then
	echo "src ip is null, exit"
	exit 1
fi

if [ "${SRC_HOST}" == "127.0.0.1" ]; then
	echo "src ip is 127.0.0.1, exit"
	exit 2
fi

if [ ! -f "${SCP_FILE}" ]; then
	echo "file: ${SCP_FILE} not exist."
	exit 3
fi

FULL_PATH=`realpath "${SCP_FILE}"`
if [ ! -f "${FULL_PATH}" ]; then
	echo "file: ${FULL_PATH} not exist.."
	exit 4
fi

if command -v ipcalc >/dev/null 2>&1; then
	if ! ipcalc -c "${DST_HOST}" >/dev/null 2>&1; then
		echo "dst_ip: ${DST_HOST} is invalid."
		exit 5
	fi
fi

if [ ! -z "${CMD_KILL}" ]; then
	DST_DIR=$(dirname "${DST_PATH}")
	CMD_JSON=$(printf '{"oper_action":"exec_cmd","dst_host":"%s","dst_path":"%s", "cmd_exec":"%s"}' "${DST_HOST}" "${DST_DIR}" "${CMD_KILL}")
	echo "${CMD_JSON}" | nc "${WIN_HOST}" "${WIN_PORT}"
	echo ""
fi

CMD_JSON=$(printf '{"oper_action":"send_file","oper_type":"file","src_host":"%s","src_path":"%s","dst_host":"%s","dst_path":"%s"}' "$SRC_HOST" "$FULL_PATH" "$DST_HOST" "$DST_PATH")
echo "${CMD_JSON}" | nc "${WIN_HOST}" "${WIN_PORT}"

echo ""



