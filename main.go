// main.go
package main

import (
	"os"
	"fmt"
	"net/http"
	"log"
)

var version = "1"

func main() {
	hostname, _ := os.Hostname()
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, from %s! My version is %s, our config is %q and our secret is %q",
			hostname,
			version,
			os.Getenv("CONFIG"),
			os.Getenv("SECRET"),
		)
	})
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
