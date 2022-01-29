package http

import (
	"expvar"
	"strconv"
	"sync/atomic"

	"github.com/go-graphite/carbonapi/cache"
	"github.com/go-graphite/carbonapi/cmd/carbonapi/config"
	zipperTypes "github.com/go-graphite/carbonapi/zipper/types"
	"go.uber.org/zap"

	"github.com/msaf1980/g2g/pkg/expvars"
	"github.com/msaf1980/g2gcounters"
)

var ApiMetrics = struct {
	Requests *expvars.Int
	// Timeouts *expvars.Int
	Errors *expvars.Int

	RenderRequests *expvars.Int
	// TODO: implement timeouts stat
	// RenderTimeouts        *expvars.Int
	RenderErrors          *expvars.Int
	RequestCacheHits      *expvars.Int
	RequestCacheMisses    *expvars.Int
	BackendCacheHits      *expvars.Int
	BackendCacheMisses    *expvars.Int
	RenderCacheOverheadNS *expvars.Int
	RequestBuckets        expvar.Func

	FindRequests *expvars.Int
	// TODO: implement timeouts stat
	// FindTimeouts        *expvars.Int
	FindErrors *expvars.Int

	// TODO: implements info stat
	// InfoRequests *expvars.Int
	// InfoTimeouts *expvars.Int
	// InfoErrors   *expvars.Int

	MemcacheTimeouts expvar.Func

	CacheSize  expvar.Func
	CacheItems expvar.Func

	RenderRequestsTime *g2gcounters.Timer // all queries (non-response cached)
	RenderRequestsSize *g2gcounters.Timer

	RenderCounter200 *g2gcounters.ERate
	RenderCounter400 *g2gcounters.ERate
	RenderCounter403 *g2gcounters.ERate
	RenderCounter404 *g2gcounters.ERate
	RenderCounter500 *g2gcounters.ERate
	RenderCounter502 *g2gcounters.ERate
	RenderCounter503 *g2gcounters.ERate
	RenderCounter504 *g2gcounters.ERate

	FindRequestsTime *g2gcounters.Timer // all queries (non-response cached)
	FindRequestsSize *g2gcounters.Timer

	FindCounter200 *g2gcounters.ERate
	FindCounter400 *g2gcounters.ERate
	FindCounter403 *g2gcounters.ERate
	FindCounter404 *g2gcounters.ERate
	FindCounter500 *g2gcounters.ERate
	FindCounter502 *g2gcounters.ERate
	FindCounter503 *g2gcounters.ERate
	FindCounter504 *g2gcounters.ERate
}{
	Requests: expvars.NewInt("requests"),
	Errors:   expvars.NewInt("errors"),
	// TODO: request_cache -> render_cache
	RenderRequests:        expvars.NewInt("render_requests"),
	RenderErrors:          expvars.NewInt("render_errors"),
	RequestCacheHits:      expvars.NewInt("request_cache_hits"),
	RequestCacheMisses:    expvars.NewInt("request_cache_misses"),
	BackendCacheHits:      expvars.NewInt("backend_cache_hits"),
	BackendCacheMisses:    expvars.NewInt("backend_cache_misses"),
	RenderCacheOverheadNS: expvars.NewInt("render_cache_overhead_ns"),

	FindRequests: expvars.NewInt("find_requests"),
	FindErrors:   expvars.NewInt("find_errors"),

	RenderRequestsTime: g2gcounters.NewTimer("render_requests_time"),
	RenderRequestsSize: g2gcounters.NewTimer("render_requests_size"),

	RenderCounter200: g2gcounters.NewERate("render_requests_status.200"),
	RenderCounter400: g2gcounters.NewERate("render_requests_status.400"),
	RenderCounter403: g2gcounters.NewERate("render_requests_status.403"),
	RenderCounter404: g2gcounters.NewERate("render_requests_status.404"),
	RenderCounter500: g2gcounters.NewERate("render_requests_status.500"),
	RenderCounter502: g2gcounters.NewERate("render_requests_status.502"),
	RenderCounter503: g2gcounters.NewERate("render_requests_status.503"),
	RenderCounter504: g2gcounters.NewERate("render_requests_status.504"),

	FindRequestsTime: g2gcounters.NewTimer("find_requests_time"),
	FindRequestsSize: g2gcounters.NewTimer("find_requests_size"),

	FindCounter200: g2gcounters.NewERate("find_requests_status.200"),
	FindCounter400: g2gcounters.NewERate("find_requests_status.400"),
	FindCounter403: g2gcounters.NewERate("find_requests_status.403"),
	FindCounter404: g2gcounters.NewERate("find_requests_status.404"),
	FindCounter500: g2gcounters.NewERate("find_requests_status.500"),
	FindCounter502: g2gcounters.NewERate("find_requests_status.502"),
	FindCounter503: g2gcounters.NewERate("find_requests_status.503"),
	FindCounter504: g2gcounters.NewERate("find_requests_status.504"),
}

