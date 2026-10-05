package main

import (
	"errors"
	"strings"
	"testing"
)

func TestAddTransactionWithinBudget(t *testing.T) {
	l := NewLedger()
	l.SetBudget(Budget{Category: "еда", Limit: 5000})

	if err := l.AddTransaction(Transaction{Amount: 3000, Category: "еда"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := l.AddTransaction(Transaction{Amount: 2000, Category: "еда"}); err != nil {
		t.Fatalf("exactly at limit must be allowed: %v", err)
	}
	if got := len(l.Transactions()); got != 2 {
		t.Fatalf("want 2 transactions, got %d", got)
	}
}

func TestAddTransactionExceedsBudget(t *testing.T) {
	l := NewLedger()
	l.SetBudget(Budget{Category: "еда", Limit: 5000})
	_ = l.AddTransaction(Transaction{Amount: 4000, Category: "еда"})

	err := l.AddTransaction(Transaction{Amount: 1500, Category: "еда"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}
	if got := len(l.Transactions()); got != 1 {
		t.Fatalf("rejected transaction must not be saved: want 1, got %d", got)
	}
}

func TestAddTransactionNoBudget(t *testing.T) {
	l := NewLedger()
	if err := l.AddTransaction(Transaction{Amount: 1e6, Category: "прочее"}); err != nil {
		t.Fatalf("category without budget must be allowed: %v", err)
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	l := NewLedger()
	l.SetBudget(Budget{Category: "транспорт", Limit: 1000})
	if err := l.AddTransaction(Transaction{Amount: 1500, Category: "транспорт"}); err == nil {
		t.Fatal("want error before limit update")
	}
	l.SetBudget(Budget{Category: "транспорт", Limit: 2000})
	if err := l.AddTransaction(Transaction{Amount: 1500, Category: "транспорт"}); err != nil {
		t.Fatalf("want success after limit update: %v", err)
	}
}

func TestLoadBudgets(t *testing.T) {
	l := NewLedger()
	err := l.LoadBudgets(strings.NewReader(`[{"category":"еда","limit":100},{"category":"связь","limit":50}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b := l.Budgets(); b["еда"].Limit != 100 || b["связь"].Limit != 50 {
		t.Fatalf("budgets not loaded: %+v", b)
	}
}

func TestLoadBudgetsErrors(t *testing.T) {
	cases := map[string]string{
		"broken json":    `[{"category":`,
		"not an array":   `{"category":"еда","limit":100}`,
		"empty category": `[{"category":"","limit":100}]`,
		"negative limit": `[{"category":"еда","limit":-1}]`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			l := NewLedger()
			if err := l.LoadBudgets(strings.NewReader(input)); err == nil {
				t.Fatal("want error, got nil")
			}
			if len(l.Budgets()) != 0 {
				t.Fatal("no budgets must be applied on error")
			}
		})
	}
}
