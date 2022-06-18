package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Simple static webserver:
	fmt.Print("starting...")
	log.Fatal(http.ListenAndServe(":8080", http.FileServer(http.Dir("E:\\Roy\\Projects\\Acronis Assignment\\"))))
}