var ZipperMetrics = struct {
	FindRequests *expvars.Int
	FindTimeouts *expvars.Int
	FindErrors   *expvars.Int

	SearchRequests *expvars.Int

	RenderRequests *expvars.Int
	RenderTimeouts *expvars.Int
	RenderErrors   *expvars.Int

	InfoRequests *expvars.Int
	InfoTimeouts *expvars.Int
	InfoErrors   *expvars.Int

	Timeouts *expvars.Int

	CacheSize   expvar.Func
	CacheItems  expvar.Func
	CacheMisses *expvars.Int
	CacheHits   *expvars.Int
}{
	FindRequests: expvars.NewInt("zipper_find_requests"),
	FindTimeouts: expvars.NewInt("zipper_find_timeouts"),
	FindErrors:   expvars.NewInt("zipper_find_errors"),

	SearchRequests: expvars.NewInt("zipper_search_requests"),

	RenderRequests: expvars.NewInt("zipper_render_requests"),
	RenderTimeouts: expvars.NewInt("zipper_render_timeouts"),
	RenderErrors:   expvars.NewInt("zipper_render_errors"),

	InfoRequests: expvars.NewInt("zipper_info_requests"),
	InfoTimeouts: expvars.NewInt("zipper_info_timeouts"),
	InfoErrors:   expvars.NewInt("zipper_info_errors"),

	Timeouts: expvars.NewInt("zipper_timeouts"),

	CacheHits:   expvars.NewInt("zipper_cache_hits"),
	CacheMisses: expvars.NewInt("zipper_cache_misses"),
}

func ZipperStats(stats *zipperTypes.Stats) {
	if stats == nil {
		return
	}
	ZipperMetrics.Timeouts.Add(stats.Timeouts)
	ZipperMetrics.FindRequests.Add(stats.FindRequests)
	ZipperMetrics.FindTimeouts.Add(stats.FindTimeouts)
	ZipperMetrics.FindErrors.Add(stats.FindErrors)
	ZipperMetrics.RenderRequests.Add(stats.RenderRequests)
	ZipperMetrics.RenderTimeouts.Add(stats.RenderTimeouts)
	ZipperMetrics.RenderErrors.Add(stats.RenderErrors)
	ZipperMetrics.InfoRequests.Add(stats.InfoRequests)
	ZipperMetrics.InfoTimeouts.Add(stats.InfoTimeouts)
	ZipperMetrics.InfoErrors.Add(stats.InfoErrors)
	ZipperMetrics.SearchRequests.Add(stats.SearchRequests)
	ZipperMetrics.CacheMisses.Add(stats.CacheMisses)
	ZipperMetrics.CacheHits.Add(stats.CacheHits)
}

type BucketEntry int

var TimeBuckets []int64

func (b BucketEntry) String() string {
	return strconv.Itoa(int(atomic.LoadInt64(&TimeBuckets[b])))
}

func RenderTimeBuckets() interface{} {
	return TimeBuckets
}

func SetupMetrics(logger *zap.Logger) {
	switch config.Config.ResponseCacheConfig.Type {
	case "memcache":
		mcache := config.Config.ResponseCache.(*cache.MemcachedCache)

		ApiMetrics.MemcacheTimeouts = expvar.Func(func() interface{} {
			return mcache.Timeouts()
		})
		expvar.Publish("memcache_timeouts", ApiMetrics.MemcacheTimeouts)

	case "mem":
		qcache := config.Config.ResponseCache.(*cache.ExpireCache)

		ApiMetrics.CacheSize = expvar.Func(func() interface{} {
			return qcache.Size()
		})
		expvar.Publish("cache_size", ApiMetrics.CacheSize)

		ApiMetrics.CacheItems = expvar.Func(func() interface{} {
			return qcache.Items()
		})
		expvar.Publish("cache_items", ApiMetrics.CacheItems)
	default:
	}

	// +1 to track every over the number of buckets we track
	TimeBuckets = make([]int64, config.Config.Upstreams.Buckets+1)
	expvar.Publish("requestBuckets", expvar.Func(RenderTimeBuckets))
}
