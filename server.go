package main

import (
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"github.com/heyLu/mu/database"
	"github.com/heyLu/mu/index"
	"html/template"
	"net/http"
	"os"
	"strconv"
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

	http.HandleFunc("/list", ListPosts)
	fmt.Println("listening on", serverConfig.addr)
	return http.ListenAndServe(serverConfig.addr, nil)
}

type Post struct {
	database.Entity
}

func (p Post) Id() string {
	return p.Get(mu.Keyword("note", "id")).(string)
}

func (p Post) Title() string {
	return p.Get(mu.Keyword("note", "title")).(string)
}

func (p Post) Content() string {
	return p.Get(mu.Keyword("note", "content")).(string)
}

func (p Post) Date() time.Time {
	return p.Get(mu.Keyword("note", "date")).(time.Time)
}

func (p Post) Tags() []Tag {
	rawTags := p.Get(mu.Keyword("note", "tags")).([]interface{})
	if len(rawTags) == 0 {
		return nil
	}

	tags := make([]Tag, len(rawTags))
	for i, rawTag := range rawTags {
		tags[i] = Tag{rawTag.(database.Entity)}
	}
	return tags
}

type Tag struct {
	database.Entity
}

func (t Tag) Name() string {
	return t.Get(mu.Keyword("tag", "name")).(string)
}

func (t Tag) String() string {
	return t.Name()
}

func ListPosts(w http.ResponseWriter, req *http.Request) {
	db := serverConfig.conn.Db()
	posts := listPosts(db, fromQueryInt(req, "n", 100))
	listPostsTemplate.Execute(w, posts)
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

var listPostsTemplate = template.Must(template.New("").Parse(listPostsTemplateStr))

var listPostsTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>All posts</title>
	</head>

	<body>
		{{ range . }}
		<div class="post">
			<h1>{{ .Title }}</h1>
			<time>{{ .Date }}</time>
			<div>{{ .Tags }}</div>
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
