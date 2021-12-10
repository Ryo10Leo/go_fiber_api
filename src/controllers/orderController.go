package controllers

import (
	"admin/src/database"
	"admin/src/internal"
	"admin/src/models"
	"context"
	"fmt"
	"log"
	"net/smtp"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/checkout/session"
)

func Orders(ctx *fiber.Ctx) error {
	var orders []models.Order

	// DB検索
	database.DB.Preload("OrderItems").Find(&orders)

	for i, order := range orders {
		orders[i].Name = order.FullName()
		orders[i].Total = order.GetTotal()
	}

	return ctx.JSON(orders)
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

	// stripeパラメーター
	var lineItems []*stripe.CheckoutSessionLineItemParams

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

		// トランザクション
		// OrderItemをDBに保存
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			ctx.Status(fiber.StatusBadRequest)
			return ctx.JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		// stripeアイテムセット
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Name:        stripe.String(product.Title),
			Description: stripe.String(product.Description),
			Images:      []*string{stripe.String(product.Image)},
			Amount:      stripe.Int64(100 * int64(product.Price)),
			Currency:    stripe.String("usd"),
			Quantity:    stripe.Int64(int64(requestProduct["quantity"])),
		})
	}

	// stripe checkout
	stripe.Key = internal.StripeSecretKey()
	log.Println(stripe.Key)
	params := stripe.CheckoutSessionParams{
		// http://localhost:5000はフロントエンド側のリンク(まだ作成していない)
		SuccessURL:         stripe.String("http://localhost:5000/success?source={CHECKOUT_SESSION_ID}"),
		CancelURL:          stripe.String("http://localhost:5000/error"),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems:          lineItems,
	}

	source, err := session.New(&params)
	if err != nil {
		tx.Rollback()
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// トランザクションIDを登録
	order.TransactionID = source.ID
	// データを書き換えたので、上書き保存
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		ctx.Status(fiber.StatusBadRequest)
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// 実行
	tx.Commit()

	return ctx.JSON(source)
}

func CompleteOrder(ctx *fiber.Ctx) error {
	var data map[string]string

	// リクエストデータを取得
	if err := ctx.BodyParser(&data); err != nil {
		return err
	}

	order := models.Order{}
	// OrderItemを検索
	database.DB.Preload("OrderItems").First(&order, models.Order{
		TransactionID: data["source"],
	})

	if order.ID == 0 {
		ctx.Status((fiber.StatusNotFound))
		return ctx.JSON(fiber.Map{
			"message": "オーダーが見つかりません",
		})
	}

	// Orderを保存
	order.Complete = true
	database.DB.Save(&order)

	// Redisキャッシュに保存したRankingも更新
	// Rankingはユーザーの購入金額を示す
	go func(order models.Order) {
		ambassadorRevenue := 0.0
		adminRevenue := 0.0

		for _, item := range order.OrderItems {
			ambassadorRevenue += item.AmbassadorRevenue
			adminRevenue += item.AdminRevenue
		}

		user := models.User{}
		user.ID = order.UserID

		database.DB.First(&user)
		// https://redis.io/commands/zincrby
		database.Cache.ZIncrBy(context.Background(), "rankings", ambassadorRevenue, user.Name())

		// Email to ambassador
		ambassadorMessage := []byte(
			fmt.Sprintf("貴方の収益は、$%fです。Link: #%s",
				ambassadorRevenue,
				order.Code))
		// MailhogへのPort
		smtp.SendMail(
			"smtp:1025",                     // 宛先のアドレス
			nil,                             // Authentication
			"no-reply@email.com",            // from
			[]string{order.AmbassadorEmail}, // To
			ambassadorMessage)               // Message

		// Email to admin
		adminMessage := []byte(
			fmt.Sprintf("Order: #%d, Total: %f",
				order.ID,
				adminRevenue))
		// MailhogへのPort
		smtp.SendMail(
			"smtp:1025",                 // 宛先のアドレス
			nil,                         // Authentication
			"no-reply@email.com",        // from
			[]string{"admin@admin.com"}, // To
			adminMessage)                // Message
	}(order)

	return ctx.JSON(fiber.Map{
		"message": "success",
	})
}
