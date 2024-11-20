package main

import (
	"log"

	"github.com/saiddis/todev/postgres"
)

func main() {
	db := postgres.New("postgres://saiddis:__1dIslo_@localhost:5432/todev?sslmode=disable")
	err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
}
