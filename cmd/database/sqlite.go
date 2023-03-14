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
