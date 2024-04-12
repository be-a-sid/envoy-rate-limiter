package main

import (
	"fmt"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
	"github.com/go-redis/redis"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/types/known/anypb"
)

const Name = "rate-limiter"

func init() {
	fmt.Printf("Registering Plugin - %v \n", Name)
	http.RegisterHttpFilterConfigFactoryAndParser(Name, ConfigFactory, &parser{})
}

type parser struct {
}

type configuration struct {
	routeSpecificRateLimitConfig []routeSpecificRateLimitConfigT
	redisConfig                  redisConfigT
	redisClient                  *redis.Client
}

type routeSpecificRateLimitConfigT struct {
	Key             string `json:key`
	BucketSize      int    `json:bucketSize`
	RefillRateInSec int    `json:refillRateInSec`
}

type redisConfigT struct {
	Address string `json:key`
}

func (p parser) Parse(any *anypb.Any, callbacks api.ConfigCallbackHandler) (interface{}, error) {
	configStruct := &xds.TypedStruct{}
	if err := any.UnmarshalTo(configStruct); err != nil {
		return nil, err
	}

	destructedValue := configStruct.Value
	var config configuration
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	if routeSpecificRateLimitConfigString, ok := destructedValue.AsMap()["routeSpecificRateLimitConfig"].(string); ok {
		routeSpecificRateLimitConfigVal := make([]routeSpecificRateLimitConfigT, 0)
		err := json.UnmarshalFromString(routeSpecificRateLimitConfigString, &routeSpecificRateLimitConfigVal)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the routeSpecificRateLimitConfig")
		}

		// Check if the cache key contains valid configuration
		errorValidateJWTClaimInKey := validateRouteConfigKeys(routeSpecificRateLimitConfigVal)
		if errorValidateJWTClaimInKey != nil {
			return nil, errorValidateJWTClaimInKey
		}
		config.routeSpecificRateLimitConfig = routeSpecificRateLimitConfigVal
	}

	if redisConfigString, ok := destructedValue.AsMap()["redisConfig"].(string); ok {
		redisConfigVal := redisConfigT{}
		err := json.UnmarshalFromString(redisConfigString, &redisConfigVal)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the routeSpecificRateLimitConfig")
		}
		config.redisConfig = redisConfigVal
	}

	if config.redisConfig.Address == "" {
		return nil, fmt.Errorf("redisConfig Address cannot be empty")
	}

	redisClient := createRedisClient(config.redisConfig.Address)
	if redisClient != nil {
		config.redisClient = redisClient
	}

	fmt.Printf("Parsed Config %v \n", config)
	return &config, nil
}

func (p parser) Merge(parentConfig interface{}, childConfig interface{}) interface{} {
	panic("not implemented")
}

func ConfigFactory(c interface{}) api.StreamFilterFactory {
	conf, ok := c.(*configuration)
	if !ok {
		panic("Unexpected configuration provided")
	}
	return func(callbacks api.FilterCallbackHandler) api.StreamFilter {
		return &filter{
			callbacks: callbacks,
			conf:      *conf,
		}
	}
}
