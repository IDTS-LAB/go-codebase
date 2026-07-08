#!/bin/bash
set -e

COMMAND=${1:-up}

case $COMMAND in
    up)
        echo "Running migrations up..."
        goose -dir migrations up
        ;;
    down)
        echo "Running migrations down..."
        goose -dir migrations down
        ;;
    status)
        echo "Migration status:"
        goose -dir migrations status
        ;;
    reset)
        echo "Resetting database..."
        goose -dir migrations reset
        ;;
    *)
        echo "Usage: $0 {up|down|status|reset}"
        exit 1
        ;;
esac
