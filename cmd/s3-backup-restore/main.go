package main

import (
	"flag"
	"fmt"
	"os"
	"s3-backup-restore/internal"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

func main() {
	verbose := flag.Bool("v", false, "Show verbose output.")
	veryVerbose := flag.Bool("vv", false, "Show very verbose output.")
	showHelp := flag.Bool("h", false, "Show this screen.")
	flag.Parse()

	if *showHelp || len(flag.Args()) < 1 {
		printHelp()
		return
	}

	log.SetOutput(os.Stdout)
	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	command := flag.Arg(0)

	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigStateFromEnv,
	}))

	s3Config := aws.NewConfig()
	if *veryVerbose {
		s3Config = aws.NewConfig().WithLogLevel(aws.LogDebugWithHTTPBody)
	}
	s3Service := s3.New(awsSession, s3Config)

	switch command {
	case "backup":
		if len(flag.Args()) < 2 {
			log.Warn("The backup operation requires a backup-type.")
			printHelp()
			return
		}

		b := internal.Backup{
			HourlyBackups:  getEnvIntOrDefault("NUM_HOURLY_BACKUPS", 3),
			DailyBackups:   getEnvIntOrDefault("NUM_DAILY_BACKUPS", 3),
			WeeklyBackups:  getEnvIntOrDefault("NUM_WEEKLY_BACKUPS", 3),
			MonthlyBackups: getEnvIntOrDefault("NUM_MONTHLY_BACKUPS", 3),
			S3Bucket:       getEnvOrFatal("S3_BUCKET"),
			S3Path:         trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:  trimTrailingSlash(getEnvStrOrDefault("DATA_DIRECTORY", "/data")),
			AwsSession:     awsSession,
			S3Service:      s3Service,
		}

		b.Run(flag.Arg(1))
	case "restore":
		r := internal.Restore{
			S3Bucket:                getEnvOrFatal("S3_BUCKET"),
			S3Path:                  trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:           trimTrailingSlash(getEnvStrOrDefault("DATA_DIRECTORY", "/data")),
			NewDirectoryPermissions: os.FileMode(getEnvIntOrDefault("DIRECTORY_PERMISSIONS", 0755)),
			ChownEnable:             getEnvStrOrDefault("CHOWN_ENABLE", "false") == "true",
			ChownUID:                getEnvIntOrDefault("CHOWN_UID", 1000),
			ChownGID:                getEnvIntOrDefault("CHOWN_GID", 1000),
			ForceRestore:            getEnvBoolOrDefault("RESTORE_FORCE", false),
			RestoreFile:             os.Getenv("RESTORE_FILE"),
			AwsSession:              awsSession,
			S3Service:               s3Service,
		}

		r.Run()
	case "cron":
		c := internal.Cron{
			HourlyCadence:  getEnvStrOrDefault("CADENCE_HOURLY", "0 * * * *"),
			DailyCadence:   getEnvStrOrDefault("CADENCE_DAILY", "10 1 * * *"),
			WeeklyCadence:  getEnvStrOrDefault("CADENCE_WEEKLY", "10 2 * * 0"),
			MonthlyCadence: getEnvStrOrDefault("CADENCE_MONTHLY", "10 3 1 * *"),
			HourlyBackups:  getEnvIntOrDefault("NUM_BACKUPS_HOURLY", 3),
			DailyBackups:   getEnvIntOrDefault("NUM_BACKUPS_DAILY", 3),
			WeeklyBackups:  getEnvIntOrDefault("NUM_BACKUPS_WEEKLY", 3),
			MonthlyBackups: getEnvIntOrDefault("NUM_BACKUPS_MONTHLY", 3),
			S3Bucket:       getEnvOrFatal("S3_BUCKET"),
			S3Path:         trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
			DataDirectory:  trimTrailingSlash(getEnvStrOrDefault("DATA_DIRECTORY", "/data")),
			AwsSession:     awsSession,
			S3Service:      s3Service,
		}

		c.Run()
	}
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  s3-backup-restore [options] backup (hourly|daily|weekly|monthly)")
	fmt.Println("  s3-backup-restore [options] restore")
	fmt.Println("  s3-backup-restore [options] cron")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h  Show this screen.")
	fmt.Println("  -v  Show verbose output.")
	fmt.Println("  -vv Show very verbose output.")
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

func getEnvBoolOrDefault(envName string, defaultValue bool) bool {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return defaultValue
	}

	return envValue == "true"
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
