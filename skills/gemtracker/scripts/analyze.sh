#!/bin/bash
# Gemtracker skill implementation
# Wraps the gemtracker CLI for Claude Code integration

set -e

# Default values
path="${1:-.}"
report_format="${2:text}"
report_flag=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --report)
      report_format="$2"
      report_flag="--report $report_format"
      shift 2
      ;;
    --json)
      report_format="json"
      report_flag="--report json"
      shift
      ;;
    --csv)
      report_format="csv"
      report_flag="--report csv"
      shift
      ;;
    *)
      path="$1"
      shift
      ;;
  esac
done

# Check if gemtracker is installed
if ! command -v gemtracker &> /dev/null; then
  echo "Error: gemtracker CLI not found in PATH"
  echo "Install with: brew install spaquet/gemtracker/gemtracker"
  exit 1
fi

# Check if path exists
if [ ! -d "$path" ] && [ ! -f "$path" ]; then
  echo "Error: Path not found: $path"
  exit 1
fi

# Run gemtracker
gemtracker "$path" $report_flag
