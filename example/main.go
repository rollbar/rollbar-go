package main

import (
  "os"
	"encoding/json"
	"fmt"
	"github.com/rollbar/rollbar-go"
	"log"
	"net/http"
	"strings"
)

func helloJson(w http.ResponseWriter, r *http.Request) {
	var u map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	for k, v := range u {
		fmt.Println("key:", k)
		fmt.Println("val:", v)
	}
	rollbar.RequestMessage(rollbar.INFO, r, "Example message json")
	fmt.Fprintf(w, "Hello world!")
}

func helloForm(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, " "))
	}
	rollbar.RequestMessage(rollbar.INFO, r, "Example message form")
	fmt.Fprintf(w, "Hello world!")
}

// In one terminal: TOKEN=POST_SERVER_ITEM_ACCESS_TOKEN go run example/main.go
// In another:
//    curl -X POST -H "Content-Type: application/x-www-form-urlencoded" \
//      http://localhost:9090/form -d "password=foobar&fuzz=buzz"
func main() {
  var token = os.Getenv("TOKEN")
	rollbar.SetToken(token)
	rollbar.SetEnvironment("test")
	http.HandleFunc("/json", helloJson)
	http.HandleFunc("/form", helloForm)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	rollbar.Wait()
}
