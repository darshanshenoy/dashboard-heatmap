
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"io/ioutil"
	"sync"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

)

// Struct to represent OHLCV data from Binance API
type OHLCVData struct {
	Symbol                   string `json:"symbol"`
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	NumberOfTrades           int    `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

// Struct to represent 24-hour ticker data from Binance API
type TickerData struct {
	Symbol      string `json:"symbol"`
	QuoteVolume string `json:"quoteVolume"` // Trading volume in USDT
}

// Global variable to store OHLCV data for all coins
var ohlcvData []OHLCVData
var mu sync.Mutex

func main() {
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/ohlcv", getOHLCVData).Methods("GET")

	// Add CORS middleware
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"http://104.199.197.142:3000"}, // Allow your frontend origin
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
    })

    // Wrap the router with the CORS middleware
    handler := c.Handler(router)


	// Start the server
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// Handler to fetch OHLCV data for top 50 coins by trading volume from Binance API
func getOHLCVData(w http.ResponseWriter, r *http.Request) {
	// Fetch 24-hour ticker data for all USDT pairs
	tickerData, err := fetch24hTickerData()
	if err != nil {
		http.Error(w, "Failed to fetch 24-hour ticker data", http.StatusInternalServerError)
		return
	}

	// Filter and sort the ticker data to get the top 50 coins by quote volume
	top50Symbols := getTop50SymbolsByQuoteVolume(tickerData)

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	
	// Reset data safely using mutex
	mu.Lock()
	ohlcvData = make([]OHLCVData, 0)
	mu.Unlock()

	// Loop through each trading pair and fetch OHLCV data
	for _, symbol := range top50Symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			fetchOHLCVForSymbol(symbol)
		}(symbol)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Log data before sending response
	mu.Lock()
	log.Printf("Sending %d records to frontend\n", len(ohlcvData))
	for i, data := range ohlcvData {
		log.Printf("Symbol: %s, Open: %s, Close: %s", data.Symbol, data.Open, data.Close)
		if i >= 4 { // Limit log output to first 5 items
			break
		}
	}
	mu.Unlock()

	// Send the OHLCV data as JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ohlcvData)
}

// Fetch 24-hour ticker data for all USDT pairs
func fetch24hTickerData() ([]TickerData, error) {
	url := "https://api.binance.com/api/v3/ticker/24hr"

	response, err := http.Get(url)
	if err != nil {
		log.Println("Failed to fetch ticker data:", err)
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Failed to read ticker data response:", err)
		return nil, err
	}

	var tickerData []TickerData
	err = json.Unmarshal(body, &tickerData)
	if err != nil {
		log.Println("Failed to parse ticker data JSON:", err)
		return nil, err
	}

	// Filter only USDT pairs
	var usdtTickerData []TickerData
	for _, ticker := range tickerData {
		if len(ticker.Symbol) >= 4 && ticker.Symbol[len(ticker.Symbol)-4:] == "USDT" {
			usdtTickerData = append(usdtTickerData, ticker)
		}
	}

	return usdtTickerData, nil
}

// Get the top 50 symbols by quote volume
func getTop50SymbolsByQuoteVolume(tickerData []TickerData) []string {
	// Sort the ticker data by quote volume in descending order
	sort.Slice(tickerData, func(i, j int) bool {
		volumeI, _ := strconv.ParseFloat(tickerData[i].QuoteVolume, 64)
		volumeJ, _ := strconv.ParseFloat(tickerData[j].QuoteVolume, 64)
		return volumeI > volumeJ
	})

	// Extract the top 50 symbols
	var top50Symbols []string
	for i, ticker := range tickerData {
		if i >= 50 {
			break
		}
		top50Symbols = append(top50Symbols, ticker.Symbol)
	}

	return top50Symbols
}

// Fetch OHLCV data for a specific trading pair
func fetchOHLCVForSymbol(symbol string) {
	url := "https://api.binance.com/api/v3/klines?symbol=" + symbol + "&interval=1m"

	response, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch data for symbol %s: %v\n", symbol, err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read response body for symbol %s: %v\n", symbol, err)
		return
	}

	// Log the raw JSON response to debug malformed data
	log.Printf("Raw OHLCV response for %s: %s\n", symbol, string(body))

	var rawData [][]interface{}
	err = json.Unmarshal(body, &rawData)
	if err != nil {
		log.Printf("Failed to parse JSON response for symbol %s: %v\n", symbol, err)
		return
	}

	// Convert the raw data into OHLCVData struct
	for _, data := range rawData {
		// Ensure that data has at least 11 elements
		if len(data) < 11 {
			log.Printf("Skipping invalid OHLCV data for %s: %+v\n", symbol, data)
			continue
		}

		ohlcv := OHLCVData{
			Symbol:                   symbol,
			OpenTime:                 int64(data[0].(float64)),
			Open:                     data[1].(string),
			High:                     data[2].(string),
			Low:                      data[3].(string),
			Close:                    data[4].(string),
			Volume:                   data[5].(string),
			CloseTime:                int64(data[6].(float64)),
			QuoteAssetVolume:         data[7].(string),
			NumberOfTrades:           int(data[8].(float64)),
			TakerBuyBaseAssetVolume:  data[9].(string),
			TakerBuyQuoteAssetVolume: data[10].(string),
		}

		// Use a mutex to safely append data to the global slice
		mu.Lock()
		ohlcvData = append(ohlcvData, ohlcv)
		mu.Unlock()
	}
}
