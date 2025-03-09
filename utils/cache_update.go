package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-tutorial/cache"
	"go-tutorial/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RedisUpdateJob struct {
	db       *mongo.Client
	database string
}

func NewRedisUpdateJob(db *mongo.Client, database string) *RedisUpdateJob {
	return &RedisUpdateJob{
		db:       db,
		database: database,
	}
}

func (j *RedisUpdateJob) Start() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			j.updateProductsCache()
			j.updateUsersCache()
		}
	}()
}

func (j *RedisUpdateJob) updateProductsCache() {
	ctx := context.Background()
	productsCollection := j.db.Database(j.database).Collection("products")

	// Get all products
	cursor, err := productsCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error fetching products for cache update: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		log.Printf("Error decoding products for cache update: %v", err)
		return
	}

	// Update products list cache
	dataToCache := struct {
		Products []models.Product `json:"products"`
		Total    int64            `json:"total"`
	}{
		Products: products,
		Total:    int64(len(products)),
	}

	if err := cache.SetCache(ctx, "products:all", dataToCache, 15*time.Minute); err != nil {
		log.Printf("Failed to update products cache: %v", err)
		return
	}

	// Update individual product caches
	for _, product := range products {
		cacheKey := fmt.Sprintf(cache.ProductDetailPattern, product.ID.Hex())
		if err := cache.SetCache(ctx, cacheKey, product, 15*time.Minute); err != nil {
			log.Printf("Failed to update product cache for ID %s: %v", product.ID.Hex(), err)
		}
	}

	log.Printf("Successfully updated Redis cache for %d products", len(products))
}

func (j *RedisUpdateJob) updateUsersCache() {
	ctx := context.Background()
	usersCollection := j.db.Database(j.database).Collection("users")

	// Get all users
	cursor, err := usersCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Error fetching users for cache update: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var users []models.UserResponse
	if err := cursor.All(ctx, &users); err != nil {
		log.Printf("Error decoding users for cache update: %v", err)
		return
	}

	// Update users list cache
	dataToCache := struct {
		Users []models.UserResponse `json:"users"`
		Total int64                 `json:"total"`
	}{
		Users: users,
		Total: int64(len(users)),
	}

	if err := cache.SetCache(ctx, "users:all", dataToCache, 15*time.Minute); err != nil {
		log.Printf("Failed to update users cache: %v", err)
		return
	}

	// Update individual user caches
	for _, user := range users {
		cacheKey := fmt.Sprintf(cache.UserDetailPattern, user.ID.Hex())
		if err := cache.SetCache(ctx, cacheKey, user, 15*time.Minute); err != nil {
			log.Printf("Failed to update user cache for ID %s: %v", user.ID.Hex(), err)
		}
	}

	log.Printf("Successfully updated Redis cache for %d users", len(users))
}
