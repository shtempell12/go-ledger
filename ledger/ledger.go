// Package ledger содержит бизнес-логику учёта расходов и бюджетов.
package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

// ErrBudgetExceeded возвращается, когда транзакция превышает лимит бюджета категории.
var ErrBudgetExceeded = errors.New("budget exceeded")

// ErrInvalidTransaction возвращается для некорректных транзакций.
var ErrInvalidTransaction = errors.New("invalid transaction")

// ErrInvalidBudget возвращается для некорректных бюджетов.
var ErrInvalidBudget = errors.New("invalid budget")

// maxCents — граница суммы в копейках: столбцы денег имеют тип NUMERIC(14,2).
const maxCents = 1e14

// Transaction — одна операция расхода.
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        time.Time
}

// Budget — лимит расходов на категорию.
type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

// CategoryTotal — сумма расходов по одной категории.
type CategoryTotal struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// ReportSummary — отчёт о расходах за период; даты в формате YYYY-MM-DD.
type ReportSummary struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	Categories []CategoryTotal `json:"categories"`
	Total      float64         `json:"total"`
}

// Validate проверяет поля транзакции. Пустая дата допустима: AddTransaction подставит текущую.
func (t Transaction) Validate() error {
	if c := cents(t.Amount); c <= 0 || c >= maxCents {
		return fmt.Errorf("%w: amount must be between 0.01 and 999999999999.99, got %v", ErrInvalidTransaction, t.Amount)
	}
	if strings.TrimSpace(t.Category) == "" {
		return fmt.Errorf("%w: category is empty", ErrInvalidTransaction)
	}
	return nil
}

// Validate проверяет поля бюджета.
func (b Budget) Validate() error {
	if strings.TrimSpace(b.Category) == "" {
		return fmt.Errorf("%w: category is empty", ErrInvalidBudget)
	}
	if c := cents(b.Limit); c <= 0 || c >= maxCents {
		return fmt.Errorf("%w: limit must be between 0.01 and 999999999999.99, got %v", ErrInvalidBudget, b.Limit)
	}
	return nil
}

// cents переводит сумму в копейки, чтобы сравнивать деньги без погрешности float64.
func cents(amount float64) float64 {
	return math.Round(amount * 100)
}

// exceedsLimit сообщает, выйдет ли сумма за лимит после добавления новой транзакции.
func exceedsLimit(spent, amount, limit float64) bool {
	return cents(spent)+cents(amount) > cents(limit)
}

// LoadBudgets читает JSON-массив бюджетов из r и добавляет их через SetBudget.
// Если хотя бы один бюджет некорректен, ни один не применяется.
func LoadBudgets(r io.Reader) error {
	var budgets []Budget
	if err := json.NewDecoder(r).Decode(&budgets); err != nil {
		return fmt.Errorf("load budgets: decode JSON: %w", err)
	}

	for i, b := range budgets {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("load budgets: item %d: %w", i, err)
		}
	}

	for _, b := range budgets {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("load budgets: %w", err)
		}
	}
	return nil
}
