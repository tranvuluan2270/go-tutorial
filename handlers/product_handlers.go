package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-tutorial/cache"
	"go-tutorial/models"
	"go-tutorial/utils"
)

// GetProducts handles retrieving a list of products with basic filtering and sorting
func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page := 1
	limit := 10

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get basic filter parameters
	category := r.URL.Query().Get("category")
	searchQuery := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort") // Possible values: price_asc, price_desc, name_asc, name_desc

	// Create cache key
	cacheKey := fmt.Sprintf("products:p%d:l%d:cat%s:q%s:sort%s",
		page, limit, category, searchQuery, sortBy)

	// Try to get from cache
	var cachedData struct {
		Products []models.Product `json:"products"`
		Total    int64            `json:"total"`
	}

	err := cache.GetCache(ctx, cacheKey, &cachedData)
	if err == nil {
		w.Header().Set("X-Cache", "HIT")
		h.ResponseHdlr.Paginated(w, "Products fetched from cache", cachedData.Products, page, limit, int(cachedData.Total))
		return
	}

	w.Header().Set("X-Cache", "MISS")

	// Build filter query
	filterQuery := bson.M{}

	// Add category filter if provided
	if category != "" {
		filterQuery["category"] = category
	}

	// Add search filter if provided
	if searchQuery != "" {
		filterQuery["$or"] = []bson.M{
			{"name": bson.M{"$regex": searchQuery, "$options": "i"}},
			{"description": bson.M{"$regex": searchQuery, "$options": "i"}},
		}
	}

	// Get total count with filters
	productsCollection := h.DB.Database(h.Database).Collection("products")
	total, err := productsCollection.CountDocuments(ctx, filterQuery)
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error counting products")
		return
	}

	// Calculate skip for pagination
	skip := (page - 1) * limit

	// Build sort options
	sortOptions := bson.D{}
	switch sortBy {
	case "price_asc":
		sortOptions = append(sortOptions, bson.E{Key: "price", Value: 1})
	case "price_desc":
		sortOptions = append(sortOptions, bson.E{Key: "price", Value: -1})
	case "name_asc":
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: 1})
	case "name_desc":
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: -1})
	default:
		// Default sorting by name ascending
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: 1})
	}

	// Find products with filters and sort
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(sortOptions)

	cursor, err := productsCollection.Find(ctx, filterQuery, opts)
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error fetching products")
		return
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error processing products data")
		return
	}

	// Store in cache
	dataToCache := struct {
		Products []models.Product `json:"products"`
		Total    int64            `json:"total"`
	}{
		Products: products,
		Total:    total,
	}

	if err := cache.SetCache(ctx, cacheKey, dataToCache, 5*time.Minute); err != nil {
		log.Printf("Failed to cache products list: %v", err)
	}

	h.ResponseHdlr.Paginated(w, "Products fetched successfully", products, page, limit, int(total))
}

// GetProductDetails handles retrieving a single product by ID
func (h *Handler) GetProductDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["id"]

	// Try to get from cache first
	var product models.Product
	ctx := r.Context()
	cacheKey := fmt.Sprintf("product:%s", productID)

	err := cache.GetCache(ctx, cacheKey, &product)
	if err == nil {
		w.Header().Set("X-Cache", "HIT")
		h.ResponseHdlr.Success(w, "Product details fetched from cache", product)
		return
	}

	w.Header().Set("X-Cache", "MISS")

	// Get from database if not in cache
	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid product ID")
		return
	}

	err = h.DB.Database(h.Database).Collection("products").
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&product)

	if err != nil {
		h.ErrorHdlr.HandleNotFound(w, "Product not found")
		return
	}

	// Store in cache
	if err := cache.SetCache(ctx, cacheKey, product, 30*time.Minute); err != nil {
		log.Printf("Failed to cache product data: %v", err)
	}

	h.ResponseHdlr.Success(w, "Product details fetched successfully", product)
}

// CreateProduct handles creating a new product
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []utils.ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, utils.ErrorDetail{
				Field:   err.Field(),
				Message: utils.FormatValidationError(err),
			})
		}
		h.ErrorHdlr.HandleValidationError(w, validationErrors)
		return
	}

	// Create new product
	newProduct := models.Product{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
		Stock:       req.Stock,
	}

	// Insert into database
	_, err := h.DB.Database(h.Database).Collection("products").
		InsertOne(r.Context(), newProduct)

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error creating product")
		return
	}

	h.ResponseHdlr.Created(w, "Product created successfully", newProduct)
}

// UpdateProduct handles updating an existing product
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	productID := vars["id"]

	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid product ID")
		return
	}

	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []utils.ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, utils.ErrorDetail{
				Field:   err.Field(),
				Message: utils.FormatValidationError(err),
			})
		}
		h.ErrorHdlr.HandleValidationError(w, validationErrors)
		return
	}

	// Build update document
	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.Price > 0 {
		update["price"] = req.Price
	}
	if req.Category != "" {
		update["category"] = req.Category
	}
	if req.Stock >= 0 {
		update["stock"] = req.Stock
	}

	if len(update) == 0 {
		h.ErrorHdlr.HandleBadRequest(w, "No fields to update")
		return
	}

	// Update product in database
	result, err := h.DB.Database(h.Database).Collection("products").
		UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error updating product")
		return
	}

	if result.MatchedCount == 0 {
		h.ErrorHdlr.HandleNotFound(w, "Product not found")
		return
	}

	// Invalidate cache
	// 1. Delete specific product cache
	detailCacheKey := fmt.Sprintf(cache.ProductDetailPattern, productID)
	if err := cache.DeleteCache(ctx, detailCacheKey); err != nil {
		log.Printf("Failed to invalidate product detail cache: %v", err)
	}

	// 2. Delete all product list caches
	if err := cache.DeleteByPattern(ctx, cache.ProductListPattern); err != nil {
		log.Printf("Failed to invalidate product list cache: %v", err)
	}

	// Get updated product
	var updatedProduct models.Product
	err = h.DB.Database(h.Database).Collection("products").
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&updatedProduct)

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error getting updated product")
		return
	}

	h.ResponseHdlr.Success(w, "Product updated successfully", updatedProduct)
}

// DeleteProduct handles deleting a product
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	productID := vars["id"]

	objID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid product ID")
		return
	}

	result, err := h.DB.Database(h.Database).Collection("products").
		DeleteOne(ctx, bson.M{"_id": objID})

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error deleting product")
		return
	}

	if result.DeletedCount == 0 {
		h.ErrorHdlr.HandleNotFound(w, "Product not found")
		return
	}

	// Invalidate cache
	// 1. Delete specific product cache
	detailCacheKey := fmt.Sprintf(cache.ProductDetailPattern, productID)
	if err := cache.DeleteCache(ctx, detailCacheKey); err != nil {
		log.Printf("Failed to invalidate product detail cache: %v", err)
	}

	// 2. Delete all product list caches
	if err := cache.DeleteByPattern(ctx, cache.ProductListPattern); err != nil {
		log.Printf("Failed to invalidate product list cache: %v", err)
	}

	h.ResponseHdlr.Success(w, "Product successfully deleted", nil)
}
