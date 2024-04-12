package main

import (
	"fmt"
	"strings"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type filter struct {
	callbacks api.FilterCallbackHandler
	conf      configuration
}

func (f *filter) DecodeHeaders(headerMap api.RequestHeaderMap, endStream bool) api.StatusType {
	httpMethod := headerMap.Method()
	httpPath := strings.Trim(headerMap.Path(), "/")
	reqHttpMethodRouteKey := strings.ToLower(fmt.Sprintf("%v--%v", httpMethod, httpPath))
	// f.callbacks.Log(api.Info, fmt.Sprintf(" Inside Decode Headers . Key : %v , conf : %v", reqKey, f.conf.routeSpecificRateLimitConfig))

	jwtAuthn := f.callbacks.StreamInfo().DynamicMetadata().Get("envoy.filters.http.jwt_authn")
	decodedJWTToken, _ := jwtAuthn["decodedJWTToken"].(map[string]interface{})
	rateLimitConfigByKey := *(getRateLimitConfigByKey(&f.conf, httpMethod, httpPath, &decodedJWTToken))
	rateLimitRequest := false
	endpointKey := fmt.Sprintf("%v--%v", httpMethod, httpPath)
	f.callbacks.Log(api.Info, fmt.Sprintf("For the endpoint : %v, finalized rateLimitConfig is Key : %v, BucketSize : %v , RefillRateInSec : %v \n", endpointKey, rateLimitConfigByKey.Key, rateLimitConfigByKey.BucketSize, rateLimitConfigByKey.RefillRateInSec))

	// If there is a valid configuration
	if (rateLimitConfigByKey != routeSpecificRateLimitConfigT{}) {
		rateLimitRequest = shouldRateLimitRequest(f.conf.redisClient, rateLimitConfigByKey)
		fmt.Printf("Based on the configuration, reqHttpMethodRouteKey %v -- shouldRateLimit %v\n", reqHttpMethodRouteKey, rateLimitRequest)
	}

	if rateLimitRequest {
		customResponse := generateRateLimitResponse()
		f.callbacks.Log(api.Info, fmt.Sprintf("RateLimiting the Request with key : %v ", rateLimitConfigByKey.Key))
		resHeaderMap := map[string][]string{
			"content-type": {"application/json"},
		}
		f.callbacks.SendLocalReply(429, customResponse, resHeaderMap, 0, "")
		return api.LocalReply
	}

	return api.Continue
}

func (f *filter) DecodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) DecodeTrailers(trailerMap api.RequestTrailerMap) api.StatusType {
	return api.Continue
}

func (f filter) EncodeHeaders(headerMap api.ResponseHeaderMap, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeData(buffer api.BufferInstance, endStream bool) api.StatusType {
	return api.Continue
}

func (f *filter) EncodeTrailers(trailerMap api.ResponseTrailerMap) api.StatusType {
	return api.Continue
}

func (f *filter) OnLog() {
}

func (f *filter) OnDestroy(reason api.DestroyReason) {
}

func (f *filter) OnLogDownstreamStart() {
}

func (f *filter) OnLogDownstreamPeriodic() {
}

func main() {
}
