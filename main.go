package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	l := NewLedger()

	// 1. Начальные бюджеты через SetBudget.
	fmt.Println("=== 1. Начальные бюджеты ===")
	l.SetBudget(Budget{Category: "еда", Limit: 5000})
	l.SetBudget(Budget{Category: "транспорт", Limit: 2000})
	printBudgets(l)

	// 2. Транзакции: в пределах лимита, сверх лимита, без бюджета, некорректная.
	fmt.Println("\n=== 2. Добавление транзакций ===")
	try(l, Transaction{Amount: 3000, Category: "еда", Description: "продукты"})         // ок: 3000 <= 5000
	try(l, Transaction{Amount: 1500, Category: "еда", Description: "ресторан"})         // ок: 4500 <= 5000
	try(l, Transaction{Amount: 1000, Category: "еда", Description: "доставка"})         // ошибка: 5500 > 5000
	try(l, Transaction{Amount: 500, Category: "еда", Description: "кофе"})              // ок: ровно 5000
	try(l, Transaction{Amount: 2500, Category: "транспорт", Description: "такси"})      // ошибка: 2500 > 2000
	try(l, Transaction{Amount: 10000, Category: "развлечения", Description: "концерт"}) // ок: бюджета нет
	try(l, Transaction{Amount: -100, Category: "еда", Description: "минус"})            // ошибка: некорректная сумма
	printTransactions(l)

	// 3. Обновление лимита через SetBudget и повтор отклонённой транзакции.
	fmt.Println("\n=== 3. Увеличиваем бюджет «транспорт» до 3000 и повторяем ===")
	l.SetBudget(Budget{Category: "транспорт", Limit: 3000})
	try(l, Transaction{Amount: 2500, Category: "транспорт", Description: "такси"}) // теперь ок

	// 4. Загрузка бюджетов из файла.
	fmt.Println("\n=== 4. Загрузка бюджетов из budgets.json ===")
	if err := loadBudgetsFromFile(l, "budgets.json"); err != nil {
		fmt.Println("  ОШИБКА:", err)
	} else {
		fmt.Println("  загружено успешно")
	}
	printBudgets(l)
	try(l, Transaction{Amount: 1500, Category: "одежда", Description: "куртка"})  // ок: 1500 <= 3000
	try(l, Transaction{Amount: 2000, Category: "одежда", Description: "ботинки"}) // ошибка: 3500 > 3000

	// 5. Обработка ошибок ввода-вывода и парсинга.
	fmt.Println("\n=== 5. Ошибки загрузки ===")
	if err := loadBudgetsFromFile(l, "no_such_file.json"); err != nil {
		fmt.Println("  несуществующий файл:", err)
	}
	if err := l.LoadBudgets(strings.NewReader(`[{"category": "еда", "limit": `)); err != nil {
		fmt.Println("  битый JSON:", err)
	}
	if err := l.LoadBudgets(strings.NewReader(`{"category": "еда", "limit": 100}`)); err != nil {
		fmt.Println("  объект вместо массива:", err)
	}
	if err := l.LoadBudgets(strings.NewReader(`[{"category": "", "limit": 100}]`)); err != nil {
		fmt.Println("  пустая категория:", err)
	}
	if err := l.LoadBudgets(strings.NewReader(`[{"category": "еда", "limit": -5}]`)); err != nil {
		fmt.Println("  отрицательный лимит:", err)
	}

	fmt.Println("\n=== Итог ===")
	printBudgets(l)
	printTransactions(l)
}

// loadBudgetsFromFile открывает файл и передаёт его в LoadBudgets через bufio.Reader.
func loadBudgetsFromFile(l *Ledger, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open budgets file: %w", err)
	}
	defer f.Close()

	return l.LoadBudgets(bufio.NewReader(f))
}

func try(l *Ledger, tx Transaction) {
	err := l.AddTransaction(tx)
	switch {
	case err == nil:
		fmt.Printf("  OK     %-12s %8.2f  (%s)\n", tx.Category, tx.Amount, tx.Description)
	case errors.Is(err, ErrBudgetExceeded):
		fmt.Printf("  ОТКАЗ  %-12s %8.2f  (%s): %v\n", tx.Category, tx.Amount, tx.Description, err)
	default:
		fmt.Printf("  ОШИБКА %-12s %8.2f  (%s): %v\n", tx.Category, tx.Amount, tx.Description, err)
	}
}

func printBudgets(l *Ledger) {
	budgets := l.Budgets()
	categories := make([]string, 0, len(budgets))
	for c := range budgets {
		categories = append(categories, c)
	}
	sort.Strings(categories)

	fmt.Println("  Бюджеты:")
	for _, c := range categories {
		fmt.Printf("    %-12s потрачено %8.2f из %8.2f\n", c, l.SpentByCategory(c), budgets[c].Limit)
	}
}

func printTransactions(l *Ledger) {
	fmt.Println("  Сохранённые транзакции:")
	for _, tx := range l.Transactions() {
		fmt.Printf("    #%d %-12s %8.2f  %s\n", tx.ID, tx.Category, tx.Amount, tx.Description)
	}
}
