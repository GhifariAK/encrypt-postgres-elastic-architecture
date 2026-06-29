package utils

import (
	"encoding/json"
	"math"
	"net/http"
)

// JSONResponse adalah blueprint universal untuk semua endpoint API
type JSONResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`       // omitempty: field hilang otomatis jika nil
	Pagination interface{} `json:"pagination,omitempty"` // omitempty: field hilang otomatis jika nil
	RequestID  string      `json:"request_id,omitempty"` // omitempty: hilang kalau kosong
}

// PaginationMeta menyimpan metadata halaman
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalData  int `json:"total_data"`
	TotalPages int `json:"total_pages"`
}

// SendSuccess mengirim respons sukses berformat JSON standar
func SendSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SendError mengirim respons error berformat JSON standar (menggantikan http.Error text biasa)
func SendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: false,
		Message: message,
	})
}

// SendSuccessWithPagination khusus untuk response berbentuk list/array berhalaman
func SendSuccessWithPagination(w http.ResponseWriter, statusCode int, message string, data interface{}, page int, limit int, totalData int, reqID string) {
	// Menghitung total halaman pembulatan ke atas
	totalPages := int(math.Ceil(float64(totalData) / float64(limit)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(JSONResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: reqID,
		Pagination: PaginationMeta{
			Page:       page,
			Limit:      limit,
			TotalData:  totalData,
			TotalPages: totalPages,
		},
	})
}
