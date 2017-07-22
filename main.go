package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/dominicphillips/amazing"
)

var client *amazing.Amazing

func main() {
	var port string
	var awsAccess, awsSecret, awsTag, awsDomain string

	flag.StringVar(&awsAccess, "access", "", "aws access id")
	flag.StringVar(&awsSecret, "secret", "", "aws secretkey")
	flag.StringVar(&awsTag, "tag", "", "amazon tag")
	flag.StringVar(&awsDomain, "domain", "JP", "amazon domain")
	flag.StringVar(&port, "port", "8080", "port number")
	flag.Parse()

	var err error
	client, err = amazing.NewAmazing("JP", awsTag, awsAccess, awsSecret)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", amazonHandler)
	log.Printf("Starting Server at %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

type Item struct {
	ASIN        string
	Title       string
	Brand       string
	URL         string
	SmallImage  string
	MediumImage string
	LargeImage  string
}

func amazonHandler(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("invalid form: %v", err)))
		return
	}

	itemID := req.FormValue("item_id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("invalid item id: %s", itemID)))
		return
	}
	params := url.Values{
		"IdType":        []string{"ASIN"},
		"ItemId":        []string{itemID},
		"Operation":     []string{"ItemLookup"},
		"ResponseGroup": []string{"Medium"},
	}
	res, err := client.ItemLookup(params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to get item infomation: %v", err)))
		return
	}

	item, err := resToItem(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to get item from response: %v", err)))
		return
	}

	b, err := json.Marshal(item)
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed to marshal item to json: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func resToItem(res *amazing.AmazonItemLookupResponse) (*Item, error) {
	items := res.AmazonItems.Items
	if len(items) == 0 {
		return nil, errors.New("empty amazon items")
	}

	aitem := items[0]

	item := &Item{
		ASIN:        aitem.ASIN,
		Title:       aitem.ItemAttributes.Title,
		Brand:       aitem.ItemAttributes.Brand,
		URL:         aitem.DetailPageURL,
		SmallImage:  aitem.SmallImage.URL,
		MediumImage: aitem.MediumImage.URL,
		LargeImage:  aitem.LargeImage.URL,
	}

	return item, nil
}
