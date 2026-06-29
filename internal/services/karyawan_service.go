package services

import (
	"fmt"

	"data-encrypt-be/internal/crypto"
	"data-encrypt-be/internal/repository/elastic"
	"data-encrypt-be/internal/repository/postgres"
	"database/sql"
	"math/rand"

	es "github.com/elastic/go-elasticsearch/v8"
)

type KaryawanService struct {
	db        *sql.DB
	esClient  *es.Client
	secretKey string
}

// NewKaryawanService adalah constructor untuk inisialisasi service
func NewKaryawanService(db *sql.DB, esClient *es.Client, secretKey string) *KaryawanService {
	return &KaryawanService{
		db:        db,
		esClient:  esClient,
		secretKey: secretKey,
	}
}

// Helper untuk Menyusun ulang urutan array karyawan mengikuti urutan ID dari Elasticsearch (Sort)
func (s *KaryawanService) reconstructAndDecrypt(ids []int, karyawans []postgres.Karyawan) []postgres.Karyawan {
	karyawanMap := make(map[int]postgres.Karyawan)
	for i := range karyawans {
		if decNIK, err := crypto.DecryptAES(karyawans[i].NIKEncrypted, s.secretKey); err == nil {
			karyawans[i].NIKDecrypted = decNIK
		} else {
			karyawans[i].NIKDecrypted = "GAGAL_DEKRIPSI"
		}

		if decPhone, err := crypto.DecryptAES(karyawans[i].PhoneEncrypted, s.secretKey); err == nil {
			karyawans[i].PhoneDecrypted = decPhone
		} else {
			karyawans[i].PhoneDecrypted = "GAGAL_DEKRIPSI"
		}
		karyawanMap[karyawans[i].ID] = karyawans[i]
	}

	var result []postgres.Karyawan
	for _, id := range ids {
		if k, exists := karyawanMap[id]; exists {
			result = append(result, k)
		}
	}
	return result
}

// RegisterKaryawan mengatur alur enkripsi data sebelum dikirim ke DB
func (s *KaryawanService) RegisterKaryawan(nama, divisi, nikAsli, phoneAsli string) (int, error) {
	// 1. Enkripsi NIK menjadi ciphertext AES
	nikEncrypted, err := crypto.EncryptAES(nikAsli, s.secretKey)
	if err != nil {
		return 0, fmt.Errorf("gagal enkripsi NIK: %v", err)
	}

	// Enkripsi phone number
	phoneEncrypted, err := crypto.EncryptAES(phoneAsli, s.secretKey)
	if err != nil {
		return 0, fmt.Errorf("gagal enkripsi nomor telepon: %v", err)
	}

	// 2. Simpan Ciphertext ke PostgreSQL dan tangkap ID-nya
	karyawanID, err := postgres.InsertKaryawan(s.db, nama, divisi, nikEncrypted, phoneEncrypted)
	if err != nil {
		return 0, err
	}

	// Di versi PoC ini, kita menembak Elastic secara sinkron (langsung).
	// Saat naik ke Production, baris kode di bawah ini harus dihapus.
	// Golang cukup mem-publish "Event" ke RabbitMQ/Kafka bahwa ada data baru.
	// Biarkan ada service/worker lain di background yang bertugas mengambil data
	// dari antrean RabbitMQ untuk dimasukkan ke Elasticsearch.
	// Ini memastikan Postgres tidak perlu di-rollback jika Elastic sedang down.
	// Hal tersebut dilakukan untuk method Create, Update, dan Delete.

	// 3. Simpan Plaintext ke Elasticsearch menggunakan ID dari Postgres
	err = elastic.IndexKaryawan(s.esClient, karyawanID, nama, nikAsli, phoneAsli)
	if err != nil {
		return 0, fmt.Errorf("Data berhasil masuk Postgres, tapi gagal ke Elastic: %v", err)
	}

	return karyawanID, nil
}

