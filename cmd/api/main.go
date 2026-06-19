package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"data-encrypt-be/internal/config"
	"data-encrypt-be/internal/handlers"
	"data-encrypt-be/internal/repository/elastic" // Untuk memanggil setup N-Gram
	"data-encrypt-be/internal/services"

	"github.com/elastic/go-elasticsearch/v8"
	_ "github.com/lib/pq"
)

func main() {
	// 1. Load Konfigurasi Tersentralisasi
	appConfig := config.LoadConfig()

	// 2. Inisialisasi Koneksi Database
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		appConfig.DBHost, appConfig.DBPort, appConfig.DBUser,
		appConfig.DBPass, appConfig.DBName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Gagal membuka gerbang database: %v", err)
	}
	defer db.Close()

	// Test Koneksi ke PostgreSQL
	log.Println("⏳ Menunggu PostgreSQL siap...")
	for i := 1; i <= 10; i++ {
		err = db.Ping()
		if err == nil {
			log.Println("✅ Database PostgreSQL Terhubung!")
			break
		}
		log.Printf("   -> Belum merespon (Percobaan %d/10). Coba lagi dalam 3 detik...\n", i)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("❌ PostgreSQL mati total setelah 30 detik: %v", err)
	}

	// 3. Inisialisasi Koneksi Elasticsearch
	esConfig := elasticsearch.Config{
		Addresses: []string{appConfig.ElasticURL},
	}
	esClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		log.Fatalf("Gagal membuat client Elasticsearch: %v", err)
	}

	// Test koneksi ke Elasticsearch
	log.Println("⏳ Menunggu Elasticsearch siap...")
	for i := 1; i <= 15; i++ {
		res, errES := esClient.Info()
		if errES == nil && res.StatusCode == 200 {
			res.Body.Close()
			log.Println("✅ Elasticsearch Terhubung!")
			break
		}
		log.Printf("   -> Belum merespon (Percobaan %d/15). Coba lagi dalam 3 detik...\n", i)
		time.Sleep(3 * time.Second)
	}

	// Menjalankan set up index
	err = elastic.SetupIndex(esClient)
	if err != nil {
		log.Fatalf("Gagal setup index N-Gram di Elasticsearch: %v", err)
	}

	// 4. Dependency Injection
	// encryption key diambil dari struct appConfig
	karyawanService := services.NewKaryawanService(db, esClient, appConfig.EncryptionKey)
	karyawanHandler := handlers.NewKaryawanHandler(karyawanService)

	// 5. Pengaturan Route API
	http.HandleFunc("/api/karyawan", karyawanHandler.GetAllKaryawanHandler)                    // GET ALL
	http.HandleFunc("/api/karyawan/detail", karyawanHandler.GetKaryawanByIDHandler)            // GET BY ID
	http.HandleFunc("/api/karyawan/create", karyawanHandler.CreateKaryawanHandler)             // CREATE
	http.HandleFunc("/api/karyawan/update", karyawanHandler.UpdateKaryawanHandler)             // UPDATE
	http.HandleFunc("/api/karyawan/delete", karyawanHandler.DeleteKaryawanHandler)             // DELETE
	http.HandleFunc("/api/karyawan/search/nik", karyawanHandler.GetKaryawanByNIKHandler)       // SEARCH VIA ES
	http.HandleFunc("/api/karyawan/search/nama", karyawanHandler.GetKaryawanByNameHandler)     // SEARCH BY NAMA
	http.HandleFunc("/api/karyawan/sorted/nik", karyawanHandler.GetKaryawanSortedByNIKHandler) // GET SORTED BY NIK
	http.HandleFunc("/api/karyawan/stats/provider", karyawanHandler.GetProviderStatsHandler)   // STATS PROVIDER
	http.HandleFunc("/api/karyawan/sync", karyawanHandler.SyncKaryawanHandler)                 // SYNC POSTGRES KE ELASTIC (BACKGROUND)

	// Menjalankan Server HTTP
	port := ":8080"
	log.Printf("Server 'Dat-Encrypt-BE' berjalan di http://localhost%s\n", port)

	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}
