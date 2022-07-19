#!/bin/sh

SERVICE_EXEC_PATH="/var/packages/velux-nibe/target/bin/velux-nibe"
CONFIG_FILE="/var/packages/velux-nibe/target/velux-nibe.conf"
TOKEN_FILE="/var/packages/velux-nibe/target/nibe-token.json"
LOG_FILE="/var/packages/velux-nibe/target/velux-nibe.log"

# Import config file
# shellcheck disable=SC1090
. "${CONFIG_FILE}"

exec "$SERVICE_EXEC_PATH" \
  -nibe-token "${TOKEN_FILE}" \
  -targetTemp "${TARGET_TEMP}" \
  -interval "${POLL_INTERVAL}" > "${LOG_FILE}" 2>&1
