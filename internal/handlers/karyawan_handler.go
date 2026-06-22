package handlers

import (
	"data-encrypt-be/internal/services"
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
		http.Error(w, "Metode HTTP harus POST", http.StatusMethodNotAllowed)
		return
	}

	var req CreateKaryawanReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Format JSON salah: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Kirim data ke layer service
	err = h.service.RegisterKaryawan(req.Nama, req.Jabatan, req.NIK, req.Phone)
	if err != nil {
		http.Error(w, "Gagal memproses data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Sukses! Data karyawan berhasil disimpan (Postgres) dan diindeks (Elastic).",
	})
}

// GetKaryawanHandler menangani endpoint GET /api/karyawan/search?nik=...
func (h *KaryawanHandler) GetKaryawanByNIKHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	// Mengambil parameter NIK dari URL Query
	nikAsli := r.URL.Query().Get("nik")
	if nikAsli == "" {
		http.Error(w, "Parameter 'nik' wajib diisi di query URL", http.StatusBadRequest)
		return
	}

	// Memanggil layer service
	karyawans, err := h.service.GetKaryawanByNIK(nikAsli)
	if err != nil {
		http.Error(w, "Data tidak ditemukan atau error: "+err.Error(), http.StatusNotFound)
		return
	}

	// Info jika array kosong (tidak ditemukan)
	if len(karyawans) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Data dengan NIK tersebut tidak ditemukan",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(karyawans)
}

// GetAllKaryawanHandler menangani GET /api/karyawan
func (h *KaryawanHandler) GetAllKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
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

	karyawans, err := h.service.GetAllKaryawan(page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// (Opsional tapi Bagus) Tambahkan meta data pagination di balasan JSON
	response := map[string]interface{}{
		"page":                 page,
		"limit":                limit,
		"Total_data_This_Page": len(karyawans),
		"data":                 karyawans,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetKaryawanByIDHandler menangani GET /api/karyawan/detail?id=X
func (h *KaryawanHandler) GetKaryawanByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID harus berupa angka valid", http.StatusBadRequest)
		return
	}

	k, err := h.service.GetKaryawanByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if k == nil {
		http.Error(w, "Data karyawan tidak ditemukan", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(k)
}

// UpdateKaryawanHandler menangani PUT /api/karyawan/update?id=X
func (h *KaryawanHandler) UpdateKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Metode HTTP harus PUT", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID harus berupa angka valid", http.StatusBadRequest)
		return
	}

	var req CreateKaryawanReq // Struktur data request body-nya sama
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON Body salah", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateKaryawan(id, req.Nama, req.Jabatan, req.NIK, req.Phone); err != nil {
		// Jika pesan error mengandung kata "tidak ditemukan", kembalikan 404
		if strings.Contains(err.Error(), "tidak ditemukan") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Data karyawan berhasil diperbarui secara sinkron!"})
}

// DeleteKaryawanHandler menangani DELETE /api/karyawan/delete?id=X
func (h *KaryawanHandler) DeleteKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Metode HTTP harus DELETE", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID harus berupa angka valid", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteKaryawan(id); err != nil {
		// Jika pesan error mengandung kata "tidak ditemukan", kembalikan 404
		if strings.Contains(err.Error(), "tidak ditemukan") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Data karyawan berhasil dihapus permanen dari Postgres dan Elastic!"})
}

// GetKaryawanByTeleponHandler menangani endpoint GET /api/karyawan/search/telepon?telp=...
func (h *KaryawanHandler) GetKaryawanByTeleponHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	telpQuery := r.URL.Query().Get("telp")
	if telpQuery == "" {
		http.Error(w, "Parameter 'telp' wajib diisi", http.StatusBadRequest)
		return
	}

	karyawans, err := h.service.GetKaryawanByPhone(telpQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(karyawans)
}

// GetKaryawanByNamaHandler menangani endpoint GET /api/karyawan/search/nama?nama=...
func (h *KaryawanHandler) GetKaryawanByNameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	namaQuery := r.URL.Query().Get("nama")
	if namaQuery == "" {
		http.Error(w, "Parameter 'nama' wajib diisi", http.StatusBadRequest)
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

	karyawans, err := h.service.GetKaryawanByName(namaQuery, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(karyawans)
}

// GetKaryawanSortedByNIKHandler menangani endpoint GET /api/karyawan/sorted/nik
func (h *KaryawanHandler) GetKaryawanSortedByNIKHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	karyawans, err := h.service.GetKaryawanSortedByNIK()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(karyawans)
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
func (h *KaryawanHandler) SearchPGHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Metode HTTP harus GET", http.StatusMethodNotAllowed)
		return
	}

	namaQuery := r.URL.Query().Get("nama")
	if namaQuery == "" {
		http.Error(w, "Parameter 'nama' wajib diisi", http.StatusBadRequest)
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

	karyawans, err := h.service.SearchNamaPG(namaQuery, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"source": "PostgreSQL (ILIKE)",
		"total":  len(karyawans),
		"data":   karyawans,
	})
}
