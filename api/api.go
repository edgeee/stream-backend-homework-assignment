package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GetStream/stream-backend-homework-assignment/api/validator"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// A DB provides a storage layer that persists messages.
type DB interface {
	ListMessages(ctx context.Context, limit int, offset int, excludeMsgIDs ...string) ([]Message, error)
	InsertMessage(ctx context.Context, msg Message) (Message, error)
	InsertReaction(ctx context.Context, reaction Reaction) (Reaction, error)
}

// A Cache provides a storage layer that caches messages.
type Cache interface {
	ListMessages(ctx context.Context) ([]Message, error)
	InsertMessage(ctx context.Context, msg Message) error
	InsertReaction(ctx context.Context, msgId string, reaction Reaction) error
}

type ErrorResponse struct {
	Kind   string                      `json:"kind"`
	Errors []validator.ValidationError `json:"errors"`
}

// API provides the REST endpoints for the application.
type API struct {
	Logger *slog.Logger
	DB     DB
	Cache  Cache
	Val    *validator.Validator

	once sync.Once
	mux  *http.ServeMux
}

// pageSize defines the number of items displayed on a single page in pagination.
var pageSize = 10

func (a *API) setupRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /messages", a.listMessages)
	mux.HandleFunc("POST /messages", a.createMessage)
	mux.HandleFunc("POST /messages/{messageID}/reactions", a.createReaction)

	a.mux = mux
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.once.Do(a.setupRoutes)
	a.Logger.Info("Request received", "method", r.Method, "path", r.URL.Path)
	a.mux.ServeHTTP(w, r)
}

func (a *API) respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		a.Logger.Error("Could not encode JSON body", "error", err.Error())
	}
}

func (a *API) respondError(w http.ResponseWriter, status int, err error, msg string) {
	type response struct {
		Error string `json:"error"`
	}
	a.Logger.Error("Error", "error", err.Error())
	a.respond(w, status, response{Error: msg})
}

func (a *API) validateBody(w http.ResponseWriter, s interface{}) bool {
	errs := a.Val.ValidateStruct(s)
	if errs != nil {
		a.respond(w, http.StatusBadRequest, &ErrorResponse{
			Errors: errs,
			Kind:   "body",
		})
		return false
	}
	return true
}

func (a *API) validateParam(w http.ResponseWriter, s interface{}, tag string) bool {
	errs := a.Val.Validate(s, tag)
	if errs != nil {
		a.respond(w, http.StatusBadRequest, &ErrorResponse{
			Errors: errs,
			Kind:   "param",
		})
		return false
	}
	return true
}

func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Messages []Message `json:"messages"`
	}

	p := r.URL.Query().Get("page")
	if p == "" {
		p = "1"
	}
	page, err := strconv.Atoi(p)

	if err != nil {
		a.respondError(w, http.StatusBadRequest, err, "Invalid page number")
		return
	}

	// Get messages from cache
	msgs, err := a.Cache.ListMessages(r.Context())
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, "Could not list messages")
		return
	}
	a.Logger.Info("Got messages from cache", "count", len(msgs))

	// Get any remaining messages from DB
	msgIDs := make([]string, len(msgs))
	for i, msg := range msgs {
		msgIDs[i] = msg.ID
	}

	dbMsgs, err := a.DB.ListMessages(r.Context(), pageSize, pageSize*(page-1), msgIDs...)
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, "Could not list messages")
		return
	}

	a.Logger.Info("Got remaining messages from DB", "count", len(dbMsgs))
	msgs = append(msgs, dbMsgs...)

	if msgs == nil {
		msgs = []Message{}
	}

	res := response{
		Messages: msgs,
	}

	a.respond(w, http.StatusOK, res)
}

func (a *API) createMessage(w http.ResponseWriter, r *http.Request) {
	type (
		request struct {
			Text   string `json:"text" validate:"required"`
			UserID string `json:"user_id" validate:"required"`
		}
		response struct {
			ID        string `json:"id"`
			Text      string `json:"text"`
			UserID    string `json:"user_id"`
			CreatedAt string `json:"created_at"`
		}
	)

	var body request
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		a.respondError(w, http.StatusBadRequest, err, "Could not decode request body")
		return
	}

	if valid := a.validateBody(w, &body); !valid {
		return
	}

	err = r.Body.Close()
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, "Could not close request body")
		return
	}

	msg, err := a.DB.InsertMessage(r.Context(), Message{
		Text:      body.Text,
		UserID:    body.UserID,
		CreatedAt: time.Now(),
	})
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, "Could not insert message")
		return
	}

	if err := a.Cache.InsertMessage(r.Context(), msg); err != nil {
		a.Logger.Error("Could not cache message", "error", err.Error())
	}

	res := response{
		ID:        msg.ID,
		Text:      msg.Text,
		UserID:    msg.UserID,
		CreatedAt: msg.CreatedAt.Format(time.RFC1123),
	}

	a.respond(w, http.StatusCreated, res)
}

func (a *API) createReaction(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Type   string `json:"type" validate:"required"`
		Score  int    `json:"score"`
		UserID string `json:"user_id" validate:"required"`
	}

	messageID := r.PathValue("messageID")
	if !a.validateParam(w, messageID, "required,uuid") {
		return
	}

	var body request
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		a.respondError(w, http.StatusBadRequest, err, "Could not decode request body")
		return
	}

	err = r.Body.Close()
	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, "Invalid request body")
		return
	}

	if !a.validateBody(w, &body) {
		return
	}

	reaction, err := a.DB.InsertReaction(r.Context(), Reaction{
		MessageID: messageID,
		Type:      body.Type,
		Score:     body.Score,
		UserID:    body.UserID,
		CreatedAt: time.Now(),
	})

	if err != nil {
		a.respondError(w, http.StatusInternalServerError, err, fmt.Sprintf("could not create reaction for message with id %s", messageID))
		return
	}

	err = a.Cache.InsertReaction(r.Context(), messageID, reaction)
	if err != nil {
		a.Logger.Error("Could not cache reaction", "error", err.Error())
		a.respondError(w, http.StatusInternalServerError, err, "Internal server error")
		return
	}

	a.respond(w, http.StatusCreated, Reaction{
		ID:        reaction.ID,
		MessageID: reaction.MessageID,
		Type:      reaction.Type,
		Score:     reaction.Score,
		UserID:    reaction.UserID,
		CreatedAt: reaction.CreatedAt,
	})
}
