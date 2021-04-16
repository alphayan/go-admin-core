package cache

import (
	"context"
	"time"

	"github.com/alphayan/redisqueue/v3"
	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v8"
)

var rctx = context.Background()

// Redis cache implement
type Redis struct {
	ConnectOption   *redis.Options
	ConsumerOptions *redisqueue.ConsumerOptions
	ProducerOptions *redisqueue.ProducerOptions
	client          *redis.Client
	consumer        *redisqueue.Consumer
	producer        *redisqueue.Producer
	mutex           *redislock.Client
}

func (*Redis) String() string {
	return "redis"
}

// Connect Setup
func (r *Redis) Connect() error {
	var err error
	r.client = redis.NewClient(r.ConnectOption)
	_, err = r.client.Ping(rctx).Result()
	if err != nil {
		return err
	}
	r.mutex = redislock.New(r.client)
	r.producer, err = r.newProducer(r.client)
	if err != nil {
		return err
	}
	r.consumer, err = r.newConsumer(r.client)
	return err
}

func (r *Redis) SetPrefix(string) {}

// Get from key
func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(context.Background(), key).Result()
}

// Set value with key and expire time
func (r *Redis) Set(key string, val interface{}, expire int) error {
	return r.client.Set(rctx, key, val, time.Duration(expire)*time.Second).Err()
}

// Del delete key in redis
func (r *Redis) Del(key string) error {
	return r.client.Del(rctx, key).Err()
}

// HashGet from key
func (r *Redis) HashGet(hk, key string) (string, error) {
	return r.client.HGet(rctx, hk, key).Result()
}

// HashDel delete key in specify redis's hashtable
func (r *Redis) HashDel(hk, key string) error {
	return r.client.HDel(rctx, hk, key).Err()
}

// Increase
func (r *Redis) Increase(key string) error {
	return r.client.Incr(rctx, key).Err()
}

func (r *Redis) Decrease(key string) error {
	return r.client.Decr(rctx, key).Err()
}

// Set ttl
func (r *Redis) Expire(key string, dur time.Duration) error {
	return r.client.Expire(rctx, key, dur).Err()
}

func (r *Redis) Append(message Message) error {
	err := r.producer.Enqueue(&redisqueue.Message{
		ID:     message.GetID(),
		Stream: message.GetStream(),
		Values: message.GetValues(),
	})
	return err
}

func (r *Redis) Register(name string, f ConsumerFunc) {
	r.consumer.Register(name, func(message *redisqueue.Message) error {
		m := new(RedisMessage)
		m.SetValues(message.Values)
		m.SetStream(message.Stream)
		m.SetID(message.ID)
		return f(m)
	})
}

func (r *Redis) Run() {
	r.consumer.Run()
}

func (r *Redis) Shutdown() {
	r.consumer.Shutdown()
}

func (r *Redis) newConsumer(client *redis.Client) (*redisqueue.Consumer, error) {
	if r.ConsumerOptions == nil {
		r.ConsumerOptions = &redisqueue.ConsumerOptions{}
	}
	r.ConsumerOptions.RedisClient = client
	return redisqueue.NewConsumerWithOptions(r.ConsumerOptions)
}

func (r *Redis) newProducer(client *redis.Client) (*redisqueue.Producer, error) {
	if r.ProducerOptions == nil {
		r.ProducerOptions = &redisqueue.ProducerOptions{}
	}
	r.ProducerOptions.RedisClient = client
	return redisqueue.NewProducerWithOptions(r.ProducerOptions)
}

func (r *Redis) Lock(key string, ttl int64, options *redislock.Options) (*redislock.Lock, error) {
	if r.mutex == nil {
		r.mutex = redislock.New(r.client)
	}
	return r.mutex.Obtain(rctx,key, time.Duration(ttl)*time.Second, options)
}

// GetClient 暴露原生client
func (r *Redis) GetClient() *redis.Client {
	return r.client
}

type RedisMessage struct {
	redisqueue.Message
}

func (m *RedisMessage) GetID() string {
	return m.ID
}

func (m *RedisMessage) GetStream() string {
	return m.Stream
}

func (m *RedisMessage) GetValues() map[string]interface{} {
	return m.Values
}

func (m *RedisMessage) SetID(id string) {
	m.ID = id
}

func (m *RedisMessage) SetStream(stream string) {
	m.Stream = stream
}

func (m *RedisMessage) SetValues(values map[string]interface{}) {
	m.Values = values
}

func (m *RedisMessage) GetPrefix() (prefix string) {
	if m.Values == nil {
		return
	}
	v, _ := m.Values[prefixKey]
	prefix, _ = v.(string)
	return
}

func (m *RedisMessage) SetPrefix(prefix string) {
	if m.Values == nil {
		m.Values = make(map[string]interface{})
	}
	m.Values[prefixKey] = prefix
}
