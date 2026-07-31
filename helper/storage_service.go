package helper

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"KPI_System_Backend/config"
	"KPI_System_Backend/logger"
	"go.uber.org/zap"

	storage_go "github.com/supabase-community/storage-go"
)

// UploadToSupabase menerima file dan menguploadnya ke Supabase Storage.
// Mengembalikan URL publik dari file tersebut.
func UploadToSupabase(file *multipart.FileHeader, bucketName, fileName string) (string, error) {
	src, err := file.Open()
	if err != nil {
		logger.Error("Failed to open uploaded file", zap.Error(err))
		return "", err
	}
	defer src.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, src); err != nil {
		logger.Error("Failed to copy file to buffer", zap.Error(err))
		return "", err
	}

	storageClient := storage_go.NewClient(config.SupabaseURL+"/storage/v1", config.SupabaseKey, nil)

	// contentType bisa diambil dari Header, atau kita kosongkan (Supabase sering kali bisa detect otomatis atau pakai param)
	// Kita akan menggunakan byte buffer untuk upload
	_, err = storageClient.UploadFile(bucketName, fileName, buf)
	if err != nil {
		logger.Error("Failed to upload to Supabase Storage", zap.Error(err))
		return "", err
	}

	// URL Publik secara default di Supabase
	publicUrl := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", config.SupabaseURL, bucketName, fileName)
	return publicUrl, nil
}

// DeleteFromSupabase menghapus file dari bucket.
func DeleteFromSupabase(bucketName, fileName string) error {
	if config.SupabaseURL == "" || config.SupabaseKey == "" {
		return nil
	}
	storageClient := storage_go.NewClient(config.SupabaseURL+"/storage/v1", config.SupabaseKey, nil)
	_, err := storageClient.RemoveFile(bucketName, []string{fileName})
	if err != nil {
		logger.Error("Failed to delete from Supabase Storage", zap.Error(err))
		return err
	}
	return nil
}
