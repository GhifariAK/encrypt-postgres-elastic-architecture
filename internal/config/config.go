package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig adalah struct untuk menyimpan seluruh konfigurasi aplikasi
type AppConfig struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPass        string
	DBName        string
	EncryptionKey string
	ElasticURL    string
}

// LoadConfig bertugas membaca .env dan memvalidasi nilainya
func LoadConfig() *AppConfig {
	// Coba baca file .env di root folder tempat aplikasi dijalankan
	err := godotenv.Load(".env")
	if err != nil {
		// Fallback: Jika gagal, coba cari di dua tingkat folder ke atas
		err = godotenv.Load("../../.env")
		if err != nil {
			log.Println("File .env tidak ditemukan, menggunakan variabel OS default.")
		}
	}

	// Validasi secret key
	secretKey := os.Getenv("ENCRYPTION_KEY")
	if len(secretKey) != 32 {
		log.Fatal("❌ ENCRYPTION_KEY di .env harus tepat 32 karakter!")
	}

	// Kembalikan dalam wujud Struct yang rapi
	return &AppConfig{
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPass:        os.Getenv("DB_PASS"),
		DBName:        os.Getenv("DB_NAME"),
		EncryptionKey: secretKey,
		ElasticURL:    os.Getenv("ELASTIC_URL"),
	}
}
