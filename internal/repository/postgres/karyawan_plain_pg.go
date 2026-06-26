package postgres

import (
	"database/sql"
	"fmt"
	"strings"
)

// duplikat fungsi getSafeOrderBy
func getSafeOrderByPlain(sortBy string, sortOrder string) string {
	allowedFields := map[string]string{
		"id":         "id",
		"nama":       "nama",
		"jabatan":    "jabatan",
		"nik":        "nik",   // NIK bisa diurutkan karena datanya teks asli
		"phone":      "phone", // Phone bisa diurutkan karena datanya teks asli
		"created_at": "created_at",
	}

	field, ok := allowedFields[sortBy]
	if !ok {
		field = "id"
	}

	order := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		order = "ASC"
	}

	return fmt.Sprintf("ORDER BY %s %s", field, order)
}

// GetAllKaryawanPlain menarik seluruh data dari tabel plaintext
func GetAllKaryawanPlain(db *sql.DB, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderByPlain(sortBy, sortOrder)

	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at 
	FROM karyawan_plaintext
	%s
	LIMIT $1 OFFSET $2`, orderClause)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		// Masukkan nik dan phone plain ke wadah Decrypted dan Encrypted agar struktur JSON tetap konsisten
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// GetKaryawanByIDPlain mengambil 1 data spesifik dari tabel plaintext
func GetKaryawanByIDPlain(db *sql.DB, id int) (*Karyawan, error) {
	query := `SELECT id, nama, jabatan, nik, phone, is_active, created_at FROM karyawan_plaintext WHERE id = $1`
	row := db.QueryRow(query, id)

	var k Karyawan
	if err := row.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	k.NIKEncrypted = k.NIKDecrypted
	k.PhoneEncrypted = k.PhoneDecrypted
	return &k, nil
}

// SearchKaryawanByNamePlain mencari data nama (ILIKE) di tabel plaintext
func SearchKaryawanByNamePlain(db *sql.DB, nama string, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {

	orderClause := getSafeOrderByPlain(sortBy, sortOrder)

	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at 
	FROM karyawan_plaintext 
	WHERE nama ILIKE $1
	%s
	LIMIT $2 OFFSET $3`, orderClause)

	searchParam := fmt.Sprintf("%%%s%%", nama)
	rows, err := db.Query(query, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// SearchKaryawanByNIKPlain mencari data NIK menggunakan LIKE (Murni Postgres)
func SearchKaryawanByNIKPlain(db *sql.DB, nik string, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderByPlain(sortBy, sortOrder)
	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at FROM karyawan_plaintext WHERE nik LIKE $1 %s LIMIT $2 OFFSET $3`, orderClause)

	searchParam := fmt.Sprintf("%%%s%%", nik)
	rows, err := db.Query(query, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// SearchKaryawanByPhonePlain mencari data Phone menggunakan LIKE (Murni Postgres)
func SearchKaryawanByPhonePlain(db *sql.DB, phone string, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderByPlain(sortBy, sortOrder)
	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at FROM karyawan_plaintext WHERE phone LIKE $1 %s LIMIT $2 OFFSET $3`, orderClause)

	searchParam := fmt.Sprintf("%%%s%%", phone)
	rows, err := db.Query(query, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// SearchKaryawanByJabatanPlain mencari data Jabatan menggunakan ILIKE (Murni Postgres)
func SearchKaryawanByJabatanPlain(db *sql.DB, jabatan string, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderByPlain(sortBy, sortOrder)
	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at FROM karyawan_plaintext WHERE jabatan ILIKE $1 %s LIMIT $2 OFFSET $3`, orderClause)
	searchParam := fmt.Sprintf("%%%s%%", jabatan)

	rows, err := db.Query(query, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	//... (sama seperti di atas)
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// SearchKaryawanByIsActivePlain mencari data IsActive menggunakan boolean (Murni Postgres)
func SearchKaryawanByIsActivePlain(db *sql.DB, isActive bool, limit int, offset int, sortBy string, sortOrder string) ([]Karyawan, error) {
	orderClause := getSafeOrderByPlain(sortBy, sortOrder)
	query := fmt.Sprintf(`SELECT id, nama, jabatan, nik, phone, is_active, created_at FROM karyawan_plaintext WHERE is_active = $1 %s LIMIT $2 OFFSET $3`, orderClause)

	rows, err := db.Query(query, isActive, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []Karyawan
	for rows.Next() {
		var k Karyawan
		err := rows.Scan(&k.ID, &k.Nama, &k.Jabatan, &k.NIKDecrypted, &k.PhoneDecrypted, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		k.NIKEncrypted = k.NIKDecrypted
		k.PhoneEncrypted = k.PhoneDecrypted
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// GetKaryawanCountPlain menghitung total seluruh data plain
func GetKaryawanCountPlain(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(id) FROM karyawan_plaintext`).Scan(&count)
	return count, err
}

// GetCountByFieldPlain menghitung total data pencarian (Dinamis untuk Nama, NIK, Phone)
func GetCountByFieldPlain(db *sql.DB, fieldName string, queryValue string) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(id) FROM karyawan_plaintext WHERE %s ILIKE $1`, fieldName)
	searchParam := fmt.Sprintf("%%%s%%", queryValue)
	var count int
	err := db.QueryRow(query, searchParam).Scan(&count)
	return count, err
}
