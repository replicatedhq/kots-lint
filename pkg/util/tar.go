package util

import (
	"archive/tar"
	"bytes"
)

func IsTarFile(value []byte) bool {
	strReader := bytes.NewReader(value)
	tr := tar.NewReader(strReader)
	_, err := tr.Next()
	if err != nil {
		return false
	}
	return true
}
