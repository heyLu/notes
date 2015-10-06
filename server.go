package main

import (
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/connection"
	"github.com/heyLu/mu/index"
	tx "github.com/heyLu/mu/transactor"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"./renderable"
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

	http.HandleFunc("/notes/", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/edit") {
			if req.Method == "POST" {
				NewPost(w, req)
				return
			}

			renderable.HandleRequest(EditPost)(w, req)
		} else {
			renderable.HandleRequest(GetPost)(w, req)
		}
	})
	http.HandleFunc("/notes", renderable.HandleRequest(ListPosts))
	http.HandleFunc("/search", renderable.HandleRequest(SearchPosts))
	http.HandleFunc("/tags/", renderable.HandleRequest(GetTag))
	http.HandleFunc("/tags", renderable.HandleRequest(ListTags))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("listening on", serverConfig.addr)
	return http.ListenAndServe(serverConfig.addr, nil)
}

func GetPost(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	parts := strings.SplitN(req.URL.Path, "/", 3)
	if len(parts) != 3 || parts[2] == "" {
		return renderable.RenderableStatus(http.StatusBadRequest), nil
	}
	noteId := parts[2]

	db := serverConfig.conn.Db()
	aid := db.Entid(mu.Keyword("note", "id"))
	if aid == -1 {
		panic("db not initialized")
	}
	minDatom := index.NewDatom(index.MinDatom.E(), aid, noteId, index.MaxDatom.Tx(), index.MinDatom.Added())
	maxDatom := index.NewDatom(index.MaxDatom.E(), aid, noteId, index.MinDatom.Tx(), index.MaxDatom.Added())
	iter := db.Avet().DatomsAt(minDatom, maxDatom)
	datom := iter.Next()
	if datom == nil {
		return renderable.RenderableStatus(http.StatusNotFound), nil
	}

	post := Post{db.Entity(datom.E())}
	return renderable.Renderable{
		Metadata: map[string]interface{}{
			"Title": post.Title(),
		},
		Data:     []Post{post},
		Template: listPostsTemplate,
	}, nil
}

func EditPost(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	parts := strings.SplitN(req.URL.Path, "/", 4)
	if len(parts) != 4 || parts[2] == "" {
		return renderable.RenderableStatus(http.StatusBadRequest), nil
	}
	noteId := parts[2]

	db := serverConfig.conn.Db()
	aid := db.Entid(mu.Keyword("note", "id"))
	if aid == -1 {
		panic("db not initialized")
	}
	minDatom := index.NewDatom(index.MinDatom.E(), aid, noteId, index.MaxDatom.Tx(), index.MinDatom.Added())
	maxDatom := index.NewDatom(index.MaxDatom.E(), aid, noteId, index.MinDatom.Tx(), index.MaxDatom.Added())
	iter := db.Avet().DatomsAt(minDatom, maxDatom)
	datom := iter.Next()
	if datom == nil {
		return renderable.RenderableStatus(http.StatusNotFound), nil
	}

	post := Post{db.Entity(datom.E())}
	return renderable.Renderable{
		Metadata: map[string]interface{}{
			"Title": post.Title(),
		},
		Data:     post,
		Template: createPostTemplate,
	}, nil
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

	id := req.FormValue("id")
	if id == "" {
		id = generateId()
	}
	title := req.FormValue("title")
	content := req.FormValue("content")
	date := time.Now().Round(time.Second)

	noteId := mu.Id(mu.Tempid(mu.DbPartUser, -1))
	txData := make([]tx.TxDatum, 4)
	txData[0] = tx.Datum{Op: tx.Assert, E: noteId, A: mu.Keyword("note", "id"), V: tx.NewValue(id)}
	txData[1] = tx.Datum{Op: tx.Assert, E: noteId, A: mu.Keyword("note", "title"), V: tx.NewValue(title)}
	txData[2] = tx.Datum{Op: tx.Assert, E: noteId, A: mu.Keyword("note", "content"), V: tx.NewValue(content)}
	txData[3] = tx.Datum{Op: tx.Assert, E: noteId, A: mu.Keyword("note", "date"), V: tx.NewValue(date)}

	url := req.FormValue("url")
	if url != "" {
		txData = append(txData, tx.Datum{
			Op: tx.Assert,
			E:  noteId,
			A:  mu.Keyword("note", "url"),
			V:  tx.NewValue(url),
		})
	}

	n := -100
	rawTags := req.FormValue("tags")
	for _, tag := range strings.Split(rawTags, " ") {
		if tag == "" {
			continue
		}

		tagId := mu.Id(mu.Tempid(mu.DbPartUser, n))
		txData = append(txData,
			tx.Datum{Op: tx.Assert, E: noteId, A: mu.Keyword("note", "tags"), V: tx.NewValue(tagId)},
			tx.Datum{Op: tx.Assert, E: tagId, A: mu.Keyword("tag", "name"), V: tx.NewValue(tag)})
		n -= 1
	}

	_, err = mu.Transact(serverConfig.conn, txData)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error: ", err)
		status := http.StatusInternalServerError
		http.Error(w, http.StatusText(status), status)
		return
	}

	http.Redirect(w, req, "/notes", http.StatusSeeOther)
}

