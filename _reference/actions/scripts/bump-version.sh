#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Check if the correct number of arguments is provided
if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <version> <action1> [action2 ...]"
  echo "Example: $0 v1.2.3 prebuild-devcontainer my-other-action"
  exit 1
fi

# Assign the first argument to VERSION, then shift to get the rest as actions
VERSION=$1
shift
ACTIONS=("$@")

# Validate the version format (must be vX.Y.Z)
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "❌ Error: Version must be in the semantic format vX.Y.Z (e.g., v1.2.3)"
  exit 1
fi

# Extract the major and minor versions
# e.g., if VERSION=v1.2.3, MAJOR_VERSION=v1 and MINOR_VERSION=v1.2
MAJOR_VERSION=$(echo "$VERSION" | cut -d'.' -f1)
MINOR_VERSION=$(echo "$VERSION" | cut -d'.' -f1,2)

# Format the actions array into a readable comma-separated string
ACTIONS_LIST=$(IFS=, ; echo "${ACTIONS[*]}")
MESSAGE="Release $VERSION updating: $ACTIONS_LIST"

echo "🔖 Creating specific tag: $VERSION"
echo "📝 Message: $MESSAGE"
git tag -a "$VERSION" -m "$MESSAGE"

echo "🔄 Updating floating minor tag: $MINOR_VERSION"
git tag -fa "$MINOR_VERSION" -m "Update $MINOR_VERSION to point to $VERSION"

echo "🔄 Updating floating major tag: $MAJOR_VERSION"
git tag -fa "$MAJOR_VERSION" -m "Update $MAJOR_VERSION to point to $VERSION"

echo "🚀 Pushing tags to origin..."
git push origin "$VERSION"
git push origin "refs/tags/$MINOR_VERSION" --force
git push origin "refs/tags/$MAJOR_VERSION" --force

echo "✅ Successfully bumped to $VERSION and updated $MINOR_VERSION and $MAJOR_VERSION!"
