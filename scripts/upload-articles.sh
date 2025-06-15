#!/bin/bash

# Script to upload all markdown articles from a directory to the flashcards endpoint
# Usage: ./upload-articles.sh <directory_path> [endpoint_url]

set -e

# Default endpoint URL
DEFAULT_ENDPOINT="http://localhost:8080/notes"

# Check if directory argument is provided
if [ $# -lt 1 ]; then
    echo "Usage: $0 <directory_path> [endpoint_url]"
    echo "Example: $0 ./booknotes/system-design"
    exit 1
fi

DIRECTORY="$1"
ENDPOINT="${2:-$DEFAULT_ENDPOINT}"

# Check if directory exists
if [ ! -d "$DIRECTORY" ]; then
    echo "Error: Directory '$DIRECTORY' does not exist"
    exit 1
fi

# Counter for uploaded files
uploaded_count=0
failed_count=0

echo "Uploading articles from: $DIRECTORY"
echo "Target endpoint: $ENDPOINT"
echo "---"

# Function to upload a single file
upload_file() {
    local file_path="$1"
    local relative_path="${file_path#$DIRECTORY/}"
    
    echo "Uploading: $relative_path"
    
    # Read file content
    content=$(cat "$file_path")
    
    # Create JSON payload with proper escaping
    json_payload=$(printf '%s' "$content" | jq -R -s '{content: .}')
    
    # Upload to endpoint
    if curl --request POST \
        --url "$ENDPOINT" \
        --header 'Content-Type: application/json' \
        --data "$json_payload" \
        --silent \
        --show-error \
        --fail > /dev/null; then
        echo "✓ Successfully uploaded: $relative_path"
        ((uploaded_count++))
    else
        echo "✗ Failed to upload: $relative_path"
        ((failed_count++))
    fi
    
    # Small delay to avoid overwhelming the server
    sleep 0.5
}

# Find and upload all markdown files recursively in subdirectories
while IFS= read -r -d '' file; do
    upload_file "$file"
done < <(find "$DIRECTORY" -name "*.md" -type f -print0)

echo "---"
echo "Upload complete!"
echo "Successfully uploaded: $uploaded_count files"
echo "Failed uploads: $failed_count files"

if [ $failed_count -gt 0 ]; then
    exit 1
fi
