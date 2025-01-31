package main

import (
	"context"
	"github.com/go-playground/validator/v10"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Event struct {
	Event      string `json:"event" validate:"required,oneof=impression add_tocart page_view"`
	VisitorID  string `json:"visitorId" validate:"required"`
	CustomerID string `json:"customerId" validate:"required"`
	PageURL    string `json:"pageUrl" validate:"required,url"`
	AdID       string `json:"adId" validate:"required"`
	Timestamp  string `json:"timestamp" validate:"required,datetime=2006-01-02T15:04:05Z"`
	UserAgent  string `json:"userAgent" validate:"required"`
}

type RequestBody struct {
	Events []Event `json:"events" validate:"gt=0,required,dive"`
}

var validate *validator.Validate
var client *mongo.Client

func main() {
	validate = validator.New()

	// Setup a configs for api from environemnt
	mongoHost := os.Getenv("MONGODB_HOST")
	mongoUser := os.Getenv("MONGODB_USER")
	mongoPass := os.Getenv("MONGODB_PASS")
	mongoDb := os.Getenv("MONGODB_DB")
	apiPort := os.Getenv("API_PORT")

	mongoURI := "mongodb://" + mongoUser + ":" + mongoPass + "@" + mongoHost + "/" + mongoDb
	mongoURI = mongoURI + "?authSource=admin"

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	r := gin.Default()
	r.PUT("/", handlePutRequest)
	err = r.Run(":" + apiPort)

	if err != nil {
		log.Fatalf(err.Error())
	}
}

func handlePutRequest(c *gin.Context) {
	var requestBody RequestBody

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validate.Struct(requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(requestBody.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No events found"})
		return
	}

	events := make([]interface{}, len(requestBody.Events))
	for idx, event := range requestBody.Events {
		if err := validate.Struct(event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		format := "2006-01-02T15:04:05Z"
		timestamp, err := time.Parse(format, event.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format"})
			return
		}
		if timestamp.After(time.Now().UTC()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Timestamp cannot be in the future"})
			return
		}
		event.Timestamp = timestamp.Format(format)
		events[idx] = event

	}

	mongoDb := os.Getenv("MONGODB_DB")
	collection := client.Database(mongoDb).Collection("events")
	_, err := collection.InsertMany(context.TODO(), events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Events synced successfully!"})
}
