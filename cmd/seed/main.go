package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	customerCount   = 500
	productCount    = 200
	orderCount      = 5000
	stressRowCount  = 3000
	wideColumnCount = 40
)

type seedConfig struct {
	dbPath string
}

type productSeed struct {
	priceCents int
}

type wideColumn struct {
	name string
	kind string
}

func main() {
	config := parseSeedArgs(os.Args[1:])
	err := run(config)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseSeedArgs(args []string) seedConfig {
	dbPath := "data/seed.db"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--db=") {
			dbPath = strings.TrimPrefix(arg, "--db=")
			continue
		}

		if strings.HasPrefix(arg, "-") {
			continue
		}

		dbPath = arg
	}

	return seedConfig{dbPath: dbPath}
}

func run(config seedConfig) error {
	err := ensureDir(config.dbPath)
	if err != nil {
		return err
	}

	_ = os.Remove(config.dbPath)

	db, err := sql.Open("sqlite", makeWriteDsn(config.dbPath))
	if err != nil {
		return err
	}
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, closeErr)
		}
	}()

	err = setupPragmas(db)
	if err != nil {
		return err
	}

	err = createSchema(db)
	if err != nil {
		return err
	}

	products, err := seedCoreTables(db)
	if err != nil {
		return err
	}

	err = seedStressTables(db, products)
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("Seeded DB: %s\n", config.dbPath)
	return nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}

func makeWriteDsn(path string) string {
	if strings.HasPrefix(path, "file:") {
		if strings.Contains(path, "mode=") {
			return path
		}

		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}

		return path + separator + "mode=rwc"
	}

	escaped := url.PathEscape(path)
	return "file:" + escaped + "?mode=rwc"
}

func setupPragmas(db *sql.DB) error {
	_, err := db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		return err
	}

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return nil
}

