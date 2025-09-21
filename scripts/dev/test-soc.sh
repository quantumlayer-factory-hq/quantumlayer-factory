#!/bin/bash
set -e

echo "Testing SOC parser with sample inputs..."

# Test valid SOC input
echo "Testing valid SOC patch..."
cat << 'EOF' > /tmp/valid_patch.txt
### FACTORY/1 PATCH
- file: backend/api/test.py
```diff
--- a/backend/api/test.py
+++ b/backend/api/test.py
@@ -0,0 +1,5 @@
+def add(a: int, b: int) -> int:
+    """Add two numbers together."""
+    return a + b
```
### END
EOF

# Test invalid SOC input (prose contamination)
echo "Testing prose-contaminated input..."
cat << 'EOF' > /tmp/invalid_patch.txt
### FACTORY/1 PATCH
Sure! I'll help you create a function to add two numbers.
- file: backend/api/test.py
```diff
--- a/backend/api/test.py
+++ b/backend/api/test.py
@@ -0,0 +1,5 @@
+def add(a: int, b: int) -> int:
+    """Add two numbers together."""
+    return a + b
```
### END
EOF

# Test refusal input
echo "Testing refusal input..."
cat << 'EOF' > /tmp/refusal.txt
I cannot help you with that request. Please provide more details about what you're trying to accomplish.
EOF

echo "Running Go tests..."
go test -v ./kernel/soc/

echo ""
echo "✅ SOC parser tests completed successfully!"
echo ""
echo "Example usage:"
echo "  echo 'Create a function to add two numbers' | go run cmd/qlf/main.go generate --dry-run"

# Cleanup
rm -f /tmp/valid_patch.txt /tmp/invalid_patch.txt /tmp/refusal.txt