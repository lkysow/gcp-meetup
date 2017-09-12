// main.go
package main

import (
	"os"
	"fmt"
	"net/http"
	"log"
)

func main() {
	hostname, _ := os.Hostname()
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, from %s! Our config is %q and our secret is %q\n",
			hostname,
			os.Getenv("CONFIG"),
			os.Getenv("SECRET"),
		)
	})
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
