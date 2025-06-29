package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"flashcards/models"
	"flashcards/services/quiz"

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
	service *quiz.Service
}

func NewQuizHandler(service *quiz.Service) *QuizHandler {
	return &QuizHandler{service: service}
}

func (h *QuizHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/quiz/generate", h.GenerateQuiz).Methods("POST")
	router.HandleFunc("/quiz/generate/stream", h.GenerateQuizStream).Methods("POST")
	router.HandleFunc("/quiz/configure", h.ConfigureQuiz).Methods("POST")
	router.HandleFunc("/quiz/rank", h.RankNotes).Methods("POST")
	router.HandleFunc("/quiz/conduct", h.ConductQuiz).Methods("POST")
	router.HandleFunc("/quiz/v2/configure", h.ConfigureQuizV2).Methods("POST")
	router.HandleFunc("/quiz/v2/conduct", h.ConductQuizV2).Methods("POST")
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

	updatedMessages := append(req.Messages, models.Message{
		Role:    "assistant",
		Content: result.Content,
	})

	response := QuizResponse{
		NoteIDs:  req.NoteIDs,
		Messages: updatedMessages,
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

func (h *QuizHandler) ConfigureQuiz(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received quiz configuration request")

	var req models.QuizConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode quiz config request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	result, err := h.service.ConfigureQuiz(req.Messages)
	if err != nil {
		log.Printf("[ERROR] Quiz configuration failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[INFO] Quiz configuration completed successfully, type: %s", result.Type)
	h.writeJSONResponse(w, http.StatusOK, result)
}

func (h *QuizHandler) RankNotes(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received note ranking request")

	var req models.NoteRankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode note ranking request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(req.NoteIDs) == 0 {
		log.Printf("[ERROR] No note IDs provided in ranking request")
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one note ID is required")
		return
	}

	if len(req.Topics) == 0 {
		log.Printf("[ERROR] No topics provided in ranking request")
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one topic is required")
		return
	}

	result, err := h.service.RankNotes(req.NoteIDs, req.Topics)
	if err != nil {
		log.Printf("[ERROR] Note ranking failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[INFO] Note ranking completed successfully, ranked %d notes", len(result.RankedNotes))
	h.writeJSONResponse(w, http.StatusOK, result)
}

func (h *QuizHandler) ConductQuiz(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received quiz conduct request")

	var req models.QuizConductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode quiz conduct request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if len(req.NoteIDs) == 0 {
		log.Printf("[ERROR] No note IDs provided in quiz conduct request")
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one note ID is required")
		return
	}

	if len(req.Topics) == 0 {
		log.Printf("[ERROR] No topics provided in quiz conduct request")
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one topic is required")
		return
	}

	result, err := h.service.ConductQuiz(req.NoteIDs, req.Topics, req.Messages)
	if err != nil {
		log.Printf("[ERROR] Quiz conduct failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[INFO] Quiz conduct completed successfully, response type: %s", result.Type)
	h.writeJSONResponse(w, http.StatusOK, result)
}

func (h *QuizHandler) ConfigureQuizV2(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received quiz v2 configuration request")

	var req models.QuizV2ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode quiz v2 config request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	result, err := h.service.ConfigureQuizV2(req.Messages)
	if err != nil {
		log.Printf("[ERROR] Quiz v2 configuration failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[INFO] Quiz v2 configuration completed successfully, type: %s", result.Type)
	h.writeJSONResponse(w, http.StatusOK, result)
}

func (h *QuizHandler) ConductQuizV2(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Received quiz v2 conduct request")

	var req models.QuizV2ConductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Failed to decode quiz v2 conduct request JSON: %v", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.QuizID <= 0 {
		log.Printf("[ERROR] Invalid quiz ID in quiz v2 conduct request: %d", req.QuizID)
		h.writeErrorResponse(w, http.StatusBadRequest, "Valid quiz ID is required")
		return
	}

	result, err := h.service.ConductQuizV2(req.QuizID, req.Messages)
	if err != nil {
		log.Printf("[ERROR] Quiz v2 conduct failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[INFO] Quiz v2 conduct completed successfully, response type: %s", result.Type)
	h.writeJSONResponse(w, http.StatusOK, result)
}
