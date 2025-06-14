package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"flashcards/models"
	"flashcards/services"

	"github.com/gorilla/mux"
)

type QuizRequest struct {
	NoteIDs  []int            `json:"note_ids"`
	Messages []models.Message `json:"messages"`
}

type QuizResponse struct {
	NoteIDs  []int            `json:"note_ids"`
	Messages []models.Message `json:"messages"`
}

type QuizHandler struct {
	service *services.QuizService
}

func NewQuizHandler(service *services.QuizService) *QuizHandler {
	return &QuizHandler{service: service}
}

func (h *QuizHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/quiz/generate", h.GenerateQuiz).Methods("POST")
	router.HandleFunc("/quiz/generate/stream", h.GenerateQuizStream).Methods("POST")
}

func (h *QuizHandler) GenerateQuiz(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received quiz generation request")

	var req QuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode quiz request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	result, err := h.service.GenerateQuizResponse(req.NoteIDs, req.Messages)
	if err != nil {
		log.Printf("[ERROR] Quiz generation failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := QuizResponse{
		NoteIDs:  result.NoteIDs,
		Messages: result.Messages,
	}

	log.Printf("[INFO] Quiz generation completed successfully")
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *QuizHandler) GenerateQuizStream(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received streaming quiz generation request")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req QuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode streaming quiz request JSON: %v", err)
		fmt.Fprintf(w, "Error: Invalid JSON payload\n\n")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[ERROR] Streaming not supported")
		fmt.Fprintf(w, "Error: Streaming not supported\n\n")
		return
	}

	err := h.service.GenerateQuizResponseStream(req.NoteIDs, req.Messages, func(token string) {
		fmt.Fprintf(w, "%s", token)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("[ERROR] Streaming quiz generation failed: %v", err)
		fmt.Fprintf(w, "Error: %s", err.Error())
		return
	}

	log.Printf("[INFO] Streaming quiz generation completed successfully")
}

func (h *QuizHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *QuizHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
