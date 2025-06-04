package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flashcards/models"
	"flashcards/services"

	"github.com/gorilla/mux"
)

type FlashcardHandler struct {
	service *services.FlashcardService
}

func NewFlashcardHandler(service *services.FlashcardService) *FlashcardHandler {
	return &FlashcardHandler{service: service}
}

func (h *FlashcardHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/flashcards", h.CreateFlashcard).Methods("POST")
	router.HandleFunc("/flashcards", h.GetAllFlashcards).Methods("GET")
	router.HandleFunc("/flashcards/{id:[0-9]+}", h.GetFlashcardByID).Methods("GET")
	router.HandleFunc("/flashcards/{id:[0-9]+}", h.UpdateFlashcard).Methods("PUT")
	router.HandleFunc("/flashcards/{id:[0-9]+}", h.DeleteFlashcard).Methods("DELETE")
}

func (h *FlashcardHandler) CreateFlashcard(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFlashcardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	flashcard, err := h.service.CreateFlashcard(&req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, flashcard)
}

func (h *FlashcardHandler) GetAllFlashcards(w http.ResponseWriter, r *http.Request) {
	flashcards, err := h.service.GetAllFlashcards()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve flashcards")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, flashcards)
}

func (h *FlashcardHandler) GetFlashcardByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid flashcard ID")
		return
	}

	flashcard, err := h.service.GetFlashcardByID(id)
	if err != nil {
		if containsFlashcardNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve flashcard")
		}
		return
	}

	h.writeJSONResponse(w, http.StatusOK, flashcard)
}

func (h *FlashcardHandler) UpdateFlashcard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid flashcard ID")
		return
	}

	var req models.UpdateFlashcardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	flashcard, err := h.service.UpdateFlashcard(id, &req)
	if err != nil {
		if containsFlashcardNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	h.writeJSONResponse(w, http.StatusOK, flashcard)
}

func (h *FlashcardHandler) DeleteFlashcard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid flashcard ID")
		return
	}

	err = h.service.DeleteFlashcard(id)
	if err != nil {
		if containsFlashcardNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete flashcard")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FlashcardHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *FlashcardHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func containsFlashcardNotFound(message string) bool {
	return len(message) > 0 && (message[len(message)-9:] == "not found" ||
		message[:len("flashcard with id")] == "flashcard with id")
}
