package main

import (
	"os"
	"fmt"
	"net/http"
	"log"
)

var version = "1"
var count = make(map[string]int)

func main() {
	hostname, _ := os.Hostname()
	nodeName := os.Getenv("NODE_NAME")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		greeting := r.URL.Query().Get("greeting")

		// increment the key by one
		num := count[greeting]
		count[greeting] = num + 1

		fmt.Fprintf(w, "Hello, from %s on %q!\nI have seen that greeting %d times.\nVersion: %s\n",
			hostname,
			nodeName,
			num,
			version,
		)
	})
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}



// with redis
//package main
//
//import (
//	"os"
//	"fmt"
//	"net/http"
//	"log"
//	"github.com/go-redis/redis"
//)
//
//var version = "2"
//
//func main() {
//	hostname, _ := os.Hostname()
//	nodeName := os.Getenv("NODE_NAME")
//	client := redis.NewClient(&redis.Options{
//		Addr: "redis:6379",
//	})
//
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprint(w, "OK")
//	})
//	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
//		greeting := r.URL.Query().Get("greeting")
//
//		// increment the key by one
//		num, err := client.Incr(greeting).Result()
//		if err != nil {
//			w.WriteHeader(503)
//			fmt.Fprintf(w, err.Error())
//		}
//
//		fmt.Fprintf(w, "Hello, from %s on %q!\nI have seen that greeting %d times.\nVersion: %s\n",
//			hostname,
//			nodeName,
//			num,
//			version,
//		)
//	})
//	log.Println("Starting server...")
//	log.Fatal(http.ListenAndServe(":8080", nil))
//}
