package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-graphite/carbonapi/cmd/carbonapi/config"
	"github.com/go-graphite/carbonapi/cmd/carbonapi/http"
	"github.com/go-graphite/carbonapi/mstats"
	"github.com/msaf1980/g2g"
	"go.uber.org/zap"
)

func setupGraphiteMetrics(logger *zap.Logger) {
	var host string
	if envhost := os.Getenv("GRAPHITEHOST") + ":" + os.Getenv("GRAPHITEPORT"); envhost != ":" || config.Config.Graphite.Host != "" {
		switch {
		case envhost != ":" && config.Config.Graphite.Host != "":
			host = config.Config.Graphite.Host
		case envhost != ":":
			host = envhost
		case config.Config.Graphite.Host != "":
			host = config.Config.Graphite.Host
		}
	}

	logger.Info("starting carbonapi",
		zap.String("build_version", BuildVersion),
		zap.Any("config", config.Config),
	)

	if host != "" {
		// register our metrics with graphite
		graphite := g2g.NewGraphiteBatch(host, config.Config.Graphite.Interval, 10*time.Second, config.Config.Graphite.BatchSize)

		hostname, _ := os.Hostname()
		hostname = strings.ReplaceAll(hostname, ".", "_")

		prefix := config.Config.Graphite.Prefix

		pattern := config.Config.Graphite.Pattern
		pattern = strings.ReplaceAll(pattern, "{prefix}", prefix)
		pattern = strings.ReplaceAll(pattern, "{fqdn}", hostname)

		graphite.Register(fmt.Sprintf("%s.requests", pattern), http.ApiMetrics.Requests)
		graphite.Register(fmt.Sprintf("%s.errors", pattern), http.ApiMetrics.Errors)
		graphite.Register(fmt.Sprintf("%s.request_cache_hits", pattern), http.ApiMetrics.RequestCacheHits)
		graphite.Register(fmt.Sprintf("%s.request_cache_misses", pattern), http.ApiMetrics.RequestCacheMisses)
		graphite.Register(fmt.Sprintf("%s.request_cache_overhead_ns", pattern), http.ApiMetrics.RenderCacheOverheadNS)
		graphite.Register(fmt.Sprintf("%s.backend_cache_hits", pattern), http.ApiMetrics.BackendCacheHits)
		graphite.Register(fmt.Sprintf("%s.backend_cache_misses", pattern), http.ApiMetrics.BackendCacheMisses)

		for i := 0; i <= config.Config.Upstreams.Buckets; i++ {
			graphite.Register(fmt.Sprintf("%s.requests_in_%dms_to_%dms", pattern, i*100, (i+1)*100), http.BucketEntry(i))
		}

		graphite.Register(fmt.Sprintf("%s.find_requests", pattern), http.ApiMetrics.FindRequests)
		graphite.Register(fmt.Sprintf("%s.find_errors", pattern), http.ApiMetrics.FindErrors)
		graphite.Register(fmt.Sprintf("%s.render_requests", pattern), http.ApiMetrics.RenderRequests)
		graphite.Register(fmt.Sprintf("%s.render_errors", pattern), http.ApiMetrics.RenderErrors)

		if config.Config.Graphite.ExtendedStat {
			graphite.MRegister(fmt.Sprintf("%s.render_requests_time", pattern), http.ApiMetrics.RenderRequestsTime)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_size", pattern), http.ApiMetrics.RenderRequestsSize)

			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.200", pattern), http.ApiMetrics.RenderCounter200)

			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.400", pattern), http.ApiMetrics.RenderCounter400)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.403", pattern), http.ApiMetrics.RenderCounter403)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.404", pattern), http.ApiMetrics.RenderCounter404)

			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.500", pattern), http.ApiMetrics.RenderCounter500)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.502", pattern), http.ApiMetrics.RenderCounter502)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.503", pattern), http.ApiMetrics.RenderCounter503)
			graphite.MRegister(fmt.Sprintf("%s.render_requests_status.504", pattern), http.ApiMetrics.RenderCounter504)

			graphite.MRegister(fmt.Sprintf("%s.find_requests_time", pattern), http.ApiMetrics.FindRequestsTime)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_size", pattern), http.ApiMetrics.FindRequestsSize)

			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.200", pattern), http.ApiMetrics.FindCounter200)

			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.400", pattern), http.ApiMetrics.FindCounter400)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.403", pattern), http.ApiMetrics.FindCounter403)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.404", pattern), http.ApiMetrics.FindCounter404)

			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.500", pattern), http.ApiMetrics.FindCounter500)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.502", pattern), http.ApiMetrics.FindCounter502)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.503", pattern), http.ApiMetrics.FindCounter503)
			graphite.MRegister(fmt.Sprintf("%s.find_requests_status.504", pattern), http.ApiMetrics.FindCounter504)
		}

		if http.ApiMetrics.MemcacheTimeouts != nil {
			graphite.Register(fmt.Sprintf("%s.memcache_timeouts", pattern), http.ApiMetrics.MemcacheTimeouts)
		}

		if http.ApiMetrics.CacheSize != nil {
			graphite.Register(fmt.Sprintf("%s.cache_size", pattern), http.ApiMetrics.CacheSize)
			graphite.Register(fmt.Sprintf("%s.cache_items", pattern), http.ApiMetrics.CacheItems)
		}

		graphite.Register(fmt.Sprintf("%s.zipper.find_requests", pattern), http.ZipperMetrics.FindRequests)
		graphite.Register(fmt.Sprintf("%s.zipper.find_errors", pattern), http.ZipperMetrics.FindErrors)

		graphite.Register(fmt.Sprintf("%s.zipper.render_requests", pattern), http.ZipperMetrics.RenderRequests)
		graphite.Register(fmt.Sprintf("%s.zipper.render_errors", pattern), http.ZipperMetrics.RenderErrors)

		graphite.Register(fmt.Sprintf("%s.zipper.info_requests", pattern), http.ZipperMetrics.InfoRequests)
		graphite.Register(fmt.Sprintf("%s.zipper.info_errors", pattern), http.ZipperMetrics.InfoErrors)

		graphite.Register(fmt.Sprintf("%s.zipper.timeouts", pattern), http.ZipperMetrics.Timeouts)

		graphite.Register(fmt.Sprintf("%s.zipper.cache_hits", pattern), http.ZipperMetrics.CacheHits)
		graphite.Register(fmt.Sprintf("%s.zipper.cache_misses", pattern), http.ZipperMetrics.CacheMisses)

		go mstats.Start(config.Config.Graphite.Interval)
		graphite.Register(fmt.Sprintf("%s.alloc", pattern), &mstats.Alloc)
		graphite.Register(fmt.Sprintf("%s.total_alloc", pattern), &mstats.TotalAlloc)
		graphite.Register(fmt.Sprintf("%s.num_gc", pattern), &mstats.NumGC)
		graphite.Register(fmt.Sprintf("%s.pause_ns", pattern), &mstats.PauseNS)
	}
}
