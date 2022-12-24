package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/net/html"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Listing represents a product for auction/sale on shopgoodwill.com
type Listing struct {
	Href        string
	Class       string
	ProductNo   string
	Description string
	Bids        string
	Price       string
	Thumb       string
	Ends        string
}

func main() {
	http.HandleFunc("/", scrape)
	fmt.Println("Starting the server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func scrape(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	sa := option.WithCredentialsFile("/home/brasey/.gcp/shopgoodwill-scraper-3ef981e163aa.json")
	client, err := firestore.NewClient(ctx, "shopgoodwill-scraper", sa)
	//client, err := firestore.NewClient(ctx, "shopgoodwill-scraper")
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	search := client.Doc("config/search")
	doc, err := search.Get(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	var searchRead map[string][]string
	var terms []string
	if err = doc.DataTo(&searchRead); err != nil {
		log.Fatalln(err)
	}
	for _, t := range searchRead["terms"] {
		// I think this is where the double term is happening
		terms = append(terms, t)
	}

	var listings []Listing

	for _, term := range terms {
		term := url.PathEscape(term)
		t := time.Now()
		date := fmt.Sprintf("%d/%d/%d", t.Month(), t.Day(), t.Year())
		url := "https://shopgoodwill.com/categories/listing?st=" + term + "&sg=&c=27&s=&lp=0&hp=999999&sbn=1&spo=false&snpo=true&socs=false&sd=false&sca=false&caed=" + date + "&cadb=7&scs=false&sis=false&col=1&p=1&ps=40&desc=false&ss=0&UseBuyerPrefs=true&sus=true&cln=2&catIds=10,27&pn=&wc=false&mci=false&hmt=false&layout=list"

		resp, err := http.Get(url)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()

		theseListings, err := parse(resp.Body)
		if err != nil {
			log.Println(err)
		}
		listings = append(listings, theseListings...)
	}

	err = deleteCollection(ctx, client, client.Collection("listings"), 100)
	if err != nil {
		log.Println(err)
	}

	err = writeListings(ctx, client, client.Collection("listings"), listings, 100)
	if err != nil {
		log.Println(err)
	}
}

// parse takes in an HTML document and returns a slice
// of listings parsed from it.
func parse(r io.Reader) ([]Listing, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	linkNodes := collectNodes(doc)
	var listings []Listing
	for _, n := range linkNodes {
		l := buildListing(n)
		if (Listing{}) != l {
			listings = append(listings, buildListing(n))
		}
	}
	return listings, nil
}

// collectNodes takes in an *html.Node and returns a
// slice of all *html.Nodes under the called node.
func collectNodes(n *html.Node) []*html.Node {
	var ret []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret = append(ret, c)
		ret = append(ret, collectNodes(c)...)
	}
	return ret
}

// buildListing takes in an <a> *html.Node and returns
// a fully-populated Listing type
func buildListing(n *html.Node) Listing {
	var ret, tmp Listing
	for _, attr := range n.Attr {
		if attr.Key == "href" {
			tmp.Href = attr.Val
		} else if attr.Key == "class" {
			tmp.Class = attr.Val
		}
	}
	if tmp.Class != "product" {
		return ret
	}

	childNodes := collectNodes(n)
	for _, c := range childNodes {
		if c.Type == html.ElementNode && c.Data == "img" {
			tmp.Thumb = thumb(c)
		} else if c.Type == html.ElementNode && c.Data == "div" {
			var class, string string
			for _, attr := range c.Attr {
				switch attr.Key {
				case "class":
					class = attr.Val
				case "data-countdown":
					string = attr.Val
				}
			}
			if class == "timer countdown product-countdown" {
				tmp.Ends = string
			}
		}
	}

	t := text(n)
	tmp.ProductNo, tmp.Description, tmp.Bids, tmp.Price = extractText(t)
	ret = tmp
	return ret
}

// thumb takes in an <a> *html.Node and returns a thumbnail
// URL string
func thumb(n *html.Node) string {
	for _, attr := range n.Attr {
		if attr.Key == "src" {
			return attr.Val
		}
	}
	return ""
}

// text takes in an <a> *html.Node and returns a string of
// all the text contained in the <a> link, concatenated
func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type != html.ElementNode {
		return ""
	}
	var ret string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ret += text(c) + " "
	}
	return strings.Join(strings.Fields(ret), " ")
}

// extractText takes in a string of text and returns strings
// of product number, description, number of bids and item price
func extractText(t string) (string, string, string, string) {
	var productNo, desc, bids, price string
	// Remove unwanted text from text
	r, _ := regexp.Compile("(?:Place a Bid)|(?:Watch)|(?:Buy [I,i]t Now)")
	t = r.ReplaceAllString(t, "")
	r, _ = regexp.Compile("[[:space:]]+$")
	t = r.ReplaceAllString(t, "")

	// Extract Product Number
	r, _ = regexp.Compile("^.*Product #: ([[:digit:]]+) ")
	n := r.FindStringSubmatch(t)
	if len(n) != 0 {
		productNo = n[1]
	}
	t = r.ReplaceAllString(t, "")

	// Extract Bids
	r, _ = regexp.Compile(" Bids: ([[:digit:]]+)")
	b := r.FindStringSubmatch(t)
	if len(b) != 0 {
		bids = b[1]
	} else {
		bids = "Buy it now"
	}
	t = r.ReplaceAllString(t, "")

	// Extract Price
	r, _ = regexp.Compile(" (\\$[[:digit:]]+\\.[[:digit:]]{2})$")
	p := r.FindStringSubmatch(t)
	if len(p) != 0 {
		price = p[1]
	} else {
		price = "$0.00"
	}
	t = r.ReplaceAllString(t, "")

	desc = t
	return productNo, desc, bids, price
}

// deleteCollection deletes a Firestore collection safely.
// It takes in context, client, collectionRef and batch size
// and returns error.
func deleteCollection(ctx context.Context, client *firestore.Client,
	ref *firestore.CollectionRef, batchSize int) error {

	for {
		// Get a batch of documents
		iter := ref.Limit(batchSize).Documents(ctx)
		numDeleted := 0

		// Iterate through the documents, adding
		// a delete operation for each one to a
		// WriteBatch.
		batch := client.Batch()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}

			batch.Delete(doc.Ref)
			numDeleted++
		}

		// If there are no documents to delete,
		// the process is over.
		if numDeleted == 0 {
			return nil
		}

		_, err := batch.Commit(ctx)
		if err != nil {
			return err
		}
	}
}

// writeListings writes a []Listing to Firestore in batches.
// It takes in context, client, collectionRef, []Listing and
// batch size and returns error.
func writeListings(ctx context.Context, client *firestore.Client,
	ref *firestore.CollectionRef, listings []Listing, batchSize int) error {

	c := 1
	batch := client.Batch()
	for i, l := range listings {
		name := fmt.Sprintf("%6d", i)
		batch.Create(ref.Doc(name), l)
		switch {
		case i == len(listings)-1:
			_, err := batch.Commit(ctx)
			if err != nil {
				return err
			}
		case c < batchSize:
			c++
		case c == batchSize:
			_, err := batch.Commit(ctx)
			batch = client.Batch()
			if err != nil {
				return err
			}
			c = 1
		}
	}
	return nil
}
