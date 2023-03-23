// Copyright (C) 2022 NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ClusterCockpit/cc-backend/pkg/schema"
	"github.com/ClusterCockpit/cc-backend/pkg/units"
)

var ar FsArchive

func loadJobData(filename string) (*JobData, error) {

	f, err := os.Open(filename)
	if err != nil {
		return &JobData{}, fmt.Errorf("fsBackend loadJobData()- %v", err)
	}
	defer f.Close()

	return DecodeJobData(bufio.NewReader(f))
}

func deepCopyJobMeta(j *JobMeta) schema.JobMeta {
	var jn schema.JobMeta

	jn.StartTime = j.StartTime
	jn.User = j.User
	jn.Project = j.Project
	jn.Cluster = j.Cluster
	jn.SubCluster = j.SubCluster
	jn.Partition = j.Partition
	jn.ArrayJobId = j.ArrayJobId
	jn.NumNodes = j.NumNodes
	jn.NumHWThreads = j.NumHWThreads
	jn.NumAcc = j.NumAcc
	jn.Exclusive = j.Exclusive
	jn.MonitoringStatus = j.MonitoringStatus
	jn.SMT = j.SMT
	jn.Duration = j.Duration
	jn.Walltime = j.Walltime
	jn.State = schema.JobState(j.State)
	jn.Exclusive = j.Exclusive
	jn.Exclusive = j.Exclusive
	jn.Exclusive = j.Exclusive

	for _, ro := range j.Resources {
		var rn schema.Resource
		rn.Hostname = ro.Hostname
		rn.Configuration = ro.Configuration
		hwt := make([]int, len(ro.HWThreads))
		copy(hwt, ro.HWThreads)
		acc := make([]string, len(ro.Accelerators))
		copy(acc, ro.Accelerators)
		jn.Resources = append(jn.Resources, &rn)
	}

	for k, v := range j.MetaData {
		jn.MetaData[k] = v
	}

	return jn
}

func deepCopyJobData(d *JobData) schema.JobData {
	var dn = make(schema.JobData)

	for k, v := range *d {
		for mk, mv := range v {
			var mn schema.JobMetric
			mn.Unit = units.ConvertUnitString(mv.Unit)
			mn.Timestep = mv.Timestep

			for _, v := range mv.Series {
				var sn schema.Series
				sn.Hostname = v.Hostname
				if v.Id != nil {
					var id = new(string)
					*id = fmt.Sprint(*v.Id)
					sn.Id = id
				}
				if v.Statistics != nil {
					sn.Statistics = schema.MetricStatistics{
						Avg: v.Statistics.Avg,
						Min: v.Statistics.Min,
						Max: v.Statistics.Max}
				}

				sn.Data = make([]schema.Float, len(v.Data))
				copy(sn.Data, v.Data)
				mn.Series = append(mn.Series, sn)
			}

			dn[k] = make(map[schema.MetricScope]*schema.JobMetric)
			dn[k][mk] = &mn
		}
	}

	return dn
}

func deepCopyClusterConfig(co *Cluster) schema.Cluster {
	var cn schema.Cluster

	cn.Name = co.Name
	for _, sco := range co.SubClusters {
		var scn schema.SubCluster
		scn.Name = sco.Name
		if sco.Nodes == "" {
			scn.Nodes = "*"
		} else {
			scn.Nodes = sco.Nodes
		}
		scn.ProcessorType = sco.ProcessorType
		scn.SocketsPerNode = sco.SocketsPerNode
		scn.CoresPerSocket = sco.CoresPerSocket
		scn.ThreadsPerCore = sco.ThreadsPerCore
		var prefix = new(string)
		*prefix = "G"
		scn.FlopRateScalar = schema.MetricValue{
			Unit:  schema.Unit{Base: "F/s", Prefix: prefix},
			Value: float64(sco.FlopRateScalar)}
		scn.FlopRateSimd = schema.MetricValue{
			Unit:  schema.Unit{Base: "F/s", Prefix: prefix},
			Value: float64(sco.FlopRateSimd)}
		scn.MemoryBandwidth = schema.MetricValue{
			Unit:  schema.Unit{Base: "B/s", Prefix: prefix},
			Value: float64(sco.MemoryBandwidth)}
		scn.Topology = *sco.Topology
		cn.SubClusters = append(cn.SubClusters, &scn)
	}

	for _, mco := range co.MetricConfig {
		var mcn schema.MetricConfig
		mcn.Name = mco.Name
		mcn.Scope = mco.Scope
		if mco.Aggregation == "" {
			fmt.Println("Property aggregation missing! Please review file!")
			mcn.Aggregation = "sum"
		} else {
			mcn.Aggregation = mco.Aggregation
		}
		mcn.Timestep = mco.Timestep
		mcn.Unit = units.ConvertUnitString(mco.Unit)
		mcn.Peak = mco.Peak
		mcn.Normal = mco.Normal
		mcn.Caution = mco.Caution
		mcn.Alert = mco.Alert
		mcn.SubClusters = mco.SubClusters
		cn.MetricConfig = append(cn.MetricConfig, &mcn)
	}

	return cn
}

func main() {
	var srcPath string
	var dstPath string

	flag.StringVar(&srcPath, "s", "./var/job-archive", "Specify the source job archive path. Default is ./var/job-archive")
	flag.StringVar(&dstPath, "d", "./var/job-archive-new", "Specify the destination job archive path. Default is ./var/job-archive-new")

	srcConfig := fmt.Sprintf("{\"path\": \"%s\"}", srcPath)
	err := ar.Init(json.RawMessage(srcConfig))
	if err != nil {
		log.Fatal(err)
	}

	err = initClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	// setup new job archive
	err = os.Mkdir(dstPath, 0750)
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range Clusters {
		path := fmt.Sprintf("%s/%s", dstPath, c.Name)
		fmt.Println(path)
		err = os.Mkdir(path, 0750)
		if err != nil {
			log.Fatal(err)
		}
		cn := deepCopyClusterConfig(c)

		f, err := os.Create(fmt.Sprintf("%s/%s/cluster.json", dstPath, c.Name))
		if err != nil {
			log.Fatal(err)
		}
		if err := EncodeCluster(f, &cn); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}

	for job := range ar.Iter() {
		fmt.Printf("Job %d\n", job.JobID)

		path := getPath(job, dstPath, "meta.json")
		err = os.MkdirAll(filepath.Dir(path), 0750)
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		jmn := deepCopyJobMeta(job)
		if err = EncodeJobMeta(f, &jmn); err != nil {
			log.Fatal(err)
		}
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}

		f, err = os.Create(getPath(job, dstPath, "data.json"))
		if err != nil {
			log.Fatal(err)
		}

		var jd *JobData
		jd, err = loadJobData(getPath(job, srcPath, "data.json"))
		if err != nil {
			log.Fatal(err)
		}
		jdn := deepCopyJobData(jd)
		if err := EncodeJobData(f, &jdn); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
