package main

import (
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"github.com/heyLu/mu/database"
	"github.com/heyLu/mu/index"
	tx "github.com/heyLu/mu/transactor"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var serverConfig struct {
	conn connection.Connection
	addr string
}

func init() {
	serverConfig.addr = "localhost:9999"
}

func RunServer(conn connection.Connection) error {
	serverConfig.conn = conn

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			http.Redirect(w, req, "/new", http.StatusSeeOther)
			return
		}

		status := http.StatusNotFound
		http.Error(w, http.StatusText(status), status)
	})
	http.HandleFunc("/notes/", GetPost)
	http.HandleFunc("/notes", ListPosts)
	http.HandleFunc("/new", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			CreatePost(w, req)
		case "POST":
			NewPost(w, req)
		default:
			status := http.StatusMethodNotAllowed
			http.Error(w, http.StatusText(status), status)
		}
	})
	fmt.Println("listening on", serverConfig.addr)
	return http.ListenAndServe(serverConfig.addr, nil)
}

func GetPost(w http.ResponseWriter, req *http.Request) {
	parts := strings.SplitN(req.URL.Path, "/", 3)
	if len(parts) != 3 || parts[2] == "" {
		status := http.StatusBadRequest
		http.Error(w, http.StatusText(status), status)
		return
	}
	noteId := parts[2]

	db := serverConfig.conn.Db()
	aid := db.Entid(mu.Keyword("note", "id"))
	if aid == -1 {
		panic("db not initialized")
	}
	minDatom := index.NewDatom(index.MinDatom.E(), aid, noteId, index.MinDatom.Tx(), index.MinDatom.Added())
	maxDatom := index.NewDatom(index.MaxDatom.E(), aid, noteId, index.MaxDatom.Tx(), index.MaxDatom.Added())
	iter := db.Avet().DatomsAt(minDatom, maxDatom)
	datom := iter.Next()
	if datom == nil {
		status := http.StatusNotFound
		http.Error(w, http.StatusText(status), status)
		return
	}

	post := Post{db.Entity(datom.E())}
	data := struct {
		Title string
		Posts []Post
	}{
		post.Title(),
		[]Post{post},
	}
	listPostsTemplate.Execute(w, data)
}

func CreatePost(w http.ResponseWriter, req *http.Request) {
	createPostTemplate.Execute(w, nil)
}

func NewPost(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	title := req.FormValue("title")
	content := req.FormValue("content")
	date := time.Now().Round(time.Second)

	txData := make([]tx.TxDatum, 1)
	txData[0] = tx.TxMap{
		Id: mu.Id(mu.Tempid(mu.DbPartUser, -1)),
		Attributes: map[database.Keyword][]tx.Value{
			mu.Keyword("note", "id"):      []tx.Value{tx.NewValue(generateId())},
			mu.Keyword("note", "title"):   []tx.Value{tx.NewValue(title)},
			mu.Keyword("note", "content"): []tx.Value{tx.NewValue(content)},
			mu.Keyword("note", "date"):    []tx.Value{tx.NewValue(date)},
		},
	}

	_, err = mu.Transact(serverConfig.conn, txData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error:", err)
		status := http.StatusInternalServerError
		http.Error(w, http.StatusText(status), status)
		return
	}

	http.Redirect(w, req, "/notes", http.StatusSeeOther)
}

func ListPosts(w http.ResponseWriter, req *http.Request) {
	db := serverConfig.conn.Db()
	posts := listPosts(db, fromQueryInt(req, "n", 100))
	data := struct {
		Title string
		Posts []Post
	}{
		"All posts",
		posts,
	}
	listPostsTemplate.Execute(w, data)
}

func listPosts(db *database.Db, n int) []Post {
	id := db.Entid(mu.Keyword("note", "date"))
	if id == -1 {
		fmt.Fprintf(os.Stderr, "Error: :note/date not present\n")
		os.Exit(1)
	}

	min, max := index.MinDatom, index.MaxDatom
	start := index.NewDatom(max.E(), id, min.V(), min.Tx(), min.Added())
	end := index.NewDatom(min.E(), id, max.V(), max.Tx(), max.Added())
	iter := db.Avet().DatomsAt(start, end)

	postIds := make([]int, 0)
	for datom := iter.Next(); datom != nil; datom = iter.Next() {
		postIds = append(postIds, datom.E())
	}

	posts := make([]Post, 0)
	l := len(postIds)
	for i := 0; i < l && i < n; i++ {
		entity := db.Entity(postIds[l-i-1])
		posts = append(posts, Post{entity})
	}

	return posts
}

var templateFuncs = template.FuncMap{
	"joinTags": func(tags []Tag) string {
		if len(tags) == 0 {
			return ""
		}

		joined := ""
		first := true
		for _, tag := range tags {
			if !first {
				joined += ", "
			}
			joined += tag.Name()
			first = false
		}
		return joined
	},
}

var createPostTemplate = template.Must(template.New("").Parse(createPostTemplateStr))
var createPostTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>Write a new post</title>
		<style>
		textarea {
			width: 40em;
			font-family: "Liberation Mono", monospace;
			font-size: smaller;
			white-space: pre-wrap;
		}

		.field label {
			display: inline-block;
			width: 5em;
		}

		.field input[type="text"], .field input[type="url"] {
			width: 40em;
		}

		.field textarea {
			height: 50vh;
		}

		.submit {
			margin-top: 2em;
			margin-left: 5em;
		}
		</style>
	</head>

	<body>
		<form method="POST">
			<div class="field">
				<label for="url">url</label>
				<input name="url" type="url" />
			</div>
			<div class="field">
				<label for="title">title</label>
				<input name="title" type="text" />
			</div>
			<div class="field">
				<label for="content">content</label>
				<textarea name="content"></textarea>
			</div>
			<div class="field">
				<label for="tags">tags</label>
				<input name="tags" type="text" />
			</div>

			<!--<div class="field">
				<label for="private">private</label>
				<input type="checkbox" name="private" checked />
			</div>

			<div class="field">
				<label for="read-later">read later</label>
				<input type="checkbox" name="read-later" />
			</div>-->

			<div class="submit">
				<input type="submit" value="Create post" />
			</div>
		</form>
	</body>
</html>
`

var listPostsTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(listPostsTemplateStr))
var listPostsTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>{{ .Title }}</title>
		<style>
		#new-note {
			position: fixed;
			left: 60em;
			top: 0;
			padding: 1ex;
			background-color: #eee;
		}

		.post .permalink {
			float: left;
			padding: 0.5ex;
		}

		.post h1 {
			margin-bottom: 0;
		}

		.post pre {
			max-width: 40em;
			font-family: "Liberation Mono", monospace;
			font-size: smaller;
			white-space: pre-wrap;
		}
		</style>
	</head>

	<body>
		<a id="new-note" href="/new">Write a note</a>

		{{ range .Posts }}
		<div class="post">
			<a class="permalink" href="/notes/{{ .Id }}">⚓</a>
			{{ if .URL }}
			<h1><a href="{{ .URL }}">{{ .Title }}</a></h1>
			{{ else }}
			<h1>{{ .Title }}</h1>
			{{ end }}
			<time>{{ .Date }}</time>
			{{ if .Tags }}<div>{{ .Tags | joinTags }}</div>{{ end }}
			<pre>{{ .Content }}</pre>
		</div>
		{{ end }}
	</body>
</html>
`

func fromQueryInt(req *http.Request, param string, n int) int {
	val := req.URL.Query().Get(param)
	if val != "" {
		n, err := strconv.Atoi(val)
		if err == nil {
			return n
		}
	}
	return n
}
