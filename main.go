package main

import (
	"bytes"
	"crypto/sha1"
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
	run("http://shop.nordstrom.com/c/men")
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

	ih := &imageHelper{
		baseURL:         resp.Request.URL,
		urlToHash:       make(map[string]string, 20),
		imageDownloaded: make(map[string]struct{}, 20),
	}

	node := getNodeByID(doc, "main-content")
	forEachNode(node, ih.downloadImages)
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

func getNodeByID(n *html.Node, id string) *html.Node {
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

type imageHelper struct {
	baseURL         *url.URL
	urlToHash       map[string]string
	imageDownloaded map[string]struct{}
}

func (ih *imageHelper) downloadImages(n *html.Node) bool {
	if n.Type != html.ElementNode || n.Data != "img" {
		return false
	}
	for _, a := range n.Attr {
		if a.Key != "src" {
			continue
		}
		u, err := ih.baseURL.Parse(a.Val)
		if err != nil {
			fmt.Fprintf(os.Stderr, "downloadImages: %v\n", err)
			continue
		}
		name := ih.downloadImage(u.String())
		n.Data = name
	}
	return false
}

func (ih *imageHelper) downloadImage(imgURL string) string {
	// if we already downloaded an image from this url return the filename
	if v, ok := ih.urlToHash[imgURL]; ok {
		return v
	}

	u, err := url.Parse(imgURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: %v\n", err)
		return ""
	}
	ext := path.Ext(u.Path)

	resp, err := http.Get(imgURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	// create a hash of the contents
	h := sha1.New()
	var buf bytes.Buffer
	w := io.MultiWriter(h, &buf)
	io.Copy(w, resp.Body)

	// save url to hash mapping
	filename := fmt.Sprintf("%x%v", h.Sum(nil), ext)
	ih.urlToHash[imgURL] = filename

	// if content was downloaded from a different url don't save a new file
	if _, ok := ih.imageDownloaded[filename]; ok {
		return filename
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloadImage: opening file: %v\n", err)
		return ""
	}
	defer f.Close()

	io.Copy(f, &buf)
	ih.imageDownloaded[filename] = struct{}{}
	return filename
}
