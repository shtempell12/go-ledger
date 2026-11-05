package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/shtempell12/go-ledger/ledger/internal/cache"
	"github.com/shtempell12/go-ledger/ledger/internal/db"
)

const (
	budgetsCacheKey = "budgets:all"
	budgetsCacheTTL = 30 * time.Second
	reportCacheTTL  = 30 * time.Second
	reportDateFmt   = "2006-01-02"
)

// Подключения, общие для всего пакета; заполняются в Init.
var (
	conn  *sql.DB
	store *cache.Cache
)

// Init подключается к PostgreSQL и Redis. Без базы работа невозможна, поэтому её ошибка
// возвращается вызывающему. Redis необязателен: без него Ledger работает напрямую с базой.
func Init() error {
	opened, err := db.Open()
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	conn = opened
	log.Println("ledger: connected to PostgreSQL")

	store, err = cache.Open()
	if err != nil {
		log.Printf("ledger: cache disabled, Redis is unavailable: %v", err)
		return nil
	}
	log.Println("ledger: connected to Redis")
	return nil
}

// Close закрывает подключения.
func Close() {
	if store != nil {
		store.Close()
	}
	if conn != nil {
		conn.Close()
	}
}

// SetBudget добавляет новый бюджет или обновляет существующий для категории.
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return err
	}
	_, err := conn.Exec(
		`INSERT INTO budgets(category, limit_amount) VALUES($1, $2)
		 ON CONFLICT(category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`,
		b.Category, b.Limit,
	)
	if err != nil {
		return fmt.Errorf("set budget: %w", err)
	}
	if store != nil {
		if err := store.Del(context.Background(), budgetsCacheKey); err != nil {
			log.Printf("ledger: %v", err)
		}
	}
	return nil
}

// ListBudgets возвращает бюджеты, отсортированные по категории.
func ListBudgets() ([]Budget, error) {
	ctx := context.Background()
	var budgets []Budget
	if cacheGet(ctx, budgetsCacheKey, &budgets) {
		return budgets, nil
	}

	rows, err := conn.Query(`SELECT category, limit_amount FROM budgets ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	budgets = []Budget{}
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("list budgets: %w", err)
		}
		budgets = append(budgets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	cacheSet(ctx, budgetsCacheKey, budgets, budgetsCacheTTL)
	return budgets, nil
}

// AddTransaction проверяет транзакцию и бюджет её категории.
// Если бюджет будет превышен, транзакция не сохраняется и возвращается ошибка.
// Категория без бюджета лимитом не ограничена. Возвращает сохранённую транзакцию
// с заполненными ID и датой.
func AddTransaction(tx Transaction) (Transaction, error) {
	if err := tx.Validate(); err != nil {
		return Transaction{}, err
	}

	var limit float64
	err := conn.QueryRow(`SELECT limit_amount FROM budgets WHERE category = $1`, tx.Category).Scan(&limit)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Бюджета нет: лимит не проверяем.
	case err != nil:
		return Transaction{}, fmt.Errorf("read budget: %w", err)
	default:
		var spent float64
		err := conn.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE category = $1`, tx.Category).Scan(&spent)
		if err != nil {
			return Transaction{}, fmt.Errorf("read spent amount: %w", err)
		}
		if exceedsLimit(spent, tx.Amount, limit) {
			return Transaction{}, fmt.Errorf("%w: category %q, spent %.2f + new %.2f > limit %.2f",
				ErrBudgetExceeded, tx.Category, spent, tx.Amount, limit)
		}
	}

	if tx.Date.IsZero() {
		tx.Date = time.Now()
	}
	err = conn.QueryRow(
		`INSERT INTO expenses(amount, category, description, date) VALUES($1, $2, $3, $4) RETURNING id`,
		tx.Amount, tx.Category, tx.Description, tx.Date,
	).Scan(&tx.ID)
	if err != nil {
		return Transaction{}, fmt.Errorf("insert transaction: %w", err)
	}
	return tx, nil
}

// ListTransactions возвращает транзакции: сначала новые.
func ListTransactions() ([]Transaction, error) {
	rows, err := conn.Query(
		`SELECT id, amount, category, COALESCE(description, ''), date FROM expenses ORDER BY date DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &tx.Description, &tx.Date); err != nil {
			return nil, fmt.Errorf("list transactions: %w", err)
		}
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return transactions, nil
}

// GetReportSummary возвращает расходы по категориям за период включительно.
// Результат кешируется в Redis на короткий TTL, поэтому новые транзакции
// могут попасть в отчёт с задержкой до reportCacheTTL.
func GetReportSummary(ctx context.Context, from, to time.Time) (ReportSummary, error) {
	key := reportCacheKey(from, to)
	var report ReportSummary
	if cacheGet(ctx, key, &report) {
		log.Printf("ledger: report cache hit: %s", key)
		return report, nil
	}
	log.Printf("ledger: report cache miss: %s", key)

	rows, err := conn.QueryContext(ctx,
		`SELECT category, SUM(amount) FROM expenses
		 WHERE date BETWEEN $1 AND $2
		 GROUP BY category ORDER BY category`,
		from, to,
	)
	if err != nil {
		return ReportSummary{}, fmt.Errorf("report summary: %w", err)
	}
	defer rows.Close()

	report = ReportSummary{
		From:       from.Format(reportDateFmt),
		To:         to.Format(reportDateFmt),
		Categories: []CategoryTotal{},
	}
	var totalCents float64
	for rows.Next() {
		var item CategoryTotal
		if err := rows.Scan(&item.Category, &item.Total); err != nil {
			return ReportSummary{}, fmt.Errorf("report summary: %w", err)
		}
		report.Categories = append(report.Categories, item)
		totalCents += cents(item.Total)
	}
	if err := rows.Err(); err != nil {
		return ReportSummary{}, fmt.Errorf("report summary: %w", err)
	}
	report.Total = totalCents / 100

	cacheSet(ctx, key, report, reportCacheTTL)
	return report, nil
}

func reportCacheKey(from, to time.Time) string {
	return fmt.Sprintf("report:summary:%s:%s", from.Format(reportDateFmt), to.Format(reportDateFmt))
}

// cacheGet читает кеш. Сбой Redis не должен ломать запрос, поэтому он считается промахом.
func cacheGet(ctx context.Context, key string, dst any) bool {
	if store == nil {
		return false
	}
	found, err := store.Get(ctx, key, dst)
	if err != nil {
		log.Printf("ledger: %v", err)
		return false
	}
	return found
}

// cacheSet пишет в кеш. Сбой Redis только логируется.
func cacheSet(ctx context.Context, key string, value any, ttl time.Duration) {
	if store == nil {
		return
	}
	if err := store.Set(ctx, key, value, ttl); err != nil {
		log.Printf("ledger: %v", err)
	}
}
