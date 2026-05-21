#!/bin/bash

# Run this after spec execution is complete and validated

set -e

echo "Starting legacy cleanup..."

# Files to remove
REMOVE_FILES=(
    ".kiro/prompts/plan-with-team.md"
    ".kiro/prompts/plan-with-team.md.deprecated"
    ".kiro/agents/team-lead.json"
    ".kiro/agents/team-lead-prompt.md"
    ".kiro/agents/builder.json"
    ".kiro/agents/builder-prompt.md"
    ".kiro/agents/validator.json"
    ".kiro/agents/validator-prompt.md"
    ".kiro/agents/documenter.json"
    ".kiro/agents/documenter-prompt.md"
    ".kiro/templates/incident-report.md"
    "scripts/worktree-create.sh"
    "scripts/worktree-merge.sh"
)

# Directories to remove
REMOVE_DIRS=(
    ".kiro/skills/plan-with-team"
)

# Remove files
echo "Removing files:"
for file in "${REMOVE_FILES[@]}"; do
    if [ -f "$file" ]; then
        rm "$file"
        echo "  - Removed: $file"
    fi
done

# Remove directories
echo "Removing directories:"
for dir in "${REMOVE_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        rm -rf "$dir"
        echo "  - Removed: $dir"
    fi
done

# Create .kiro-krew directory
mkdir -p .kiro-krew
echo "Created: .kiro-krew/"

# Copy new agent files
if [ -f "templates/agents/krew-lead.json" ]; then
    cp "templates/agents/krew-lead.json" ".kiro/agents/"
    echo "Added: .kiro/agents/krew-lead.json"
fi

if [ -f "templates/agents/krew-lead-prompt.md" ]; then
    cp "templates/agents/krew-lead-prompt.md" ".kiro/agents/"
    echo "Added: .kiro/agents/krew-lead-prompt.md"
fi

# Update .gitignore
echo "Updating .gitignore..."
if ! grep -q "^\.kiro-krew/$" .gitignore 2>/dev/null; then
    echo ".kiro-krew/" >> .gitignore
    echo "Added .kiro-krew/ to .gitignore"
fi

if ! grep -q "^\.worktrees/$" .gitignore 2>/dev/null; then
    echo ".worktrees/" >> .gitignore
    echo "Added .worktrees/ to .gitignore"
fi

echo "Legacy cleanup complete!"