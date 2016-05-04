package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

func main() {
	resp, err := http.Get("http://shop.nordstrom.com")
	if err != nil {
		fmt.Fprintf(os.Stderr, "noder: %v\n", err)
	}
	doc, err := html.Parse(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "outline: %v\n", err)
		os.Exit(1)
	}
	filter := func(n *html.Node) bool {
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

	node := getNodeById(doc, "main-content")
	forEachNode(node, filter)
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
