package main

import (
	//"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
	"github.com/samber/lo"
)

const rateLimiterLuaScript = `
	local key = KEYS[1]
	local capacity = tonumber(KEYS[2]) -- Maximum allowed requests within the time window
	local rate = tonumber(KEYS[3])  -- Rate of token generation (per second)

	local currentTime = redis.call("TIME")[1]

  	local retrievedValue = redis.call("HMGET", key, "last_access", "tokens")
  	local lastAccess = tonumber(retrievedValue[1] or 0)
	local tokens = tonumber(retrievedValue[2] or capacity)

	local elapsedTime = math.max(0, currentTime - lastAccess)

	local refillAmount = math.min(elapsedTime * rate, capacity - tokens)

	tokens = math.min(capacity, tokens + refillAmount)

	if tokens > 0 then
    	redis.call("HMSET", key, "last_access", currentTime, "tokens", tokens - 1)
    	return 1
	else
    	return 0
	end
  `

func hasMatchingMethodPath(apiEndpoint, testEndpoint string) bool {
	// Split the paths by '/'
	apiParts := strings.Split(apiEndpoint, "/")
	testParts := strings.Split(testEndpoint, "/")

	// Ensure the lengths are compatible
	if len(testParts) != len(apiParts) {
		return false
	}

	// Check each segment for equality or placeholder
	for i := range apiParts {
		if apiParts[i] != testParts[i] && !strings.HasPrefix(apiParts[i], ":") {
			return false
		}
	}

	return true
}

func getRateLimitConfigByKey(conf *configuration, reqHttpMethod, reqHttpPath string, decodedJWTToken *map[string]interface{}) *routeSpecificRateLimitConfigT {
	var shortListedConfig = routeSpecificRateLimitConfigT{}
	rateLimitMap := conf.routeSpecificRateLimitConfig
	reqKey := strings.ToLower(fmt.Sprintf("%v--%v", reqHttpMethod, reqHttpPath))
	for _, configPair := range rateLimitMap {
		keyMapping := strings.Split(configPair.Key, "--")
		routePathKey := fmt.Sprintf("%v--%v", keyMapping[0], keyMapping[1])
		isMatch := hasMatchingMethodPath(routePathKey, reqKey)
		if isMatch {
			if len(keyMapping) == 3 {
				jwtBasedClaimKey := strings.Replace(keyMapping[2], "jwt.", "", 1)
				// fmt.Printf("jwtBasedClaimKey : %v \n", jwtBasedClaimKey)
				claimVal, claimValExists := (*decodedJWTToken)[jwtBasedClaimKey]
				if claimValExists {
					shortListedConfig.Key = fmt.Sprintf("%v--%v", routePathKey, claimVal)
					shortListedConfig.BucketSize = configPair.BucketSize
					shortListedConfig.RefillRateInSec = configPair.RefillRateInSec
				}
			} else {
				shortListedConfig = configPair
			}
			// fmt.Printf("shortListedConfig is : %v \n", shortListedConfig)
			break
		}
	}
	return &shortListedConfig
}

func shouldRateLimitRequest(redisClient *redis.Client, rConfig routeSpecificRateLimitConfigT) bool {
	// fmt.Printf("Inside blockRequest %v \n", rConfig)
	keys := []string{rConfig.Key, fmt.Sprint(rConfig.BucketSize), fmt.Sprint(rConfig.RefillRateInSec)}
	args := []string{}
	isAllowedVal, err := redisClient.Eval(rateLimiterLuaScript, keys, args).Result()
	if err != nil {
		// Redis client returns nil if key is not present
		if err == redis.Nil {
			// fmt.Println("Key is not present")
			return false
		}
		fmt.Printf("error occurred while retrieving the key %v", err)
		return false //don't rateLimit in case of errors
	}
	isAllowedIntVal := isAllowedVal.(int64)
	// fmt.Printf("retrieved val is %v \n", isAllowedIntVal)
	// Allow for 1 & deny for any other
	return isAllowedIntVal != 1
}

func generateRateLimitResponse() string {
	rateLimitResponse := map[string]string{
		"message": "Too many Requests",
	}
	bodyBytes, _ := json.Marshal(rateLimitResponse)
	stringifiedResponse := string(bodyBytes[:])
	return stringifiedResponse
}

func createRedisClient(address string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:       address,
		MaxRetries: 1,
	})

	// Ping Redis to check if the connection is working
	_, err := client.Ping().Result()
	if err != nil {
		fmt.Printf("Unable to connect to Redis %v \n", err)
		return nil
	}

	return client
}

func validateRouteConfigKeys(routeSpecificRateLimitConfigVal []routeSpecificRateLimitConfigT) error {
	validJWTKeys := []string{"jwt.sub", "jwt.iss"}
	validHTTPMethods := []string{"get", "post", "patch", "delete", "put"}

	for _, routeConfig := range routeSpecificRateLimitConfigVal {

		rateLimitKeySet := strings.Split(routeConfig.Key, "--")
		rateLimitKeyLen := len(rateLimitKeySet)

		if rateLimitKeyLen < 1 || rateLimitKeyLen > 3 {
			return fmt.Errorf("invalid rateLimitKey %v", routeConfig.Key)
		}

		rateLimitKeyMethod := rateLimitKeySet[0]
		if !lo.Contains(validHTTPMethods, rateLimitKeyMethod) {
			return fmt.Errorf("invalid rateLimitKey : %v - Unsupported HTTP Method : %v ", routeConfig.Key, rateLimitKeyMethod)
		}

		if rateLimitKeyLen == 3 {
			rateLimitKeyJWT := rateLimitKeySet[2]
			if !lo.Contains(validJWTKeys, rateLimitKeyJWT) {
				return fmt.Errorf("invalid rateLimitKey : %v -  Unsupported JWT Claim : %v", routeConfig.Key, rateLimitKeyJWT)
			}
		}
	}
	return nil
}
