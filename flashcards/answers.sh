# incorrect
curl -X POST http://localhost:8080/quiz/generate \
    -H "Content-Type: application/json" \
    -d '{"messages": [{"role": "assistant", "content": "**Quiz Question:**\n\nWhich significant event in English history occurred in 1215 and is considered a foundational document limiting the powers of the monarchy?\n\nA) The signing of the Magna Carta  \nB) The execution of King Charles I  \nC) The establishment of the Commonwealth  \nD) The defeat of the Spanish Armada  \n\n**Correct Answer:** A) The signing of the Magna Carta"}, {"role": "user", "content": "B) The execution of King Charles I"}]}' | jq

# correct
  curl -X POST http://localhost:8080/quiz/generate \
    -H "Content-Type: application/json" \
    -d '{"messages": [{"role": "assistant", "content": "**Quiz Question:**\n\nWhich significant event in English history occurred in 1215 and is considered a foundational document limiting the powers of the monarchy?\n\nA) The signing of the Magna Carta  \nB) The execution of King Charles I  \nC) The establishment of the Commonwealth  \nD) The defeat of the Spanish Armada  \n\n**Correct Answer:** A) The signing of the Magna Carta"}, {"role": "user", "content": "A) The signing of the Magna Carta"}]}' | jq


