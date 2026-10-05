package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// ErrBudgetExceeded возвращается, когда транзакция превышает лимит бюджета категории.
var ErrBudgetExceeded = errors.New("budget exceeded")

// ErrInvalidTransaction возвращается для некорректных транзакций.
var ErrInvalidTransaction = errors.New("invalid transaction")

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
	Period   string  `json:"period,omitempty"` // например "2026-10"; пока только информационное поле
}

// Ledger хранит транзакции и бюджеты.
type Ledger struct {
	transactions []Transaction
	budgets      map[string]Budget
	nextID       int
}

// NewLedger создаёт пустой Ledger.
func NewLedger() *Ledger {
	return &Ledger{
		budgets: make(map[string]Budget),
		nextID:  1,
	}
}

// SetBudget добавляет новый бюджет или обновляет существующий для категории.
func (l *Ledger) SetBudget(b Budget) {
	l.budgets[b.Category] = b
}

// Budgets возвращает копию хранилища бюджетов.
func (l *Ledger) Budgets() map[string]Budget {
	out := make(map[string]Budget, len(l.budgets))
	for k, v := range l.budgets {
		out[k] = v
	}
	return out
}

// SpentByCategory возвращает сумму всех транзакций по категории.
func (l *Ledger) SpentByCategory(category string) float64 {
	var total float64
	for _, tx := range l.transactions {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total
}

// AddTransaction проверяет транзакцию и бюджет её категории.
// Если бюджет будет превышен, транзакция не сохраняется и возвращается ошибка.
func (l *Ledger) AddTransaction(tx Transaction) error {
	if tx.Amount <= 0 {
		return fmt.Errorf("%w: amount must be positive, got %.2f", ErrInvalidTransaction, tx.Amount)
	}
	if tx.Category == "" {
		return fmt.Errorf("%w: category is empty", ErrInvalidTransaction)
	}

	if b, ok := l.budgets[tx.Category]; ok {
		spent := l.SpentByCategory(tx.Category)
		if spent+tx.Amount > b.Limit {
			return fmt.Errorf("%w: category %q, spent %.2f + new %.2f > limit %.2f",
				ErrBudgetExceeded, tx.Category, spent, tx.Amount, b.Limit)
		}
	}

	tx.ID = l.nextID
	l.nextID++
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}
	l.transactions = append(l.transactions, tx)
	return nil
}

// Transactions возвращает копию списка сохранённых транзакций.
func (l *Ledger) Transactions() []Transaction {
	out := make([]Transaction, len(l.transactions))
	copy(out, l.transactions)
	return out
}

// LoadBudgets читает JSON-массив бюджетов из r и добавляет их через SetBudget.
// Если хотя бы один бюджет некорректен, ни один не применяется.
func (l *Ledger) LoadBudgets(r io.Reader) error {
	var budgets []Budget
	if err := json.NewDecoder(r).Decode(&budgets); err != nil {
		return fmt.Errorf("load budgets: decode JSON: %w", err)
	}

	for i, b := range budgets {
		if b.Category == "" {
			return fmt.Errorf("load budgets: item %d: category is empty", i)
		}
		if b.Limit <= 0 {
			return fmt.Errorf("load budgets: item %d (%q): limit must be positive, got %.2f", i, b.Category, b.Limit)
		}
	}

	for _, b := range budgets {
		l.SetBudget(b)
	}
	return nil
}
