package main

import (
	"admin/src/database"
	"admin/src/models"
	"fmt"
	"log"

	"github.com/bxcodec/faker/v3"
)

func main() {
	// DB接続
	database.Connect()

	log.Println("Creating Test User...")

	for i := 0; i < 30; i++ {
		ambassador := models.User{
			FirstName:    faker.FirstName(),
			LastName:     faker.LastName(),
			Email:        faker.Email(),
			IsAmbassador: true,
		}

		ambassador.SetPassword("1234")
		database.DB.Create(&ambassador)
		log.Println(fmt.Sprintf("Created Test User %d", i+1))
	}

	log.Println("Finish Test User!")
}
