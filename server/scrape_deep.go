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

const apiURL2 = "https://pali-thai-dictionary.onrender.com/api/dict"
const outputDir2 = `d:\Program\TIPITAKA\server\server-data\dicts`

var newDicts2 = map[string]string{
	"NG":     "th-ng.db",
	"DMC":    "th-dmc.db",
	"THATU":  "th-thatu.db",
	"THATU2": "th-thatu2.db",
}

type DictEntry2b struct {
	DictName string `json:"dictName"`
	Word     string `json:"word"`
	Meaning  string `json:"meaning"`
}

func searchAPI2(word, dictCode string) ([]DictEntry2b, error) {
	payload := map[string]interface{}{"word": word, "limit": 500, "dicts": []string{dictCode}}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(apiURL2, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var results []DictEntry2b
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse error: %.100s", data)
	}
	return results, nil
}

func createDB2b(path string) (*sql.DB, error) {
	os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE dictionary (word TEXT NOT NULL, meaning TEXT NOT NULL)`)
	db.Exec(`CREATE INDEX word ON dictionary(word)`)
	return db, nil
}

// All 2nd-character options after a Sinhala consonant:
// - Dependent vowel marks: า(0x0DCF), ิ(0x0DD0), ี(0x0DD1), ุ(0x0DD4), ู(0x0DD6), เ(0x0DD9), โ(0x0DDC) 
// - Virama (hal): ้(0x0DCA) for consonant clusters
// - Another consonant: ก(0x0D9A)-ฟ(0x0DC6) for clusters
func getSecondChars() []string {
	var chars []string
	// Sinhala dependent vowel marks (after consonant)
	vowelMarks := []rune{
		0x0DCA, // virama/hal - consonant cluster like ก้...
		0x0DCF, // ā  - กา...
		0x0DD0, // ă
		0x0DD1, // i  - กิ...
		0x0DD2, // ī
		0x0DD3, // ĭ
		0x0DD4, // u  - กุ...
		0x0DD6, // ū  - กู...
		0x0DD8, // ṛ
		0x0DD9, // e  - กเ...
		0x0DDA, // ea
		0x0DDC, // o  - กโ...
		0x0DDD, // oa
		0x0DDE, // au
		0x0DDF, // ä
	}
	for _, r := range vowelMarks {
		chars = append(chars, string(r))
	}
	// Sinhala consonants (for clusters like กก, กข, ...)
	for r := rune(0x0D9A); r <= rune(0x0DC6); r++ {
		chars = append(chars, string(r))
	}
	// Niggahita (anusvara)
	chars = append(chars, string(rune(0x0D82)))
	return chars
}

type dbContext struct {
	db   *sql.DB
	tx   *sql.Tx
	stmt *sql.Stmt
	seen map[string]bool
	total int
}

func (ctx *dbContext) insert(word, meaning string) {
	key := word + "|"
	if len(meaning) > 60 {
		key += meaning[:60]
	} else {
		key += meaning
	}
	if ctx.seen[key] {
		return
	}
	ctx.seen[key] = true
	ctx.stmt.Exec(word, meaning)
	ctx.total++
	if ctx.total%500 == 0 {
		ctx.tx.Commit()
		ctx.tx, _ = ctx.db.Begin()
		ctx.stmt, _ = ctx.tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	}
}

func (ctx *dbContext) flush() {
	ctx.tx.Commit()
}

func scrapeWithDepth(prefix, dictCode string, ctx *dbContext, depth int) {
	results, err := searchAPI2(prefix, dictCode)
	if err != nil {
		log.Printf("    Error %q depth=%d: %v", prefix, depth, err)
		time.Sleep(1 * time.Second)
		return
	}
	for _, r := range results {
		ctx.insert(r.Word, r.Meaning)
	}

	// If we hit the limit and depth allows, go deeper
	if len(results) >= 490 && depth < 3 {
		if depth == 1 {
			fmt.Printf("    %q: %d results (going deeper)...\n", prefix, len(results))
		}
		basePrefix := prefix[:len(prefix)-1] // remove trailing %
		for _, ch := range getSecondChars() {
			newPrefix := basePrefix + ch + "%"
			time.Sleep(80 * time.Millisecond)
			scrapeWithDepth(newPrefix, dictCode, ctx, depth+1)
		}
	}
}

func scrapeDict2(dictCode, dbPath string) {
	fmt.Printf("\n=== Scraping %s → %s ===\n", dictCode, dbPath)
	db, err := createDB2b(dbPath)
	if err != nil {
		log.Printf("Failed to create DB: %v", err)
		return
	}

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	ctx := &dbContext{db: db, tx: tx, stmt: stmt, seen: make(map[string]bool)}

	// Level 1 prefixes: all Sinhala starting characters
	var prefixes []string
	// Independent vowels (words starting with vowels like "อ...", "อา...")
	for r := rune(0x0D85); r <= rune(0x0D96); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	// Consonants
	for r := rune(0x0D9A); r <= rune(0x0DC6); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}

	for i, prefix := range prefixes {
		if i%10 == 0 {
			fmt.Printf("  Progress: %d/%d prefixes, %d entries\n", i+1, len(prefixes), ctx.total)
		}
		scrapeWithDepth(prefix, dictCode, ctx, 1)
		time.Sleep(150 * time.Millisecond)
	}
	ctx.flush()
	fmt.Printf("  Done! Total: %d entries\n", ctx.total)
	db.Close()
}

func main() {
	fmt.Println("=== Re-scraping NG, DMC, THATU, THATU2 with deep prefix search ===")
	for dictCode, dbFile := range newDicts2 {
		scrapeDict2(dictCode, outputDir2+`\`+dbFile)
	}
	fmt.Println("\n=== All Done! ===")
}
