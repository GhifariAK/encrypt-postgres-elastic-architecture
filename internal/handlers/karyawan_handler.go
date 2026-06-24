package handlers

import (
	"data-encrypt-be/internal/services"
	"data-encrypt-be/internal/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type KaryawanHandler struct {
	service *services.KaryawanService
}

// Constructor
func NewKaryawanHandler(service *services.KaryawanService) *KaryawanHandler {
	return &KaryawanHandler{service: service}
}

// Struct untuk membaca body request JSON dari Postman saat INSERT
type CreateKaryawanReq struct {
	Nama    string `json:"nama"`
	Jabatan string `json:"jabatan"`
	NIK     string `json:"nik"`
	Phone   string `json:"phone"`
}

// CreateKaryawanHandler menangani endpoint POST /api/karyawan/create
func (h *KaryawanHandler) CreateKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus POST")
		return
	}

	var req CreateKaryawanReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.SendError(w, http.StatusBadRequest, "Format JSON salah: "+err.Error())
		return
	}

	// Kirim data ke layer service
	newID, err := h.service.RegisterKaryawan(req.Nama, req.Jabatan, req.NIK, req.Phone)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Gagal memproses data: "+err.Error())
		return
	}

	// Menyiapkan payload ringkas untuk dimasukkan ke field "data" JSON
	createdData := map[string]interface{}{
		"id":      newID,
		"nama":    req.Nama,
		"jabatan": req.Jabatan,
	}

	utils.SendSuccess(
		w,
		http.StatusCreated,
		"Sukses! Data karyawan berhasil disimpan (Postgres) dan diindeks (Elastic).",
		createdData,
	)
}

// GetKaryawanHandler menangani endpoint GET /api/karyawan/search?nik=...
func (h *KaryawanHandler) GetKaryawanByNIKHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	// Mengambil parameter NIK dari URL Query
	nikAsli := r.URL.Query().Get("nik")
	if nikAsli == "" {
		utils.SendError(w, http.StatusBadRequest, "Parameter 'nik' wajib diisi di query URL")
		return
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	// Memanggil layer service
	karyawans, totalData, err := h.service.GetKaryawanByNIK(nikAsli, limit, offset, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Terjadi kesalahan: "+err.Error())
		return
	}

	// Info jika array kosong (tidak ditemukan)
	if len(karyawans) == 0 {
		utils.SendError(w, http.StatusNotFound, "Data dengan NIK tersebut tidak ditemukan")
		return
	}

	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		"Data ditemukan (Sumber: Elasticsearch NIK Wildcard)",
		karyawans,
		page,
		limit,
		totalData,
	)
}

// GetAllKaryawanHandler menangani GET /api/karyawan
func (h *KaryawanHandler) GetAllKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	// Tangkap parameter dari URL
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Konversi string ke int, dengan nilai default jika kosong
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // Default halaman 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default 10 data per halaman
	}

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	karyawans, totalData, err := h.service.GetAllKaryawan(limit, offset, sortBy, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Menggunakan SendSuccessWithPagination pusat
	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		"Data berhasil diambil",
		karyawans,
		page,
		limit,
		totalData,
	)
}

// GetKaryawanByIDHandler menangani GET /api/karyawan/detail?id=X
func (h *KaryawanHandler) GetKaryawanByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(w, http.StatusBadRequest, "ID harus berupa angka valid")
		return
	}

	k, err := h.service.GetKaryawanByID(id)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if k == nil {
		utils.SendError(w, http.StatusNotFound, "Data karyawan tidak ditemukan")
		return
	}

	utils.SendSuccess(w, http.StatusOK, "Data karyawan ditemukan", k)
}

// UpdateKaryawanHandler menangani PUT /api/karyawan/update?id=X
func (h *KaryawanHandler) UpdateKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus PUT")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(w, http.StatusBadRequest, "ID harus berupa angka valid")
		return
	}

	var req CreateKaryawanReq // Struktur data request body-nya sama
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "JSON Body salah")
		return
	}

	if err := h.service.UpdateKaryawan(id, req.Nama, req.Jabatan, req.NIK, req.Phone); err != nil {
		// Jika pesan error mengandung kata "tidak ditemukan", kembalikan 404
		if strings.Contains(err.Error(), "tidak ditemukan") {
			utils.SendError(w, http.StatusNotFound, err.Error())
			return
		}

		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updatedData := map[string]interface{}{
		"id":      id,
		"nama":    req.Nama,
		"jabatan": req.Jabatan,
	}

	utils.SendSuccess(w, http.StatusOK, "Data karyawan berhasil diperbarui secara sinkron!", updatedData)
}

// DeleteKaryawanHandler menangani DELETE /api/karyawan/delete?id=X
func (h *KaryawanHandler) DeleteKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus DELETE")
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.SendError(w, http.StatusBadRequest, "ID harus berupa angka valid")
		return
	}

	karyawan, err := h.service.GetKaryawanByID(id)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Gagal mengambil data sebelum dihapus: "+err.Error())
		return
	}
	if karyawan == nil {
		utils.SendError(w, http.StatusNotFound, "Data tidak ditemukan")
		return
	}

	if err := h.service.DeleteKaryawan(id); err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Gagal menghapus data: "+err.Error())
		return
	}

	deletedData := map[string]interface{}{
		"id":              karyawan.ID,
		"nama":            karyawan.Nama,
		"jabatan":         karyawan.Jabatan,
		"nik_encrypted":   karyawan.NIKEncrypted,
		"phone_encrypted": karyawan.PhoneEncrypted,
	}

	utils.SendSuccess(w, http.StatusOK, "Data karyawan berhasil dihapus permanen dari Postgres dan Elastic!", deletedData)
}

