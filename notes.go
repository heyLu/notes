package main

import (
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"io/ioutil"
	"os"
)

var config struct {
	dbUrl string
}

func init() {
	config.dbUrl = "files://db?name=posts"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <cmd> [<args>]\n", os.Args[0])
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "import":
		conn := ConnectOrInit(config.dbUrl)
		err := ImportFromPinboard(args[0], conn)
		if err != nil {
			panic(err)
		}
	case "server":
		conn := ConnectOrInit(config.dbUrl)
		err := RunServer(conn)
		if err != nil {
			panic(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command '%s'\n", cmd)
		os.Exit(1)
	}
}

func ConnectOrInit(dbUrl string) connection.Connection {
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

	return conn
}
