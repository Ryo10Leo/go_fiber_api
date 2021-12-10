package main

import (
	"admin/src/database"
	"admin/src/models"
	"fmt"
	"log"
	"math/rand"

	"github.com/bxcodec/faker/v3"
)

func main() {
	// DB接続
	database.Connect()

	log.Println("Creating Test Product...")

	for i := 0; i < 30; i++ {
		product := models.Product{
			Title:       faker.Username(),
			Description: faker.Username(),
			Image:       faker.URL(),
			Price:       float64(rand.Intn(90) + 10),
		}

		database.DB.Create(&product)
		log.Println(fmt.Sprintf("Created Test Product %d", i+1))
	}

	log.Println("Finish Test Product!")
}
