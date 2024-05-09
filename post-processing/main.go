package main

import (
    "bufio"
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "regexp"

    _ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

func main() {
    db, err := sql.Open("sqlite3", "history.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    createTables(db)

    path := "../history" // directory containing the files
    err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() {
            processFile(db, path)
        }
        return nil
    })

    if err != nil {
        fmt.Printf("Error walking the path %q: %v\n", path, err)
    }
}

func createTables(db *sql.DB) {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS totals (
            date TEXT PRIMARY KEY,
            total INTEGER
        );
        CREATE TABLE IF NOT EXISTS repositories (
            repository TEXT PRIMARY KEY,
            first_seen TEXT
        );
    `)
    if err != nil {
        panic(err)
    }
}

func processFile(db *sql.DB, filePath string) {
    file, err := os.Open(filePath)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    date := filepath.Base(filePath) // Use file name as date

    var total int
    repoRegex := regexp.MustCompile(`^[\w-]+/[\w-]+$`)
    firstLine := true

    tx, err := db.Begin()
    if err != nil {
        panic(err)
    }

    for scanner.Scan() {
        line := scanner.Text()
        if firstLine {
            // Assume first line is always the total count
            fmt.Sscanf(line, "Total unique repositories found: %d", &total)
            _, err = tx.Exec("INSERT OR REPLACE INTO totals (date, total) VALUES (?, ?)", date, total)
            if err != nil {
                panic(err)
            }
            firstLine = false
        } else if repoRegex.MatchString(line) {
            _, err = tx.Exec("INSERT INTO repositories (repository, first_seen) VALUES (?, ?) ON CONFLICT(repository) DO UPDATE SET first_seen = MIN(first_seen, ?)", line, date, date)
            if err != nil {
                panic(err)
            }
        }
    }

    if err := scanner.Err(); err != nil {
        panic(err)
    }

    err = tx.Commit()
    if err != nil {
        panic(err)
    }
}

