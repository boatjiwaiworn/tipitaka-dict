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

const apiBase = "https://pali-thai-dictionary.onrender.com"
const dictsDir = `d:\Program\TIPITAKA\server\server-data\dicts`

type DictEntry struct {
	DictName string `json:"dictName"`
	Word     string `json:"word"`
	Meaning  string `json:"meaning"`
}

func postAPI(path string, body map[string]interface{}) ([]DictEntry, error) {
	bodyBytes, _ := json.Marshal(body)
	resp, err := http.Post(apiBase+path, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var results []DictEntry
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse error: %v, body: %.100s", err, data)
	}
	return results, nil
}

func createDB(path string) (*sql.DB, error) {
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE dictionary (word TEXT NOT NULL, meaning TEXT NOT NULL)`)
	db.Exec(`CREATE INDEX word ON dictionary(word)`)
	return db, nil
}

func scrapeReverseDict(dictCode, dbPath string) {
	fmt.Printf("\n=== Scraping reverse dict %s → %s ===\n", dictCode, dbPath)
	db, err := createDB(dbPath)
	if err != nil {
		log.Printf("Failed to create DB: %v", err)
		return
	}
	defer db.Close()

	seen := make(map[string]bool)
	totalInserted := 0

	// Thai consonants ก-ฮ (0x0E01-0x0E2E)
	// Thai vowel patterns
	var prefixes []string
	for r := rune(0x0E01); r <= rune(0x0E2E); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	// Additional Thai starting patterns
	extraPrefixes := []string{
		"เ%", "แ%", "โ%", "ไ%", "ใ%", "อ%",
		"ฤ%", "ฦ%",
	}
	prefixes = append(prefixes, extraPrefixes...)

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	batchCount := 0

	for i, prefix := range prefixes {
		results, err := postAPI("/api/dict", map[string]interface{}{
			"word":  prefix,
			"limit": 500,
			"dicts": []string{dictCode},
		})
		if err != nil {
			log.Printf("  Error %q: %v", prefix, err)
			time.Sleep(2 * time.Second)
			continue
		}
		for _, r := range results {
			key := r.Word + "|" + r.Meaning
			if len(key) > 100 {
				key = key[:100]
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			stmt.Exec(r.Word, r.Meaning)
			totalInserted++
			batchCount++
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
	fmt.Println("=== Scraping PD and PS (reverse dicts) ===")
	scrapeReverseDict("PD", dictsDir+`\th-pd.db`)
	scrapeReverseDict("PS", dictsDir+`\th-ps.db`)
	fmt.Println("\n=== Done! ===")
}
