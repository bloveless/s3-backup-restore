package internal

import (
	"archive/tar"
	"compress/gzip"
	"errors"
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
)

type byTimestamp []string

func (s byTimestamp) Len() int {
	return len(s)
}

func (s byTimestamp) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byTimestamp) Less(i, j int) bool {
	iSplit := strings.Index(s[i], "/")
	jSplit := strings.Index(s[j], "/")

	// format is hourly/2020-04-15T12:35:00Z.tar.gz
	// compare only the timestamp and not the backup type
	return s[i][iSplit:] < s[j][jSplit:]
}

type Restore struct {
	S3Bucket                string
	S3Path                  string
	DataDirectory           string
	AwsSession              *session.Session
	S3Service               s3iface.S3API
	NewFilePermissions      os.FileMode
	NewDirectoryPermissions os.FileMode
}

func (r Restore) Run() {
	latestBackup, err := r.getLatestBackup()
	if err != nil {
		log.Fatal(err)
	}

	backupFilePath, err := r.downloadBackup(latestBackup)
	if err != nil {
		log.Fatal("Unable to download backup file ", err)
	}

	err = r.restoreBackup(backupFilePath)
	if err != nil {
		log.Fatal(err)
	}
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
		Key: aws.String(backupName),
	}

	n, err := downloader.Download(f, i)
	if err != nil {
		return "", err
	}

	log.Debugf("Downloaded %d bytes", n)
	return f.Name(), nil
}

// restoreBackup will replace any files in the directory that overlap with the backup
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
			log.Debugf("Create directory %s", name)
			continue
		case tar.TypeReg:
			dir := filepath.Dir(name)
			err := os.MkdirAll(dir, r.NewDirectoryPermissions)
			if err != nil {
				return err
			}

			log.Debugf("File %s size %d", name, header.Size)

			outFile, err := os.Create(name)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			if err != nil {
				return err
			}
		default:
			fmt.Printf("%s : %c %s %s\n",
				"Yikes! Unable to figure out type",
				header.Typeflag,
				"in file",
				name,
			)
		}
	}

	return nil
}
