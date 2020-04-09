package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/robfig/cron/v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if os.Getenv("RUN_CRON") == "" {
		backupJob("hourly")
	} else {
		startCron()
	}
}

func startCron() {
	hourlyCadence := os.Getenv("CADENCE_HOURLY")
	if hourlyCadence == "" {
		hourlyCadence = "0 * * * *"
	}

	dailyCadence := os.Getenv("CADENCE_DAILY")
	if dailyCadence == "" {
		dailyCadence = "10 1 * * *"
	}

	weeklyCadence := os.Getenv("CADENCE_WEEKLY")
	if weeklyCadence == "" {
		weeklyCadence = "10 2 * * 0"
	}

	monthlyCadence := os.Getenv("CADENCE_MONTHLY")
	if monthlyCadence == "" {
		monthlyCadence = "10 3 1 * *"
	}

	c := cron.New()
	c.AddFunc(hourlyCadence, func() { backupJob("hourly") })
	c.AddFunc(dailyCadence, func() { backupJob("daily") })
	c.AddFunc(weeklyCadence, func() { backupJob("weekly") })
	c.AddFunc(monthlyCadence, func() { backupJob("monthly") })
	c.Start()
}

func backupJob(backupType string) {
	dataDirectory := os.Getenv("DATA_DIRECTORY")
	if dataDirectory == "" {
		dataDirectory = "/data"
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	backupFile, err := os.OpenFile(dataDirectory + "/BACKUP_DATE", os.O_WRONLY|os.O_CREATE, 0644)
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

	fpErr := filepath.Walk(
		dataDirectory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fmt.Println("Adding file: ", path)

			afErr := addFile(tw, path)
			if afErr != nil {
				return afErr
			}

			return nil
		},
	)

	if fpErr != nil {
		log.Println(fpErr)
	}

	file.Stat()
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
