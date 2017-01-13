package main

import (
	"net/http"
	"log"
	"github.com/gorilla/mux"
	"fmt"
	"bufio"
	"github.com/nu7hatch/gouuid"
	"gopkg.in/redis.v5"
	"encoding/json"
)

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB
	})

func check(err error){
	if err != nil {
		panic(err)
		return
	}
}

type Request struct {
	Uuid string `redis:"uuid"`
	Urls  map[string]string `redis:"urls"`
	Status string `redis:"status"`
}

func getUrl(url string, thread chan string) string{
	r, _ := http.Get(url)
	return string(r.StatusCode)

}

func exchanger(id string, c chan string) {
	obj, err := client.Get(id).Result()
	check(err)
	log.Printf("In Exchanger %T", obj)
	t, _ := json.Unmarshal([]byte(obj), )
	log.Printf("In Exchanger %T after unmarshall ", t)
	log.Println(obj)
	//thread := make(chan string)
	//for url := range obj["Urls"] {
	//	go getUrl(url, thread)
	//	obj["Urls"][url] = <-thread
	//}
	//
	//err = client.Set(id, obj, 0).Err()
	//check(err)

}

func finalizer(id string, c chan string) {

}

func processNewRequest(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	defer file.Close()

	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	u, err := uuid.NewV4()
	check(err)
	reqId := u.String()
	newRequest := Request{Uuid: reqId, Urls: make(map[string]string), Status: "Requesting"}
	for _, l := range lines {
		newRequest.Urls[l] = "to call"
	}
	serialRequest, err := json.Marshal(newRequest)

	err = client.Set(reqId, serialRequest, 0).Err()
	check(err)

	// Respond with unique Request Id
	w.Header().Set("req_id", reqId)
	c := make(chan string)

	log.Println("Calling Exchanger")
	go exchanger(reqId, c)
	go finalizer(reqId, c)

}


func processReqId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	result, err := client.Get(vars["reqId"]).Result()
	check(err)
	w.Write([]byte(result))
}

func main() {
	r := mux.NewRouter()
	// Routes for new Request
	r.HandleFunc("/requests", processNewRequest).
	Methods("POST")

	// Route for existing request
	r.HandleFunc("/requests/{reqId}", processReqId).
	Methods("GET")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}