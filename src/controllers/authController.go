package controllers

import (
	"admin/src/database"
	"admin/src/middleware"
	"admin/src/models"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Register(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	if data["password"] != data["password_confirm"] {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "パスワードに誤りがあります。",
		})
	}

	user := models.User{
		FirstName:    data["first_name"],
		LastName:     data["last_name"],
		Email:        data["email"],
		IsAmbassador: strings.Contains(ctx.Path(), "/api/ambassador"),
	}

	// パスワードセット
	user.SetPassword(data["password"])

	result := database.DB.Create(&user)
	if result.Error != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "そのEmailは既に登録されています",
		})
	}

	return ctx.JSON(user)
}

func Login(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	var user models.User
	database.DB.Where("email = ?", data["email"]).First(&user)

	if user.ID == 0 {
		ctx.Status((fiber.StatusBadRequest))
		return ctx.JSON(fiber.Map{
			"message": "ユーザーは見つかりませんでした。",
		})
	}

	err := user.ComparePassword(data["password"])
	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "パスワードに誤りがあります。",
		})
	}

	isAmbassador := strings.Contains(ctx.Path(), "/api/ambassador")

	var scope string

	if isAmbassador {
		scope = "ambassador"
	} else {
		scope = "admin"
	}

	if !isAmbassador && user.IsAmbassador {
		ctx.Status(fiber.StatusUnauthorized) // 401
		return ctx.JSON(fiber.Map{
			"message": "認証が許可されていません",
		})
	}

	token, err := middleware.GenerateJWT(user.ID, scope)

	if err != nil {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "ログイン情報に誤りがあります。",
		})
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}

	ctx.Cookie(&cookie)

	return ctx.JSON(token)
}

func Logout(ctx *fiber.Ctx) error {
	log.Println("Do Logout")
	// cookieをクリアする
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour * 24),
		HTTPOnly: true,
	}

	ctx.Cookie(&cookie)
	return ctx.JSON(fiber.Map{
		"message": "success",
	})
}

func User(ctx *fiber.Ctx) error {
	log.Println("Do User")
	id, _ := middleware.GetUserID(ctx)

	// ユーザー検索
	var user models.User
	database.DB.Where("id = ?", id).First(&user)

	if strings.Contains(ctx.Path(), "/api/ambassador") {
		ambassador := models.Ambassador(user)
		ambassador.CalculateRevenue(database.DB)
		return ctx.JSON(ambassador)
	}

	return ctx.JSON(user)
}

func UpdateInfo(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	id, _ := middleware.GetUserID(ctx)
	user := models.User{
		FirstName: data["first_name"],
		LastName:  data["last_name"],
		Email:     data["email"],
	}

	user.ID = id

	database.DB.Model(&user).Updates(&user)
	return ctx.JSON(user)
}

func UpdatePassword(ctx *fiber.Ctx) error {
	var data map[string]string

	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	if data["password"] != data["password_confirm"] {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "パスワードに誤りがあります。",
		})
	}

	id, _ := middleware.GetUserID(ctx)
	user := models.User{}
	user.ID = id

	user.SetPassword(data["password"])

	database.DB.Model(&user).Updates(&user)
	return ctx.JSON(user)
}
