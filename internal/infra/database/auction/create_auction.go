package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go func() {
		timer := time.NewTimer(getAuctionClosureInterval())
		defer timer.Stop()

		select {
		case <-timer.C:
			_, err := ar.Collection.UpdateOne(ctx, bson.M{
				"_id": auctionEntityMongo.Id,
			}, bson.M{
				"$set": bson.M{"status": auction_entity.Completed},
			})
			if err != nil {
				logger.Error("failed to update auction status", err)
			}
		case <-ctx.Done():
			logger.Info("auction closure canceled by context")
		}
	}()

	return nil
}

func getAuctionClosureInterval() time.Duration {
	intervalStr := os.Getenv("AUCTION_INTERVAL")
	if d, err := time.ParseDuration(intervalStr); err == nil {
		return d
	}
	logger.Info("Invalid or missing AUCTION_INTERVAL, defaulting to 10s")
	return 10 * time.Second
}
