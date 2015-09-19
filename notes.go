package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/database"
	tx "github.com/heyLu/mu/transactor"
	"io/ioutil"
	"net/url"
	"os"
	"time"
)

type Posts struct {
	XMLName xml.Name `xml:"posts"`
	User    string   `xml:"user,attr"`
	Posts   []Post   `xml:"post"`
}

type Post struct {
	XMLName xml.Name  `xml:"post"`
	Title   string    `xml:"description,attr"`
	Content string    `xml:"extended,attr"`
	Date    time.Time `xml:"time,attr"`
	URL     *url.URL  `xml:"href,attr"`
}

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	decoder := xml.NewDecoder(f)
	var posts Posts
	err = decoder.Decode(&posts)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(posts.Posts))

	dbUrl := "files://db?name=posts"
	isNew, err := mu.CreateDatabase(dbUrl)
	if err != nil {
		panic(err)
	}

	conn, err := mu.Connect(dbUrl)
	if err != nil {
		panic(err)
	}

	if isNew {
		schema, err := ioutil.ReadFile("schema.edn")
		if err != nil {
			panic(err)
		}

		_, err = mu.TransactString(conn, string(schema))
		if err != nil {
			panic(err)
		}
	}

	n := 0
	nextId := func() int {
		n -= 1
		return n
	}

	txData := make([]tx.TxDatum, 0)
	for _, post := range posts.Posts {
		txDatum := tx.TxMap{
			Id: mu.Id(mu.Tempid(mu.DbPartUser, nextId())),
			Attributes: map[database.Keyword][]tx.Value{
				mu.Keyword("note", "id"):      []tx.Value{tx.NewValue(generateId())},
				mu.Keyword("note", "title"):   []tx.Value{tx.NewValue(post.Title)},
				mu.Keyword("note", "content"): []tx.Value{tx.NewValue(post.Content)},
				mu.Keyword("note", "date"):    []tx.Value{tx.NewValue(post.Date)},
			},
		}
		txData = append(txData, txDatum)
	}

	txRes, err := mu.Transact(conn, txData)
	if err != nil {
		panic(err)
	}

	fmt.Println("added", len(txRes.Datoms), "datoms")
}

func generateId() string {
	buf := make([]byte, 5)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