// GetKaryawanByNIK mengambil data berdasarkan query NIK yang dimasukkan
func (s *KaryawanService) GetKaryawanByNIK(nikQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	// 1. Cari di Elastic: Dapatkan kumpulan ID karyawan yang NIK-nya mengandung "nikQuery"
	ids, totalData, err := elastic.SearchNIK(s.esClient, nikQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// Jika Elastic tidak menemukan apa-apa, langsung kembalikan array kosong
	if len(ids) == 0 {
		return []postgres.Karyawan{}, 0, nil
	}

	// 2. Cari di Postgres: Tarik data utuh berdasarkan kumpulan ID tadi
	karyawans, err := postgres.GetKaryawanByIDs(s.db, ids)
	if err != nil {
		return nil, 0, err
	}

	// 3. Melakukan Dekripsi di memori RAM
	sortedResult := s.reconstructAndDecrypt(ids, karyawans)

	return sortedResult, totalData, nil
}

// GetAllKaryawan mengambil semua data lalu mendekripsinya satu per satu
func (s *KaryawanService) GetAllKaryawan(limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {

	// 1. Tarik data halaman ini dari Postgres
	karyawans, err := postgres.GetAllKaryawan(s.db, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// 2. Tarik jumlah total seluruh data dari Postgres
	totalData, err := postgres.GetKaryawanCount(s.db)
	if err != nil {
		return nil, 0, err
	}

	for i := range karyawans {
		if decryptedNIK, err := crypto.DecryptAES(karyawans[i].NIKEncrypted, s.secretKey); err == nil {
			karyawans[i].NIKDecrypted = decryptedNIK
		}
		if decryptedPhone, err := crypto.DecryptAES(karyawans[i].PhoneEncrypted, s.secretKey); err == nil {
			karyawans[i].PhoneDecrypted = decryptedPhone
		}
	}
	return karyawans, totalData, nil

}

// GetKaryawanByID mengambil satu data spesifik dan mendekripsinya
func (s *KaryawanService) GetKaryawanByID(id int) (*postgres.Karyawan, error) {
	k, err := postgres.GetKaryawanByID(s.db, id)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, nil
	}

	decryptedNIK, err := crypto.DecryptAES(k.NIKEncrypted, s.secretKey)
	if err == nil {
		k.NIKDecrypted = decryptedNIK
	}

	decryptedPhone, err := crypto.DecryptAES(k.PhoneEncrypted, s.secretKey)
	if err == nil {
		k.PhoneDecrypted = decryptedPhone
	}
	return k, nil
}

// UpdateKaryawan mengonstruksi ulang enkripsi data baru, lalu melakukan Dual-Write Update
func (s *KaryawanService) UpdateKaryawan(id int, nama, divisi, nikAsli, phoneAsli string) error {
	// 1. Enkripsi ulang yang baru dimasukkan
	nikEncrypted, err := crypto.EncryptAES(nikAsli, s.secretKey)
	if err != nil {
		return err
	}

	phoneEncrypted, err := crypto.EncryptAES(phoneAsli, s.secretKey)
	if err != nil {
		return err
	}

	// 2. Update Postgres
	if err := postgres.UpdateKaryawan(s.db, id, nama, divisi, nikEncrypted, phoneEncrypted); err != nil {
		return err
	}

	// 3. Update Elastic menggunakan fungsi Index yang sama
	return elastic.IndexKaryawan(s.esClient, id, nama, nikAsli, phoneAsli)
}

// DeleteKaryawan menghapus data dari kedua database (Dual-Delete)
func (s *KaryawanService) DeleteKaryawan(id int) error {
	// 1. Hapus dari Postgres
	if err := postgres.DeleteKaryawan(s.db, id); err != nil {
		return err
	}
	// 2. Hapus dari Elastic
	return elastic.DeleteKaryawan(s.esClient, id)
}

func (s *KaryawanService) GetKaryawanByPhone(phoneQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	ids, totalData, err := elastic.SearchPhone(s.esClient, phoneQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []postgres.Karyawan{}, 0, nil
	}

	karyawans, err := postgres.GetKaryawanByIDs(s.db, ids)
	if err != nil {
		return nil, 0, err
	}

	sortedResult := s.reconstructAndDecrypt(ids, karyawans)

	return sortedResult, totalData, nil
}

// GetKaryawanByName mengambil data berdasarkan query nama
func (s *KaryawanService) GetKaryawanByName(namaQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {

	ids, totalData, err := elastic.SearchNama(s.esClient, namaQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []postgres.Karyawan{}, 0, nil
	}

	karyawans, err := postgres.GetKaryawanByIDs(s.db, ids)
	if err != nil {
		return nil, 0, err
	}

	sortedResult := s.reconstructAndDecrypt(ids, karyawans)

	return sortedResult, totalData, nil
}

// GetKaryawanByJabatan mengambil data berdasarkan query jabatan langsung dari Postgres (BARU)
func (s *KaryawanService) GetKaryawanByJabatan(jabatanQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	// 1. Jalur murni Postgres karena jabatan tidak diindeks di Elastic
	karyawans, err := postgres.SearchKaryawanByJabatanPG(s.db, jabatanQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// 2. Hitung total data pencarian jabatan untuk keperluan pagination metadata
	totalData, err := postgres.GetCountByFieldPG(s.db, "jabatan", jabatanQuery)
	if err != nil {
		return nil, 0, err
	}

	// 3. Lakukan dekripsi AES untuk data sensitif sebelum dikembalikan ke Handler
	for i := range karyawans {
		if decNIK, err := crypto.DecryptAES(karyawans[i].NIKEncrypted, s.secretKey); err == nil {
			karyawans[i].NIKDecrypted = decNIK
		}
		if decPhone, err := crypto.DecryptAES(karyawans[i].PhoneEncrypted, s.secretKey); err == nil {
			karyawans[i].PhoneDecrypted = decPhone
		}
	}

	return karyawans, totalData, nil
}

// GetProviderStats mengambil analitik jumlah karyawan per provider telepon
func (s *KaryawanService) GetProviderStats() (map[string]int, error) {
	return elastic.GetPhoneProviderStats(s.esClient)
}

// SyncAllPostgresToElastic memindahkan seluruh data dari Postgres ke Elastic secara otomatis
func (s *KaryawanService) SyncAllPostgresToElastic() (int, error) {
	jumlahData := 0
	limit := 1000
	offset := 0

	fmt.Println("🔄 Memulai sinkronisasi dari Postgres ke Elastic...")

	for {
		// 1. Tarik semua data dari Postgres
		karyawans, err := postgres.GetAllKaryawan(s.db, limit, offset, "id", "asc")
		if err != nil {
			return jumlahData, fmt.Errorf("gagal mengambil data dari postgres: %v", err)
		}

		// Jika array kosong, berarti semua data sudah habis ditarik dari Postgres
		if len(karyawans) == 0 {
			break
		}

		// 2. Loop semua data dan kirim ke Elastic
		for _, k := range karyawans {
			// Dekripsi data sensitif agar Elastic bisa menyimpannya sebagai plaintext katalog
			decNIK, errNIK := crypto.DecryptAES(k.NIKEncrypted, s.secretKey)
			decPhone, errPhone := crypto.DecryptAES(k.PhoneEncrypted, s.secretKey)

			if errNIK != nil || errPhone != nil {
				// Jika ada data yang gagal didekripsi, lewati data ini agar tidak merusak Elastic
				continue
			}

			// 3. Masukkan ke Elastic memakai fungsi IndexKaryawan
			errStr := elastic.IndexKaryawan(s.esClient, k.ID, k.Nama, decNIK, decPhone)
			if errStr != nil {
				fmt.Printf("Gagal sync karyawan ID %d ke Elastic: %v\n", k.ID, errStr)
				continue
			}
			jumlahData++
		}
		fmt.Printf("⏳ sync %d data...\n", jumlahData)

		// 3. Majukan offset sejauh 1000 langkah untuk batch selanjutnya
		offset += limit
	}

	fmt.Printf("✅ Sinkronisasi selesai! Total %d data berhasil dikirim ke Elastic.\n", jumlahData)
	return jumlahData, nil
}

// SeedDummyData menyuntikkan puluhan ribu data dummy secara background
func (s *KaryawanService) SeedDummyData() {
	jumlahData := 20000

	// Jalankan Goroutine agar Postman tidak loading panjang
	go func() {
		fmt.Printf("🚀 MULAI: Menyuntikkan %d data dummy...\n", jumlahData)

		namaDepan := []string{"Andi", "Budi", "Citra", "Dewi", "Eko", "Fajar", "Gita", "Hadi", "Indah", "Joko"}
		namaBelakang := []string{"Saputra", "Wijaya", "Kusuma", "Lestari", "Nugroho", "Pratama", "Sari", "Setiawan", "Hidayat", "Putri"}

		for i := 1; i <= jumlahData; i++ {
			// Mengacak indeks dari 0 sampai 9
			idxDepan := rand.Intn(len(namaDepan))
			idxBelakang := rand.Intn(len(namaBelakang))

			// Hasilnya akan bervariasi: "Andi Wijaya", "Citra Kusuma", dll ditambah angka agar unik
			namaAsli := fmt.Sprintf("%s %s %d", namaDepan[idxDepan], namaBelakang[idxBelakang], i)

			nikAsli := fmt.Sprintf("327000%08d", i)
			phoneAsli := fmt.Sprintf("0812%08d", i)

			// Enkripsi
			nikEncrypted, _ := crypto.EncryptAES(nikAsli, s.secretKey)
			phoneEncrypted, _ := crypto.EncryptAES(phoneAsli, s.secretKey)

			// Simpan ke Postgres pakai fungsi InsertKaryawan yang sudah di buat
			karyawanID, err := postgres.InsertKaryawan(s.db, namaAsli, "Staff", nikEncrypted, phoneEncrypted)
			if err != nil {
				fmt.Printf("Gagal insert ke PG baris %d: %v\n", i, err)
				continue
			}

			// Sinkron ke Elastic
			err = elastic.IndexKaryawan(s.esClient, karyawanID, namaAsli, nikAsli, phoneAsli)
			if err != nil {
				fmt.Printf("Gagal index ke ES baris %d: %v\n", i, err)
			}

			// Tampilkan log tiap 5000 data
			if i%5000 == 0 {
				fmt.Printf("⏳ PROGRES: %d / %d data berhasil masuk...\n", i, jumlahData)
			}
		}
		fmt.Println("✅ SELESAI: 20.000 data dummy berhasil disuntikkan!")
	}()
}

// SearchNamaPG mencari nama murni lewat Postgres untuk pembanding performa
func (s *KaryawanService) SearchNamaPG(namaQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {

	karyawans, err := postgres.SearchKaryawanByNamePG(s.db, namaQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	// Tarik jumlah total seluruh data yang cpcook dari postgres
	totalData, err := postgres.GetKaryawanCountByName(s.db, namaQuery)
	if err != nil {
		return nil, 0, err
	}

	// Dekripsi data balasan agar formatnya sama dengan balasan Elastic
	for i := range karyawans {
		if decNIK, err := crypto.DecryptAES(karyawans[i].NIKEncrypted, s.secretKey); err == nil {
			karyawans[i].NIKDecrypted = decNIK
		}
		if decPhone, err := crypto.DecryptAES(karyawans[i].PhoneEncrypted, s.secretKey); err == nil {
			karyawans[i].PhoneDecrypted = decPhone
		}
	}
	return karyawans, totalData, nil
}

// Service decrypt helper
func (s *KaryawanService) DecryptText(ciphertext string) (string, error) {
	return crypto.DecryptAES(ciphertext, s.secretKey)
}

// CloneToPlaintext menyalin seluruh data ke tabel benchmark secara background
func (s *KaryawanService) CloneToPlaintext() {
	fmt.Println("🚀 Memulai cloning data ke tabel Plaintext...")
	limit := 1000
	offset := 0
	count := 0

	for {
		// 1. Tarik data batch per 1000 dari tabel lama (fungsi ini otomatis melakukan decrypt di layer service milikmu)
		karyawans, err := postgres.GetAllKaryawan(s.db, limit, offset, "id", "asc")
		if err != nil || len(karyawans) == 0 {
			break // Berhenti jika data habis
		}

		// 2. Masukkan satu per satu ke tabel baru
		for _, k := range karyawans {
			// Kita dekripsi manual lagi untuk memastikan keamanan (karena di GetAllKaryawan hasil dekripsi ada di struct NIKDecrypted)
			decNIK, _ := crypto.DecryptAES(k.NIKEncrypted, s.secretKey)
			decPhone, _ := crypto.DecryptAES(k.PhoneEncrypted, s.secretKey)

			errInsert := postgres.InsertKaryawanPlaintext(s.db, k.ID, k.Nama, k.Jabatan, decNIK, decPhone)
			if errInsert == nil {
				count++
			}
		}

		offset += limit
		fmt.Printf("⏳ Progres: %d data berhasil di-copy...\n", count)
	}
	fmt.Println("✅ SELESAI! Seluruh data sukses disalin ke tabel karyawan_plaintext.")
}

// Struct khusus untuk membalas hasil Reveal (Hanya data esensial)
type RevealResponse struct {
	ID      int    `json:"id"`
	Nama    string `json:"nama"`
	Jabatan string `json:"jabatan"`
	NIK     string `json:"nik_asli"`
	Phone   string `json:"phone_asli"`
}

// RevealKaryawan mengambil ID spesifik dari Postgres dan mengembalikan data yang sudah terdekripsi (Plaintext)
func (s *KaryawanService) RevealKaryawan(ids []int) ([]RevealResponse, error) {
	if len(ids) == 0 {
		return []RevealResponse{}, nil
	}

	// 1. Ambil data mentah dari Postgres berdasarkan kumpulan ID
	karyawans, err := postgres.GetKaryawanByIDs(s.db, ids)
	if err != nil {
		return nil, err
	}

	// 2. Dekripsi dan ubah bentuknya ke Struct Reveal
	var revealedData []RevealResponse
	for _, k := range karyawans {
		nikDec, _ := crypto.DecryptAES(k.NIKEncrypted, s.secretKey)
		phoneDec, _ := crypto.DecryptAES(k.PhoneEncrypted, s.secretKey)

		revealedData = append(revealedData, RevealResponse{
			ID:      k.ID,
			Nama:    k.Nama,
			Jabatan: k.Jabatan,
			NIK:     nikDec,
			Phone:   phoneDec,
		})
	}

	return revealedData, nil
}
