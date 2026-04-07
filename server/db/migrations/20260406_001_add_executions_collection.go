package migrations

import (
	"context"
	"open-nirmata/db"
	"open-nirmata/db/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	db.RegisterMigration(db.MigrationInfo{
		Version:     "20260406_001",
		Name:        "add_executions_collection",
		Description: "Create indexes for the executions collection to support efficient querying",
		Up:          addExecutionsCollectionUp,
		Down:        addExecutionsCollectionDown,
	})
}

func addExecutionsCollectionUp(ctx context.Context, database db.DB) error {
	execModel := models.GetExecutionModel()
	
	// Get the underlying MongoDB database and collection
	// This requires accessing the MongoDB-specific implementation
	mongoCollection := database.GetCollection(execModel.Name())

	// Create indexes for efficient querying
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "agent_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_agent_id_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "prompt_flow_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_prompt_flow_id_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_status_created_at"),
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_created_at"),
		},
	}

	_, err := mongoCollection.Indexes().CreateMany(ctx, indexes)
	return err
}

func addExecutionsCollectionDown(ctx context.Context, database db.DB) error {
	execModel := models.GetExecutionModel()
	mongoCollection := database.GetCollection(execModel.Name())

	// Drop the indexes
	indexNames := []string{
		"idx_agent_id_created_at",
		"idx_prompt_flow_id_created_at",
		"idx_status_created_at",
		"idx_created_at",
	}

	for _, indexName := range indexNames {
		_, _ = mongoCollection.Indexes().DropOne(ctx, indexName)
		// Ignore errors as index may not exist
	}

	return nil
}
