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

type ExecutorContainer struct {
	VMID      string
	Auth      types.NamePasswordAuthentication
	FilePaths []string
	Command   string
	Interval  int32
}

type Account struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

type Storage[T any] struct {
	RedisClient *redis.Client
	Container   T
}

func NewStorage[T any](redisAddress string, redisPassword string) Storage[T] {
	return Storage[T]{
		RedisClient: redis.NewClient(&redis.Options{
			Addr:     redisAddress,
			Password: redisPassword,
			DB:       0,
		}),
		Container: *new(T),
	}
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

func (storage *Storage[T]) Create(ctx context.Context, itemKey string) error {

	marshaledFeature, err := MarshalBinary(storage.Container)
	if err != nil {
		return err
	}

	_, err = storage.RedisClient.HSetNX(ctx, itemKey, hashField(), marshaledFeature).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating Redis entry, %v", err))

	}
	return nil
}

func (storage *Storage[T]) Get(ctx context.Context, itemKey string) (T, error) {

	result, err := storage.RedisClient.HGet(ctx, itemKey, hashField()).Result()
	if err != nil {
		return *new(T), status.Error(codes.Internal, fmt.Sprintf("Error getting Redis entry, %v", err))
	} else if result == "" {
		return *new(T), status.Error(codes.Internal, fmt.Sprintf("Redis entry not found, %v", err))
	}

	if err = UnmarshalBinary([]byte(result), &storage.Container); err != nil {
		return *new(T), err
	}
	return storage.Container, nil
}

func (storage *Storage[T]) Update(ctx context.Context, featureID string) error {

	marshalledEntry, err := MarshalBinary(storage.Container)
	if err != nil {
		return err
	}
	_, err = storage.RedisClient.HSet(ctx, featureID, hashField(), marshalledEntry).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error updating Redis entry, %v", err))
	}
	return nil
}

func (storage *Storage[_]) Delete(ctx context.Context, featureID string) error {

	_, err := storage.RedisClient.Del(ctx, featureID).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error deleting Redis entry, %v", err))
	}
	return nil
}
