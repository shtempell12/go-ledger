package ledger

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func mustSetBudget(t *testing.T, b Budget) {
	t.Helper()
	if err := SetBudget(b); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
}

func listTransactions(t *testing.T) []Transaction {
	t.Helper()
	txs, err := ListTransactions()
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	return txs
}

func TestAddTransactionWithinBudget(t *testing.T) {
	reset(t)
	mustSetBudget(t, Budget{Category: "еда", Limit: 5000})

	first, err := AddTransaction(Transaction{Amount: 3000, Category: "еда", Description: "продукты"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.ID == 0 || first.Date.IsZero() {
		t.Fatalf("id and date must be filled: %+v", first)
	}
	if _, err := AddTransaction(Transaction{Amount: 2000, Category: "еда"}); err != nil {
		t.Fatalf("exactly at limit must be allowed: %v", err)
	}
	if got := len(listTransactions(t)); got != 2 {
		t.Fatalf("want 2 transactions, got %d", got)
	}
}

func TestAddTransactionExceedsBudget(t *testing.T) {
	reset(t)
	mustSetBudget(t, Budget{Category: "еда", Limit: 5000})
	if _, err := AddTransaction(Transaction{Amount: 4000, Category: "еда"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := AddTransaction(Transaction{Amount: 1500, Category: "еда"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("want ErrBudgetExceeded, got %v", err)
	}
	if got := len(listTransactions(t)); got != 1 {
		t.Fatalf("rejected transaction must not be saved: want 1, got %d", got)
	}
}

func TestAddTransactionCentsAtLimit(t *testing.T) {
	reset(t)
	mustSetBudget(t, Budget{Category: "кофе", Limit: 0.3})
	if _, err := AddTransaction(Transaction{Amount: 0.1, Category: "кофе"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := AddTransaction(Transaction{Amount: 0.2, Category: "кофе"}); err != nil {
		t.Fatalf("0.1 + 0.2 equals the limit 0.3 and must be allowed: %v", err)
	}
}

func TestAddTransactionNoBudget(t *testing.T) {
	reset(t)
	if _, err := AddTransaction(Transaction{Amount: 1e6, Category: "прочее"}); err != nil {
		t.Fatalf("category without budget must be allowed: %v", err)
	}
}

func TestAddTransactionInvalid(t *testing.T) {
	reset(t)
	cases := map[string]Transaction{
		"zero amount":     {Amount: 0, Category: "еда"},
		"negative amount": {Amount: -100, Category: "еда"},
		"below one cent":  {Amount: 0.001, Category: "еда"},
		"too large":       {Amount: 1e12, Category: "еда"},
		"empty category":  {Amount: 100, Category: "  "},
	}
	for name, tx := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := AddTransaction(tx); !errors.Is(err, ErrInvalidTransaction) {
				t.Fatalf("want ErrInvalidTransaction, got %v", err)
			}
		})
	}
	if got := len(listTransactions(t)); got != 0 {
		t.Fatalf("invalid transactions must not be saved, got %d", got)
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	reset(t)
	mustSetBudget(t, Budget{Category: "транспорт", Limit: 1000})
	if _, err := AddTransaction(Transaction{Amount: 1500, Category: "транспорт"}); err == nil {
		t.Fatal("want error before limit update")
	}
	mustSetBudget(t, Budget{Category: "транспорт", Limit: 2000})
	if _, err := AddTransaction(Transaction{Amount: 1500, Category: "транспорт"}); err != nil {
		t.Fatalf("want success after limit update: %v", err)
	}
	got, err := ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(got) != 1 || got[0].Limit != 2000 {
		t.Fatalf("budget must be updated without duplicates: %+v", got)
	}
}

func TestSetBudgetInvalid(t *testing.T) {
	reset(t)
	for _, b := range []Budget{{Category: "", Limit: 100}, {Category: "еда", Limit: 0}, {Category: "еда", Limit: -1}} {
		if err := SetBudget(b); !errors.Is(err, ErrInvalidBudget) {
			t.Fatalf("%+v: want ErrInvalidBudget, got %v", b, err)
		}
	}
}

func TestListTransactionsNewestFirst(t *testing.T) {
	reset(t)
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	for _, d := range []int{1, 3, 2} {
		if _, err := AddTransaction(Transaction{Amount: 10, Category: "еда", Date: day(d)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	txs := listTransactions(t)
	if txs[0].Date.Day() != 3 || txs[1].Date.Day() != 2 || txs[2].Date.Day() != 1 {
		t.Fatalf("wrong order: %+v", txs)
	}
}

func TestLoadBudgets(t *testing.T) {
	reset(t)
	err := LoadBudgets(strings.NewReader(`[{"category":"еда","limit":100},{"category":"связь","limit":50}]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(got) != 2 || got[0].Category != "еда" || got[0].Limit != 100 || got[1].Limit != 50 {
		t.Fatalf("budgets not loaded: %+v", got)
	}
}

func TestLoadBudgetsErrors(t *testing.T) {
	cases := map[string]string{
		"broken json":    `[{"category":`,
		"not an array":   `{"category":"еда","limit":100}`,
		"empty category": `[{"category":"ок","limit":1},{"category":"","limit":100}]`,
		"negative limit": `[{"category":"ок","limit":1},{"category":"еда","limit":-1}]`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			reset(t)
			if err := LoadBudgets(strings.NewReader(input)); err == nil {
				t.Fatal("want error, got nil")
			}
			got, err := ListBudgets()
			if err != nil {
				t.Fatalf("ListBudgets: %v", err)
			}
			if len(got) != 0 {
				t.Fatalf("no budgets must be applied on error: %+v", got)
			}
		})
	}
}

func TestValidateWithoutDatabase(t *testing.T) {
	if err := (Transaction{Amount: 0.01, Category: "еда"}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := (Transaction{Amount: 0, Category: "еда"}).Validate(); !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("want ErrInvalidTransaction, got %v", err)
	}
	if err := (Budget{Category: "еда", Limit: -1}).Validate(); !errors.Is(err, ErrInvalidBudget) {
		t.Fatalf("want ErrInvalidBudget, got %v", err)
	}
}

func TestGetReportSummaryUsesCache(t *testing.T) {
	reset(t)
	if store == nil {
		t.Skip("Redis is unavailable")
	}
	ctx := context.Background()
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	from, to := day(1), day(31)
	clearReportCache(t, from, to)

	for _, tx := range []Transaction{
		{Amount: 100.10, Category: "еда", Date: day(1)},
		{Amount: 50.20, Category: "еда", Date: day(31)},
		{Amount: 30, Category: "связь", Date: day(15)},
		{Amount: 999, Category: "еда", Date: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if _, err := AddTransaction(tx); err != nil {
			t.Fatalf("AddTransaction: %v", err)
		}
	}

	report, err := GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("GetReportSummary: %v", err)
	}
	if report.Total != 180.30 || len(report.Categories) != 2 || report.Categories[0].Total != 150.30 {
		t.Fatalf("wrong report: %+v", report)
	}

	if _, err := AddTransaction(Transaction{Amount: 1, Category: "еда", Date: day(2)}); err != nil {
		t.Fatalf("AddTransaction: %v", err)
	}
	cached, err := GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("GetReportSummary: %v", err)
	}
	if cached.Total != 180.30 {
		t.Fatalf("second call must come from cache: %+v", cached)
	}

	clearReportCache(t, from, to)
	fresh, err := GetReportSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("GetReportSummary: %v", err)
	}
	if fresh.Total != 181.30 {
		t.Fatalf("report must be recalculated after a cache miss: %+v", fresh)
	}
}

func TestListBudgetsCacheInvalidatedBySetBudget(t *testing.T) {
	reset(t)
	mustSetBudget(t, Budget{Category: "еда", Limit: 100})
	if _, err := ListBudgets(); err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	mustSetBudget(t, Budget{Category: "еда", Limit: 200})
	got, err := ListBudgets()
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(got) != 1 || got[0].Limit != 200 {
		t.Fatalf("stale budgets: %+v", got)
	}
}
