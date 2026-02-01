#!/bin/bash
#
# Script de release pour Edgeo Drivers
# Usage: ./scripts/release.sh <major|minor|patch>
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
VERSION_FILE="$ROOT_DIR/modbus/version.go"
CHANGELOG_FILE="$ROOT_DIR/website/docs/changelog.md"
INDEX_FILE="$ROOT_DIR/website/docs/index.md"
CONFIG_FILE="$ROOT_DIR/website/docusaurus.config.ts"

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Vérifier les arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <major|minor|patch>"
    echo ""
    echo "Exemples:"
    echo "  $0 patch   # 1.0.0 -> 1.0.1"
    echo "  $0 minor   # 1.0.0 -> 1.1.0"
    echo "  $0 major   # 1.0.0 -> 2.0.0"
    exit 1
fi

BUMP_TYPE=$1

if [[ ! "$BUMP_TYPE" =~ ^(major|minor|patch)$ ]]; then
    log_error "Type de version invalide: $BUMP_TYPE (utilisez major, minor, ou patch)"
fi

# Lire la version actuelle
CURRENT_VERSION=$(grep 'Version = ' "$VERSION_FILE" | sed 's/.*"\(.*\)".*/\1/')
CURRENT_MAJOR=$(grep 'VersionMajor = ' "$VERSION_FILE" | sed 's/.*= \([0-9]*\).*/\1/')
CURRENT_MINOR=$(grep 'VersionMinor = ' "$VERSION_FILE" | sed 's/.*= \([0-9]*\).*/\1/')
CURRENT_PATCH=$(grep 'VersionPatch = ' "$VERSION_FILE" | sed 's/.*= \([0-9]*\).*/\1/')

log_info "Version actuelle: $CURRENT_VERSION"

# Calculer la nouvelle version
case $BUMP_TYPE in
    major)
        NEW_MAJOR=$((CURRENT_MAJOR + 1))
        NEW_MINOR=0
        NEW_PATCH=0
        ;;
    minor)
        NEW_MAJOR=$CURRENT_MAJOR
        NEW_MINOR=$((CURRENT_MINOR + 1))
        NEW_PATCH=0
        ;;
    patch)
        NEW_MAJOR=$CURRENT_MAJOR
        NEW_MINOR=$CURRENT_MINOR
        NEW_PATCH=$((CURRENT_PATCH + 1))
        ;;
esac

NEW_VERSION="$NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
log_info "Nouvelle version: $NEW_VERSION"

# Confirmer
read -p "Continuer avec la release v$NEW_VERSION? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warn "Release annulée"
    exit 0
fi

# Mettre à jour version.go
log_info "Mise à jour de $VERSION_FILE..."
sed -i '' "s/Version = \".*\"/Version = \"$NEW_VERSION\"/" "$VERSION_FILE"
sed -i '' "s/VersionMajor = .*/VersionMajor = $NEW_MAJOR/" "$VERSION_FILE"
sed -i '' "s/VersionMinor = .*/VersionMinor = $NEW_MINOR/" "$VERSION_FILE"
sed -i '' "s/VersionPatch = .*/VersionPatch = $NEW_PATCH/" "$VERSION_FILE"

# Mettre à jour index.md (badge version)
log_info "Mise à jour de $INDEX_FILE..."
sed -i '' "s/version-[0-9]*\.[0-9]*\.[0-9]*/version-$NEW_VERSION/" "$INDEX_FILE"
sed -i '' "s/@v[0-9]*\.[0-9]*\.[0-9]*/@v$NEW_VERSION/" "$INDEX_FILE"

# Mettre à jour docusaurus.config.ts
log_info "Mise à jour de $CONFIG_FILE..."
sed -i '' "s/label: '[0-9]*\.[0-9]*\.[0-9]*'/label: '$NEW_VERSION'/" "$CONFIG_FILE"

# Ajouter une entrée dans le changelog
log_info "Mise à jour de $CHANGELOG_FILE..."
TODAY=$(date +%Y-%m-%d)
CHANGELOG_ENTRY="## [$NEW_VERSION] - $TODAY

### Ajouté

-

### Modifié

-

### Corrigé

-

---

"

# Insérer après la première ligne "## ["
sed -i '' "/^## \[/,/^## \[/{
    /^## \[/{
        i\\
$CHANGELOG_ENTRY
    }
}" "$CHANGELOG_FILE" 2>/dev/null || {
    # Fallback: ajouter après "adhère au"
    sed -i '' "/adhère au/a\\
\\
$CHANGELOG_ENTRY" "$CHANGELOG_FILE"
}

log_info "Fichiers mis à jour!"
echo ""
echo "Prochaines étapes:"
echo "  1. Éditer $CHANGELOG_FILE pour documenter les changements"
echo "  2. git add -A && git commit -m \"Release v$NEW_VERSION\""
echo "  3. git tag v$NEW_VERSION"
echo "  4. git push origin main --tags"
echo ""

# Optionnel: créer une version Docusaurus
read -p "Créer une version Docusaurus archivée pour v$CURRENT_VERSION? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Création de la version Docusaurus $CURRENT_VERSION..."
    cd "$ROOT_DIR/website"
    npm run docusaurus docs:version "$CURRENT_VERSION"
    log_info "Version $CURRENT_VERSION archivée dans website/versioned_docs/"
fi

log_info "Release v$NEW_VERSION préparée!"
