#!/bin/bash

# Quiz API Evaluation Script
# Runs test scenarios against the quiz endpoint and accumulates results

API_URL="http://localhost:8080/quiz/generate"
OUTPUT_FILE="quiz_evaluation_results.txt"

# Check if jq is available for JSON formatting
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required for JSON formatting. Please install jq."
    exit 1
fi

# Function to clear results file and add header
clear_results_file() {
    cat > "$OUTPUT_FILE" << EOF
=== QUIZ EVALUATION RESULTS ===
Generated: $(date)

EOF
}

# Function to call the quiz API
call_quiz_api() {
    local payload="$1"
    curl -s -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "$payload"
}

# Function to run a test scenario
run_test_scenario() {
    local description="$1"
    local payload="$2"
    
    echo "Running: $description"
    
    # Add scenario header to output file
    cat >> "$OUTPUT_FILE" << EOF
SCENARIO: $description
REQUEST: $payload
RESPONSE:
EOF
    
    # Call API and format response
    local response=$(call_quiz_api "$payload")
    
    if [ $? -eq 0 ] && [ -n "$response" ]; then
        echo "$response" | jq '.' >> "$OUTPUT_FILE" 2>/dev/null
        if [ $? -ne 0 ]; then
            echo "Raw response (JSON parsing failed): $response" >> "$OUTPUT_FILE"
        fi
    else
        echo "API call failed or returned empty response" >> "$OUTPUT_FILE"
    fi
    
    # Add separator
    echo -e "\n---\n" >> "$OUTPUT_FILE"
}

# Initialize results file
clear_results_file

echo "Starting quiz API evaluation..."

# Test 1: Initial quiz generation (empty messages)
run_test_scenario \
    "Fresh quiz start - should generate first question based on notes" \
    '{"messages": []}'

# Test 2: Correct answer follow-up
run_test_scenario \
    "Student provides correct answer - should ask follow-up question" \
    '{"messages": [
        {"role": "assistant", "content": "What were the major Anglo-Saxon kingdoms established in England?"},
        {"role": "user", "content": "The major Anglo-Saxon kingdoms were Wessex, Mercia, Northumbria, and East Anglia, collectively known as the Heptarchy."}
    ]}'

# Test 3: Incorrect answer follow-up
run_test_scenario \
    "Student provides wrong answer - should guide/correct and continue" \
    '{"messages": [
        {"role": "assistant", "content": "In what year did the Norman Conquest occur?"},
        {"role": "user", "content": "The Norman Conquest happened in 1067."}
    ]}'

# Test 4: Off-topic response
run_test_scenario \
    "Student goes completely off-topic - should redirect to quiz content" \
    '{"messages": [
        {"role": "assistant", "content": "Who was King Alfred the Great and what was he known for?"},
        {"role": "user", "content": "I think pizza is the best food ever invented. What do you think about Italian cuisine?"}
    ]}'

# Test 5: Partial/vague answer
run_test_scenario \
    "Student gives incomplete answer - should probe for more detail" \
    '{"messages": [
        {"role": "assistant", "content": "What was the significance of the Magna Carta?"},
        {"role": "user", "content": "It was important for limiting power."}
    ]}'

# Test 6: Multi-turn conversation
run_test_scenario \
    "Extended quiz conversation - should maintain context and progression" \
    '{"messages": [
        {"role": "assistant", "content": "What were the Wars of the Roses?"},
        {"role": "user", "content": "Civil wars between the Houses of Lancaster and York for control of the English throne."},
        {"role": "assistant", "content": "Excellent! Can you tell me how these wars ended?"},
        {"role": "user", "content": "Henry Tudor defeated Richard III at the Battle of Bosworth Field in 1487."}
    ]}'

# Test 7: Student asks a question back
run_test_scenario \
    "Student asks a question instead of answering - should handle gracefully" \
    '{"messages": [
        {"role": "assistant", "content": "What led to the English Reformation under Henry VIII?"},
        {"role": "user", "content": "Can you tell me more about Catherine of Aragon first?"}
    ]}'

# Test 8: Very short/minimal response
run_test_scenario \
    "Student gives minimal one-word answer - should encourage elaboration" \
    '{"messages": [
        {"role": "assistant", "content": "What was significant about Elizabeth I'\''s reign?"},
        {"role": "user", "content": "Good."}
    ]}'

echo "Quiz API evaluation completed. Results saved to $OUTPUT_FILE"
echo "Review the output file to analyze AI behavior across different scenarios."