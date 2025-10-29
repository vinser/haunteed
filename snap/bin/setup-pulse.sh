#!/bin/bash
# setup-pulse.sh — executed before launching haunteed

export XDG_RUNTIME_DIR="/run/user/$(id -u)"
export PULSE_SERVER="unix:$XDG_RUNTIME_DIR/pulse/native"

exec "$@"
