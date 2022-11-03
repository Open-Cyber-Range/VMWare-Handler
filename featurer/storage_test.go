package main

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"
)

func createRedisClient() Storage {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     testConfiguration.RedisAddress,
		Password: testConfiguration.RedisPassword,
		DB:       0,
	})

	return NewStorage(redisClient)
}

func TestRedisCRUD(t *testing.T) {

	ctx := context.Background()
	storage := createRedisClient()

	featureID := "123456789"
	featureHolder := FeatureContainer{
		VMID: "vmid",
		Auth: types.NamePasswordAuthentication{
			Username: "username",
			Password: "password",
		},
		FilePaths: []string{"im a path", "im a string two"},
	}

	err := storage.Create(ctx, featureID, featureHolder)
	if err != nil {
		panic(err)
	}
	log.Infof("Redis Create success")

	featureContainer, err := storage.Get(ctx, featureID)
	if err != nil {
		panic(err)
	}
	log.Infof("Redis Read success")

	updatedPath := "im updated!"
	featureContainer.FilePaths = []string{updatedPath}

	err = storage.Update(ctx, featureID, *featureContainer)
	if err != nil {
		panic(err)
	}

	featureContainer, err = storage.Get(ctx, featureID)
	if err != nil {
		panic(err)
	} else if featureContainer.FilePaths[0] != updatedPath {
		panic("Unexpected result from Redis Update")
	}
	log.Infof("Redis Update success")

	err = storage.Delete(ctx, featureID)
	if err != nil {
		panic(err)
	}

	_, err = storage.Get(ctx, featureID)
	if err == nil {
		panic("Redis entry was not deleted")
	}
	log.Infof("Redis Delete success")

}