func ListPosts(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	db := serverConfig.conn.Db()

	id := db.Entid(mu.Keyword("note", "date"))
	if id == -1 {
		fmt.Fprintf(os.Stderr, "Error: :note/date not present\n")
		os.Exit(1)
	}

	min, max := index.MinDatom, index.MaxDatom
	start := index.NewDatom(min.E(), id, min.V(), max.Tx(), min.Added())
	end := index.NewDatom(max.E(), id, max.V(), min.Tx(), max.Added())
	iter := db.Avet().DatomsAt(start, end)

	postIds := make([]int, 0)
	for datom := iter.Next(); datom != nil; datom = iter.Next() {
		postIds = append(postIds, datom.E())
	}

	posts := make([]Post, 0)
	l := len(postIds)
	n := fromQueryInt(req, "n", 100)
	for i := 0; i < l && i < n; i++ {
		entity := db.Entity(postIds[l-i-1])
		posts = append(posts, Post{entity})
	}

	return renderable.Renderable{
		Metadata: map[string]interface{}{
			"Title": "All notes",
		},
		Data:     posts,
		Template: listPostsTemplate,
	}, nil
}

func SearchPosts(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	query := req.FormValue("q")
	if query == "" {
		return renderable.Renderable{
			Metadata: map[string]interface{}{"Title": "Search notes"},
			Template: listPostsTemplate,
		}, nil
	}

	db := serverConfig.conn.Db()
	id := db.Entid(mu.Keyword("note", "title"))
	if id == -1 {
		panic("db not initialized")
	}

	min, max := index.MinDatom, index.MaxDatom
	start := index.NewDatom(min.E(), id, min.V(), max.Tx(), min.Added())
	end := index.NewDatom(max.E(), id, max.V(), min.Tx(), max.Added())
	iter := db.Aevt().DatomsAt(start, end)

	posts := make([]Post, 0)
	for datom := iter.Next(); datom != nil; datom = iter.Next() {
		if strings.Contains(strings.ToLower(datom.V().Val().(string)), strings.ToLower(query)) {
			posts = append(posts, Post{db.Entity(datom.E())})
		}
	}
	sort.Sort(sort.Reverse(postsByDate(posts)))

	n := fromQueryInt(req, "n", 100)
	if n < 1 || n > len(posts) {
		n = len(posts)
	}
	return renderable.Renderable{
		Metadata: map[string]interface{}{
			"Title": fmt.Sprintf("Search for '%s'", query),
		},
		Data:     posts[0:n],
		Template: listPostsTemplate,
	}, nil
}

func GetTag(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	parts := strings.SplitN(req.URL.Path, "/", 3)
	if len(parts) != 3 || parts[2] == "" {
		return renderable.RenderableStatus(http.StatusBadRequest), nil
	}
	tagName := parts[2]

	db := serverConfig.conn.Db()

	id := db.Entid(mu.Keyword("note", "tags"))
	if id == -1 {
		panic("db not initialized")
	}

	tagId := db.Entid(mu.LookupRef(mu.Keyword("tag", "name"), tagName))
	if tagId == -1 {
		return renderable.RenderableStatus(http.StatusNotFound), nil
	}

	min, max := index.MinDatom, index.MaxDatom
	start := index.NewDatom(min.E(), id, tagId, max.Tx(), min.Added())
	end := index.NewDatom(max.E(), id, tagId, min.Tx(), max.Added())
	iter := db.Vaet().DatomsAt(start, end)

	posts := make([]Post, 0)
	for datom := iter.Next(); datom != nil; datom = iter.Next() {
		posts = append(posts, Post{db.Entity(datom.E())})
	}
	sort.Sort(sort.Reverse(postsByDate(posts)))

	n := fromQueryInt(req, "n", 100)
	if n < 1 || n > len(posts) {
		n = len(posts)
	}
	return renderable.Renderable{
		Metadata: map[string]interface{}{
			"Title": fmt.Sprintf("Notes tagged '%s'", tagName),
		},
		Data:     posts[0:n],
		Template: listPostsTemplate,
	}, nil
}

type postsByDate []Post

func (p postsByDate) Len() int           { return len(p) }
func (p postsByDate) Less(i, j int) bool { return p[i].Date().Unix() < p[j].Date().Unix() }
func (p postsByDate) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func ListTags(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	db := serverConfig.conn.Db()

	aid := db.Entid(mu.Keyword("tag", "name"))
	if aid == -1 {
		panic("db not initialized")
	}
	minDatom := index.NewDatom(index.MinDatom.E(), aid, index.MinValue, index.MaxDatom.Tx(), index.MinDatom.Added())
	maxDatom := index.NewDatom(index.MaxDatom.E(), aid, index.MaxValue, index.MinDatom.Tx(), index.MaxDatom.Added())
	iter := db.Avet().DatomsAt(minDatom, maxDatom)

	tags := make([]string, 0)
	for datom := iter.Next(); datom != nil; datom = iter.Next() {
		tags = append(tags, datom.V().Val().(string))
	}

	return renderable.Renderable{
		Metadata: map[string]interface{}{"Title": "All tags"},
		Data:     tags,
		Template: listTagsTemplate,
	}, nil
}

