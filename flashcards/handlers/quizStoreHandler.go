package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"flashcards/models"
	"flashcards/services"

	"github.com/gorilla/mux"
)

type QuizStoreHandler struct {
	service *services.QuizStoreService
}

func NewQuizStoreHandler(service *services.QuizStoreService) *QuizStoreHandler {
	return &QuizStoreHandler{service: service}
}

func (h *QuizStoreHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/quizzes", h.CreateQuiz).Methods("POST")
	router.HandleFunc("/quizzes", h.GetAllQuizzes).Methods("GET")
	router.HandleFunc("/quizzes/{id:[0-9]+}", h.GetQuizByID).Methods("GET")
	router.HandleFunc("/quizzes/{id:[0-9]+}", h.UpdateQuiz).Methods("PUT")
	router.HandleFunc("/quizzes/{id:[0-9]+}", h.DeleteQuiz).Methods("DELETE")
}

func (h *QuizStoreHandler) CreateQuiz(w http.ResponseWriter, r *http.Request) {
	var req models.CreateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	quiz, err := h.service.CreateQuiz(&req)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, quiz)
}

func (h *QuizStoreHandler) GetAllQuizzes(w http.ResponseWriter, r *http.Request) {
	quizzes, err := h.service.GetAllQuizzes()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve quizzes")
		return
	}

	h.writeJSONResponse(w, http.StatusOK, quizzes)
}

func (h *QuizStoreHandler) GetQuizByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	quiz, err := h.service.GetQuizByID(id)
	if err != nil {
		if containsQuizNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve quiz")
		}
		return
	}

	h.writeJSONResponse(w, http.StatusOK, quiz)
}

func (h *QuizStoreHandler) UpdateQuiz(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	var req models.UpdateQuizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	err = h.service.UpdateQuiz(id, &req)
	if err != nil {
		if containsQuizNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *QuizStoreHandler) DeleteQuiz(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid quiz ID")
		return
	}

	err = h.service.DeleteQuiz(id)
	if err != nil {
		if containsQuizNotFound(err.Error()) {
			h.writeErrorResponse(w, http.StatusNotFound, err.Error())
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete quiz")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *QuizStoreHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *QuizStoreHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func containsQuizNotFound(message string) bool {
	return len(message) > 0 && (message[len(message)-9:] == "not found" ||
		message[:len("quiz with id")] == "quiz with id")
}