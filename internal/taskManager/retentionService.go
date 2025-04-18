// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package taskManager

import (
	"time"

	"github.com/ClusterCockpit/cc-backend/pkg/archive"
	"github.com/ClusterCockpit/cc-backend/pkg/log"
	"github.com/go-co-op/gocron/v2"
)

func RegisterRetentionDeleteService(age int, includeDB bool) {
	log.Info("Register retention delete service")

	s.NewJob(gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(04, 0, 0))),
		gocron.NewTask(
			func() {
				startTime := time.Now().Unix() - int64(age*24*3600)
				jobs, err := jobRepo.FindJobsBetween(0, startTime)
				if err != nil {
					log.Warnf("Error while looking for retention jobs: %s", err.Error())
				}
				archive.GetHandle().CleanUp(jobs)

				if includeDB {
					cnt, err := jobRepo.DeleteJobsBefore(startTime)
					if err != nil {
						log.Errorf("Error while deleting retention jobs from db: %s", err.Error())
					} else {
						log.Infof("Retention: Removed %d jobs from db", cnt)
					}
					if err = jobRepo.Optimize(); err != nil {
						log.Errorf("Error occured in db optimization: %s", err.Error())
					}
				}
			}))
}

func RegisterRetentionMoveService(age int, includeDB bool, location string) {
	log.Info("Register retention move service")

	s.NewJob(gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(04, 0, 0))),
		gocron.NewTask(
			func() {
				startTime := time.Now().Unix() - int64(age*24*3600)
				jobs, err := jobRepo.FindJobsBetween(0, startTime)
				if err != nil {
					log.Warnf("Error while looking for retention jobs: %s", err.Error())
				}
				archive.GetHandle().Move(jobs, location)

				if includeDB {
					cnt, err := jobRepo.DeleteJobsBefore(startTime)
					if err != nil {
						log.Errorf("Error while deleting retention jobs from db: %v", err)
					} else {
						log.Infof("Retention: Removed %d jobs from db", cnt)
					}
					if err = jobRepo.Optimize(); err != nil {
						log.Errorf("Error occured in db optimization: %v", err)
					}
				}
			}))
}
