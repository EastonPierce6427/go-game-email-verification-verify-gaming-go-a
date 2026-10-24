package main

import (
	"log"
	"net/http"
)

func main() {
	sender, err := NewEmailSender()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/signup", (SignupService{Sender: sender}).Signup)
	log.Println("game signup listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
