//
// usage:
//   e.g) echo "{}" | SECRET_TOKEN=$(cat test/secret_token.txt) go run client.go
//
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func PostRequest() {
	secret := os.Getenv("SECRET_TOKEN")
	if len(secret) == 0 {
		log.Fatalf("SECRET_TOKEN is empty")
	}

	//log.Printf("secret: %v\n", secret)
	key := []byte(secret)
	body, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("%v", err)
	}
	//log.Printf("body: %v", body)

	mac := hmac.New(sha1.New, key)
	mac.Write(body)
	actual := mac.Sum(nil)

	sign := fmt.Sprintf("sha1=%s", hex.EncodeToString(actual))

	req, err := http.NewRequest("POST", "http://localhost:9981/serf/query/hoko?webhook=github", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("%v", err)
	}
	req.Header.Add("x-hub-signature", sign)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("%v", err)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("%v\n", res.Status)
	fmt.Printf("%v\n", string(b))
}
