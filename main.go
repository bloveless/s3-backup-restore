package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type app struct {
	debug          bool
	hourlyCadence  string
	dailyCadence   string
	weeklyCadence  string
	monthlyCadence string
	hourlyBackups  int
	dailyBackups   int
	monthlyBackups int
	weeklyBackups  int
	s3Bucket       string
	s3Path         string
	dataDirectory  string
	logLevel       int
	awsSession     *session.Session
	s3Service      *s3.S3
}

func (a app) startCron() {
	c := cron.New(
		cron.WithLogger(
			cron.VerbosePrintfLogger(
				log.New(),
			),
		),
	)

	c.AddFunc(a.hourlyCadence, func() { a.backupJob("hourly") })
	c.AddFunc(a.dailyCadence, func() { a.backupJob("daily") })
	c.AddFunc(a.weeklyCadence, func() { a.backupJob("weekly") })
	c.AddFunc(a.monthlyCadence, func() { a.backupJob("monthly") })

	log.Info("Starting cron")
	c.Run()
}

func (a app) backupJob(backupType string) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	archivePath := a.compressDirectory(now, backupType)

	a.uploadToS3(now, backupType, archivePath)

	a.pruneS3(backupType)
}

func (a app) compressDirectory(now string, backupType string) string {
	dataDirectory := os.Getenv("DATA_DIRECTORY")
	if dataDirectory == "" {
		dataDirectory = "/data"
	}

	backupFile, err := os.OpenFile(dataDirectory+"/BACKUP_DATE", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	backupFile.WriteString(fmt.Sprintf("%s/%s", backupType, now))

	file, err := ioutil.TempFile("", "output*.tar.gz")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(
		dataDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			log.Debugf("Adding file: %s", path)

			afErr := addFile(tw, path)
			if afErr != nil {
				return afErr
			}

			return nil
		},
	)

	if err != nil {
		log.Println(err)
	}

	log.Debugf("Output file: %s", file.Name())
	return file.Name()
}

func (a app) uploadToS3(now string, backupType string, path string) {
	uploader := s3manager.NewUploader(a.awsSession)
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to open archive: ", path)
	}
	defer file.Close()

	log.Infof("Output path: %s", path)

	uploadPath := fmt.Sprintf("%s/%s/%s.tar.gz", a.s3Path, backupType, now)

	uploadInput := s3manager.UploadInput{
		Body:   file,
		Bucket: aws.String(a.s3Bucket),
		Key:    aws.String(uploadPath),
	}

	_, err = uploader.Upload(&uploadInput)
	if err != nil {
		log.Fatal("Unable to upload", err)
	}

	log.Info("Uploaded backup successfully")
}

func (a app) pruneS3(backupType string) {
	objects := a.getBucketObjects(backupType)
	var keys []string
	for _, o := range objects {
		keys = append(keys, aws.StringValue(o.Key))
	}

	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	numberToKeep := a.hourlyBackups
	switch backupType {
	case "daily":
		numberToKeep = a.dailyBackups
	case "weekly":
		numberToKeep = a.weeklyBackups
	case "monthly":
		numberToKeep = a.monthlyBackups
	}

	var deleteObjects []s3manager.BatchDeleteObject
	for _, k := range keys[numberToKeep:] {
		deleteObjects = append(deleteObjects, s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Bucket: aws.String(a.s3Bucket),
				Key: aws.String(k),
			},
		})
	}

	batcher := s3manager.NewBatchDeleteWithClient(a.s3Service)
	err := batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: deleteObjects,
	})

	if err != nil {
		log.Fatal("Unable to delete old", err)
	}
}

func (a app) getBucketObjects(backupType string) []*s3.Object {
	i := &s3.ListObjectsInput{
		Bucket: aws.String("s3-backup-restore-dev-test"),
		Prefix: aws.String(fmt.Sprintf("%s%s", a.s3Path, backupType)),
	}

	fmt.Println(i)

	o, err := a.s3Service.ListObjects(i)
	if err != nil {
		log.Fatal(err)
	}

	return o.Contents
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigStateFromEnv,
	}))

	s3Service := s3.New(awsSession, aws.NewConfig().WithLogLevel(aws.LogDebugWithHTTPBody))

	a := app{
		debug:          true,
		hourlyCadence:  getEnvStrOrDefault("CADENCE_HOURLY", "0 * * * *"),
		dailyCadence:   getEnvStrOrDefault("CADENCE_DAILY", "10 1 * * *"),
		weeklyCadence:  getEnvStrOrDefault("CADENCE_WEEKLY", "10 2 * * 0"),
		monthlyCadence: getEnvStrOrDefault("CADENCE_MONTHLY", "10 3 1 * *"),
		hourlyBackups:  getEnvIntOrDefault("NUM_HOURLY_BACKUPS", 3),
		dailyBackups:   getEnvIntOrDefault("NUM_DAILY_BACKUPS", 3),
		weeklyBackups:  getEnvIntOrDefault("NUM_WEEKLY_BACKUPS", 3),
		monthlyBackups: getEnvIntOrDefault("NUM_MONTHLY_BACKUPS", 3),
		s3Bucket:       getEnvOrFatal("S3_BUCKET"),
		s3Path:         trimTrailingSlash(getEnvStrOrDefault("S3_PATH", "/")),
		dataDirectory:  getEnvOrFatal("DATA_DIRECTORY"),
		awsSession:     awsSession,
		s3Service:      s3Service,
	}

	if os.Getenv("ENABLE_CRON") == "" {
		a.backupJob("hourly")
	} else {
		a.startCron()
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
		log.Fatal("Unable to get required environment variable", envName)
	}

	return envValue
}

func trimTrailingSlash(path string) string {
	return strings.TrimRight(path, "/")
}

func addFile(tw *tar.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if stat, err := file.Stat(); err == nil {
		header, err := tar.FileInfoHeader(stat, path)
		if err != nil {
			return err
		}

		header.Name = path

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}

	return nil
}