var templateFuncs = template.FuncMap{
	"joinTags": func(tags []Tag) template.HTML {
		if len(tags) == 0 {
			return ""
		}

		joined := template.HTML("")
		first := true
		for _, tag := range tags {
			if !first {
				joined += ", "
			}
			tagLink := fmt.Sprintf("<a href=\"/tags/%s\">%s</a>",
				template.JSEscapeString(tag.Name()), template.HTMLEscapeString(tag.Name()))
			joined += template.HTML(tagLink)
			first = false
		}
		return joined
	},
	"join": func(tags []Tag, separator string) string {
		joined := ""
		first := true
		for _, tag := range tags {
			if !first {
				joined += separator
			}
			joined += tag.Name()
			first = false
		}
		return joined
	},
}

var createPostTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(createPostTemplateStr))
var createPostTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>Write a note!</title>

		<link rel="stylesheet" href="/static/codemirror/lib/codemirror.css" />
		<link rel="stylesheet" href="/static/codemirror/addon/scroll/simplescrollbars.css" />
		<link rel="stylesheet" href="/static/write.css" />
	</head>

	<body>
		<div id="editor">
			<div id="editor-stats">
				<span id="stats-words">0 words</span>
				<span id="stats-chars">0 characters</span>
				<span id="stats-time">0 minutes</span>
			</div>
		</div>

		<div id="sidebar">
			<form method="POST" action="/new">
				<div class="field">
					<label for="url">url</label>
					<input id="url" name="url" type="url" value="{{ if .Data.URL }}{{ .Data.URL }}{{ end }}" />
				</div>
				<div class="field">
					<label for="title">title</label>
					<input id="title" name="title" type="text" required />
				</div>
				<div class="field">
					<label for="tags">tags</label>
					<input id="tags" name="tags" type="text" autocomplete="off" value="{{ join .Data.Tags " " }}" />
				</div>

				<input name="id" type="hidden" value="{{ .Data.Id }}" />
				<input id="content" name="content" type="hidden" />

				<div class="field">
					<div style="display: inline-block; width: 3em;"></div>
					<input id="submit" type="submit" value="{{ if .Data }}Edit note{{ else }}Create note{{ end }}" />
				</div>
			</form>
		</div>

		<script src="/static/codemirror/lib/codemirror.js"></script>
		<script src="/static/codemirror/mode/markdown/markdown.js"></script>
		<script src="/static/codemirror/addon/scroll/simplescrollbars.js"></script>
		<script src="/static/write.js"></script>
		<script src="/static/tags.js"></script>
		{{ if .Data }}
		<script>
			if (codeMirror.getValue() == "") {
				codeMirror.setValue({{ printf "# %s\n%s" .Data.Title .Data.Content }});
			}
		</script>
		{{ end }}
	</body>
</html>
`

var listPostsTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(listPostsTemplateStr))
var listPostsTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>{{ .Metadata.Title }}</title>
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

		.post .tags a {
			text-decoration: none;
			color: black;
		}

		.post .tags a:hover {
			text-decoration: underline;
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
		<script>
			window.addEventListener("keydown", function(ev) {
				if (ev.ctrlKey && ev.key == "n") {
					ev.preventDefault();
					window.location = "/new";
				}
			});
		</script>

		<form method="GET" action="/search">
			<input id="search" name="q" type="search" />
		</form>

		<a id="new-note" href="/new">Write a note</a>

		{{ range .Data }}
		<div class="post">
			<a class="permalink" href="/notes/{{ .Id }}">⚓</a>
			{{ if .URL }}
			<h1><a href="{{ .URL }}">{{ .Title }}</a></h1>
			{{ else }}
			<h1>{{ .Title }}</h1>
			{{ end }}
			<time>{{ .Date }}</time>
			{{ if .Tags }}<div class="tags">{{ .Tags | joinTags }}</div>{{ end }}
			<pre>{{ .Content }}</pre>
		</div>
		{{ end }}
	</body>
</html>
`

var listTagsTemplate = template.Must(template.New("").Funcs(templateFuncs).Parse(listTagsTemplateStr))
var listTagsTemplateStr = `<!doctype html>
<html>
	<head>
		<meta charset="utf-8" />
		<title>{{ .Metadata.Title }}</title>
	</head>

	<body>
		<ul class="tags">
			{{ range .Data }}
			<li><a href="/tags/{{ . }}">{{ . }}</a></li>
			{{ end }}
		</ul>
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
