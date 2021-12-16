// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/ClusterCockpit/cc-jobarchive/schema"
)

type FilterRanges struct {
	Duration  *IntRangeOutput  `json:"duration"`
	NumNodes  *IntRangeOutput  `json:"numNodes"`
	StartTime *TimeRangeOutput `json:"startTime"`
}

type FloatRange struct {
	From float64 `json:"from"`
	To   float64 `json:"to"`
}

type HistoPoint struct {
	Count int `json:"count"`
	Value int `json:"value"`
}

type IntRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type IntRangeOutput struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type Job struct {
	ID               string                `json:"Id"`
	JobID            int                   `json:"JobId"`
	User             string                `json:"User"`
	Project          string                `json:"Project"`
	Cluster          string                `json:"Cluster"`
	StartTime        time.Time             `json:"StartTime"`
	Duration         int                   `json:"Duration"`
	NumNodes         int                   `json:"NumNodes"`
	NumHWThreads     int                   `json:"NumHWThreads"`
	NumAcc           int                   `json:"NumAcc"`
	Smt              int                   `json:"SMT"`
	Exclusive        int                   `json:"Exclusive"`
	Partition        string                `json:"Partition"`
	ArrayJobID       int                   `json:"ArrayJobId"`
	MonitoringStatus int                   `json:"MonitoringStatus"`
	State            JobState              `json:"State"`
	Tags             []*JobTag             `json:"Tags"`
	Resources        []*schema.JobResource `json:"Resources"`
	LoadAvg          *float64              `json:"LoadAvg"`
	MemUsedMax       *float64              `json:"MemUsedMax"`
	FlopsAnyAvg      *float64              `json:"FlopsAnyAvg"`
	MemBwAvg         *float64              `json:"MemBwAvg"`
	NetBwAvg         *float64              `json:"NetBwAvg"`
	FileBwAvg        *float64              `json:"FileBwAvg"`
}

type JobFilter struct {
	Tags        []string     `json:"tags"`
	JobID       *StringInput `json:"jobId"`
	User        *StringInput `json:"user"`
	Project     *StringInput `json:"project"`
	Cluster     *StringInput `json:"cluster"`
	Duration    *IntRange    `json:"duration"`
	NumNodes    *IntRange    `json:"numNodes"`
	StartTime   *TimeRange   `json:"startTime"`
	JobState    []JobState   `json:"jobState"`
	FlopsAnyAvg *FloatRange  `json:"flopsAnyAvg"`
	MemBwAvg    *FloatRange  `json:"memBwAvg"`
	LoadAvg     *FloatRange  `json:"loadAvg"`
	MemUsedMax  *FloatRange  `json:"memUsedMax"`
}

type JobMetricWithName struct {
	Name   string            `json:"name"`
	Metric *schema.JobMetric `json:"metric"`
}

type JobResultList struct {
	Items  []*Job `json:"items"`
	Offset *int   `json:"offset"`
	Limit  *int   `json:"limit"`
	Count  *int   `json:"count"`
}

type JobsStatistics struct {
	ID             string        `json:"id"`
	TotalJobs      int           `json:"totalJobs"`
	ShortJobs      int           `json:"shortJobs"`
	TotalWalltime  int           `json:"totalWalltime"`
	TotalCoreHours int           `json:"totalCoreHours"`
	HistWalltime   []*HistoPoint `json:"histWalltime"`
	HistNumNodes   []*HistoPoint `json:"histNumNodes"`
}

type MetricConfig struct {
	Name     string `json:"Name"`
	Unit     string `json:"Unit"`
	Timestep int    `json:"Timestep"`
	Peak     int    `json:"Peak"`
	Normal   int    `json:"Normal"`
	Caution  int    `json:"Caution"`
	Alert    int    `json:"Alert"`
	Scope    string `json:"Scope"`
}

type MetricFootprints struct {
	Name       string         `json:"name"`
	Footprints []schema.Float `json:"footprints"`
}

type NodeMetric struct {
	Name string         `json:"name"`
	Data []schema.Float `json:"data"`
}

type NodeMetrics struct {
	ID      string        `json:"id"`
	Metrics []*NodeMetric `json:"metrics"`
}

type OrderByInput struct {
	Field string            `json:"field"`
	Order SortDirectionEnum `json:"order"`
}

type PageRequest struct {
	ItemsPerPage int `json:"itemsPerPage"`
	Page         int `json:"page"`
}

type StringInput struct {
	Eq         *string `json:"eq"`
	Contains   *string `json:"contains"`
	StartsWith *string `json:"startsWith"`
	EndsWith   *string `json:"endsWith"`
}

type TimeRange struct {
	From *time.Time `json:"from"`
	To   *time.Time `json:"to"`
}

type TimeRangeOutput struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type Aggregate string

const (
	AggregateUser    Aggregate = "USER"
	AggregateProject Aggregate = "PROJECT"
	AggregateCluster Aggregate = "CLUSTER"
)

var AllAggregate = []Aggregate{
	AggregateUser,
	AggregateProject,
	AggregateCluster,
}

func (e Aggregate) IsValid() bool {
	switch e {
	case AggregateUser, AggregateProject, AggregateCluster:
		return true
	}
	return false
}

func (e Aggregate) String() string {
	return string(e)
}

func (e *Aggregate) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Aggregate(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Aggregate", str)
	}
	return nil
}

func (e Aggregate) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type JobState string

const (
	JobStateRunning   JobState = "running"
	JobStateCompleted JobState = "completed"
	JobStateFailed    JobState = "failed"
	JobStateCanceled  JobState = "canceled"
	JobStateStopped   JobState = "stopped"
	JobStateTimeout   JobState = "timeout"
)

var AllJobState = []JobState{
	JobStateRunning,
	JobStateCompleted,
	JobStateFailed,
	JobStateCanceled,
	JobStateStopped,
	JobStateTimeout,
}

func (e JobState) IsValid() bool {
	switch e {
	case JobStateRunning, JobStateCompleted, JobStateFailed, JobStateCanceled, JobStateStopped, JobStateTimeout:
		return true
	}
	return false
}

func (e JobState) String() string {
	return string(e)
}

func (e *JobState) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = JobState(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid JobState", str)
	}
	return nil
}

func (e JobState) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type SortDirectionEnum string

const (
	SortDirectionEnumDesc SortDirectionEnum = "DESC"
	SortDirectionEnumAsc  SortDirectionEnum = "ASC"
)

var AllSortDirectionEnum = []SortDirectionEnum{
	SortDirectionEnumDesc,
	SortDirectionEnumAsc,
}

func (e SortDirectionEnum) IsValid() bool {
	switch e {
	case SortDirectionEnumDesc, SortDirectionEnumAsc:
		return true
	}
	return false
}

func (e SortDirectionEnum) String() string {
	return string(e)
}

func (e *SortDirectionEnum) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SortDirectionEnum(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SortDirectionEnum", str)
	}
	return nil
}

func (e SortDirectionEnum) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
