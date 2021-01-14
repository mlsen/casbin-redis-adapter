package redisadapter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2/model"

	"github.com/casbin/casbin/v2"

	redisadapter "github.com/mlsen/casbin-redis-adapter"

	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	redisImage   = "redis"
	redisVersion = "6-alpine"
)

type RedisAdapterTestSuite struct {
	suite.Suite
	client     *redis.Client
	purgeRedis func() error
}

func (suite *RedisAdapterTestSuite) TestNewAdapterFromClient() {
	_, err := redisadapter.NewFromClient(suite.client)
	suite.NoError(err)
}

func (suite *RedisAdapterTestSuite) TestNewAdapterFromURL() {
	url := "redis://localhost:6379/0"
	_, err := redisadapter.NewFromURL(url)
	suite.NoError(err)
}

func (suite *RedisAdapterTestSuite) TestSavePolicy() {
	// Load the model + policies from the example files
	e, err := casbin.NewEnforcer("files/model.conf", "files/policy.csv")
	suite.NoError(err)

	// Save the file model for later comparison
	fileModel := e.GetModel()

	// Create the adapter
	a, err := redisadapter.NewFromClient(suite.client)
	suite.NoError(err)

	// Save the file model to redis
	err = a.SavePolicy(fileModel)
	suite.NoError(err)

	// Create a new Enforcer, this time with the redis adapter
	e, err = casbin.NewEnforcer("files/model.conf", a)
	suite.NoError(err)

	// Load policies from redis
	err = e.LoadPolicy()
	suite.NoError(err)

	// Compare the policies from redis to the ones from the file
	assertModelEqual(suite.T(), fileModel, e.GetModel())

	// Check if the new policies overwrite the old ones
	_ = e.SavePolicy()
	polLength, err := suite.client.LLen(context.Background(), redisadapter.PolicyKey).Result()
	suite.NoError(err)
	suite.EqualValues(3, polLength)

	// Delete current policies
	e.ClearPolicy()

	// Save empty model for comparison
	emptyModel := e.GetModel()

	// Save empty model
	err = e.SavePolicy()
	suite.NoError(err)

	// Load empty model again
	err = e.LoadPolicy()
	suite.NoError(err)

	// Check if the loaded model equals the empty model from before
	assertModelEqual(suite.T(), emptyModel, e.GetModel())
}

func (suite *RedisAdapterTestSuite) TestLoadPolicy() {
	suite.True(true)
}

func (suite *RedisAdapterTestSuite) TestAddPolicy() {
	// Create the adapter
	a, err := redisadapter.NewFromClient(suite.client)
	suite.NoError(err)

	// Create a new Enforcer, this time with the redis adapter
	e, err := casbin.NewEnforcer("files/model.conf", a)
	suite.NoError(err)

	// Add policies
	_, err = e.AddPolicy("bob", "data1", "read")
	suite.NoError(err)
	_, err = e.AddPolicy("alice", "data1", "write")
	suite.NoError(err)

	// Clear all policies from memory
	e.ClearPolicy()

	// Policy is deleted now
	hasPol := e.HasPolicy("bob", "data1", "read")
	suite.False(hasPol)

	// Load policies from redis
	err = e.LoadPolicy()
	suite.NoError(err)

	// Policy is there again
	hasPol = e.HasPolicy("bob", "data1", "read")
	suite.True(hasPol)
	hasPol = e.HasPolicy("alice", "data1", "write")
	suite.True(hasPol)
}

func (suite *RedisAdapterTestSuite) TestRemovePolicy() {
	// Create the adapter
	a, err := redisadapter.NewFromClient(suite.client)
	suite.NoError(err)

	// Create a new Enforcer, this time with the redis adapter
	e, err := casbin.NewEnforcer("files/model.conf", a)
	suite.NoError(err)

	// Add policy
	_, err = e.AddPolicy("bob", "data1", "read")
	suite.NoError(err)

	// Policy is available
	hasPol := e.HasPolicy("bob", "data1", "read")
	suite.True(hasPol)

	// Remove the policy
	_, err = e.RemovePolicy("bob", "data1", "read")
	suite.NoError(err)

	// Policy is gone
	hasPol = e.HasPolicy("bob", "data1", "read")
	suite.False(hasPol)
}

// SetupTest spins up a Redis instance after each test
func (suite *RedisAdapterTestSuite) SetupTest() {
	var err error

	suite.client, suite.purgeRedis, err = startRedis()
	if err != nil {
		suite.Failf("failed to start redis", err.Error())
	}
}

// TearDownTest purges the Redis instance after each test
func (suite *RedisAdapterTestSuite) TearDownTest() {
	err := suite.purgeRedis()
	if err != nil {
		suite.Failf("failed to purgeRedis redis", err.Error())
	}
}

func TestAdapter_TestSuite(t *testing.T) {
	suite.Run(t, new(RedisAdapterTestSuite))
}

// Checks if two models are equal
func assertModelEqual(t *testing.T, m0, m1 model.Model) {
	t.Helper()

	for sec, assertionMap := range m0 {
		for ptype, assertion := range assertionMap {
			for i, rule := range assertion.Policy {
				assert.Equal(t, m1[sec][ptype].Policy[i], rule)
			}
		}
	}
}

// Uses dockertest to spin up a new redis instance
func startRedis() (client *redis.Client, purge func() error, err error) {
	ctx := context.Background()

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}

	resource, err := pool.Run(redisImage, redisVersion, nil)
	if err != nil {
		return nil, nil, err
	}

	err = pool.Retry(func() error {
		client = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", resource.GetPort("6379/tcp")),
		})

		return client.Ping(ctx).Err()
	})
	if err != nil {
		return nil, nil, err
	}

	purge = func() error {
		_ = client.Close()
		if err = pool.Purge(resource); err != nil {
			return err
		}
		return nil
	}

	return
}