func createSchema(db *sql.DB) error {
	schemaStatements := []string{
		"CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY AUTOINCREMENT, sku TEXT NOT NULL UNIQUE, name TEXT NOT NULL, price_cents INTEGER NOT NULL)",
		"CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY AUTOINCREMENT, customer_id INTEGER NOT NULL REFERENCES customers(id), status TEXT NOT NULL, created_at TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS order_items (id INTEGER PRIMARY KEY AUTOINCREMENT, order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE, product_id INTEGER NOT NULL REFERENCES products(id), quantity INTEGER NOT NULL, unit_price_cents INTEGER NOT NULL)",
		"CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id)",
		"CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id)",
		"CREATE TABLE IF NOT EXISTS json_blobs (id INTEGER PRIMARY KEY AUTOINCREMENT, label TEXT NOT NULL, payload TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS long_texts (id INTEGER PRIMARY KEY AUTOINCREMENT, short_text TEXT, medium_text TEXT, long_text TEXT, note TEXT)",
		"CREATE TABLE IF NOT EXISTS mixed_shapes (id INTEGER PRIMARY KEY AUTOINCREMENT, category TEXT, count INTEGER, ratio REAL, flag INTEGER, note TEXT, payload TEXT)",
	}

	for _, stmt := range schemaStatements {
		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	wideColumns := buildWideColumns(wideColumnCount)
	wideStatement := buildWideTableStatement(wideColumns)
	_, err := db.Exec(wideStatement)
	if err != nil {
		return err
	}

	return nil
}

func seedCoreTables(db *sql.DB) ([]productSeed, error) {
	products := make([]productSeed, productCount+1)

	err := withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO customers (name, email, created_at) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		for i := 1; i <= customerCount; i += 1 {
			name := fmt.Sprintf("Customer %d", i)
			email := fmt.Sprintf("customer.%d@example.com", i)
			createdAt := isoDaysAgo(30)
			_, err = stmt.Exec(name, email, createdAt)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO products (sku, name, price_cents) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		for i := 1; i <= productCount; i += 1 {
			sku := fmt.Sprintf("SKU-%04d", i)
			name := fmt.Sprintf("Product %d", i)
			priceCents := 500 + (i%50)*75
			products[i] = productSeed{priceCents: priceCents}
			_, err = stmt.Exec(sku, name, priceCents)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = withTx(db, func(tx *sql.Tx) error {
		orderStmt, err := tx.Prepare("INSERT INTO orders (customer_id, status, created_at) VALUES (?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := orderStmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		itemStmt, err := tx.Prepare("INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents) VALUES (?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := itemStmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		statusOptions := []string{"pending", "paid", "shipped", "cancelled"}

		for i := 1; i <= orderCount; i += 1 {
			customerId := ((i - 1) % customerCount) + 1
			status := statusOptions[i%len(statusOptions)]
			createdAt := isoDaysAgo(i % 45)

			result, err := orderStmt.Exec(customerId, status, createdAt)
			if err != nil {
				return err
			}

			orderId, err := result.LastInsertId()
			if err != nil {
				return err
			}

			itemCount := 1 + (i % 4)
			for j := 0; j < itemCount; j += 1 {
				productIndex := ((i + j) % productCount) + 1
				quantity := 1 + ((i + j) % 3)
				price := products[productIndex].priceCents

				_, err = itemStmt.Exec(orderId, productIndex, quantity, price)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return products, nil
}

func seedStressTables(db *sql.DB, products []productSeed) error {
	err := seedJsonBlobs(db)
	if err != nil {
		return err
	}

	err = seedWideTable(db)
	if err != nil {
		return err
	}

	err = seedLongTexts(db)
	if err != nil {
		return err
	}

	err = seedMixedShapes(db, products)
	if err != nil {
		return err
	}

	return nil
}

func seedJsonBlobs(db *sql.DB) error {
	return withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO json_blobs (label, payload) VALUES (?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		for i := 1; i <= stressRowCount; i += 1 {
			blobSize := 2000 + (i % 3000)
			payload := buildJSONPayload(i, blobSize)
			label := fmt.Sprintf("event-%04d", i)

			_, err = stmt.Exec(label, payload)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func seedWideTable(db *sql.DB) error {
	columns := buildWideColumns(wideColumnCount)
	columnNames := []string{}
	columnKinds := []string{}
	for _, col := range columns {
		columnNames = append(columnNames, col.name)
		columnKinds = append(columnKinds, col.kind)
	}

	placeholderParts := []string{}
	for range columnNames {
		placeholderParts = append(placeholderParts, "?")
	}

	insertSql := "INSERT INTO wide_table (" + strings.Join(columnNames, ", ") + ") VALUES (" + strings.Join(placeholderParts, ", ") + ")"

	return withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(insertSql)
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		for row := 1; row <= stressRowCount; row += 1 {
			values := make([]any, len(columnNames))
			for i := 0; i < len(columnNames); i += 1 {
				values[i] = buildWideValue(row, i, columnKinds[i])
			}

			_, err = stmt.Exec(values...)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func seedLongTexts(db *sql.DB) error {
	return withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO long_texts (short_text, medium_text, long_text, note) VALUES (?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		for i := 1; i <= stressRowCount; i += 1 {
			shortText := fillToLength("short", 20+(i%10))
			mediumText := fillToLength("medium", 200+(i%50))
			longText := fillToLength("long", 2000+(i%500))
			note := fillToLength("note", 80)

			_, err = stmt.Exec(shortText, mediumText, longText, note)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func seedMixedShapes(db *sql.DB, products []productSeed) error {
	return withTx(db, func(tx *sql.Tx) error {
		stmt, err := tx.Prepare("INSERT INTO mixed_shapes (category, count, ratio, flag, note, payload) VALUES (?, ?, ?, ?, ?, ?)")
		if err != nil {
			return err
		}
		defer func() {
			closeErr := stmt.Close()
			if closeErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, closeErr)
			}
		}()

		categories := []string{"alpha", "beta", "gamma", "delta"}

		for i := 1; i <= stressRowCount; i += 1 {
			category := categories[i%len(categories)]
			count := i * 3
			ratio := float64(i) / 7.0
			flag := 0
			if i%2 == 0 {
				flag = 1
			}

			note := any(fmt.Sprintf("note-%d", i))
			if i%10 == 0 {
				note = nil
			}

			payload := buildMixedPayload(i, products)

			_, err = stmt.Exec(category, count, ratio, flag, note, payload)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func withTx(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("seed tx: %w; rollback error: %v", err, rollbackErr)
		}
		return err
	}

	return tx.Commit()
}

func isoDaysAgo(daysAgo int) string {
	now := time.Now().UTC()
	then := now.AddDate(0, 0, -daysAgo)
	return then.Format(time.RFC3339)
}

func buildWideColumns(count int) []wideColumn {
	columns := []wideColumn{}
	for i := 1; i <= count; i += 1 {
		name := fmt.Sprintf("col_%02d", i)
		kind := "TEXT"
		switch i % 5 {
		case 1:
			kind = "INTEGER"
		case 2:
			kind = "REAL"
		case 3:
			kind = "TEXT"
		case 4:
			kind = "INTEGER"
		}

		columns = append(columns, wideColumn{name: name, kind: kind})
	}

	return columns
}

func buildWideTableStatement(columns []wideColumn) string {
	parts := []string{"id INTEGER PRIMARY KEY AUTOINCREMENT"}
	for _, col := range columns {
		parts = append(parts, col.name+" "+col.kind)
	}

	return "CREATE TABLE IF NOT EXISTS wide_table (" + strings.Join(parts, ", ") + ")"
}

func buildWideValue(row int, colIndex int, kind string) any {
	if row%17 == 0 && colIndex%7 == 0 {
		return nil
	}

	if kind == "INTEGER" {
		return row*10 + colIndex
	}

	if kind == "REAL" {
		return float64(row)/10.0 + float64(colIndex)/100.0
	}

	return fmt.Sprintf("row_%d_%02d", row, colIndex+1)
}

func buildJSONPayload(index int, blobSize int) string {
	blob := strings.Repeat("x", blobSize)
	payload := map[string]any{
		"id":    index,
		"kind":  "event",
		"tags":  []string{"alpha", "beta", "gamma"},
		"count": index * 3,
		"meta": map[string]any{
			"blob": blob,
			"note": fmt.Sprintf("payload-%d", index),
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}

	return string(raw)
}

func buildMixedPayload(index int, products []productSeed) string {
	productIndex := (index % productCount) + 1
	price := 0
	if productIndex < len(products) {
		price = products[productIndex].priceCents
	}

	payload := map[string]any{
		"id":    index,
		"name":  fmt.Sprintf("mixed-%d", index),
		"price": price,
		"flags": []int{index % 3, (index + 1) % 3},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}

	return string(raw)
}

func fillToLength(word string, length int) string {
	if length <= 0 {
		return ""
	}

	var builder strings.Builder
	for builder.Len() < length {
		if builder.Len() > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(word)
	}

	text := builder.String()
	if len(text) > length {
		return text[:length]
	}

	return text
}
