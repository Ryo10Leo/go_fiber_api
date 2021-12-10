package controllers

import (
	"admin/src/database"
	"admin/src/models"
	"context"
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

var (
	products_frontend = "products_frontend"
	products_backend  = "products_backend"
)

func Products(ctx *fiber.Ctx) error {
	var products []models.Product

	database.DB.Find(&products)

	return ctx.JSON(products)
}

func CreateProducts(ctx *fiber.Ctx) error {
	var product models.Product

	if err := ctx.BodyParser(&product); err != nil {
		return err
	}

	database.DB.Create(&product)

	go database.ClearCache(products_frontend, products_backend)

	return ctx.JSON(product)
}

func GetProduct(ctx *fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	var product models.Product
	product.ID = uint(id)

	database.DB.Find(&product)

	return ctx.JSON(product)
}

func UpdateProduct(ctx *fiber.Ctx) error {
	// リクエストからIDを取得
	id, _ := strconv.Atoi(ctx.Params("id"))

	product := models.Product{}
	product.ID = uint(id)

	if err := ctx.BodyParser(&product); err != nil {
		return err
	}

	// プロダクト更新
	database.DB.Model(&product).Updates(&product)

	go database.ClearCache(products_frontend, products_backend)

	return ctx.JSON(product)
}

func DeleteProduct(ctx *fiber.Ctx) error {
	// リクエストからIDを取得
	id, _ := strconv.Atoi(ctx.Params("id"))

	product := models.Product{}
	product.ID = uint(id)

	// プロダクト削除
	database.DB.Delete(&product)

	go database.ClearCache(products_frontend, products_backend)

	return nil
}

func ProductFrontend(ctx *fiber.Ctx) error {
	var products []models.Product
	var c = context.Background()
	redisKey := "products_frontend"
	expiredTime := 30 * time.Minute

	result, err := database.Cache.Get(c, redisKey).Result()
	if err != nil {
		log.Println("---DB Search---")
		database.DB.Find(&products)

		productBytes, err := json.Marshal(&products)
		if err != nil {
			panic(err)
		}
		err = database.Cache.Set(c, redisKey, productBytes, expiredTime).Err()
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("---Redis Search---")
		json.Unmarshal([]byte(result), &products)
	}

	return ctx.JSON(products)
}

func ProductBackend(ctx *fiber.Ctx) error {
	var products []models.Product
	var c = context.Background()
	redisKey := "products_backend"
	expiredTime := 30 * time.Minute

	// キャッシュ操作
	result, err := database.Cache.Get(c, redisKey).Result()
	if err != nil {
		database.DB.Find(&products)

		productBytes, err := json.Marshal(&products)
		if err != nil {
			panic(err)
		}

		database.Cache.Set(c, redisKey, productBytes, expiredTime).Err()

	} else {
		json.Unmarshal([]byte(result), &products)
	}

	var searchProducts []models.Product

	// 検索
	// urlの?q=XXXXから文字列を取得
	if q := ctx.Query("q"); q != "" {
		// 大文字小文字の区別をなくすため、全て小文字扱いにする
		lower := strings.ToLower(q)
		for _, product := range products {
			// 検索条件1: Title
			if strings.Contains(strings.ToLower(product.Title), lower) ||
				// 検索条件2: Description
				strings.Contains(strings.ToLower(product.Description), lower) {
				searchProducts = append(searchProducts, product)
			}
		}
	} else {
		// 検索しない場合は、全てのデータを返却
		searchProducts = products
	}

	if sortParam := ctx.Query("sort"); sortParam != "" {
		sortLower := strings.ToLower(sortParam)
		if sortLower == "asc" {
			sort.Slice(searchProducts, func(i, j int) bool {
				return searchProducts[i].Price < searchProducts[j].Price
			})
		} else if sortLower == "desc" {
			sort.Slice(searchProducts, func(i, j int) bool {
				return searchProducts[i].Price > searchProducts[j].Price
			})
		}
	}

	// ページネーション
	var total = len(searchProducts)
	// デフォルトは"1"ページ
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	// 1ページ最大9個の商品
	perPage := 9

	var data []models.Product

	if total <= page*perPage && total >= (page-1)*perPage {
		data = searchProducts[(page-1)*perPage : total]
	} else if total >= page*perPage {
		data = searchProducts[(page-1)*perPage : page*perPage]
	} else {
		data = []models.Product{}
	}

	// 1ページ目 -> 0 ~ 8
	// 2パージ目 -> 9 ~ 17
	return ctx.JSON(fiber.Map{
		"data":      data,
		"total":     total,
		"page":      page,
		"last_page": total/perPage + 1,
	})
}
