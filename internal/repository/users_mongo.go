package repository

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"rest-api-crud/internal/domain"
	"rest-api-crud/pkg/logging"
)

const (
	notFound = "not found"
)

type db struct {
	collection *mongo.Collection
	logger     *logging.Logger
}

func (d *db) Create(ctx context.Context, user domain.User) (string, error) {
	result, err := d.collection.InsertOne(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to create user due to error %v", err)
	}

	d.logger.Debug("Converting InsertId to ObjectId")
	oid, ok := result.InsertedID.(bson.ObjectID)
	if ok {
		return oid.Hex(), nil
	}
	d.logger.Trace(user)
	return "", fmt.Errorf("failed to convert objectID to hex")
}

func (d *db) FindById(ctx context.Context, id string) (u domain.User, err error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return u, fmt.Errorf("failed to convert hex to objectID")
	}

	filter := bson.M{"_id": oid}

	result := d.collection.FindOne(ctx, filter)
	if result.Err() != nil {
		// todo process 404
		return u, fmt.Errorf("failed to find user by id: %s due to err %s", id, result.Err())
	}

	if err = result.Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode user from DB due to error: %v", err)
	}

	return u, nil
}

func (d *db) FindAll(ctx context.Context) (u []domain.User, err error) {
	cursor, err := d.collection.Find(ctx, bson.M{})
	if cursor.Err() != nil {
		return u, fmt.Errorf("failed to find all users due to error: %v", err)
	}

	if err = cursor.All(ctx, &u); err != nil {
		return u, fmt.Errorf("failed to read all documents from cursor. error : %v", err)
	}

	return u, nil
}

func (d *db) Update(ctx context.Context, user domain.User) error {
	id, ok := user.ID.(bson.ObjectID)
	if !ok {
		return fmt.Errorf("failed to convert user ID field to bson.ObjectID")
	}

	oid, err := bson.ObjectIDFromHex(id.Hex())
	if err != nil {
		return fmt.Errorf("failed to convert userID to ObjectID, ID = %s", user.ID)
	}

	filter := bson.M{"_id": oid}

	userBytes, err := bson.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user, error : %v", err)
	}

	var updatedUserObj bson.M
	err = bson.Unmarshal(userBytes, &updatedUserObj)
	if err != nil {
		return fmt.Errorf("failed to unmarshall user bytes, error : %v", err)
	}

	delete(updatedUserObj, "_id")

	update := bson.M{
		"$set": updatedUserObj,
	}

	result, err := d.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to execute update user query. error: %v", err)
	}

	if result.MatchedCount == 0 {
		// todo ErrEntityNotFound
		return fmt.Errorf(notFound)
	}

	d.logger.Tracef("Matched %d documents and Modified %d documents", result.MatchedCount, result.ModifiedCount)

	return nil
}

func (d *db) DeleteById(ctx context.Context, id string) error {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("failed to convert user ID to ObjectID. ID = %s", id)
	}

	filter := bson.M{"_id": oid}

	result, err := d.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to execute query. error: %v", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf(notFound)
	}

	d.logger.Tracef("Deleted %d documents", result.DeletedCount)

	return nil
}

func NewStorage(database *mongo.Database, collection string, logger *logging.Logger) Users {
	return &db{
		collection: database.Collection(collection),
		logger:     logger,
	}
}
