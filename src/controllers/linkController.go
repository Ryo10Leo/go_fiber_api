package controllers

import (
	"admin/src/database"
	"admin/src/middleware"
	"admin/src/models"
	"strconv"

	"github.com/bxcodec/faker/v3"
	"github.com/gofiber/fiber/v2"
)

func Link(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	var links []models.Link

	database.DB.Where("user_id = ?", id).Find(&links)
	// 追加
	for i, link := range links {
		var orders []models.Order
		database.DB.Where("code = ? and complete = true", link.Code).Find(&orders)

		links[i].Orders = orders
	}

	return ctx.JSON(links)
}

type CreateLinkRequest struct {
	ProductIDs []int
}

func CreateLink(ctx *fiber.Ctx) error {
	var request CreateLinkRequest

	// リクエストデータからProductIDを取得
	if err := ctx.BodyParser(&request); err != nil {
		return err
	}

	// ユーザーIDを取得
	id, _ := middleware.GetUserID(ctx)

	// リンク作成
	link := models.Link{
		UserID: id,
		Code:   faker.Username(),
	}

	// プロダクトID, ユーザーIDを紐づける
	for _, productID := range request.ProductIDs {
		product := models.Product{}
		product.ID = uint(productID)
		link.Products = append(link.Products, product)
	}

	// DB保存
	database.DB.Create(&link)

	return ctx.JSON(link)
}

func Stats(ctx *fiber.Ctx) error {
	// ユーザーIDを取得
	id, _ := middleware.GetUserID(ctx)

	// DB検索
	var links []models.Link
	database.DB.Find(&links, models.Link{
		UserID: id,
	})

	var result []interface{}
	var orders []models.Order

	for _, link := range links {
		// OrderItemsを先にロードしてからOrderデータを検索
		database.DB.Preload("OrderItems").Find(&orders, &models.Order{
			Code:     link.Code,
			Complete: true,
		})

		var revenue float64 = 0
		for _, order := range orders {
			revenue += order.GetTotal()
		}

		result = append(result, fiber.Map{
			"code":    link.Code,
			"count":   len(orders),
			"revenue": revenue,
		})
	}

	return ctx.JSON(result)
}

func GetLink(ctx *fiber.Ctx) error {
	// URL(links/:code)からcodeを取得する
	code := ctx.Params("code")

	link := models.Link{
		Code: code,
	}

	// データベース検索
	// linkデータを取得する前にUser, Productsデータを取得する
	database.DB.Preload("User").Preload("Products").First(&link)

	return ctx.JSON(link)
}

type CreateOrderRequest struct {
	Code      string
	FirstName string
	LastName  string
	Email     string
	Address   string
	Country   string
	City      string
	Zip       string
	Products  []map[string]int
}

func CreateOrder(ctx *fiber.Ctx) error {
	var request CreateOrderRequest

	// リクエストデータを取得
	if err := ctx.BodyParser(&request); err != nil {
		return err
	}

	// リクエストからコードを抜き出す
	link := models.Link{
		Code: request.Code,
	}

	// DB検索
	database.DB.Preload("User").First(&link)

	// 該当データがない場合はエラー
	if link.ID == 0 {
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": "無効なリンクです",
		})
	}

	// Orderを作成する
	order := models.Order{
		Code:            link.Code,
		UserID:          link.UserID,
		AmbassadorEmail: link.User.Email,
		FirstName:       request.FirstName,
		LastName:        request.LastName,
		Email:           request.Email,
		Address:         request.Address,
		Country:         request.Country,
		City:            request.City,
		Zip:             request.Zip,
	}

	// OrderをDBに保存
	database.DB.Create(&order)

	// トランザクション
	tx := database.DB.Begin()
	// OrderをDBに保存
	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// リクエストからプロダクトを取得
	for _, requestProduct := range request.Products {
		product := models.Product{}
		product.ID = uint(requestProduct["product_id"])

		// product検索
		database.DB.First(&product)

		// トータルを算出
		total := product.Price * float64(requestProduct["quantity"])

		// OrderItemを作成
		item := models.OrderItem{
			OrderID:           order.ID,
			ProductTitle:      product.Title,
			Price:             product.Price,
			Quantity:          uint(requestProduct["quantity"]),
			AmbassadorRevenue: 0.1 * total,
			AdminRevenue:      0.9 * total,
		}

		// OrderItemをDBに保存
		database.DB.Create(&item)
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			ctx.Status(fiber.StatusBadRequest)
			return ctx.JSON(fiber.Map{
				"message": err.Error(),
			})
		}
	}

	// 実行
	tx.Commit()

	return ctx.JSON(order)
}
