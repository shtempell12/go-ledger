package api

import (
	"net/http"
	"sort"
	"strings"
)

// NewRouter собирает маршруты /api и подключает middleware.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/transactions", byMethod(map[string]http.HandlerFunc{
		http.MethodGet:  listTransactions,
		http.MethodPost: createTransaction,
	}))
	mux.HandleFunc("/api/budgets", byMethod(map[string]http.HandlerFunc{
		http.MethodGet:  listBudgets,
		http.MethodPost: createBudget,
	}))
	mux.HandleFunc("/api/reports/summary", byMethod(map[string]http.HandlerFunc{
		http.MethodGet: reportSummary,
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	return logging(jsonContentType(mux))
}

// byMethod выбирает обработчик по HTTP-методу. Встроенные методовые шаблоны ServeMux
// не подходят: их ответы 404 и 405 идут не в едином формате {"error":"..."}.
func byMethod(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	allowed := make([]string, 0, len(handlers))
	for method := range handlers {
		allowed = append(allowed, method)
	}
	sort.Strings(allowed)

	return func(w http.ResponseWriter, r *http.Request) {
		handler, ok := handlers[r.Method]
		if !ok {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handler(w, r)
	}
}
