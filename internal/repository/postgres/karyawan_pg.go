package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq" // untuk fitur array query
)

// Karyawan adalah struktur data yang dibaca oleh aplikasi Go
type Karyawan struct {
	ID             int       `json:"id"`
	Nama           string    `json:"nama"`
	Jabatan        string    `json:"jabatan"`
	NIKEncrypted   string    `json:"nik_encrypted"`
	NIKDecrypted   string    `json:"-"`
	PhoneEncrypted string    `json:"phone_encrypted"`
	PhoneDecrypted string    `json:"-"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

// Helper Untuk Mencegah SQL Injection pada ORDER BY
// Contohnya penyerang tidak bisa memasukkan request ?sort_by=id; DROP TABLE karyawan;
func getSafeOrderBy(sortBy string, sortOrder string) string {
	// Whitelist kolom yang diizinkan untuk di-sort
	allowedFields := map[string]string{
		"id":         "id",
		"nama":       "nama",
		"jabatan":    "jabatan",
		"created_at": "created_at",
	}

	field, ok := allowedFields[sortBy]
	if !ok {
		field = "id" // Default sorting by ID
	}

	order := "DESC" // Default descending
	if strings.ToUpper(sortOrder) == "ASC" {
		order = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", field, order)
}

// InsertKaryawan bertugas memasukkan data yang SUDAH diacak ke PostgreSQL
func InsertKaryawan(db *sql.DB, nama, jabatan, nikEncrypted, phoneEncrypted string) (int, error) {
	// Returning Id penting untuk dikirimkan ke elastic sebagai penghubung
	query := `
		INSERT INTO karyawan (nama, jabatan, nik_encrypted, phone_encrypted) 
		VALUES ($1, $2, $3, $4) RETURNING id
	`

	var insertedID int
	err := db.QueryRow(query, nama, jabatan, nikEncrypted, phoneEncrypted).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("gagal insert data: %v, ke postgres", err)
	}
	return insertedID, nil
}

func GetKaryawanByIDs(db *sql.DB, ids []int) ([]Karyawan, error) {
	if len(ids) == 0 {
		return []Karyawan{}, nil
	}

	// Query ini hanya hit database sekali untuk semua ID
	query := `
		SELECT id, nama, jabatan, nik_encrypted, phone_encrypted, is_active, created_at
		FROM karyawan 
		WHERE id = ANY($1)
	`

	// Gunakan db.Query (karena hasilnya banyak baris) dan bungkus ids dengan pq.Array
	rows, err := db.Query(query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("gagal menarik batch data dari postgres: %v", err)
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKEncrypted, &k.PhoneEncrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error membaca baris data: %v", err)
		}
		hasil = append(hasil, k)
	}

	// Mengecek apakah ada error yang terjadi selama proses looping
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return hasil, nil
}

// GetAllKaryawan menarik seluruh data karyawan  yang ada di database
func GetAllKaryawan(db *sql.DB, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {

	orderClause := getSafeOrderBy(sortBy, sortOrder)

	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik_encrypted, phone_encrypted, is_active, created_at 
	FROM karyawan
	%s
	LIMIT $1 OFFSET $2
	`, orderClause)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKEncrypted, &k.PhoneEncrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// GetKaryawanByID mengambil 1 data spesifik berdasarkan Primary Key
func GetKaryawanByID(db *sql.DB, id int) (*Karyawan, error) {
	query := `SELECT id, nama, jabatan, nik_encrypted, phone_encrypted, is_active, created_at FROM karyawan WHERE id = $1`
	row := db.QueryRow(query, id)

	var k Karyawan
	if err := row.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKEncrypted, &k.PhoneEncrypted, &k.IsActive, &k.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Data memang tidak ada
		}
		return nil, err
	}
	return &k, nil
}

// UpdateKaryawan mengubah data nama, divisi, dan NIK terenkripsi di Postgres
func UpdateKaryawan(db *sql.DB, id int, nama, jabatan, nikEncrypted, phoneEncrypted string) error {
	query := `UPDATE karyawan SET nama = $1, jabatan = $2, nik_encrypted = $3, phone_encrypted = $4 WHERE id = $5`

	// 1. Tangkap objek sql.Result dari db.Exec
	result, err := db.Exec(query, nama, jabatan, nikEncrypted, phoneEncrypted, id)
	if err != nil {
		return fmt.Errorf("gagal eksekusi query update: %v", err)
	}

	// 2. Cek berapa baris yang benar-benar ter-update di Postgres
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mengecek rows affected: %v", err)
	}

	// 3. Jika 0 baris yang berubah, artinya ID tersebut sudah tidak ada (sudah dihapus)
	if rowsAffected == 0 {
		// Kita kembalikan error khusus agar layer Service tahu bahwa data ini sudah ga ada
		return fmt.Errorf("data tidak ditemukan")
	}

	return nil
}

// DeleteKaryawan menghapus data secara permanen dari Postgres
func DeleteKaryawan(db *sql.DB, id int) error {
	query := `DELETE FROM karyawan WHERE id = $1`

	result, err := db.Exec(query, id)

	if err != nil {
		return fmt.Errorf("gagal eksekusi query delete: %v", err)
	}

	// Cek apakah ada baris yang benar-benar terhapus
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mengecek rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Data tidak ditemukan")
	}

	return nil
}

// SearchKaryawanByNamePG mencari data langsung ke Postgres menggunakan ILIKE (Full Table Scan)
func SearchKaryawanByNamePG(db *sql.DB, nama string, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderBy(sortBy, sortOrder)

	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik_encrypted, phone_encrypted, is_active, created_at 
	FROM karyawan 
	WHERE nama ILIKE $1
	%s
	LIMIT $2 OFFSET $3`, orderClause)

	// Menambahkan % di awal dan akhir query agar sesuai kaidah ILIKE
	searchParam := fmt.Sprintf("%%%s%%", nama)

	rows, err := db.Query(query, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKEncrypted, &k.PhoneEncrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// GetKaryawanCount menghitung total seluruh baris data di tabel karyawan
func GetKaryawanCount(db *sql.DB) (int, error) {
	query := `SELECT COUNT(id) FROM karyawan`
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung total data karyawan: %v", err)
	}
	return count, nil
}

// GetKaryawanCountByName menghitung total baris data yang namanya mengandung query pencarian
func GetKaryawanCountByName(db *sql.DB, nama string) (int, error) {
	query := `SELECT COUNT(id) FROM karyawan WHERE nama ILIKE $1`
	searchParam := fmt.Sprintf("%%%s%%", nama)

	var count int
	err := db.QueryRow(query, searchParam).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung total data pencarian nama: %v", err)
	}
	return count, nil
}

// InsertKaryawanPlaintext memasukkan data tanpa enkripsi ke tabel khusus benchmark
func InsertKaryawanPlaintext(db *sql.DB, id int, nama, jabatan, nik, phone string) error {
	query := `INSERT INTO karyawan_plaintext (id, nama, jabatan, nik, phone) 
	          VALUES ($1, $2, $3, $4, $5) 
	          ON CONFLICT (id) DO NOTHING`
	_, err := db.Exec(query, id, nama, jabatan, nik, phone)
	return err
}
