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
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const apiURL = "https://pali-thai-dictionary.onrender.com/api/dict"
const outputDir = `d:\Program\TIPITAKA\server-data\dicts`

// New dicts to scrape (code -> output db name)
var newDicts = map[string]string{
	"NG":    "th-ng.db",
	"DMC":   "th-dmc.db",
	"THATU": "th-thatu.db",
	"THATU2": "th-thatu2.db",
}

// PD and PS have Thai headwords - different treatment
var reverseDicts = map[string]string{
	"PD": "th-pd.db",
	"PS": "th-ps.db",
}

type DictEntry struct {
	DictName string `json:"dictName"`
	Word     string `json:"word"`
	Meaning  string `json:"meaning"`
}

func searchAPI(word string, dictCode string) ([]DictEntry, error) {
	payload := map[string]interface{}{
		"word":  word,
		"limit": 500,
		"dicts": []string{dictCode},
	}
	body, _ := json.Marshal(payload)
	
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var results []DictEntry
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse error: %v, body: %s", err, string(data[:min(len(data), 200)]))
	}
	return results, nil
}

func createDB(path string) (*sql.DB, error) {
	os.Remove(path) // start fresh
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE dictionary (word TEXT NOT NULL, meaning TEXT NOT NULL)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE INDEX word ON dictionary(word)`)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func scrapeDict(dictCode string, dbPath string) {
	fmt.Printf("\n=== Scraping %s → %s ===\n", dictCode, dbPath)
	
	db, err := createDB(dbPath)
	if err != nil {
		log.Printf("Failed to create DB for %s: %v", dictCode, err)
		return
	}
	defer db.Close()
	
	seen := make(map[string]bool)
	totalInserted := 0
	
	// Sinhala consonants range: ක (U+0D9A) to ෆ (U+0DC6)
	// Plus Sinhala vowels: අ (U+0D85) to ෆ (U+0DC6)
	// And some special starting chars
	
	// All Sinhala characters to use as prefixes
	var prefixes []string
	
	// Sinhala vowels (independent): අ, ආ, ඇ, ඈ, ඉ, ඊ, උ, ඌ, ඍ, ඎ, ඏ, ඐ, එ, ඒ, ඓ, ඔ, ඕ, ඖ
	for r := rune(0x0D85); r <= rune(0x0D96); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	// Sinhala consonants: ක to ෆ
	for r := rune(0x0D9A); r <= rune(0x0DC6); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	
	// Also search common Roman Pali starting letters (some dicts may use Roman)
	for _, c := range []string{"a", "b", "c", "d", "e", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "r", "s", "t", "u", "v", "y"} {
		prefixes = append(prefixes, c+"%")
	}
	
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	
	batchCount := 0
	for i, prefix := range prefixes {
		results, err := searchAPI(prefix, dictCode)
		if err != nil {
			log.Printf("  Error searching %q: %v", prefix, err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		newEntries := 0
		for _, r := range results {
			key := r.Word + "|" + r.Meaning[:min(len(r.Meaning), 50)]
			if seen[key] {
				continue
			}
			seen[key] = true
			stmt.Exec(r.Word, r.Meaning)
			totalInserted++
			newEntries++
			batchCount++
		}
		
		if len(results) >= 490 {
			// Need deeper search with 2-char prefixes
			fmt.Printf("  %s: %d results (need deeper search)\n", prefix, len(results))
			// Get 2nd char range for Sinhala
			if len(prefix) > 0 {
				firstChar, size := utf8.DecodeRuneInString(prefix)
				_ = firstChar; _ = size
				// Add more specific prefixes
				for r2 := rune(0x0D85); r2 <= rune(0x0D96); r2++ {
					p2 := string(firstChar) + string(r2) + "%"
					results2, err2 := searchAPI(p2, dictCode)
					if err2 != nil { continue }
					for _, r := range results2 {
						key := r.Word + "|" + r.Meaning[:min(len(r.Meaning), 50)]
						if seen[key] { continue }
						seen[key] = true
						stmt.Exec(r.Word, r.Meaning)
						totalInserted++
						batchCount++
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
		
		// Commit every 1000 entries
		if batchCount >= 1000 {
			tx.Commit()
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
			batchCount = 0
		}
		
		if i % 10 == 0 {
			fmt.Printf("  Progress: %d/%d prefixes, %d entries so far\n", i+1, len(prefixes), totalInserted)
		}
		
		time.Sleep(200 * time.Millisecond) // Be polite to the server
	}
	
	tx.Commit()
	fmt.Printf("  Done! Total entries: %d\n", totalInserted)
}

func scrapeReverseDict(dictCode string, dbPath string) {
	fmt.Printf("\n=== Scraping reverse dict %s → %s ===\n", dictCode, dbPath)
	
	db, err := createDB(dbPath)
	if err != nil {
		log.Printf("Failed to create DB for %s: %v", dictCode, err)
		return
	}
	defer db.Close()
	
	seen := make(map[string]bool)
	totalInserted := 0
	
	// For Thai headword dicts, search using Thai consonants
	// Thai consonants: ก (0x0E01) to ฮ (0x0E2E)
	var prefixes []string
	for r := rune(0x0E01); r <= rune(0x0E2E); r++ {
		prefixes = append(prefixes, string(r)+"%")
	}
	// Also Thai vowels
	for _, c := range []string{"อ", "อา", "อิ", "อี", "อุ", "อู", "เ", "แ", "โ", "ไ", "ใ"} {
		prefixes = append(prefixes, c+"%")
	}
	
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
	batchCount := 0
	
	for i, prefix := range prefixes {
		results, err := searchAPI(prefix, dictCode)
		if err != nil {
			log.Printf("  Error searching %q: %v", prefix, err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		for _, r := range results {
			key := r.Word + "|" + r.Meaning[:min(len(r.Meaning), 50)]
			if seen[key] { continue }
			seen[key] = true
			stmt.Exec(r.Word, r.Meaning)
			totalInserted++
			batchCount++
		}
		
		if batchCount >= 1000 {
			tx.Commit()
			tx, _ = db.Begin()
			stmt, _ = tx.Prepare(`INSERT INTO dictionary (word, meaning) VALUES (?, ?)`)
			batchCount = 0
		}
		
		if i % 5 == 0 {
			fmt.Printf("  Progress: %d/%d prefixes, %d entries so far\n", i+1, len(prefixes), totalInserted)
		}
		time.Sleep(200 * time.Millisecond)
	}
	
	tx.Commit()
	fmt.Printf("  Done! Total entries: %d\n", totalInserted)
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

func main() {
	fmt.Println("=== Pali Dict Scraper ===")
	fmt.Println("Scraping new dictionaries from pali-thai-dictionary.onrender.com")
	
	// Scrape main Sinhala-headword dicts
	for code, dbName := range newDicts {
		scrapeDict(code, outputDir+`\`+dbName)
	}
	
	// Scrape reverse Thai-headword dicts
	for code, dbName := range reverseDicts {
		scrapeReverseDict(code, outputDir+`\`+dbName)
	}
	
	fmt.Println("\n=== All done! ===")
}
