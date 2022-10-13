#!/bin/sh

SERVICE_EXEC_PATH="/var/packages/velux-nibe/target/bin/velux-nibe"
CONFIG_FILE="/var/packages/velux-nibe/target/velux-nibe.conf"
TOKEN_FILE="/var/packages/velux-nibe/target/nibe-token.json"
LOG_FILE="/var/packages/velux-nibe/target/velux-nibe.log"

exec "$SERVICE_EXEC_PATH" \
  -nibe-token "${TOKEN_FILE}" \
  -conf "${CONFIG_FILE}" > "${LOG_FILE}" 2>&1
