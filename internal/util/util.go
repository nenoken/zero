package util

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"text/template"

	"github.com/kyokomi/emoji"
	"github.com/logrusorgru/aurora"
)

func CreateDirIfDoesNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		return err
	}
	return nil
}

var FuncMap = template.FuncMap{
	"Title":   strings.Title,
	"ToLower": strings.ToLower,
}

func createTemplatedFile(fullFilePath string, template *template.Template, wg *sync.WaitGroup, data interface{}) {
	f, err := os.Create(fullFilePath)
	if err != nil {
		log.Println(aurora.Red(emoji.Sprintf(":exclamation: Error creating file '%s' : %v", fullFilePath, err)))
	}
	wg.Add(1)
	go func() {
		err = template.Execute(f, data)
		if err != nil {
			log.Println(aurora.Red(emoji.Sprintf(":exclamation: Error templating '%s': %v", fullFilePath, err)))
		}
		log.Println(aurora.Green(emoji.Sprintf(":white_check_mark: Finished templating : %v", fullFilePath)))
		wg.Done()
	}()
}

func TemplateFileAndOverwrite(fileDir string, fileName string, template *template.Template, wg *sync.WaitGroup, data interface{}) {
	fullFilePath := fmt.Sprintf("%v/%v", fileDir, fileName)
	err := os.MkdirAll(fileDir, os.ModePerm)
	if err != nil {
		log.Println(aurora.Red(emoji.Sprintf(":exclamation: Error creating directory %v: %v", fullFilePath, err)))
	}
	createTemplatedFile(fullFilePath, template, wg, data)

}

func TemplateFileIfDoesNotExist(fileDir string, fileName string, template *template.Template, wg *sync.WaitGroup, data interface{}) {
	fullFilePath := path.Join(fileDir, fileName)

	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		if fileDir != "" {
			err := CreateDirIfDoesNotExist(fileDir)
			if err != nil {
				log.Println(aurora.Red(emoji.Sprintf(":exclamation: Error creating directory %v: %v", fullFilePath, err)))
			}
		}
		createTemplatedFile(fullFilePath, template, wg, data)
	} else {
		log.Println(aurora.Yellow(emoji.Sprintf("%v already exists. skipping.", fullFilePath)))
	}
}