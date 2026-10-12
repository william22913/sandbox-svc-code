package nexlogger

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"strings"
)

type toZipStruct struct {
	Filepath    string
	NewFileName string
}

func zipFiles(filename string, files []string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = newZipFile.Close()
	}()

	zipWriter := zip.NewWriter(newZipFile)
	defer func() {
		err = zipWriter.Close()
	}()

	for _, file := range files {
		if err = addFileToZip(zipWriter, file, ""); err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, basePath string, baseInZip string) error {
	basePath = strings.Replace(basePath, "\\", "/", -1)
	fileInfo, err := os.Stat(basePath)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		dat, err := ioutil.ReadFile(basePath)
		if err != nil {
			return err
		}

		f, err := zipWriter.Create(baseInZip + fileInfo.Name())
		if err != nil {
			return err
		}
		_, err = f.Write(dat)
		if err != nil {
			return err
		}
	} else {
		files, err := ioutil.ReadDir(basePath)
		fileName := strings.Split(basePath, "/")
		if err != nil {
			return err
		}
		if len(files) == 0 {
			zipName := baseInZip
			if baseInZip == "" {
				zipName = baseInZip + fileName[len(fileName)-1] + "/"
			}
			_, err := zipWriter.Create(zipName)
			if err != nil {
				return err
			}
		} else {
			if baseInZip == "" {
				baseInZip = fileName[len(fileName)-1] + "/"
			}
			for _, file := range files {
				if !file.IsDir() {
					newBase := basePath + "/" + file.Name()
					err = addFileToZip(zipWriter, newBase, baseInZip)
					if err != nil {
						return err
					}
				} else if file.IsDir() {
					newBase := basePath + "/" + file.Name()
					err = addFileToZip(zipWriter, newBase, baseInZip+file.Name()+"/")
					if err != nil {
						return err
					}
				}

			}
		}
	}
	return nil
}
