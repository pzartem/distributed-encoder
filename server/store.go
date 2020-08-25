package server

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// FSObjectStore represents simple file storage
type FSObjectStore struct {
	Path string
}

// WriteObjects create a file and write data from src reader
func (s *FSObjectStore) WriteObject(key string, src io.Reader) error {
	f, err := ioutil.TempFile(s.Path, key)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	defer writer.Flush()

	if _, err := io.Copy(writer, src); err != nil {
		log.Println("Error store: ", err)
		s.Remove(f.Name())

		return err
	}

	return nil
}

// Remove removes file from the fs
func (FSObjectStore) Remove(key string) {
	if err := os.Remove(key); err != nil {
		log.Println("can't remove file: ", err)
	}
}

// HasObject checks does object exists
func (FSObjectStore) HasObject(key string) bool {
	info, err := os.Stat(key)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
