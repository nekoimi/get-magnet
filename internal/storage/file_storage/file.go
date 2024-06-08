package file_storage

import (
	"gmagnet/internal/storage"
	"gmagnet/pkg/file"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type fileStorage struct {
	m       sync.Mutex
	saveDir string // save dir
}

func New(saveDir string) storage.Storage {
	Init(saveDir)
	return &fileStorage{
		m:       sync.Mutex{},
		saveDir: saveDir,
	}
}

func Init(saveDir string) {
	if exists, err := file.Exists(saveDir); err != nil {
		log.Fatal(err.Error())
	} else if !exists {
		if err := os.MkdirAll(saveDir, os.ModeDir); err != nil {
			log.Fatal(err.Error())
		}
	}
}

func OutputFile() string {
	return time.Now().Format("2006-01-02") + "." + "txt"
}

func (s *fileStorage) Save(magnetLink string) error {
	s.m.Lock()
	defer s.m.Unlock()

	outputFile := filepath.Join(s.saveDir, OutputFile())
	if exists, err := file.Exists(outputFile); err != nil {
		return err
	} else if !exists {
		_, err := os.Create(outputFile)
		if err != nil {
			return err
		}
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND, 0666)
	defer f.Close()
	if err != nil {
		return err
	}

	return file.WriteLine(f, magnetLink)
}
