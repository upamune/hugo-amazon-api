package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/upamune/amazing"
)

type service struct {
	client   *amazing.Amazing
	cacheDir string
}

func main() {
	var port, cacheDir string
	var awsAccess, awsSecret, awsTag, awsDomain string

	flag.StringVar(&awsAccess, "access", "", "aws access id")
	flag.StringVar(&awsSecret, "secret", "", "aws secretkey")
	flag.StringVar(&awsTag, "tag", "", "amazon tag")
	flag.StringVar(&awsDomain, "domain", "JP", "amazon domain")
	flag.StringVar(&port, "port", "8080", "port number")
	flag.StringVar(&cacheDir, "cache-dir", "", "for json cache")
	flag.Parse()

	client, err := amazing.NewAmazing(awsDomain, awsTag, awsAccess, awsSecret)
	if err != nil {
		log.Fatal(err)
	}

	service := &service{
		client:   client,
		cacheDir: cacheDir,
	}
	http.HandleFunc("/hc", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/", service.amazonHandler)
	log.Printf("Starting Server at %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

type Item struct {
	ASIN         string
	Brand        string
	Creator      string
	Manufacturer string
	Publisher    string
	ReleaseDate  string
	Studio       string
	Title        string
	URL          string

	SmallImage  string
	MediumImage string
	LargeImage  string
}

var ErrNotFoundFile = errors.New("file not found")

func (s *service) getItemFromCache(itemID string) ([]byte, error) {
	filename := s.getFileName(itemID)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, ErrNotFoundFile
	}

	return b, nil
}

func (s *service) getFileName(itemID string) string {
	return fmt.Sprintf("%s/%s", s.cacheDir, itemID)
}

func (s *service) saveItemToCache(item *Item) error {
	f, err := os.OpenFile(s.getFileName(item.ASIN), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(item)
}

func (s *service) amazonHandler(w http.ResponseWriter, req *http.Request) {
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

	// キャッシュされていたらキャッシュを返す
	if s.cacheDir != "" {
		json, err := s.getItemFromCache(itemID)
		if err != nil {
			if err != ErrNotFoundFile {
				log.Printf("failed get a cache: %s", err)
			}
		} else {
			// キャッシュが見つかった時
			w.Header().Set("Content-Type", "application/json")
			w.Write(json)
			log.Printf("hit cache: %s", itemID)
			return
		}
	}

	params := url.Values{
		"IdType":        []string{"ASIN"},
		"ItemId":        []string{itemID},
		"Operation":     []string{"ItemLookup"},
		"ResponseGroup": []string{"Large"},
	}
	res, err := s.client.ItemLookup(params)
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

	// キャッシュする
	if s.cacheDir != "" {
		if err := s.saveItemToCache(item); err != nil {
			log.Printf("failed to save a cache: %v", err)
		}
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
		ASIN:         aitem.ASIN,
		Brand:        aitem.ItemAttributes.Brand,
		Creator:      aitem.ItemAttributes.Creator,
		Manufacturer: aitem.ItemAttributes.Manufacturer,
		Publisher:    aitem.ItemAttributes.Publisher,
		ReleaseDate:  aitem.ItemAttributes.ReleaseDate,
		Studio:       aitem.ItemAttributes.Studio,
		Title:        aitem.ItemAttributes.Title,
		URL:          aitem.DetailPageURL,
		SmallImage:   aitem.SmallImage.URL,
		MediumImage:  aitem.MediumImage.URL,
		LargeImage:   aitem.LargeImage.URL,
	}

	return item, nil
}
