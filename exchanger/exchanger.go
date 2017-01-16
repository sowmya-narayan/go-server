package main

import (
	"log"
	"time"
	"encoding/json"
	"gopkg.in/redis.v3"
	"net/http"
)


var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB
	MaxRetries: 2,
})

type Request struct {
	Uuid string
	Urls  map[string]string
	Status string
}

func check(err error){
	if err != nil {
		panic(err)
	}
}

func getUrl(url string, thread chan string) {
	r, e := http.Get(url)
	check(e)
	thread <- string(r.Status)

}

func getExchangerQ() []string{
	q, err := client.Get("exchangerQ").Result()
	if q == "" || err != nil {
		var newQ = make([]string, 100)
		return newQ
	}
	var obj []string
	json.Unmarshal([]byte(q), &obj)
	return obj
}

func setExchangerQ(q []string) {
	q = q[1:]
	serializeExchangeQ, err := json.Marshal(q)
	err = client.Set("exchangerQ", serializeExchangeQ, 0).Err()
	check(err)
}



func appendFinalizerQ(id string) {
	q, err := client.Get("finalizerQ").Result()
	check(err)
	var finalizerQ []string
	if q == "" || err != nil {
		finalizerQ = make([]string, 100)
	} else {
		var finalizerQ []string
		json.Unmarshal([]byte(q), &finalizerQ)
	}
	finalizerQ = append(finalizerQ, id)
	serializeFinalizerQ, err := json.Marshal(finalizerQ)
	err = client.Set("finalizerQ", serializeFinalizerQ, 0).Err()
	check(err)
}

func main() {

	for {
		exchangerQ := getExchangerQ()

		if len(exchangerQ) == 0 || exchangerQ[0] == ""{
			time.Sleep(3 * time.Second)
		} else {
			log.Printf("IN Exchanger processing %v", exchangerQ)
			id := exchangerQ[0]
			log.Printf("IN Exchanger processing req: %v", exchangerQ[0])

			setExchangerQ(exchangerQ)

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
			appendFinalizerQ(id)
		}
	}
}