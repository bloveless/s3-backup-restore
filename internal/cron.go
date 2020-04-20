package internal

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type Cron struct {
	HourlyCadence  string
	DailyCadence   string
	WeeklyCadence  string
	MonthlyCadence string
	HourlyBackups  int
	DailyBackups   int
	WeeklyBackups  int
	MonthlyBackups int
	S3Bucket       string
	S3Path         string
	DataDirectory  string
	AwsSession     *session.Session
	S3Service      s3iface.S3API
}

func (c Cron) Run() {
	cr := cron.New(
		cron.WithLogger(
			cron.VerbosePrintfLogger(
				log.New(),
			),
		),
	)

	b := Backup{
		HourlyBackups:  c.HourlyBackups,
		DailyBackups:   c.DailyBackups,
		WeeklyBackups:  c.WeeklyBackups,
		MonthlyBackups: c.MonthlyBackups,
		S3Bucket:       c.S3Bucket,
		S3Path:         c.S3Path,
		DataDirectory:  c.DataDirectory,
		AwsSession:     c.AwsSession,
		S3Service:      c.S3Service,
	}

	if b.HourlyBackups > 0 {
		cr.AddFunc(c.HourlyCadence, func() { b.Run("hourly") })
	}

	if b.DailyBackups > 0 {
		cr.AddFunc(c.DailyCadence, func() { b.Run("daily") })
	}

	if b.WeeklyBackups > 0 {
		cr.AddFunc(c.WeeklyCadence, func() { b.Run("weekly") })
	}

	if b.WeeklyBackups > 0 {
		cr.AddFunc(c.MonthlyCadence, func() { b.Run("monthly") })
	}

	log.Info("Starting cron")
	cr.Run()
}
