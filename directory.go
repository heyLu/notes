package main

import (
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"github.com/heyLu/mu/database"
	tx "github.com/heyLu/mu/transactor"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func ImportFromDirectory(directory string, conn connection.Connection) error {
	f, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer f.Close()

	fis, err := f.Readdir(-1)
	if err != nil {
		return err
	}

	n := -1
	txData := make([]tx.TxDatum, 0)
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(path.Join(directory, fi.Name()))
		if err != nil {
			return err
		}

		title := fi.Name()
		content := string(data)

		if newLine := strings.IndexByte(content, '\n'); newLine != -1 {
			firstLine := content[0:newLine]
			if strings.HasPrefix(firstLine, "# ") && len(firstLine) > 2 {
				title = firstLine[2:]
				content = content[newLine+1:]
			}
		}

		txDatum := tx.TxMap{
			Id: mu.Id(mu.Tempid(mu.DbPartUser, n)),
			Attributes: map[database.Keyword][]tx.Value{
				mu.Keyword("note", "id"):      []tx.Value{tx.NewValue(generateId())},
				mu.Keyword("note", "title"):   []tx.Value{tx.NewValue(title)},
				mu.Keyword("note", "content"): []tx.Value{tx.NewValue(content)},
				mu.Keyword("note", "date"):    []tx.Value{tx.NewValue(fi.ModTime())},
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
