package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

func main() {
	run("http://shop.nordstrom.com/c/women")
}

func run(url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "noder: %v\n", err)
	}
	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "outline: %v\n", err)
		os.Exit(1)
	}

	downloadImages := func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "img" {
			return false
		}
		for _, a := range n.Attr {
			if a.Key != "src" {
				continue
			}
			u, err := resp.Request.URL.Parse(a.Val)
			if err != nil {
				fmt.Fprintf(os.Stderr, "downloadImages: %v\n", err)
			}
			name := downloadImage(u.String())
			n.Data = name
		}
		return false
	}

	node := getNodeById(doc, "main-content")
	forEachNode(node, downloadImages)
	forEachNode(node, stripCommentAndSpace)
	html.Render(os.Stdout, node)
}

func forEachNode(n *html.Node, f func(*html.Node) bool) {
	var done bool
	if f != nil {
		done = f(n)
	}
	if done {
		return
	}
	c := n.FirstChild
	for c != nil {
		s := c.NextSibling
		forEachNode(c, f)
		c = s
	}
}

func stripCommentAndSpace(n *html.Node) bool {
	switch n.Type {
	case html.CommentNode:
		par := n.Parent
		par.RemoveChild(n)
	case html.TextNode:
		var leading, trailing bool
		r, _ := utf8.DecodeRuneInString(n.Data)
		if unicode.IsSpace(r) {
			leading = true
		}
		r, _ = utf8.DecodeLastRuneInString(n.Data)
		if unicode.IsSpace(r) {
			trailing = true
		}
		n.Data = strings.TrimSpace(n.Data)
		switch {
		case len(n.Data) == 0:
			n.Data = " "
		case leading && trailing:
			n.Data = " " + n.Data + " "
		case leading:
			n.Data = " " + n.Data
		case trailing:
			n.Data += " "
		}
	}
	return false
}
func getNodeById(n *html.Node, id string) *html.Node {
	var res *html.Node
	filter := func(n *html.Node) bool {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				res = n
				return true
			}
		}
		return false
	}
	forEachNode(n, filter)
	return res
}

func downloadImage(imgURL string) string {
	u, err := url.Parse(imgURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: %v\n")
	}
	ext := path.Ext(u.Path)
	b := make([]byte, 8)
	rand.Read(b)
	name := fmt.Sprintf("%x%v", b, ext)

	resp, err := http.Get(imgURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: %v\n")
	}
	defer resp.Body.Close()

	f, err := os.Create(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: opening file: %v\n")
	}
	defer f.Close()

	io.Copy(f, resp.Body)
	return name
}
