package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "../server-data/db/dict-all.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT token FROM tokens LIMIT 20 OFFSET 958000")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		rows.Scan(&token)
		if len(token) > 0 {
			r := []rune(token)[0]
			fmt.Printf("Token: %s | First Rune: %U\n", token, r)
		}
	}
}
