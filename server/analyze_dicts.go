package main

import (
	"database/sql"
	"fmt"
	"log"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

func scriptOf(r rune) string {
	switch {
	case r >= 0x0D80 && r <= 0x0DFF:
		return "Sinhala"
	case r >= 0x0E00 && r <= 0x0E7F:
		return "Thai"
	case r >= 0x0000 && r <= 0x017F:
		return "Roman"
	case r >= 0x0900 && r <= 0x097F:
		return "Devanagari"
	default:
		return fmt.Sprintf("Other(U+%04X)", r)
	}
}

func analyzeDB(path, name string) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Printf("%s: Error: %v", name, err)
		return
	}
	defer db.Close()

	var total int
	db.QueryRow("SELECT COUNT(*) FROM dictionary").Scan(&total)
	fmt.Printf("\n[%s] Total: %d rows\n", name, total)

	// Analyze first 20 words to see what script
	rows, _ := db.Query("SELECT word FROM dictionary LIMIT 20")
	defer rows.Close()
	scriptCount := map[string]int{}
	sample := []string{}
	for rows.Next() {
		var w string
		rows.Scan(&w)
		r, _ := utf8.DecodeRuneInString(w)
		s := scriptOf(r)
		scriptCount[s]++
		if len(sample) < 5 {
			sample = append(sample, w)
		}
	}
	fmt.Printf("  Scripts: %v\n", scriptCount)
	fmt.Printf("  Samples: %v\n", sample)

	// Search for Thai "การ" directly
	var count int
	db.QueryRow("SELECT COUNT(*) FROM dictionary WHERE word LIKE ?", "การ%").Scan(&count)
	fmt.Printf("  LIKE 'การ%%': %d matches\n", count)

	// Search for Sinhala "කාර"
	db.QueryRow("SELECT COUNT(*) FROM dictionary WHERE word LIKE ?", "කාර%").Scan(&count)
	fmt.Printf("  LIKE 'කාර%%': %d matches\n", count)

	// Search for what the NG shows for "vicara" (known to work)
	db.QueryRow("SELECT COUNT(*) FROM dictionary WHERE word LIKE ?", "විචාර%").Scan(&count)
	fmt.Printf("  LIKE 'විචාර%%' (vicara): %d matches\n", count)
}

func main() {
	base := `d:\Program\TIPITAKA\server\server-data\dicts\`
	analyzeDB(base+"th-ng.db", "NG")
	analyzeDB(base+"th-dmc.db", "DMC")
	analyzeDB(base+"th-thatu.db", "THATU")
}
