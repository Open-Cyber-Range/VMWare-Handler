package library

import (
	"context"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"
)

var testConfiguration = Configuration{
	User:               os.Getenv("TEST_VMWARE_USER"),
	Password:           os.Getenv("TEST_VMWARE_PASSWORD"),
	Hostname:           os.Getenv("TEST_VMWARE_HOSTNAME"),
	Insecure:           true,
	TemplateFolderPath: os.Getenv("TEST_VMWARE_TEMPLATE_FOLDER_PATH"),
	ServerAddress:      "127.0.0.1",
	ResourcePoolPath:   os.Getenv("TEST_VMWARE_RESOURCE_POOL_PATH"),
	ExerciseRootPath:   os.Getenv("TEST_VMWARE_EXERCISE_ROOT_PATH"),
	RedisAddress:       os.Getenv("TEST_REDIS_ADDRESS"),
	RedisPassword:      os.Getenv("TEST_REDIS_PASSWORD"),
}

func createRedisClient() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     testConfiguration.RedisAddress,
		Password: testConfiguration.RedisPassword,
		DB:       0,
	})

	return redisClient
}

func TestRedisCRUD(t *testing.T) {

	ctx := context.Background()
	redisClient := createRedisClient()

	featureID := "123456789"

	storage := Storage[ExecutorContainer]{
		RedisClient: redisClient,
		Container: ExecutorContainer{
			VMID: "vmid",
			Auth: types.NamePasswordAuthentication{
				Username: "username",
				Password: "password",
			},
			FilePaths: []string{"im a path", "im a path two"},
		},
	}

	err := storage.Create(ctx, featureID)
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

	storage.Container = featureContainer
	err = storage.Update(ctx, featureID)
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
