package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type AnalyzeRequest struct {
	FEN       string `json:"fen"`
	Depth     int    `json:"depth,omitempty"`
	MoveTime  int    `json:"movetime_ms,omitempty"`
	MultiPV   int    `json:"multipv,omitempty"`
}

type AnalyzeResponse struct {
	BestMove   string   `json:"bestmove"`
	Evaluation string   `json:"evaluation,omitempty"`
	Raw        []string `json:"raw"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func registerAPIRoutes(mux *http.ServeMux, cfg *Config, log zerolog.Logger) {
	mux.HandleFunc("/analyze", func(w http.ResponseWriter, r *http.Request) {
		handleAnalyze(w, r, cfg, log)
	})
}

func handleAnalyze(w http.ResponseWriter, r *http.Request, cfg *Config, log zerolog.Logger) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed; use POST")
		return
	}

	var req AnalyzeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid JSON body: %v", err))
		return
	}

	req.FEN = strings.TrimSpace(req.FEN)
	if req.FEN == "" {
		writeJSONError(w, http.StatusBadRequest, "fen is required")
		return
	}

	if req.Depth == 0 {
		req.Depth = 15
	}
	if req.Depth < 1 || req.Depth > 20 {
		writeJSONError(w, http.StatusBadRequest, "depth must be between 1 and 20")
		return
	}
	if req.MoveTime < 0 || req.MoveTime > 30000 {
		writeJSONError(w, http.StatusBadRequest, "movetime_ms must be between 0 and 30000")
		return
	}
	if req.MultiPV < 0 || req.MultiPV > 5 {
		writeJSONError(w, http.StatusBadRequest, "multipv must be between 0 and 5")
		return
	}

	session, err := createEphemeralStockfishSession(cfg.Stockfish.Path, log)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start stockfish: %v", err))
		return
	}
	defer session.close()

	commandTimeout := cfg.Stockfish.CommandTimeout
	if commandTimeout <= 0 {
		commandTimeout = 30 * time.Second
	}

	if _, err := session.executeCommand("uci", commandTimeout); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("uci failed: %v", err))
		return
	}
	if _, err := session.executeCommand("isready", commandTimeout); err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("isready failed: %v", err))
		return
	}
	if req.MultiPV > 1 {
		if err := session.sendCommand(fmt.Sprintf("setoption name MultiPV value %d", req.MultiPV)); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("multipv option failed: %v", err))
			return
		}
		if _, err := session.executeCommand("isready", commandTimeout); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("isready after multipv failed: %v", err))
			return
		}
	}
	if err := session.sendCommand("position fen " + req.FEN); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("position setup failed: %v", err))
		return
	}

	analysisCommand := fmt.Sprintf("go depth %d", req.Depth)
	if req.MoveTime > 0 {
		analysisCommand = fmt.Sprintf("go movetime %d", req.MoveTime)
	}

	raw, err := session.executeCommand(analysisCommand, commandTimeout)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("analysis failed: %v", err))
		return
	}

	response := AnalyzeResponse{
		BestMove:   extractBestMove(raw),
		Evaluation: extractEvaluation(raw),
		Raw:        raw,
	}
	if response.BestMove == "" {
		writeJSONError(w, http.StatusInternalServerError, "stockfish did not return a bestmove")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func extractBestMove(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		if len(fields) >= 2 && fields[0] == "bestmove" {
			return fields[1]
		}
	}
	return ""
}

func extractEvaluation(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		fields := strings.Fields(lines[i])
		for j := 0; j < len(fields)-2; j++ {
			if fields[j] == "score" {
				return fields[j+1] + " " + fields[j+2]
			}
		}
	}
	return ""
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
