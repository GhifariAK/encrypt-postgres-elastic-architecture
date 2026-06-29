package handlers

import (
	"data-encrypt-be/internal/repository/postgres"
	"data-encrypt-be/internal/services"
	"data-encrypt-be/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	if sortBy == "" {
		sortBy = "nik" // API lama otomatis sort by NIK
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Memanggil layer service
	karyawans, totalData, err := h.service.GetKaryawanByNIK(nikAsli, limit, offset, sortBy, sortOrder)
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
		"",
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

	// Ambil parameter search
	idStr := r.URL.Query().Get("id")
	nik := r.URL.Query().Get("nik")
	phone := r.URL.Query().Get("phone")
	nama := r.URL.Query().Get("nama")
	jabatan := r.URL.Query().Get("jabatan")

	// Ambil parameter sorting
	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	// Default kalau sortby kosong
	defaultSortBy := "id" // Fallback paling akhir
	if nama != "" {
		defaultSortBy = "nama"
	} else if nik != "" {
		defaultSortBy = "nik"
	} else if phone != "" {
		defaultSortBy = "phone"
	} else if jabatan != "" {
		defaultSortBy = "jabatan"
	}

	// Cek apakah user sengaja memasukkan custom sort_by
	isCustomSort := sortBy != ""

	// Terapkan default jika kosong
	if sortBy == "" && sortOrder == "" {
		sortBy = defaultSortBy
		sortOrder = "desc"
	} else if sortBy != "" && sortOrder == "" {
		sortOrder = "desc"
	} else if sortBy == "" && sortOrder != "" {
		sortBy = defaultSortBy
	}

	warningMsg := ""
	isElasticRoute := idStr == "" && (nik != "" || phone != "" || nama != "")

	// Jika masuk rute Elastic, TAPI user maksa sort pakai kolom di luar Elastic
	if isElasticRoute && isCustomSort {
		if sortBy != "nama" && sortBy != "nik" && sortBy != "phone" {
			warningMsg = fmt.Sprintf(" [NOTE: Kolom '%s' tidak bisa di sort, karena tidak diindeks pada Elastic, sorting ignored]", sortBy)
		}
	}

	var karyawans interface{}
	var totalData int
	var sumber string

	// DECISION TREE
	if idStr != "" {
		id, errParse := strconv.Atoi(idStr)
		if errParse != nil {
			utils.SendError(w, http.StatusBadRequest, "Parameter 'id' harus berupa angka")
			return
		}

		var k *postgres.Karyawan
		k, err = h.service.GetKaryawanByID(id)

		if k != nil {
			karyawans = []postgres.Karyawan{*k}
			totalData = 1
		} else {
			karyawans = []postgres.Karyawan{}
			totalData = 0
		}
		sumber = "[BENCHMARK] Unified Search: By Exact ID (Postgres + Decrypt)"

	} else if nik != "" {
		karyawans, totalData, err = h.service.GetKaryawanByNIK(nik, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By NIK (Elasticsearch)" + warningMsg

	} else if phone != "" {
		karyawans, totalData, err = h.service.GetKaryawanByPhone(phone, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Phone (Elasticsearch)" + warningMsg

	} else if nama != "" {
		karyawans, totalData, err = h.service.GetKaryawanByName(nama, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Nama (Elasticsearch)" + warningMsg

	} else if jabatan != "" {
		karyawans, totalData, err = h.service.GetKaryawanByJabatan(jabatan, limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: By Jabatan (Postgres + Decrypt)"

	} else {
		// Jika semua kosong, jalankan Get All biasa
		karyawans, totalData, err = h.service.GetAllKaryawan(limit, offset, sortBy, sortOrder)
		sumber = "[BENCHMARK] Unified Search: Get All Data (Postgres + Decrypt)"
	}

	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var extractedIDs []int

	// Ekstrak ID dari hasil pencarian
	if kList, ok := karyawans.([]postgres.Karyawan); ok {
		for _, k := range kList {
			extractedIDs = append(extractedIDs, k.ID)
		}
	}

	requestID := ""
	if len(extractedIDs) > 0 {
		requestID = utils.GenerateRequestID()
		utils.RequestCache.Set(requestID, extractedIDs, 5*time.Minute)
	}

	// Menggunakan SendSuccessWithPagination pusat
	utils.SendSuccessWithPagination(
		w,
		http.StatusOK,
		sumber,
		karyawans,
		page,
		limit,
		totalData,
		requestID,
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

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	offset := (page - 1) * limit

	if sortBy == "" {
		sortBy = "phone" // API lama otomatis sort by phone
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	karyawans, totalData, err := h.service.GetKaryawanByPhone(telpQuery, limit, offset, sortBy, sortOrder)
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
		"",
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

	sortBy := r.URL.Query().Get("sort_by")
	sortOrder := r.URL.Query().Get("sort_order")

	if sortBy == "" {
		sortBy = "nama" // API lama otomatis sort by name
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	offset := (page - 1) * limit

	karyawans, totalData, err := h.service.GetKaryawanByName(namaQuery, limit, offset, sortBy, sortOrder)
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
		"",
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
		"",
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

type RevealRequest struct {
	RequestID string `json:"request_id"`
}

// RevealKaryawanHandler menangani POST /api/karyawan/reveal
func (h *KaryawanHandler) RevealKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.SendError(w, http.StatusMethodNotAllowed, "Metode HTTP harus POST")
		return
	}

	var req RevealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, http.StatusBadRequest, "Format JSON salah")
		return
	}

	if req.RequestID == "" {
		utils.SendError(w, http.StatusBadRequest, "request_id wajib dikirim")
		return
	}

	// 1. Cek didalam memori berdasarkan request_id
	ids, exists := utils.RequestCache.Get(req.RequestID)
	if !exists {
		utils.SendError(w, http.StatusNotFound, "Request ID tidak valid atau sudah kedaluwarsa (lebih dari 5 menit)")
		return
	}

	// 2. Minta Service untuk mendekripsi data spesifik tersebut
	revealedData, err := h.service.RevealKaryawan(ids)
	if err != nil {
		utils.SendError(w, http.StatusInternalServerError, "Gagal mendekripsi data: "+err.Error())
		return
	}

	utils.SendSuccess(w, http.StatusOK, "Berhasil membuka (Reveal) data rahasia", revealedData)
}
