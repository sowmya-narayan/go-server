package main

import (
	"encoding/json"
	"time"
	"gopkg.in/redis.v5"
	"log"
)

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB

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

func getFinalizerQ() []string{
	q, err := client.Get("finalizerQ").Result()
	//check(err)
	if q == "" || err != nil {
		var newQ = make([]string, 100)
		return newQ
	}
	var obj []string
	json.Unmarshal([]byte(q), &obj)
	return obj
}

func setFinalizerQ(q []string) {
	q = q[1:]
	serializeExchangeQ, err := json.Marshal(q)
	err = client.Set("finalizerQ", serializeExchangeQ, 0).Err()
	check(err)
}


func main() {
	for {
		finalizerQ := getFinalizerQ()
		if len(finalizerQ) == 0 || finalizerQ[0] == ""{
			time.Sleep(5 * time.Second)
		} else {
			log.Println("In Finalizer Processing")
			id := finalizerQ[0]
			setFinalizerQ(finalizerQ)
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
	}
}
