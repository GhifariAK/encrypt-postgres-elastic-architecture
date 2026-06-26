package handlers

import (
	"data-encrypt-be/internal/repository/postgres"
	"data-encrypt-be/internal/utils"
	"net/http"
	"strconv"
)

// GetAllKaryawanPlainHandler untuk endpoint GET /api/plain/karyawan
func (h *KaryawanHandler) GetAllKaryawanPlainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	// 1. Ambil Parameter Pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	// 2. Ambil Parameter Sorting
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")
	offset := (page - 1) * limit

	// 3. Ambil Parameter Search
	idStr := r.URL.Query().Get("id")
	nik := r.URL.Query().Get("nik")
	phone := r.URL.Query().Get("phone")
	nama := r.URL.Query().Get("nama")
	jabatan := r.URL.Query().Get("jabatan")

	// default jika kosong
	defaultSortBy := "id" // Ini fallback paling akhir
	if nama != "" {
		defaultSortBy = "nama"
	} else if nik != "" {
		defaultSortBy = "nik"
	} else if phone != "" {
		defaultSortBy = "phone"
	}

	if sortBy == "" && sortOrder == "" {
		sortBy = defaultSortBy
		sortOrder = "desc"
	} else if sortBy != "" && sortOrder == "" {
		sortOrder = "desc"
	} else if sortBy == "" && sortOrder != "" {
		sortBy = defaultSortBy
	}

	var karyawans interface{}
	var totalData int
	var err error
	var sumber string

	// Decesion Tree
	if idStr != "" {
		id, errParse := strconv.Atoi(idStr)
		if errParse != nil {
			utils.SendError(w, http.StatusBadRequest, "Parameter 'id' harus berupa angka")
			return
		}

		k, errSearch := h.service.GetKaryawanByIDPlain(id)
		err = errSearch
		if k != nil {
			// Bungkus 1 data ke dalam bentuk Slice/Array agar response JSON tetap konsisten
			karyawans = []postgres.Karyawan{*k}
			totalData = 1
		} else {
			karyawans = []postgres.Karyawan{}
			totalData = 0
		}
		sumber = "[BENCHMARK] Unified Search: By Exact ID Plaintext"

	} else if nik != "" {
		karyawans, totalData, err = h.service.SearchNIKPlain(nik, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By NIK Plaintext"

	} else if phone != "" {
		karyawans, totalData, err = h.service.SearchPhonePlain(phone, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Phone Plaintext"

	} else if nama != "" {
		karyawans, totalData, err = h.service.SearchNamaPlain(nama, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Nama Plaintext"

	} else if jabatan != "" {
		karyawans, totalData, err = h.service.SearchJabatanPlain(jabatan, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Jabatan Plaintext"

	} else {
		// Jika semua kosong, jalankan Get All biasa
		karyawans, totalData, err = h.service.GetAllKaryawanPlain(limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: Get All Data Plaintext"
	}

	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessWithPagination(w, http.StatusOK, sumber, karyawans, page, limit, totalData)
}

// GetKaryawanByIDPlainHandler untuk endpoint GET /api/plain/karyawan/detail?id=X
func (h *KaryawanHandler) GetKaryawanByIDPlainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		utils.SendError(w, http.StatusBadRequest, "ID harus berupa angka valid")
		return
	}

	k, err := h.service.GetKaryawanByIDPlain(id)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if k == nil {
		utils.SendError(w, http.StatusNotFound, "Data karyawan tidak ditemukan")
		return
	}

	utils.SendSuccess(w, http.StatusOK, "[BENCHMARK] Data detail diambil murni dari Postgres", k)
}

// SearchPlainHandler menangani parameter nama, nik, atau phone secara dinamis
func (h *KaryawanHandler) SearchPlainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	nama := r.URL.Query().Get("nama")
	nik := r.URL.Query().Get("nik")
	phone := r.URL.Query().Get("phone")

	if nama == "" && nik == "" && phone == "" {
		utils.SendError(w, http.StatusBadRequest, "Parameter 'nama', 'nik', atau 'phone' wajib diisi")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")
	offset := (page - 1) * limit

	if sortBy == "" && sortOrder == "" {
		sortBy = "id"
		sortOrder = "desc"
	} else if sortBy != "" && sortOrder == "" {
		sortOrder = "desc"
	} else if sortBy == "" && sortOrder != "" {
		sortBy = "id"
	}

	var karyawans interface{}
	var totalData int
	var err error
	var sumber string

	// Pilih service berdasarkan parameter yang diinputkan user
	if nama != "" {
		karyawans, totalData, err = h.service.SearchNamaPlain(nama, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Pencarian Nama di Postgres"
	} else if nik != "" {
		karyawans, totalData, err = h.service.SearchNIKPlain(nik, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Pencarian NIK di Postgres"
	} else if phone != "" {
		karyawans, totalData, err = h.service.SearchPhonePlain(phone, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Pencarian Phone di Postgres"
	}

	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessWithPagination(w, http.StatusOK, sumber, karyawans, page, limit, totalData)
}
