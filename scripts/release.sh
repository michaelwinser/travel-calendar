#!/bin/sh
# Create a tagged release with auto-generated changelog.
#
# Usage:
#   scripts/release.sh patch    # v2.0.0 → v2.0.1
#   scripts/release.sh minor    # v2.0.0 → v2.1.0
#   scripts/release.sh major    # v2.0.0 → v3.0.0
#   scripts/release.sh v2.1.0   # Explicit version
#
# Steps:
#   1. Runs ./dev ci (refuses to release if it fails)
#   2. Generates changelog from conventional commits since last tag
#   3. Creates annotated git tag
#   4. Optionally pushes tag and creates GitHub release

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; DIM='\033[2m'; NC='\033[0m'

die() { printf "${RED}%s${NC}\n" "$1" >&2; exit 1; }

# --- Determine version ---

BUMP="${1:-patch}"

# Find the latest v2.x tag (ignore v1.x from old app)
LAST_TAG=$(git tag -l 'v2.*' --sort=-v:refname | head -1)
if [ -z "$LAST_TAG" ]; then
    LAST_TAG="v2.0.0"
    LAST_TAG_EXISTS=false
    echo "No v2.x tags found. Starting from v2.0.0"
else
    LAST_TAG_EXISTS=true
    echo "Last release: $LAST_TAG"
fi

# Parse version
MAJOR=$(echo "$LAST_TAG" | sed 's/v//' | cut -d. -f1)
MINOR=$(echo "$LAST_TAG" | sed 's/v//' | cut -d. -f2)
PATCH=$(echo "$LAST_TAG" | sed 's/v//' | cut -d. -f3)

case "$BUMP" in
    patch)  PATCH=$((PATCH + 1)) ;;
    minor)  MINOR=$((MINOR + 1)); PATCH=0 ;;
    major)  MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    v*)     # Explicit version
            MAJOR=$(echo "$BUMP" | sed 's/v//' | cut -d. -f1)
            MINOR=$(echo "$BUMP" | sed 's/v//' | cut -d. -f2)
            PATCH=$(echo "$BUMP" | sed 's/v//' | cut -d. -f3)
            ;;
    *)      die "Usage: release.sh [patch|minor|major|vX.Y.Z]" ;;
esac

NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
echo "New version: $NEW_TAG"
echo ""

# --- Pre-flight checks ---

if [ -n "$(git status --porcelain)" ]; then
    die "Working tree is dirty. Commit or stash changes first."
fi

if git tag -l "$NEW_TAG" | grep -q "$NEW_TAG"; then
    die "Tag $NEW_TAG already exists."
fi

# --- Run CI ---

echo "Running CI checks..."
./dev ci || die "CI failed. Fix errors before releasing."
echo ""

# --- Generate changelog ---

echo "Generating changelog..."

if [ "$LAST_TAG_EXISTS" = true ]; then
    RANGE="${LAST_TAG}..HEAD"
else
    RANGE="HEAD"
fi

CHANGELOG=$(git log "$RANGE" --pretty=format:"%s" --no-merges 2>/dev/null | awk '
    /^feat:/ || /^feat\(/ { feats = feats "- " $0 "\n" }
    /^fix:/ || /^fix\(/ { fixes = fixes "- " $0 "\n" }
    /^refactor:/ || /^test:/ || /^docs:/ || /^chore:/ { other = other "- " $0 "\n" }
    END {
        if (feats) printf "### Features\n\n%s\n", feats
        if (fixes) printf "### Fixes\n\n%s\n", fixes
        if (other) printf "### Other\n\n%s\n", other
    }
')

if [ -z "$CHANGELOG" ]; then
    CHANGELOG="No conventional commits found since $LAST_TAG."
fi

echo "$CHANGELOG"
echo ""

# --- Create tag ---

printf "Create tag ${GREEN}%s${NC}? [y/N] " "$NEW_TAG"
read -r CONFIRM
case "$CONFIRM" in
    y|Y)
        git tag -a "$NEW_TAG" -m "$(cat <<EOF
Release $NEW_TAG

$CHANGELOG
EOF
)"
        echo "Tagged $NEW_TAG"
        ;;
    *)
        echo "Aborted."
        exit 0
        ;;
esac

# --- Push and create release ---

printf "Push tag and create GitHub release? [y/N] "
read -r PUSH_CONFIRM
case "$PUSH_CONFIRM" in
    y|Y)
        git push origin "$NEW_TAG"
        if command -v gh >/dev/null 2>&1; then
            gh release create "$NEW_TAG" \
                --title "$NEW_TAG" \
                --notes "$CHANGELOG"
            echo "GitHub release created."
        else
            echo "Tag pushed. Install gh CLI to create GitHub releases automatically."
        fi
        ;;
    *)
        echo "Tag created locally. Push with: git push origin $NEW_TAG"
        ;;
esac
