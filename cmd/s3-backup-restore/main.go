package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
	"os"
	"s3-backup-restore/internal"
	"strconv"
	"strings"
)

func main() {
	verbose := flag.Bool("v", false, "Verbose")
	flag.Parse()

	log.SetOutput(os.Stdout)
	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if len(flag.Args()) < 1 {
		log.Info("usage: s3-backup-restore [options] [operation:backup,restore,cron] [backup-type:hourly,daily,weekly,monthly]")
		log.Info("options:")
		log.Info("    -v verbose")
		return
	}

	command := flag.Arg(0)

	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigStateFromEnv,
	}))

	s3Config := aws.NewConfig()
	// if *verbose {
	// 	s3Config = aws.NewConfig().WithLogLevel(aws.LogDebugWithHTTPBody)
	// }
	s3Service := s3.New(awsSession, s3Config)

	switch command {
	case "backup":
		if len(flag.Args()) < 2 {
			log.Warn("The backup operation requires a backup-type.")
			log.Info("usage: s3-backup-restore [options] [operation:backup,restore,cron] [backup-type:hourly,daily,weekly,monthly]")
			log.Info("options:")
			log.Info("    -v verbose")
			return
		}

		b := internal.Backup{
			HourlyBackups:  getEnvIntOrDefault("NUM_HOURLY_BACKUPS", 3),
			DailyBackups:   getEnvIntOrDefault("NUM_DAILY_BACKUPS", 3),
			WeeklyBackups:  getEnvIntOrDefault("NUM_WEEKLY_BACKUPS", 3),
			MonthlyBackups: getEnvIntOrDefault("NUM_MONTHLY_BACKUPS", 3),
			S3Bucket:       getEnvOrFatal("S3_BUCKET"),
			S3Path:         trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:  getEnvOrFatal("DATA_DIRECTORY"),
			AwsSession:     awsSession,
			S3Service:      s3Service,
		}

		b.Run(flag.Arg(1))
	case "restore":
		r := internal.Restore{
			S3Bucket:                getEnvOrFatal("S3_BUCKET"),
			S3Path:                  trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:           getEnvOrFatal("DATA_DIRECTORY"),
			AwsSession:              awsSession,
			S3Service:               s3Service,
			NewFilePermissions:      0644,
			NewDirectoryPermissions: 0755,
		}

		r.Run()
	case "cron":
		c := internal.Cron{
			HourlyCadence:  getEnvStrOrDefault("CADENCE_HOURLY", "0 * * * *"),
			DailyCadence:   getEnvStrOrDefault("CADENCE_DAILY", "10 1 * * *"),
			WeeklyCadence:  getEnvStrOrDefault("CADENCE_WEEKLY", "10 2 * * 0"),
			MonthlyCadence: getEnvStrOrDefault("CADENCE_MONTHLY", "10 3 1 * *"),
			HourlyBackups:  getEnvIntOrDefault("NUM_HOURLY_BACKUPS", 3),
			DailyBackups:   getEnvIntOrDefault("NUM_DAILY_BACKUPS", 3),
			WeeklyBackups:  getEnvIntOrDefault("NUM_WEEKLY_BACKUPS", 3),
			MonthlyBackups: getEnvIntOrDefault("NUM_MONTHLY_BACKUPS", 3),
			S3Bucket:       getEnvOrFatal("S3_BUCKET"),
			S3Path:         trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:  getEnvOrFatal("DATA_DIRECTORY"),
			AwsSession:     awsSession,
			S3Service:      s3Service,
		}

		c.Run()
	}
}

func getEnvStrOrDefault(envName string, defaultValue string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return defaultValue
	}

	return envValue
}

func getEnvIntOrDefault(envName string, defaultValue int) int {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return defaultValue
	}

	envValueInt, err := strconv.Atoi(envValue)
	if err != nil {
		log.Fatalf("Environment variable %s expected to be integer, received %s", envName, envValue)
	}

	return envValueInt
}

func getEnvOrFatal(envName string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		log.Fatalf("Unable to get required environment variable \"%s\"", envName)
	}

	return envValue
}

func trimTrailingSlash(path string) string {
	return strings.TrimRight(path, "/")
}
