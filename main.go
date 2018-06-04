package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

const (
	cmcapURL = "https://api.coinmarketcap.com/v2/ticker/?sort=percent_change_24h&structure=array"
)

// CoinMarketCap is a struct used to work with the results of coinmarketcap ticker request
type CoinMarketCap struct {
	Data []struct {
		ID                int         `json:"id"`
		Name              string      `json:"name"`
		Symbol            string      `json:"symbol"`
		WebsiteSlug       string      `json:"website_slug"`
		Rank              int         `json:"rank"`
		CirculatingSupply interface{} `json:"circulating_supply"`
		TotalSupply       float64     `json:"total_supply"`
		MaxSupply         interface{} `json:"max_supply"`
		Quotes            struct {
			USD struct {
				Price            float64     `json:"price"`
				Volume24H        float64     `json:"volume_24h"`
				MarketCap        interface{} `json:"market_cap"`
				PercentChange1H  float64     `json:"percent_change_1h"`
				PercentChange24H float64     `json:"percent_change_24h"`
				PercentChange7D  float64     `json:"percent_change_7d"`
			} `json:"USD"`
		} `json:"quotes"`
		LastUpdated int `json:"last_updated"`
	} `json:"data"`
	Metadata struct {
		Timestamp           int         `json:"timestamp"`
		NumCryptocurrencies int         `json:"num_cryptocurrencies"`
		Error               interface{} `json:"error"`
	} `json:"metadata"`
}

func makeRequest(req *http.Request, client http.Client) []byte {
	fmt.Printf("Making request to: %s\n", req.URL)
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Could not make request to %s. Error: %s\n", req.URL, err)
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Could not read body. %s\n", err)
	}
	return data
}

func getCoinMarketCapResults(client http.Client, out chan<- CoinMarketCap) {
	fmt.Printf("Enter getCoinMarketCapResults\n")
	req, err := http.NewRequest("GET", cmcapURL, nil)
	if err != nil {
		log.Fatalf("Could not create NewRequest %s\n", err)
	}
	var cmcap CoinMarketCap
	data := makeRequest(req, client)
	err = json.Unmarshal(data, &cmcap)
	if err != nil {
		log.Fatalf("Could not Unmarshall data. %s\n", err)
	}

	out <- cmcap
}

func tweet(twttClient *twitter.Client, in <-chan CoinMarketCap) {
	// get result of go routine: getCoinMarketCapResults.
	cmcapData := <-in

	var twtt string
	for i, coin := range cmcapData.Data[:5] {
		twtt += fmt.Sprintf("%d: %s(%s) - Change in 24h: %g%%\n",
			i+1, coin.Name, coin.Symbol, coin.Quotes.USD.PercentChange24H)
	}

	tweet, _, err := twttClient.Statuses.Update(twtt, nil)
	if err != nil {
		fmt.Printf("Could not tweet :( Error: %s\n", err)
	}
	fmt.Printf("Tweeted: %s", tweet.Text)

}

func startApp(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Starting bot...")
	// Create httpClient
	httpClient := http.Client{}

	config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))

	// oauth1 http.Client will automatically authorize Requests
	oauthHTTPClient := config.Client(oauth1.NoContext, token)

	// twitter client
	twttClient := twitter.NewClient(oauthHTTPClient)

	// create channel to receive the results from Coin Market Cap api call
	cmcapResChan := make(chan CoinMarketCap)

	// call go routines getCoinMarketResults and twitter
	for {
		go getCoinMarketCapResults(httpClient, cmcapResChan)
		go tweet(twttClient, cmcapResChan)
		time.Sleep(5 * time.Minute)
	}
}

func main() {
	port := os.Getenv("PORT")
	http.HandleFunc("/", startApp)
	http.ListenAndServe(":"+port, nil)
}
