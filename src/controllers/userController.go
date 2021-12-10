package controllers

import (
	"admin/src/database"
	"admin/src/models"
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

func Ambassadors(ctx *fiber.Ctx) error {
	var users []models.User

	database.DB.Where("is_ambassador = true").Find(&users)

	return ctx.JSON(users)
}

func Ranking(ctx *fiber.Ctx) error {
	rankings, err := database.Cache.ZRevRangeByScoreWithScores(
		context.Background(),
		"rankings", // updateRankings.goでkeyとして設定した値
		&redis.ZRangeBy{
			Min: "-inf",
			Max: "+inf",
		}).Result()

	if err != nil {
		return err
	}

	result := make(map[string]float64)

	for _, ranking := range rankings {
		result[ranking.Member.(string)] = ranking.Score
	}

	return ctx.JSON(result)
}
