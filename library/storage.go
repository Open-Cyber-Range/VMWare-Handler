package library

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/vmware/govmomi/vim25/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeatureContainer struct {
	VMID      string
	Auth      types.NamePasswordAuthentication
	FilePaths []string
}

type Account struct {
	Name     string
	Password string
}

type Storage struct {
	RedisClient *redis.Client
}

func NewStorage(redisClient *redis.Client) Storage {
	return Storage{RedisClient: redisClient}
}

func hashField() string {
	return "generic-field"
}

func MarshalBinary[T any](input T) ([]byte, error) {
	result, err := json.Marshal(input)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during marshalling, %v", err))
	}
	return result, nil
}

func UnmarshalBinary[T any](data []byte, output T) error {
	if err := json.Unmarshal(data, &output); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error during unmarshalling, %v", err))
	}

	return nil
}

func Create[T any](ctx context.Context, redisClient *redis.Client, itemKey string, item T) error {

	marshaledFeature, err := MarshalBinary(item)
	if err != nil {
		return err
	}

	_, err = redisClient.HSetNX(ctx, itemKey, hashField(), marshaledFeature).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating Redis entry, %v", err))

	}
	return nil
}

func Get[T any](ctx context.Context, redisClient *redis.Client, itemKey string, resultType *T) (*T, error) {

	result, err := redisClient.HGet(ctx, itemKey, hashField()).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting Redis entry, %v", err))
	} else if result == "" {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Redis entry not found, %v", err))
	}

	if err = UnmarshalBinary([]byte(result), resultType); err != nil {
		return nil, err
	}
	return resultType, nil
}

func Update[T any](ctx context.Context, redisClient *redis.Client, featureID string, newEntry *T) error {

	marshalledEntry, err := MarshalBinary(newEntry)
	if err != nil {
		return err
	}
	_, err = redisClient.HSet(ctx, featureID, hashField(), marshalledEntry).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error updating Redis entry, %v", err))
	}
	return nil
}

func Delete(ctx context.Context, redisClient *redis.Client, featureID string) error {

	_, err := redisClient.Del(ctx, featureID).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error deleting Redis entry, %v", err))
	}
	return nil
}
