package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `d:\Program\TIPITAKA\server\server-data\dicts\th-dmc.db`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Search words starting with Sinhala ka (ก=0x0D9A)
	rows, _ := db.Query("SELECT word, meaning FROM dictionary WHERE word LIKE ? LIMIT 10", "ก%")
	fmt.Println("=== DMC words starting with Sinhala ก (ka) ===")
	count := 0
	for rows.Next() {
		var w, m string
		rows.Scan(&w, &m)
		if len(m) > 50 { m = m[:50] }
		fmt.Printf("  %q -> %q\n", w, m)
		count++
	}
	rows.Close()
	if count == 0 { fmt.Println("  None found") }

	// Check what's stored for words starting with ก
	fmt.Println("\n=== Words starting with hex 0x0D9A (Sinhala ก) ===")
	db.QueryRow("SELECT COUNT(*) FROM dictionary WHERE word >= ? AND word < ?", "\u0D9A", "\u0D9B").Scan(&count)
	fmt.Printf("  Count: %d\n", count)

	// Try with Unicode range
	var total int
	db.QueryRow("SELECT COUNT(*) FROM dictionary WHERE word LIKE ?", "\u0D9A%").Scan(&total)
	fmt.Printf("  LIKE '\\u0D9A%%': %d\n", total)

	// What are the first words alphabetically?
	rows2, _ := db.Query("SELECT word, hex(substr(word,1,3)) FROM dictionary ORDER BY word LIMIT 5")
	fmt.Println("\n=== First words (with hex codes of first 3 bytes) ===")
	for rows2.Next() {
		var w, h string
		rows2.Scan(&w, &h)
		fmt.Printf("  %q  hex: %s\n", w, h)
	}
	rows2.Close()

	// Char analysis of first word
	rows3, _ := db.Query("SELECT word FROM dictionary LIMIT 1")
	if rows3.Next() {
		var w string
		rows3.Scan(&w)
		fmt.Printf("\n=== First word analysis: %q ===\n", w)
		for i, r := range w {
			fmt.Printf("  [%d] rune=%d (0x%04X) char=%q\n", i, r, r, string(r))
		}
	}
	rows3.Close()
}
