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
	"time"
)

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB
	})

var exhangerQ = make([]string, 100)
var finalizerQ = make([]string, 100)

func check(err error){
	if err != nil {
		panic(err)
		return
	}
}

type Request struct {
	Uuid string
	Urls  map[string]string
	Status string
}

func getUrl(url string, thread chan string) {
	r, e := http.Get(url)
	check(e)
	thread <- string(r.Status)

}

func exchanger(c chan int) {
	log.Println("IN Exchanger")
	for {
		if len(exhangerQ) == 0 {
			log.Println("Exchanger Q is empty -- seepling")
			time.Sleep(2 * time.Second)
		} else {
			log.Println("IN Exchanger processing")
			id := exhangerQ[0]
			log.Println("IN Exchanger processing req: ", exhangerQ[0])
			exhangerQ = exhangerQ[1:]
			log.Println("IN Exchanger processing req: ", exhangerQ)
			obj, err := client.Get(id).Result()
			check(err)

			var req Request
			x := json.Unmarshal([]byte(obj), &req)
			check(x)
			req.Status = "Exchanging"
			serialRequest, err := json.Marshal(req)
			err = client.Set(id, serialRequest, 0).Err()
			check(err)

			thread := make(chan string)
			for url := range req.Urls {
				log.Println("In side loop :", req.Urls[url])
				go getUrl(url, thread)
				req.Urls[url] = <-thread
			}

			req.Status = "Done Exchanging"
			serialRequest, err = json.Marshal(req)
			err = client.Set(id, serialRequest, 0).Err()
			check(err)
			finalizerQ = append(finalizerQ, id)
		}
	}
	c <- 2
}

func finalizer(c chan int) {
	if len(finalizerQ) == 0 {
		time.Sleep(2 * time.Second)
	} else {
		id := finalizerQ[0]
		finalizerQ = finalizerQ[1:]
		obj, err := client.Get(id).Result()
		check(err)

		var req Request
		x := json.Unmarshal([]byte(obj), &req)
		check(x)
		req.Status = "Finalizing in process"
		serialRequest, err := json.Marshal(req)
		err = client.Set(id, serialRequest, 0).Err()
		check(err)

		reverseMap := make(map[string]string, len(req.Urls))
		i := 1
		for url := range req.Urls {
			reverseMap[req.Urls[url] + string(i)] = url
			i++
		}
		req.Urls = reverseMap
		req.Status = "Ready"
		serialRequest, err = json.Marshal(req)
		err = client.Set(id, serialRequest, 0).Err()
		check(err)
	}
	c <- 3
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
	exhangerQ = append(exhangerQ, reqId)
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

func router(c chan int) {
	r := mux.NewRouter()
	// Routes for new Request
	r.HandleFunc("/requests", processNewRequest).
	Methods("POST")

	// Route for existing request
	r.HandleFunc("/requests/{reqId}", processReqId).
	Methods("GET")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
	c <- 1
}

func main() {
	c := make(chan int, 3)
	go router(c)
	go exchanger(c)
	go finalizer(c)
	log.Println(<- c, <- c, <- c)
}