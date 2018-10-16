package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rollbar/rollbar-go"
	"log"
	"net/http"
	"os"
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
	// This person context will override the global value set with SetPerson in main for the
	// specific call that it is sent with.
	ctx := rollbar.NewPersonContext(context.TODO(), &rollbar.Person{Id: "42", Username: "Frank", Email: ""})
	rollbar.Info(r, "Example message json", ctx)
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
	// Without a context this will include the person from the global SetPerson call
	rollbar.Info(r, "Example message form")
	fmt.Fprintf(w, "Hello world!")
}

// In one terminal: TOKEN=POST_SERVER_ITEM_ACCESS_TOKEN go run example/main.go
// In another:
//    curl -X POST -H "Content-Type: application/x-www-form-urlencoded" \
//      http://localhost:9090/form -d "password=foobar&fuzz=buzz"
// Or:
//    curl -X POST -H "Content-Type: application/json" \
//      http://localhost:9090/json -d '{"password":"foobar","fuzz":"buzz"}'
func main() {
	var token = os.Getenv("TOKEN")
	rollbar.SetToken(token)
	rollbar.SetEnvironment("test")
	rollbar.SetCaptureIp(rollbar.CaptureIpAnonymize)
	rollbar.SetPerson("88", "Steve", "")
	http.HandleFunc("/json", helloJson)
	http.HandleFunc("/form", helloForm)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	rollbar.Wait()
}
