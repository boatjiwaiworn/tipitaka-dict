package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `d:\Program\TIPITAKA\server-data\dicts\th-ng.db`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Count total
	var count int
	db.QueryRow("SELECT COUNT(*) FROM dictionary").Scan(&count)
	fmt.Printf("th-ng.db total rows: %d\n", count)

	// Sample words
	fmt.Println("\nFirst 5 rows:")
	rows, _ := db.Query("SELECT word, meaning FROM dictionary LIMIT 5")
	defer rows.Close()
	for rows.Next() {
		var w, m string
		rows.Scan(&w, &m)
		if len(m) > 80 { m = m[:80] }
		fmt.Printf("  word: %q, meaning: %q\n", w, m)
	}

	// Search for vicara
	fmt.Println("\nSearch 'විචාර%':")
	rows2, _ := db.Query("SELECT word, meaning FROM dictionary WHERE word LIKE ? LIMIT 5", "විචාර%")
	defer rows2.Close()
	found := 0
	for rows2.Next() {
		var w, m string
		rows2.Scan(&w, &m)
		if len(m) > 80 { m = m[:80] }
		fmt.Printf("  word: %q, meaning: %q\n", w, m)
		found++
	}
	if found == 0 {
		fmt.Println("  No results found!")
	}
	
	// Show schema
	fmt.Println("\nSchema:")
	rows3, _ := db.Query("SELECT name, sql FROM sqlite_master WHERE type='table' OR type='index'")
	defer rows3.Close()
	for rows3.Next() {
		var name, sql string
		rows3.Scan(&name, &sql)
		fmt.Printf("  %s: %s\n", name, sql)
	}
}
