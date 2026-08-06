package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

const apiBase2 = "https://pali-thai-dictionary.onrender.com"
const dictsDir2 = `d:\Program\TIPITAKA\server\server-data\dicts`

type DictEntry2 struct {
	DictName string `json:"dictName"`
	Word     string `json:"word"`
	Meaning  string `json:"meaning"`
}

func postReverse(term string, dictCode string) ([]DictEntry2, error) {
	body := map[string]interface{}{
		"term":  term,
		"limit": 500,
		"dicts": []string{dictCode},
	}
	bodyBytes, _ := json.Marshal(body)
	resp, err := http.Post(apiBase2+"/api/reverse", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var results []DictEntry2
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}
	return results, nil
}

func createDB2(path string) (*sql.DB, error) {
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE dictionary (word TEXT NOT NULL, meaning TEXT NOT NULL)`)
	db.Exec(`CREATE INDEX word ON dictionary(word)`)
	return db, nil
}

func scrapeReverseDict2(dictCode, dbPath string) {
	fmt.Printf("\n=== Scraping reverse dict %s via /api/reverse → %s ===\n", dictCode, dbPath)
	db, err := createDB2(dbPath)
	if err != nil {
		log.Printf("Failed to create DB: %v", err)
		return
	}
	defer db.Close()

	seen := make(map[string]bool)
	totalInserted := 0

	// Thai consonants (ก=0E01 to ฮ=0E2E) + special starting chars
	var prefixes []string
	for r := rune(0x0E01); r <= rune(0x0E2E); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	// Thai vowels that start words
	for _, v := range []string{"เ", "แ", "โ", "ไ", "ใ", "อ", "ฤ"} {
		prefixes = append(prefixes, v+"%")
	}

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	batchCount := 0

	for i, prefix := range prefixes {
		results, err := postReverse(prefix, dictCode)
		if err != nil {
			log.Printf("  Error %q: %v", prefix, err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, r := range results {
			// word = Thai, meaning = Pali (as received from API)
			key := r.Word + "|" + r.Meaning
			if len(key) > 120 {
				key = key[:120]
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			stmt.Exec(r.Word, r.Meaning)
			totalInserted++
			batchCount++
		}

		if len(results) >= 490 {
			fmt.Printf("  %s: %d results (may need deeper search)\n", prefix, len(results))
		}

		if batchCount >= 500 {
			tx.Commit()
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
			batchCount = 0
		}
		if i%5 == 0 {
			fmt.Printf("  Progress: %d/%d prefixes, %d entries\n", i+1, len(prefixes), totalInserted)
		}
		time.Sleep(150 * time.Millisecond)
	}
	tx.Commit()
	fmt.Printf("  Done! Total: %d entries\n", totalInserted)
}

func main() {
	fmt.Println("=== Scraping PD and PS via /api/reverse ===")
	scrapeReverseDict2("PD", dictsDir2+`\th-pd.db`)
	scrapeReverseDict2("PS", dictsDir2+`\th-ps.db`)
	fmt.Println("\n=== Done! ===")
}
