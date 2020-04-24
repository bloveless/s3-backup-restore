package internal

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

type byTimestamp []string

func (s byTimestamp) Len() int {
	return len(s)
}

func (s byTimestamp) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byTimestamp) Less(i, j int) bool {
	iSplit := strings.LastIndex(s[i], "/")
	jSplit := strings.LastIndex(s[j], "/")

	// format is hourly/2020-04-15T12:35:00Z.tar.gz
	// compare only the timestamp and not the backup type
	return s[i][iSplit:] < s[j][jSplit:]
}

// Restore contains the config necessary to run a restore
type Restore struct {
	S3Bucket                string
	S3Path                  string
	DataDirectory           string
	NewDirectoryPermissions os.FileMode
	ChownEnable             bool
	ChownUID                int
	ChownGID                int
	ForceRestore            bool
	RestoreFile             string
	AwsSession              *session.Session
	S3Service               s3iface.S3API
}

// Run performs a restore
func (r Restore) Run() {
	log.Info("Beginning restore")
	if r.ForceRestore == false && r.isDataDirectoryIsEmpty() != true {
		log.Info("Cowardly refusing to restore to a folder full of files")
		return
	}

	backupPath := r.S3Path + "/" + r.RestoreFile
	if r.RestoreFile == "" {
		log.Info("Getting latest backup from S3")
		b, err := r.getLatestBackup()
		if err != nil {
			log.Fatal(err)
		}

		backupPath = b
	} else {
		log.Infof("Restoring user requested backup: %s", backupPath)
	}

	log.Info("Downloading backup from S3")
	backupFilePath, err := r.downloadBackup(backupPath)
	if err != nil {
		log.Fatal("Unable to download backup file ", err)
	}

	log.Info("Restoring backup")
	err = r.restoreBackup(backupFilePath)
	if err != nil {
		log.Fatal(err)
	}

	if r.ChownEnable {
		log.Info("Updating permissions on restored files")
		err = r.updatePermissions()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Info("Finished restore")
}

func (r Restore) isDataDirectoryIsEmpty() bool {
	files, err := ioutil.ReadDir(r.DataDirectory)
	if err != nil {
		return false
	}

	if len(files) > 0 {
		return false
	}

	return true
}

func (r Restore) getLatestBackup() (string, error) {
	i := &s3.ListObjectsInput{
		Bucket: aws.String(r.S3Bucket),
		Prefix: aws.String(r.S3Path),
	}

	listOutput, err := r.S3Service.ListObjects(i)
	if err != nil {
		return "", err
	}

	var keys []string
	for _, o := range listOutput.Contents {
		keys = append(keys, aws.StringValue(o.Key))
	}

	if len(keys) == 0 {
		return "", errors.New("unable to find a backup to restore from")
	}

	sort.Sort(sort.Reverse(byTimestamp(keys)))

	log.Infof("Found latest backup: %s", keys[0])
	return keys[0], nil
}

func (r Restore) downloadBackup(backupName string) (string, error) {
	downloader := s3manager.NewDownloaderWithClient(r.S3Service)

	f, err := ioutil.TempFile("", "backup*.tar.gz")
	if err != nil {
		return "", err
	}
	defer f.Close()

	i := &s3.GetObjectInput{
		Bucket: aws.String(r.S3Bucket),
		Key:    aws.String(backupName),
	}

	n, err := downloader.Download(f, i)
	if err != nil {
		return "", err
	}

	log.Infof("Downloaded %d bytes", n)
	return f.Name(), nil
}

func (r Restore) restoreBackup(backupFilePath string) error {
	f, err := os.Open(backupFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		name := fmt.Sprintf("%s/%s", r.DataDirectory, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			dir := filepath.Dir(name)
			err := os.MkdirAll(dir, r.NewDirectoryPermissions)
			if err != nil {
				return err
			}

			log.Infof("Restored file %s", name)

			outFile, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return err
			}

			outFile.Close()
		default:
			log.Errorf("Unable to figure out type: %c %s %s",
				header.Typeflag,
				"in file",
				name,
			)
		}
	}

	log.Info("Backup restored")
	return nil
}

func (r Restore) updatePermissions() error {
	err := filepath.Walk(
		r.DataDirectory,
		func(path string, info os.FileInfo, err error) error {
			return os.Chown(path, r.ChownUID, r.ChownGID)
		},
	)

	log.Info("Data directory permissions updated")
	return err
}
