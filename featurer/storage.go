package main

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

type Storage struct {
	RedisClient *redis.Client
}

func NewStorage(redisClient *redis.Client) Storage {
	return Storage{RedisClient: redisClient}
}

func createHashKey(featureID string) string {
	return "feature-container:" + featureID
}

func featureHashField() string {
	return "feature-container"
}

func (featureContainer *FeatureContainer) MarshalBinary() ([]byte, error) {
	result, err := json.Marshal(featureContainer)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error during marshalling, %v", err))
	}
	return result, nil
}

func (featureContainer *FeatureContainer) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &featureContainer); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error during unmarshalling, %v", err))
	}

	return nil
}

func (storage *Storage) Create(ctx context.Context, featureID string, featureContainer FeatureContainer) error {

	marshaledFeature, err := featureContainer.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = storage.RedisClient.HSetNX(ctx, createHashKey(featureID), featureHashField(), marshaledFeature).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error creating Redis entry, %v", err))

	}
	return nil
}

func (storage *Storage) Get(ctx context.Context, featureID string) (*FeatureContainer, error) {

	result, err := storage.RedisClient.HGet(ctx, createHashKey(featureID), featureHashField()).Result()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error getting Redis entry, %v", err))
	} else if result == "" {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Redis entry not found, %v", err))
	}
	featureContainer := &FeatureContainer{}
	if err = featureContainer.UnmarshalBinary([]byte(result)); err != nil {
		return nil, err
	}
	return featureContainer, nil
}

func (storage *Storage) Update(ctx context.Context, featureID string, newEntry FeatureContainer) error {

	_, err := storage.Get(ctx, featureID)
	if err != nil {
		return err
	}

	marshalledFeature, err := newEntry.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = storage.RedisClient.HSet(ctx, createHashKey(featureID), featureHashField(), marshalledFeature).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error updating Redis entry, %v", err))
	}
	return nil
}

func (storage *Storage) Delete(ctx context.Context, featureID string) error {

	_, err := storage.RedisClient.Del(ctx, createHashKey(featureID)).Result()
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error deleting Redis entry, %v", err))
	}
	return nil
}
