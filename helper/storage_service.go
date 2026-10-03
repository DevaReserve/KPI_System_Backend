package helper

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"

	"go.uber.org/zap"
)

// UploadToSupabase (nama dipertahankan agar caller tidak berubah) kini menyimpan
// file ke disk lokal: ./uploads/<bucketName>/<fileName>.
// Mengembalikan URL publik, disajikan oleh route static "/uploads".
func UploadToSupabase(file *multipart.FileHeader, bucketName, fileName string) (string, error) {
	src, err := file.Open()
	if err != nil {
		logger.Error("Failed to open uploaded file", zap.Error(err))
		return "", err
	}
	defer src.Close()

	dir := filepath.Join("uploads", bucketName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		logger.Error("Failed to create upload dir", zap.Error(err))
		return "", err
	}

	dst, err := os.Create(filepath.Join(dir, filepath.Base(fileName)))
	if err != nil {
		logger.Error("Failed to create local file", zap.Error(err))
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		logger.Error("Failed to write local file", zap.Error(err))
		return "", err
	}

	return fmt.Sprintf("%s/uploads/%s/%s", strings.TrimRight(config.BackendURL, "/"), bucketName, filepath.Base(fileName)), nil
}

// DeleteFromSupabase (nama dipertahankan) menghapus file dari disk lokal.
func DeleteFromSupabase(bucketName, fileName string) error {
	path := filepath.Join("uploads", bucketName, filepath.Base(fileName))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		logger.Error("Failed to delete local file", zap.Error(err))
		return err
	}
	return nil
}
