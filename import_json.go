package main

import (
	"encoding/json"
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"github.com/heyLu/mu/database"
	tx "github.com/heyLu/mu/transactor"
	"os"
	"time"
)

type jsonPost struct {
	Id      string    `json:"id"`
	Title   string    `json:"title"`
	Content string    `json:"content"`
	Date    time.Time `json:"created"`
}

func ImportFromJSON(path string, conn connection.Connection) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var posts []jsonPost
	err = decoder.Decode(&posts)
	if err != nil {
		return err
	}

	n := -1
	txData := make([]tx.TxDatum, 0)
	for _, post := range posts {
		txDatum := tx.TxMap{
			Id: mu.Id(mu.Tempid(mu.DbPartUser, n)),
			Attributes: map[database.Keyword][]tx.Value{
				mu.Keyword("note", "id"):      []tx.Value{tx.NewValue(generateId())},
				mu.Keyword("note", "title"):   []tx.Value{tx.NewValue(post.Title)},
				mu.Keyword("note", "content"): []tx.Value{tx.NewValue(post.Content)},
				mu.Keyword("note", "date"):    []tx.Value{tx.NewValue(post.Date)},
			},
		}

		n -= 1
		txData = append(txData, txDatum)
	}

	txRes, err := mu.Transact(conn, txData)
	if err != nil {
		return err
	}
	fmt.Println("added", len(txRes.Datoms), "datoms")

	return nil
}
