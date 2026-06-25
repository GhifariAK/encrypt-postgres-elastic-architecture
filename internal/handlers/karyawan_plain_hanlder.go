package handlers

import (
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

	karyawans, totalData, err := h.service.GetAllKaryawanPlain(limit, offset, sortBy, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessWithPagination(w, http.StatusOK, "[BENCHMARK] Data diambil dari tabel plaintext tanpa enkripsi", karyawans, page, limit, totalData)
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

	sortOrder := r.URL.Query().Get("sort_order")
	offset := (page - 1) * limit

	var karyawans interface{}
	var totalData int
	var err error
	var sumber string

	// Pilih service berdasarkan parameter yang diinputkan user
	if nama != "" {
		karyawans, totalData, err = h.service.SearchNamaPlain(nama, limit, offset, sortOrder)
		sumber = "[BENCHMARK] Pencarian Nama di Postgres"
	} else if nik != "" {
		karyawans, totalData, err = h.service.SearchNIKPlain(nik, limit, offset, sortOrder)
		sumber = "[BENCHMARK] Pencarian NIK di Postgres"
	} else if phone != "" {
		karyawans, totalData, err = h.service.SearchPhonePlain(phone, limit, offset, sortOrder)
		sumber = "[BENCHMARK] Pencarian Phone di Postgres"
	}

	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessWithPagination(w, http.StatusOK, sumber, karyawans, page, limit, totalData)
}
