package ledger

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Хранилище в памяти, общее для всего пакета.
var (
	mu           sync.Mutex
	transactions []Transaction
	budgets      = make(map[string]Budget)
	nextID       = 1
)

// SetBudget добавляет новый бюджет или обновляет существующий для категории.
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	budgets[b.Category] = b
	return nil
}

// ListBudgets возвращает бюджеты, отсортированные по категории.
func ListBudgets() ([]Budget, error) {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Budget, 0, len(budgets))
	for _, b := range budgets {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Category < out[j].Category })
	return out, nil
}

// AddTransaction проверяет транзакцию и бюджет её категории.
// Если бюджет будет превышен, транзакция не сохраняется и возвращается ошибка.
// Возвращает сохранённую транзакцию с заполненными ID и датой.
func AddTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}
	mu.Lock()
	defer mu.Unlock()

	if b, ok := budgets[tx.Category]; ok {
		var spent float64
		for _, t := range transactions {
			if t.Category == tx.Category {
				spent += t.Amount
			}
		}
		if exceedsLimit(spent, tx.Amount, b.Limit) {
			return Transaction{}, fmt.Errorf("%w: category %q, spent %.2f + new %.2f > limit %.2f",
				ErrBudgetExceeded, tx.Category, spent, tx.Amount, b.Limit)
		}
	}

	tx.ID = nextID
	nextID++
	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}
	transactions = append(transactions, tx)
	return tx, nil
}

// ListTransactions возвращает транзакции: сначала новые.
func ListTransactions() ([]Transaction, error) {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Transaction, len(transactions))
	copy(out, transactions)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}
