#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<EOF
Usage: $(basename "$0") <new-module-path>

Rename the Go module path across the entire project.

Examples:
    $(basename "$0") github.com/myorg/myproject
    $(basename "$0") github.com/myorg/myproject/v2

What it does:
    1. Updates go.mod module declaration
    2. Updates all Go import paths
    3. Updates references in docs, swagger, configs, and scripts
    4. Runs go mod tidy to clean up
EOF
    exit 1
}

if [[ $# -ne 1 ]]; then
    usage
fi

NEW_PATH="$1"
OLD_PATH=$(head -1 go.mod | awk '{print $2}')

if [[ -z "$OLD_PATH" ]]; then
    echo "Error: could not read current module path from go.mod" >&2
    exit 1
fi

if [[ "$OLD_PATH" == "$NEW_PATH" ]]; then
    echo "New path is the same as current path. Nothing to do." >&2
    exit 0
fi

echo "Renaming module:"
echo "  from: $OLD_PATH"
echo "  to:   $NEW_PATH"
echo ""

# Escape paths for sed
OLD_ESCAPED=$(printf '%s\n' "$OLD_PATH" | sed 's/[.[\*^$()+?{|]/\\&/g')
NEW_ESCAPED=$(printf '%s\n' "$NEW_PATH" | sed 's/[.[\*^$()+?{|]/\\&/g')

# 1. Update go.mod
echo "Updating go.mod..."
sed -i "s|${OLD_PATH}|${NEW_PATH}|g" go.mod

# 2. Update all Go files (import paths)
echo "Updating Go imports..."
find . -name '*.go' -not -path './vendor/*' -exec sed -i "s|${OLD_PATH}|${NEW_PATH}|g" {} +

# 3. Update non-Go files (docs, configs, swagger, scripts, etc.)
echo "Updating docs and configs..."
find . \( \
    -name '*.md' -o \
    -name '*.yaml' -o \
    -name '*.yml' -o \
    -name '*.json' -o \
    -name '*.toml' -o \
    -name '*.html' -o \
    -name '*.sh' -o \
    -name '*.env*' -o \
    -name 'Makefile' -o \
    -name 'Dockerfile*' -o \
    -name '*.mod' -o \
    -name '*.sum' \
\) | grep -v vendor | grep -v node_modules | grep -v '.git/' | \
    xargs sed -i "s|${OLD_PATH}|${NEW_PATH}|g" 2>/dev/null || true

# 4. Run go mod tidy
echo "Running go mod tidy..."
go mod tidy 2>/dev/null || echo "Warning: go mod tidy failed (may need manual fix)"

echo ""
echo "Done. Module renamed from $OLD_PATH to $NEW_PATH"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Run tests: go test ./..."
echo "  3. Verify build: go build ./..."