// GetKaryawanByPhoneHandler menangani endpoint GET /api/karyawan/search/phone?phone=...
func (h *KaryawanHandler) GetKaryawanByPhoneHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	telpQuery := r.URL.Query().Get("phone")
	if telpQuery == "" {
		utils.SendError(w, http.StatusBadRequest, "Parameter 'phone' wajib diisi")
		return
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	karyawans, totalData, err := h.service.GetKaryawanByPhone(telpQuery, limit, offset, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(karyawans) == 0 {
		utils.SendError(w, http.StatusNotFound, "Data dengan nomor telepon tersebut tidak ditemukan")
		return
	}

	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		"Data ditemukan (Sumber: Elasticsearch Phone Wildcard)",
		karyawans,
		page,
		limit,
		totalData,
	)
}

// GetKaryawanByNamaHandler menangani endpoint GET /api/karyawan/search/name/es/nama?nama=...
func (h *KaryawanHandler) GetKaryawanByNameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	namaQuery := r.URL.Query().Get("nama")
	if namaQuery == "" {
		utils.SendError(w, http.StatusBadRequest, "Parameter 'nama' wajib diisi")
		return
	}

	// Tangkap parameter pagination
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	karyawans, totalData, err := h.service.GetKaryawanByName(namaQuery, limit, offset, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		"Pencarian berhasil (Sumber: Elasticsearch)",
		karyawans,
		page,
		limit,
		totalData,
	)
}

// GetProviderStatsHandler endpoint untuk GET /api/karyawan/stats/provider
func (h *KaryawanHandler) GetProviderStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.service.GetProviderStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Statistik Provider Karyawan",
		"data":    stats,
	})
}

// SyncKaryawanHandler menangani endpoint POST /api/karyawan/sync
func (h *KaryawanHandler) SyncKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metode HTTP harus POST", http.StatusMethodNotAllowed)
		return
	}

	jumlah, err := h.service.SyncAllPostgresToElastic()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Sinkronisasi data database ke Elasticsearch berhasil!",
		"total_sync": jumlah,
	})
}

// RunSeederHandler menangani POST /api/karyawan/seed
func (h *KaryawanHandler) RunSeederHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metode HTTP harus POST", http.StatusMethodNotAllowed)
		return
	}

	// Panggil service yang jalan di background
	h.service.SeedDummyData()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Proses seeding 50.000 data sedang berjalan di background (Lihat log terminal Docker).",
	})
}

// SearchPGHandler menangani GET /api/karyawan/search-pg?nama=...
func (h *KaryawanHandler) SearchNamePGHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus GET")
		return
	}

	namaQuery := r.URL.Query().Get("nama")
	if namaQuery == "" {
		utils.SendError(w, http.StatusBadRequest, "Parameter 'nama' wajib diisi")
		return
	}

	// Tangkap parameter pagination
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1 // Default halaman 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // Default 10 data
	}

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	karyawans, totalData, err := h.service.SearchNamaPG(namaQuery, limit, offset, sortBy, sortOrder)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Gunakan SendSuccessWithPagination
	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		"Pencarian berhasil (Sumber: PostgreSQL)",
		karyawans,
		page,
		limit,
		totalData,
	)
}

// API DECRYPT
// Struct untuk menangkap body JSON dari user
type DecryptRequest struct {
	EncryptedText string `json:"encrypted_text"`
}

// Struct untuk membalas hasil dekripsi
type DecryptResponse struct {
	PlainText string `json:"plaintext"`
}

// DecryptDataHandler menangani endpoint POST /api/karyawan/decrypt
func (h *KaryawanHandler) DecryptDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus POST")
		return
	}

	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Format JSON salah")
		return
	}

	if req.EncryptedText == "" {
		utils.SendError(w, http.StatusBadRequest, "encrypted_text tidak boleh kosong")
		return
	}

	// Panggil fungsi jembatan dari service untuk melakukan dekripsi
	plainText, err := h.service.DecryptText(req.EncryptedText)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Gagal mendekripsi: Pastikan format ciphertext benar")
		return
	}

	res := DecryptResponse{
		PlainText: plainText,
	}

	utils.SendSuccess(w, http.StatusOK, "Data berhasil didekripsi", res)
}

// ClonePlaintextHandler menangani endpoint POST /api/karyawan/clone-plain
func (h *KaryawanHandler) ClonePlaintextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus POST")
		return
	}

	// Jalankan di background (Goroutine) agar Postman tidak macet menunggu 20.000 data
	go h.service.CloneToPlaintext()

	utils.SendSuccess(w, http.StatusOK, "Proses cloning ke tabel plaintext sedang berjalan di background. Silakan cek log terminal!", nil)
}
