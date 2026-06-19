# Tahap 1: Build (Membangun aplikasi Go)
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Salin daftar dependency dulu agar di-cache oleh Docker
COPY go.mod go.sum ./
RUN go mod download

# Salin seluruh kode proyekmu
COPY . .

# Build aplikasi menjadi file binary mandiri (tanpa dependensi OS luar)
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# Tahap 2: Rilis (Menggunakan OS kosong yang sangat ringan)
FROM alpine:3.19

WORKDIR /app

# Salin hasil build (file 'main') dari Tahap 1
COPY --from=builder /app/main .

# Expose port yang digunakan aplikasi
EXPOSE 8080

# Perintah utama saat kontainer dijalankan
CMD ["./main"]