package main
import ("database/sql"; "fmt"; _ "modernc.org/sqlite")
func main() {
    dbs := []string{"th-ng", "th-dmc", "th-thatu", "th-thatu2"}
    for _, name := range dbs {
        db, _ := sql.Open("sqlite", `d:\Program\TIPITAKA\server\server-data\dicts\` + name + ".db")
        var count int
        db.QueryRow("SELECT COUNT(*) FROM dictionary").Scan(&count)
        rows, _ := db.Query("SELECT word, meaning FROM dictionary WHERE word LIKE ? LIMIT 3", "?????%")
        fmt.Printf("[%s] total=%d, vicara matches:\n", name, count)
        found := 0
        for rows.Next() {
            var w, m string; rows.Scan(&w, &m)
            if len(m) > 60 { m = m[:60] }
            fmt.Printf("  %q -> %q\n", w, m)
            found++
        }
        if found == 0 { 
            // Try a sample
            rows2, _ := db.Query("SELECT word FROM dictionary LIMIT 3")
            fmt.Printf("  (no vicara match) Sample words: ")
            for rows2.Next() { var w string; rows2.Scan(&w); fmt.Printf("%q ", w) }
            fmt.Println()
        }
        db.Close()
    }
}
