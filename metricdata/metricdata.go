package metricdata

import (
	"context"
	"fmt"
	"time"

	"github.com/ClusterCockpit/cc-jobarchive/config"
	"github.com/ClusterCockpit/cc-jobarchive/schema"
	"github.com/iamlouk/lrucache"
)

type MetricDataRepository interface {
	// Initialize this MetricDataRepository. One instance of
	// this interface will only ever be responsible for one cluster.
	Init(url, token string) error

	// Return the JobData for the given job, only with the requested metrics.
	LoadData(job *schema.Job, metrics []string, scopes []schema.MetricScope, ctx context.Context) (schema.JobData, error)

	// Return a map of metrics to a map of nodes to the metric statistics of the job. node scope assumed for now.
	LoadStats(job *schema.Job, metrics []string, ctx context.Context) (map[string]map[string]schema.MetricStatistics, error)

	// Return a map of nodes to a map of metrics to the data for the requested time.
	LoadNodeData(clusterId string, metrics, nodes []string, from, to int64, ctx context.Context) (map[string]map[string][]schema.Float, error)
}

var metricDataRepos map[string]MetricDataRepository = map[string]MetricDataRepository{}

var JobArchivePath string

var useArchive bool

func Init(jobArchivePath string, disableArchive bool) error {
	useArchive = !disableArchive
	JobArchivePath = jobArchivePath
	for _, cluster := range config.Clusters {
		if cluster.MetricDataRepository != nil {
			switch cluster.MetricDataRepository.Kind {
			case "cc-metric-store":
				ccms := &CCMetricStore{}
				if err := ccms.Init(cluster.MetricDataRepository.Url, cluster.MetricDataRepository.Token); err != nil {
					return err
				}
				metricDataRepos[cluster.Name] = ccms
			// case "influxdb-v2":
			// 	idb := &InfluxDBv2DataRepository{}
			// 	if err := idb.Init(cluster.MetricDataRepository.Url); err != nil {
			// 		return err
			// 	}
			// 	metricDataRepos[cluster.Name] = idb
			default:
				return fmt.Errorf("unkown metric data repository '%s' for cluster '%s'", cluster.MetricDataRepository.Kind, cluster.Name)
			}
		}
	}
	return nil
}

var cache *lrucache.Cache = lrucache.New(500 * 1024 * 1024)

// Fetches the metric data for a job.
func LoadData(job *schema.Job, metrics []string, scopes []schema.MetricScope, ctx context.Context) (schema.JobData, error) {
	if job.State == schema.JobStateRunning || !useArchive {
		ckey := cacheKey(job, metrics, scopes)
		if data := cache.Get(ckey, nil); data != nil {
			return data.(schema.JobData), nil
		}

		repo, ok := metricDataRepos[job.Cluster]
		if !ok {
			return nil, fmt.Errorf("no metric data repository configured for '%s'", job.Cluster)
		}

		if scopes == nil {
			scopes = append(scopes, schema.MetricScopeNode)
		}

		if metrics == nil {
			cluster := config.GetClusterConfig(job.Cluster)
			for _, mc := range cluster.MetricConfig {
				metrics = append(metrics, mc.Name)
			}
		}

		data, err := repo.LoadData(job, metrics, scopes, ctx)
		if err != nil {
			return nil, err
		}

		// calcStatisticsSeries(job, data, 7)
		cache.Put(ckey, data, data.Size(), 2*time.Minute)
		return data, nil
	}

	data, err := loadFromArchive(job)
	if err != nil {
		return nil, err
	}

	if metrics != nil {
		res := schema.JobData{}
		for _, metric := range metrics {
			if metricdata, ok := data[metric]; ok {
				res[metric] = metricdata
			}
		}
		return res, nil
	}
	return data, nil
}

// Used for the jobsFootprint GraphQL-Query. TODO: Rename/Generalize.
func LoadAverages(job *schema.Job, metrics []string, data [][]schema.Float, ctx context.Context) error {
	if job.State != schema.JobStateRunning && useArchive {
		return loadAveragesFromArchive(job, metrics, data)
	}

	repo, ok := metricDataRepos[job.Cluster]
	if !ok {
		return fmt.Errorf("no metric data repository configured for '%s'", job.Cluster)
	}

	stats, err := repo.LoadStats(job, metrics, ctx)
	if err != nil {
		return err
	}

	for i, m := range metrics {
		nodes, ok := stats[m]
		if !ok {
			data[i] = append(data[i], schema.NaN)
			continue
		}

		sum := 0.0
		for _, node := range nodes {
			sum += node.Avg
		}
		data[i] = append(data[i], schema.Float(sum))
	}

	return nil
}

// Used for the node/system view. Returns a map of nodes to a map of metrics (at node scope).
func LoadNodeData(clusterId string, metrics, nodes []string, from, to int64, ctx context.Context) (map[string]map[string][]schema.Float, error) {
	repo, ok := metricDataRepos[clusterId]
	if !ok {
		return nil, fmt.Errorf("no metric data repository configured for '%s'", clusterId)
	}

	if metrics == nil {
		for _, m := range config.GetClusterConfig(clusterId).MetricConfig {
			metrics = append(metrics, m.Name)
		}
	}

	data, err := repo.LoadNodeData(clusterId, metrics, nodes, from, to, ctx)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, fmt.Errorf("the metric data repository for '%s' does not support this query", clusterId)
	}

	return data, nil
}

func cacheKey(job *schema.Job, metrics []string, scopes []schema.MetricScope) string {
	// Duration and StartTime do not need to be in the cache key as StartTime is less unique than
	// job.ID and the TTL of the cache entry makes sure it does not stay there forever.
	return fmt.Sprintf("%d:[%v],[%v]",
		job.ID, metrics, scopes)
}