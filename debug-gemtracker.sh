#!/bin/bash
# Debug script to capture gemtracker crash logs

LOG_FILE="${HOME}/.cache/gemtracker/crash.log"
mkdir -p "$(dirname "$LOG_FILE")"

echo "Starting gemtracker with debug logging..."
echo "Logs will be written to: $LOG_FILE"
echo ""
echo "When the app crashes, the log file will contain details."
echo ""

# Run with all output captured
./gemtracker "$@" > "$LOG_FILE" 2>&1

EXIT_CODE=$?

echo ""
echo "App exited with code: $EXIT_CODE"
echo ""
echo "Last 50 lines of log:"
tail -50 "$LOG_FILE"

if [ $EXIT_CODE -ne 0 ]; then
  echo ""
  echo "FULL LOG:"
  cat "$LOG_FILE"
fi
