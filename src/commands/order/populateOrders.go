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

	log.Println("Creating Test Order...")

	for i := 0; i < 30; i++ {
		var orderItems []models.OrderItem

		for j := 0; j < rand.Intn(5); j++ {
			price := float64(rand.Intn(90) + 10)
			qty := uint(rand.Intn(5))

			orderItems = append(orderItems, models.OrderItem{
				ProductTitle:      faker.Word(),
				Price:             price,
				Quantity:          qty,
				AdminRevenue:      0.9 * price * float64(qty),
				AmbassadorRevenue: 0.1 * price * float64(qty),
			})

			database.DB.Create(
				&models.Order{
					UserID:          uint(rand.Intn(38) + 1),
					Code:            faker.Username(),
					AmbassadorEmail: faker.Email(),
					FirstName:       faker.FirstName(),
					LastName:        faker.LastName(),
					Email:           faker.Email(),
					Complete:        true,
					OrderItems:      orderItems,
				})
			log.Println(fmt.Sprintf("Created Test Order %d", i+1))
		}
	}
	log.Println("Finish Test Order!")
}
