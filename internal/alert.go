package internal

import "context"

type Alert struct {
	Name                     string `json:"name"`
	Query                    Query  `json:"query"`
	CronJobDurationInMinutes int64  `json:"duration"`
	Threshold                int    `json:"threshold"`
	Script                   string `json:"script"`
	running                  bool
}

type AlertStats struct {
	Jobs         []string
	RunningJobs  []string
	SleepingJobs []string
}

type AlertManager interface {
	Submit(context.Context, Alert) error
	Run(context.Context) error
	List(context.Context) ([]Alert, error)
	Stats(context.Context) error
	Stop(context.Context, string) error
	Delete(context.Context, string) error
}
