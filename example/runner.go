package main

import (
	"flag"
	"fmt"
	"github.com/soh335/airshow"
	"github.com/soh335/airshow/tumblr"
)

func main() {
	host := flag.String("host", "aoi-miyazaki.tumblr.com", "host name")
	apikey := flag.String("apikey", "", "api key")
	limit := flag.Int("limit", 100, "photo limit")
	flag.Parse()

	if *apikey == "" {
		fmt.Println("require apikey")
		return
	}

	a := airshow.New()
	a.AddWorker(tumblr.NewTumblrClient(*host, *apikey, *limit))
	a.Run()
}
