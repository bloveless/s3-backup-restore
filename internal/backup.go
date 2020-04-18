package internal

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Backup struct {
	HourlyBackups  int
	DailyBackups   int
	MonthlyBackups int
	WeeklyBackups  int
	S3Bucket       string
	S3Path         string
	DataDirectory  string
	AwsSession     *session.Session
	S3Service      s3iface.S3API
}

func (b Backup) Run(backupType string) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	archivePath := b.compressDirectory(now, backupType)

	b.uploadToS3(now, backupType, archivePath)

	b.pruneS3(backupType)

	b.removeBackupDirectory()
}

func (b Backup) compressDirectory(now string, backupType string) string {
	dataDirectory := os.Getenv("DATA_DIRECTORY")
	if dataDirectory == "" {
		dataDirectory = "/data"
	}

	backupFile, err := os.OpenFile(dataDirectory+"/BACKUP_DATE", os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	backupFile.WriteString(fmt.Sprintf("%s/%s\n", backupType, now))

	tempDir := os.TempDir() + "/backups"
	os.Mkdir(tempDir, 0700)
	file, err := ioutil.TempFile(tempDir, "backup*.tar.gz")
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

			afErr := b.addFile(tw, path)
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

func (b Backup) uploadToS3(now string, backupType string, path string) {
	uploader := s3manager.NewUploader(b.AwsSession)
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to open archive: ", path)
	}
	defer file.Close()

	log.Infof("Output path: %s", path)

	uploadPath := fmt.Sprintf("%s/%s/%s.tar.gz", b.S3Path, backupType, now)

	uploadInput := s3manager.UploadInput{
		Body:   file,
		Bucket: aws.String(b.S3Bucket),
		Key:    aws.String(uploadPath),
	}

	_, err = uploader.Upload(&uploadInput)
	if err != nil {
		log.Fatal("Unable to upload", err)
	}

	log.Info("Uploaded backup successfully")
}

func (b Backup) pruneS3(backupType string) {
	objects := b.getBucketObjects(backupType)
	var keys []string
	for _, o := range objects {
		keys = append(keys, aws.StringValue(o.Key))
	}

	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	numberToKeep := b.HourlyBackups
	switch backupType {
	case "daily":
		numberToKeep = b.DailyBackups
	case "weekly":
		numberToKeep = b.WeeklyBackups
	case "monthly":
		numberToKeep = b.MonthlyBackups
	}

	if len(keys) <= numberToKeep {
		log.Debug("Nothing to prune, skipping.")
		return
	}

	var deleteObjects []s3manager.BatchDeleteObject
	for _, k := range keys[numberToKeep:] {
		deleteObjects = append(deleteObjects, s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Bucket: aws.String(b.S3Bucket),
				Key: aws.String(k),
			},
		})
	}

	batcher := s3manager.NewBatchDeleteWithClient(b.S3Service)
	err := batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: deleteObjects,
	})

	if err != nil {
		log.Fatal("Unable to delete old", err)
	}
}

func (b Backup) removeBackupDirectory() {
	dir := os.TempDir() + "/backups"
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatal("Unable to remove temp files")
	}
}

func (b Backup) getBucketObjects(backupType string) []*s3.Object {
	i := &s3.ListObjectsInput{
		Bucket: aws.String("s3-backup-restore-dev-test"),
		Prefix: aws.String(fmt.Sprintf("%s%s", b.S3Path, backupType)),
	}

	fmt.Println(i)

	o, err := b.S3Service.ListObjects(i)
	if err != nil {
		log.Fatal(err)
	}

	return o.Contents
}

func (b Backup) addFile(tw *tar.Writer, path string) error {
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

		header.Name = strings.ReplaceAll(path, b.DataDirectory + "/", "")

		log.Debugf("Adding file: %s => %s", path, header.Name)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}

	return nil
}
