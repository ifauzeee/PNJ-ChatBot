#!/bin/bash

# Script to setup quality-gate.sh as a git pre-push hook

HOOK_FILE=".git/hooks/pre-push"
SCRIPT_PATH="scripts/quality-gate.sh"

echo "🔧 Setting up pre-push hook..."

# Check if .git directory exists
if [ ! -d ".git" ]; then
    echo "❌ Error: .git directory not found. Are you in the root of the repository?"
    exit 1
fi

# Create hook file
cat > "$HOOK_FILE" <<EOF
#!/bin/bash

# Run quality gate before pushing
./$SCRIPT_PATH
EOF

# Make hook executable
chmod +x "$HOOK_FILE"
chmod +x "$SCRIPT_PATH"

echo "✅ Pre-push hook installed successfully at $HOOK_FILE"
echo "🚀 The quality gate will now run automatically before every 'git push'."
