#!/usr/bin/env bash
# otel-device — CLI for managing OTel SDK configuration on Android devices.
#
# Seed for a full control plane CLI. Currently wraps adb SharedPreferences
# operations for runtime config override.
#
# Future: will support gateway API calls for remote config management.
#
# Usage:
#   otel-device config set --endpoint URL [--mode MODE] [--auth-token TOKEN] ...
#   otel-device config show
#   otel-device config reset
#   otel-device list
#   otel-device -s SERIAL config show     # target specific device
#   otel-device --package com.my.app config show  # custom package
#
# Environment:
#   OTEL_DEVICE_PACKAGE  — override default package name
set -euo pipefail

# ── Defaults ───────────────────────────────────────────────────────────────

PKG="${OTEL_DEVICE_PACKAGE:-io.opentelemetry.android.demo}"
PREFS_FILE="shared_prefs/otel_config.xml"
SERIAL=""

# ── Helpers ────────────────────────────────────────────────────────────────

usage() {
  cat <<'USAGE'
otel-device — CLI for managing OTel SDK configuration on Android devices

COMMANDS
  config set   Set configuration values on a connected device
  config show  Show current SharedPreferences config override
  config reset Remove config override (restore bundled defaults)
  list         List connected devices/emulators

OPTIONS
  -s SERIAL        Target a specific device (from `adb devices`)
  --package PKG    Override package name (default: io.opentelemetry.android.demo)
                   Also settable via OTEL_DEVICE_PACKAGE env var

CONFIG SET FLAGS
  --endpoint URL           Collector endpoint (e.g., http://10.0.2.2:14317)
  --auth-token TOKEN       Authorization bearer token
  --dataset NAME           Dash0 dataset name
  --mode MODE              Export mode: CONDITIONAL, CONTINUOUS, or HYBRID
  --service-name NAME      OTel service.name
  --service-version VER    OTel service.version

EXAMPLES
  otel-device config set --endpoint http://10.0.2.2:14317 --mode CONTINUOUS
  otel-device config show
  otel-device config reset
  otel-device -s emulator-5554 config set --endpoint https://ingress.dash0.com:4317
  otel-device --package com.myapp list
USAGE
  exit 0
}

die() { echo "error: $*" >&2; exit 1; }

adb_cmd() {
  if [ -n "$SERIAL" ]; then
    adb -s "$SERIAL" "$@"
  else
    adb "$@"
  fi
}

# ── Parse global options ───────────────────────────────────────────────────

while [ $# -gt 0 ]; do
  case "$1" in
    -s)        SERIAL="$2"; shift 2 ;;
    --package) PKG="$2"; shift 2 ;;
    -h|--help) usage ;;
    *)         break ;;
  esac
done

[ $# -eq 0 ] && usage

COMMAND="$1"; shift

# ── list ───────────────────────────────────────────────────────────────────

cmd_list() {
  echo "Connected devices:"
  adb devices -l | tail -n +2 | while IFS= read -r line; do
    [ -z "$line" ] && continue
    echo "  $line"
  done
}

# ── config show ────────────────────────────────────────────────────────────

cmd_config_show() {
  echo "Package: $PKG"
  echo "Device:  ${SERIAL:-<default>}"
  echo ""

  local raw
  raw=$(adb_cmd shell "run-as $PKG cat $PREFS_FILE 2>/dev/null" || true)

  if [ -z "$raw" ]; then
    echo "No SharedPreferences override active. Using bundled defaults."
    return
  fi

  echo "Active SharedPreferences override:"
  echo "$raw" | while IFS= read -r line; do
    # Extract key and value from <string name="key">value</string>
    case "$line" in
      *'<string name="'*)
        key=$(echo "$line" | sed 's/.*name="\([^"]*\)".*/\1/')
        val=$(echo "$line" | sed 's/.*">\(.*\)<\/string>.*/\1/')
        printf "  %-25s %s\n" "$key" "$val"
        ;;
    esac
  done
}

# ── config reset ───────────────────────────────────────────────────────────

cmd_config_reset() {
  adb_cmd shell "run-as $PKG rm -f $PREFS_FILE"
  adb_cmd shell "am force-stop $PKG"
  echo "Config override removed. App will use bundled defaults on next launch."
}

# ── config set ─────────────────────────────────────────────────────────────

cmd_config_set() {
  local endpoint="" auth_token="" dataset="" mode="" svc_name="" svc_version=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --endpoint)        endpoint="$2"; shift 2 ;;
      --auth-token)      auth_token="$2"; shift 2 ;;
      --dataset)         dataset="$2"; shift 2 ;;
      --mode)            mode="$2"; shift 2 ;;
      --service-name)    svc_name="$2"; shift 2 ;;
      --service-version) svc_version="$2"; shift 2 ;;
      *) die "Unknown flag: $1" ;;
    esac
  done

  # Validate mode if provided
  if [ -n "$mode" ]; then
    case "$mode" in
      CONDITIONAL|CONTINUOUS|HYBRID) ;;
      *) die "Invalid mode: $mode (must be CONDITIONAL, CONTINUOUS, or HYBRID)" ;;
    esac
  fi

  # At least one value must be provided
  if [ -z "$endpoint" ] && [ -z "$auth_token" ] && [ -z "$dataset" ] && \
     [ -z "$mode" ] && [ -z "$svc_name" ] && [ -z "$svc_version" ]; then
    die "No values provided. Use --endpoint, --mode, --auth-token, --dataset, --service-name, or --service-version"
  fi

  # Build XML entries
  local entries=""
  [ -n "$endpoint" ]    && entries="${entries}  <string name=\"collector_endpoint\">$endpoint</string>\n"
  [ -n "$auth_token" ]  && entries="${entries}  <string name=\"auth_token\">$auth_token</string>\n"
  [ -n "$dataset" ]     && entries="${entries}  <string name=\"dataset\">$dataset</string>\n"
  [ -n "$mode" ]        && entries="${entries}  <string name=\"export_mode\">$mode</string>\n"
  [ -n "$svc_name" ]    && entries="${entries}  <string name=\"service_name\">$svc_name</string>\n"
  [ -n "$svc_version" ] && entries="${entries}  <string name=\"service_version\">$svc_version</string>\n"
  # Always mark as loaded so ConfigManager treats prefs as authoritative
  # Must be <boolean> tag — ConfigManager calls getBoolean() on this key
  entries="${entries}  <boolean name=\"config_loaded_from_bundle\" value=\"true\" />\n"

  local xml_content
  xml_content="<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\" ?>
<map>
$(echo -e "$entries")</map>"

  # Write to device — pipe through stdin to avoid heredoc temp file issues in run-as sandbox
  adb_cmd shell "run-as $PKG mkdir -p shared_prefs"
  echo "$xml_content" | adb_cmd shell "run-as $PKG sh -c 'cat > $PREFS_FILE'"

  # Force-stop so app picks up new config on relaunch
  adb_cmd shell "am force-stop $PKG"

  echo "Config set. App will use new values on next launch."
  echo ""
  cmd_config_show
}

# ── Dispatch ───────────────────────────────────────────────────────────────

case "$COMMAND" in
  list) cmd_list ;;
  config)
    [ $# -eq 0 ] && die "Missing subcommand: set, show, or reset"
    SUB="$1"; shift
    case "$SUB" in
      set)   cmd_config_set "$@" ;;
      show)  cmd_config_show ;;
      reset) cmd_config_reset ;;
      *)     die "Unknown config subcommand: $SUB (use set, show, or reset)" ;;
    esac
    ;;
  *) die "Unknown command: $COMMAND (use config or list)" ;;
esac
