package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Db struct {
	db *sql.DB
}

func NewConnection(dbName string) *Db {
	if _, err := os.Stat(dbName); errors.Is(err, os.ErrNotExist) {
		err := os.WriteFile(dbName, []byte(""), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		panic(err)
	}
	return &Db{
		db: db,
	}
}

func (d *Db) CreateTableIfNotExists() {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY UNIQUE,
		telegram_id INTEGER NOT NULL UNIQUE,
		command TEXT NOT NULL
	)`)
	if err != nil {
		log.Panicln(err)
		return
	}
	_, err = d.db.Exec(`CREATE TABLE IF NOT EXISTS lists (
		id INTEGER PRIMARY KEY UNIQUE,
		telegram_id INTEGER NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL
	)`)
	if err != nil {
		log.Panicln(err)
		return
	}
}

func (d *Db) GetCommandById(id int64) (string, error) {
	rows, err := d.db.Prepare(`
		SELECT command FROM users WHERE telegram_id = ?
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var command string
	err = rows.QueryRow(id).Scan(&command)
	if err != nil {
		fmt.Println(err)
	}
	return command, err
}

func (d *Db) SaveCommandById(command string, id int64) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (telegram_id, command) VALUES ($1, $2)`, id, command)
	return err
}

func (d *Db) ClearCommandById(id int64) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO users (telegram_id, command) VALUES ($1, '')`, id)
	return err
}

func (d *Db) SaveList(id int64, key, value string) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO lists (telegram_id, key, value) VALUES ($1, $2, $3)`, id, key, value)
	return err
}

func (d *Db) GetList(id int64, key string) ([]string, error) {
	result := make([]string, 0, 10)
	rows, err := d.db.Query(`
		SELECT value FROM lists WHERE telegram_id = $1 AND key = $2
	`, id, key)
	defer rows.Close()
	if err != nil {
		return []string{}, err
	}
	for rows.Next() {
		var value string
		rows.Scan(&value)
		result = append(result, value)
	}
	return result, nil
}

func (d *Db) GetListLists(id int64) ([]string, error) {
	result := make([]string, 0, 10)
	rows, err := d.db.Query(`
		SELECT key FROM lists WHERE telegram_id = $1 GROUP BY key
	`, id)
	defer rows.Close()
	if err != nil {
		return []string{}, err
	}
	for rows.Next() {
		var key string
		rows.Scan(&key)
		result = append(result, key)
	}
	if len(result) == 0 {
		return result, errors.New("List is empty")
	}
	return result, nil
}

func (d *Db) DeleteList(id int64, key string) error {
	_, err := d.db.Exec(`DELETE FROM lists WHERE telegram_id = $1 AND key = $2`, id, key)
	return err
}
