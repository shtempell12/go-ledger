// Package api — HTTP-слой Gateway: DTO, обработчики, маршруты и middleware.
package api

import (
	"fmt"
	"time"

	"github.com/shtempell12/go-ledger/ledger"
)

const dateLayout = "2006-01-02"

// CreateTransactionRequest — тело POST /api/transactions.
type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// TransactionResponse — транзакция в ответах API.
type TransactionResponse struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// CreateBudgetRequest — тело POST /api/budgets.
type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

// BudgetResponse — бюджет в ответах API.
type BudgetResponse struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

func (r CreateTransactionRequest) toDomain() (ledger.Transaction, error) {
	tx := ledger.Transaction{
		Amount:      r.Amount,
		Category:    r.Category,
		Description: r.Description,
	}
	if r.Date != "" {
		date, err := time.Parse(dateLayout, r.Date)
		if err != nil {
			return ledger.Transaction{}, fmt.Errorf("date must be in YYYY-MM-DD format, got %q", r.Date)
		}
		tx.Date = date
	}
	return tx, nil
}

func (r CreateBudgetRequest) toDomain() ledger.Budget {
	return ledger.Budget{Category: r.Category, Limit: r.Limit}
}

func newTransactionResponse(tx ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date.Format(dateLayout),
	}
}

func newBudgetResponse(b ledger.Budget) BudgetResponse {
	return BudgetResponse{Category: b.Category, Limit: b.Limit}
}

// CategoryTotalResponse — сумма расходов по категории в отчёте.
type CategoryTotalResponse struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// ReportSummaryResponse — ответ GET /api/reports/summary.
type ReportSummaryResponse struct {
	From       string                  `json:"from"`
	To         string                  `json:"to"`
	Categories []CategoryTotalResponse `json:"categories"`
	Total      float64                 `json:"total"`
}

func newReportSummaryResponse(report ledger.ReportSummary) ReportSummaryResponse {
	categories := make([]CategoryTotalResponse, 0, len(report.Categories))
	for _, c := range report.Categories {
		categories = append(categories, CategoryTotalResponse{Category: c.Category, Total: c.Total})
	}
	return ReportSummaryResponse{From: report.From, To: report.To, Categories: categories, Total: report.Total}
}
