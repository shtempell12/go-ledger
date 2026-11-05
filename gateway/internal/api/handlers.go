package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/shtempell12/go-ledger/ledger"
)

const maxBodyBytes = 1 << 20

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	tx, err := req.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tx.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := ledger.AddTransaction(tx)
	switch {
	case errors.Is(err, ledger.ErrBudgetExceeded):
		writeError(w, http.StatusConflict, ledger.ErrBudgetExceeded.Error())
	case err != nil:
		writeInternalError(w, err)
	default:
		writeJSON(w, http.StatusCreated, newTransactionResponse(created))
	}
}

func listTransactions(w http.ResponseWriter, r *http.Request) {
	txs, err := ledger.ListTransactions()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	resp := make([]TransactionResponse, 0, len(txs))
	for _, tx := range txs {
		resp = append(resp, newTransactionResponse(tx))
	}
	writeJSON(w, http.StatusOK, resp)
}

func createBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	b := req.toDomain()
	if err := b.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ledger.SetBudget(b); err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newBudgetResponse(b))
}

func listBudgets(w http.ResponseWriter, r *http.Request) {
	budgets, err := ledger.ListBudgets()
	if err != nil {
		writeInternalError(w, err)
		return
	}
	resp := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		resp = append(resp, newBudgetResponse(b))
	}
	writeJSON(w, http.StatusOK, resp)
}

// decodeJSON читает тело запроса в dst. При ошибке сам отвечает 400 и возвращает false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeInternalError логирует причину, а клиенту отдаёт общее сообщение.
func writeInternalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
