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
	}
}

type Request struct {
	Uuid string
	Urls  map[string]string
	Status string
}

func appendExchangerQ(reqId string) {
	q, err := client.Get("exchangerQ").Result()
	check(err)
	var exchangerQ []string
	if q == "" || err != nil {
		exchangerQ = make([]string, 100)
	} else {
		var exchangerQ []string
		json.Unmarshal([]byte(q), &exchangerQ)
	}
	exchangerQ = append(exchangerQ, reqId)
	serializeExchangerQ, err := json.Marshal(exchangerQ)
	err = client.Set("exchangerQ", serializeExchangerQ, 0).Err()
	check(err)
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
	appendExchangerQ(reqId)
	// Respond with unique Request Id
	w.Header().Set("req_id", reqId)


}


func processReqId(w http.ResponseWriter, r *http.Request) {
	log.Println("In Track Req")
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