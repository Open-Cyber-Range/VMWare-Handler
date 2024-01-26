package library

import (
	"container/heap"
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Request struct {
	VmId          string
	ActionId      string
	RunCount      int
	Interval      int
	index         int
	TotalRunCount int
	Priority      float64
}

type PriorityQueue []*Request

type Mutex struct {
	PoolId         string
	Distributor    *MutexDistributor
	Mutex          *redsync.Mutex
	RedisClient    *redis.Client
	MaxConnections int64
	Options        MutexOptions
}

type MutexOptions struct {
	VmId     string
	ActionId string
	Interval int
}

type MutexDistributor struct {
	Id             string
	Redsync        *redsync.Redsync
	RedisClient    *redis.Client
	MaxConnections int64
	Counter        map[string]map[string]int
	CounterMutex   *redsync.Mutex
	ActionQueue    PriorityQueue
	QueueMutex     *redsync.Mutex
	VmQueue        map[string][]*MutexOptions
}

func NewRequest(mutexOptions *MutexOptions, runCount int, totalRunCount int) *Request {
	frequencyPriority := float64(mutexOptions.Interval) / (float64(runCount) + 1)
	totalVmRunCountPriority := 1 / float64(totalRunCount+1)
	priority := (frequencyPriority + totalVmRunCountPriority) / 2

	return &Request{
		VmId:          mutexOptions.VmId,
		ActionId:      mutexOptions.ActionId,
		RunCount:      runCount,
		Interval:      mutexOptions.Interval,
		TotalRunCount: totalRunCount,
		Priority:      priority,
	}
}

func NewMutexOptions(vmId string, actionId string, interval ...int) *MutexOptions {
	defaultVmId := "general-mutex"
	defaultActionId := ""
	defaultInterval := 1

	if vmId != "" {
		defaultVmId = "vm-mutex-" + vmId
	}

	if actionId != "" {
		defaultActionId = actionId
	}

	if len(interval) > 0 {
		defaultInterval = interval[0]
	}

	return &MutexOptions{
		VmId:     defaultVmId,
		ActionId: defaultActionId,
		Interval: defaultInterval,
	}
}

func (queue PriorityQueue) Len() int { return len(queue) }

func (queue PriorityQueue) Less(i, j int) bool {
	return queue[i].Priority > queue[j].Priority
}

func (queue PriorityQueue) Swap(i, j int) {
	queue[i], queue[j] = queue[j], queue[i]
	queue[i].index = i
	queue[j].index = j
}

func (queue *PriorityQueue) Push(x interface{}) {
	n := len(*queue)
	item := x.(*Request)
	item.index = n
	*queue = append(*queue, item)
}

func (queue *PriorityQueue) Pop() interface{} {
	old := *queue
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*queue = old[0 : n-1]
	return item
}

func (queue *PriorityQueue) Update(item *Request, value string, runCount int, interval int) {
	item.VmId = value
	item.RunCount = runCount
	item.Interval = interval
	heap.Fix(queue, item.index)
}

func (distributor *MutexDistributor) IncrementVMCounter(vmId string) {
	distributor.CounterMutex.Lock()
	defer distributor.CounterMutex.Unlock()

	if _, ok := distributor.Counter[vmId]; !ok {
		distributor.Counter[vmId] = make(map[string]int)
	}
	distributor.Counter[vmId]["vm"]++
}

func (distributor *MutexDistributor) GetVMCounter(vmId string) int {
	distributor.CounterMutex.Lock()
	defer distributor.CounterMutex.Unlock()

	if _, ok := distributor.Counter[vmId]; !ok {
		return 0
	}
	return distributor.Counter[vmId]["vm"]
}

func (distributor *MutexDistributor) IncrementActionCounter(vmId string, actionId string) {
	distributor.CounterMutex.Lock()
	defer distributor.CounterMutex.Unlock()

	if _, ok := distributor.Counter[vmId]; !ok {
		distributor.Counter[vmId] = make(map[string]int)
	}
	distributor.Counter[vmId][actionId]++
	distributor.IncrementVMCounter(vmId)
}

func (distributor *MutexDistributor) GetActionCounter(vmId string, actionId string) int {
	distributor.CounterMutex.Lock()
	defer distributor.CounterMutex.Unlock()

	if _, ok := distributor.Counter[vmId]; !ok {
		return 0
	}
	return distributor.Counter[vmId][actionId]
}

func (distributor *MutexDistributor) GetMutex(ctx context.Context, options *MutexOptions) (*Mutex, error) {
	return distributor.acquireMutex(options)
}

func (distributor *MutexDistributor) addToQueue(ctx context.Context, options MutexOptions) error {
	distributor.CounterMutex.Lock()
	runCount := distributor.GetActionCounter(options.VmId, options.ActionId)
	distributor.CounterMutex.Unlock()

	for {
		distributor.QueueMutex.Lock()
		currentConnections, err := distributor.RedisClient.HLen(ctx, distributor.Id).Result()
		if err != nil {
			distributor.QueueMutex.Unlock()
			return fmt.Errorf("error getting Redis entry: %w", err)
		}
		if currentConnections < distributor.MaxConnections {
			request := NewRequest(&options, runCount, distributor.GetVMCounter(options.VmId))

			heap.Push(&distributor.ActionQueue, request)
			distributor.VmQueue[options.VmId] = append(distributor.VmQueue[options.VmId], &options)

			distributor.QueueMutex.Unlock()
			return nil
		}

		distributor.QueueMutex.Unlock()
		time.Sleep(time.Duration(rand.Intn(50)+20) * time.Millisecond)
	}
}

func (distributor *MutexDistributor) waitUntilFrontOfQueue(vmId string, actionId string) error {
	for {
		distributor.QueueMutex.Lock()
		queue := distributor.VmQueue[vmId]
		if len(queue) == 0 || queue[0].ActionId == actionId {
			distributor.QueueMutex.Unlock()
			return nil
		}
		distributor.QueueMutex.Unlock()
		time.Sleep(time.Duration(rand.Intn(20)+10) * time.Millisecond)
	}
}

func (distributor *MutexDistributor) acquireMutex(options *MutexOptions) (*Mutex, error) {
	mutex := &Mutex{
		Distributor:    distributor,
		PoolId:         distributor.Id,
		Mutex:          distributor.Redsync.NewMutex(options.VmId),
		RedisClient:    distributor.RedisClient,
		MaxConnections: distributor.MaxConnections,
		Options:        *options,
	}

	return mutex, nil
}

func (mutex *Mutex) Lock(ctx context.Context) error {
	mutex.Distributor.addToQueue(ctx, mutex.Options)

	if err := mutex.Distributor.waitUntilFrontOfQueue(mutex.Options.VmId, mutex.Options.ActionId); err != nil {
		return err
	}

	for {
		lockAlreadyExists, err := mutex.Distributor.RedisClient.HExists(ctx, mutex.Distributor.Id, mutex.Mutex.Name()).Result()
		if err != nil {
			return fmt.Errorf("error checking existence of Redis entry: %w", err)
		}

		if !lockAlreadyExists {
			err := mutex.Mutex.Lock()
			if err != nil {
				if strings.HasPrefix(err.Error(), "lock already taken") {
					time.Sleep(time.Duration(rand.Intn(50)+20) * time.Millisecond)
					continue
				} else {
					return err
				}
			}
			return nil
		}
	}
}

func (mutex *Mutex) Unlock() (err error) {
	mutex.Mutex.Unlock()

	mutex.Distributor.QueueMutex.Lock()
	defer mutex.Distributor.QueueMutex.Unlock()

	for i, item := range mutex.Distributor.ActionQueue {
		if item.VmId == mutex.Options.VmId && item.ActionId == mutex.Options.ActionId {
			heap.Remove(&mutex.Distributor.ActionQueue, i)
			break
		}
	}

	queue := mutex.Distributor.VmQueue[mutex.Options.VmId]
	for i, options := range queue {
		if options.ActionId == mutex.Options.ActionId {
			mutex.Distributor.VmQueue[mutex.Options.VmId] = append(queue[:i], queue[i+1:]...)
			break
		}
	}

	_, delErr := mutex.RedisClient.HDel(context.Background(), mutex.PoolId, mutex.Mutex.Name()).Result()
	if delErr != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Error deleting Redis entry, %v", delErr))
	}

	return nil
}

func NewMutexDistributor(ctx context.Context, poolIdentifier string, redsync redsync.Redsync, redisClient redis.Client, maxConnections int64) (distributor MutexDistributor, err error) {
	distributor = MutexDistributor{
		Id:             poolIdentifier,
		Redsync:        &redsync,
		RedisClient:    &redisClient,
		MaxConnections: maxConnections,
		Counter:        make(map[string]map[string]int),
		CounterMutex:   redsync.NewMutex("counterMutex"),
		ActionQueue:    make(PriorityQueue, 0),
		VmQueue:        make(map[string][]*MutexOptions),
		QueueMutex:     redsync.NewMutex("queueMutex"),
	}
	_, err = redisClient.Del(ctx, poolIdentifier).Result()
	if err != nil {
		return MutexDistributor{}, status.Error(codes.Internal, fmt.Sprintf("Error cleaning up MutexPool, %v", err))
	}
	return distributor, nil
}
