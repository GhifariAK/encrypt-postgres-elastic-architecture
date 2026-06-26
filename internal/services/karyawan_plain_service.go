package services

import (
	"data-encrypt-be/internal/repository/postgres"
)

// GetAllKaryawanPlain jalur bypass tanpa AES
func (s *KaryawanService) GetAllKaryawanPlain(limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	karyawans, err := postgres.GetAllKaryawanPlain(s.db, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	totalData, err := postgres.GetKaryawanCountPlain(s.db)
	return karyawans, totalData, err
}

// GetKaryawanByIDPlain jalur bypass tanpa AES
func (s *KaryawanService) GetKaryawanByIDPlain(id int) (*postgres.Karyawan, error) {
	return postgres.GetKaryawanByIDPlain(s.db, id)
}

// SearchNamaPlain jalur bypass pencarian nama tanpa AES
func (s *KaryawanService) SearchNamaPlain(namaQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	karyawans, err := postgres.SearchKaryawanByNamePlain(s.db, namaQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	totalData, err := postgres.GetCountByFieldPlain(s.db, "nama", namaQuery)
	return karyawans, totalData, err
}

// SearchNIKPlain jalur bypass pencarian NIK tanpa Elastic dan tanpa AES
func (s *KaryawanService) SearchNIKPlain(nikQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	karyawans, err := postgres.SearchKaryawanByNIKPlain(s.db, nikQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	totalData, err := postgres.GetCountByFieldPlain(s.db, "nik", nikQuery)
	return karyawans, totalData, err
}

// SearchPhonePlain jalur bypass pencarian Phone tanpa Elastic dan tanpa AES
func (s *KaryawanService) SearchPhonePlain(phoneQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	karyawans, err := postgres.SearchKaryawanByPhonePlain(s.db, phoneQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	totalData, err := postgres.GetCountByFieldPlain(s.db, "phone", phoneQuery)
	return karyawans, totalData, err
}

func (s *KaryawanService) SearchJabatanPlain(jabatanQuery string, limit int, offset int, sortBy string, sortOrder string) ([]postgres.Karyawan, int, error) {
	karyawans, err := postgres.SearchKaryawanByJabatanPlain(s.db, jabatanQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}
	totalData, err := postgres.GetCountByFieldPlain(s.db, "jabatan", jabatanQuery)
	return karyawans, totalData, err
}
