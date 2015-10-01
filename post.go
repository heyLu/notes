package main

import (
	"bytes"
	"fmt"
	"github.com/heyLu/mu"
	"github.com/heyLu/mu/database"
	"net/url"
	"time"
)

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

func (p Post) URL() *url.URL {
	u := p.Get(mu.Keyword("note", "url"))
	if u == nil {
		return nil
	}
	return u.(*url.URL)
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

func (p Post) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "{")
	fmt.Fprintf(buf, "\"id\": %q, ", p.Id())
	fmt.Fprintf(buf, "\"title\": %q, ", p.Title())
	fmt.Fprintf(buf, "\"content\": %q, ", p.Content())
	fmt.Fprintf(buf, "\"date\": \"%s\"", p.Date().Format(time.RFC3339))
	u := p.URL()
	if u != nil {
		fmt.Fprintf(buf, " ,\"url\": %q", u)
	}
	tags := p.Tags()
	if tags != nil {
		fmt.Fprint(buf, " ,\"tags\": [")
		first := true
		for _, tag := range tags {
			if !first {
				fmt.Fprint(buf, ", ")
			}
			first = false
			fmt.Fprintf(buf, "%q", tag)
		}
		fmt.Fprintf(buf, "]")
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
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
