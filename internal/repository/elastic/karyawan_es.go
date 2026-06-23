package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// SetupIndex memastikan tabel Elasticsearch dibuat menggunakan tipe data "wildcard"
func SetupIndex(es *elasticsearch.Client) error {
	indexName := "karyawan_index"

	res, err := es.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	if res.StatusCode == 200 {
		return nil // Index sudah ada
	}

	// Untuk alternatif query "order By" dari data yang ter encrypt di postgr
	// adalah menambahkan fields keyword
	mappingBody := `{
		"mappings": {
			"properties": {
				"nama": { 
					"type": "text",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				},
				"nik": { 
					"type": "wildcard",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				},
				"phone": { 
					"type": "wildcard",
					"fields": {
						"keyword": { "type": "keyword" }
					}
				}
			}
		}
	}`

	req := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mappingBody),
	}

	createRes, err := req.Do(context.Background(), es)
	if err != nil || createRes.IsError() {
		return fmt.Errorf("gagal membuat index: %v", createRes.String())
	}

	fmt.Println("✅ Elasticsearch: Index dengan tipe 'wildcard' berhasil dibuat!")
	return nil
}

func IndexKaryawan(es *elasticsearch.Client, id int, nama, nikAsli, phoneAsli string) error {
	doc := map[string]string{
		"nama":  nama,
		"nik":   nikAsli, // dibuat Plaintext di elastic agar bisa melakukan query LIKE dan sebagainya
		"phone": phoneAsli,
	}
	docBytes, _ := json.Marshal(doc)

	req := esapi.IndexRequest{
		Index: "karyawan_index",
		// Menyamakan DocumentID Elastic dengan ID Postgres.
		DocumentID: fmt.Sprintf("%d", id),
		Body:       bytes.NewReader(docBytes),

		// Khusus untuk uji coba : Memaksa Elastic langsung me-refresh index agar data instan bisa dicari.
		// namun pada production jangan diaktifkan, karena Elastic punya mekanisme auto-refresh tiap 1 detik
		// Memasksa elastic refresh tiap kali ada data masuk akan membuat beban disk I/O server melonjak tajam.
		Refresh: "true",
	}

	res, err := req.Do(context.Background(), es)
	if err != nil || res.IsError() {
		return fmt.Errorf("gagal indexing ke elastic")
	}
	defer res.Body.Close()
	return nil
}

func SearchNIK(es *elasticsearch.Client, nikQuery string, limit int, offset int) ([]int, int, error) {
	// Query pencarian dengan wildcard
	queryBody := fmt.Sprintf(`{
		"from": %d,
        "size": %d,
		"query": {
			"wildcard": {
				"nik": {
					"value": "*%s*"
				}
			}
		}
	}`, offset, limit, nikQuery)

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("karyawan_index"),
		es.Search.WithBody(bytes.NewReader([]byte(queryBody))),
	)

	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("gagal mencari di elastic")
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	var ids []int
	totalData := 0

	// Menghindari panic jika hasil pencarian kosong (nil)
	if hitsData, ok := result["hits"].(map[string]interface{}); ok {
		// Ambil total data dari elastic
		if totalMap, ok := hitsData["total"].(map[string]interface{}); ok {
			if totalVal, ok := totalMap["value"].(float64); ok {
				totalData = int(totalVal)
			}
		}

		if hitsList, ok := hitsData["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				docID := hit.(map[string]interface{})["_id"].(string)
				var id int
				fmt.Sscanf(docID, "%d", &id)
				ids = append(ids, id)
			}
		}
	}

	return ids, totalData, nil
}

// SearchPhone mencari dokumen berdasarkan potongan nomor telepon
func SearchPhone(es *elasticsearch.Client, phoneQuery string, limit int, offset int) ([]int, int, error) {
	queryBody := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"wildcard": {
				"phone": {
					"value": "*%s*"
				}
			}
		}
	}`, offset, limit, phoneQuery)

	// untuk default "wildcard" sama seperti query Like (case sensitive)
	// Jika ingin seperti query "ILIKE" (case insensitive) maka harus menggunakan
	// "case_insensitive": true diletakan di bawah "value": "*%s*"

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("karyawan_index"),
		es.Search.WithBody(bytes.NewReader([]byte(queryBody))),
	)

	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("gagal mencari di elastic")
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	var ids []int
	totalData := 0

	if hitsData, ok := result["hits"].(map[string]interface{}); ok {
		// Ambil total data dari elastic
		if totalMap, ok := hitsData["total"].(map[string]interface{}); ok {
			if totalVal, ok := totalMap["value"].(float64); ok {
				totalData = int(totalVal)
			}
		}

		if hitsList, ok := hitsData["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				docID := hit.(map[string]interface{})["_id"].(string)
				var id int
				fmt.Sscanf(docID, "%d", &id)
				ids = append(ids, id)
			}
		}
	}

	return ids, totalData, nil
}

// SearchNama mencari dokumen berdasarkan nama dengan toleransi salah ketik (typo)
func SearchNama(es *elasticsearch.Client, namaQuery string, limit int, offset int) ([]int, int, error) {
	// // Menggunakan "match" dan "fuzziness" agar "Andy" tetap cocok dengan "Andi"
	// queryBody := fmt.Sprintf(`{
	// 	"query": {
	// 		"match": {
	// 			"nama": {
	// 				"query": "%s",
	// 				"fuzziness": "AUTO"
	// 			}
	// 		}
	// 	}
	// }`, namaQuery)

	// Menggunakan wildcard pada "nama.keyword" dipadukan dengan case_insensitive: true.
	// Jika user mengetik "an", maka "Andi", "Anton", "Hasan" akan langsung ditemukan.
	queryBody := fmt.Sprintf(`{
		"from": %d,
		"size": %d,
		"query": {
			"wildcard": {
				"nama.keyword": {
					"value": "*%s*",
					"case_insensitive": true
				}
			}
		}
	}`, offset, limit, namaQuery)

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("karyawan_index"),
		es.Search.WithBody(bytes.NewReader([]byte(queryBody))),
	)

	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("gagal mencari nama di elastic")
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	var ids []int
	totalData := 0

	// Menghindari panic jika hasil pencarian kosong
	if hitsData, ok := result["hits"].(map[string]interface{}); ok {
		// Ambil total data dari hits.total.value
		if totalMap, ok := hitsData["total"].(map[string]interface{}); ok {
			if totalVal, ok := totalMap["value"].(float64); ok {
				totalData = int(totalVal)
			}
		}

		// Ambil ID dokumen dari hits.hits
		if hitsList, ok := hitsData["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				docID := hit.(map[string]interface{})["_id"].(string)
				var id int
				fmt.Sscanf(docID, "%d", &id)
				ids = append(ids, id)
			}
		}
	}

	return ids, totalData, nil
}

// DeleteKaryawan menghapus data katalog pencarian di Elasticsearch berdasarkan ID
func DeleteKaryawan(es *elasticsearch.Client, id int) error {
	req := esapi.DeleteRequest{
		Index:      "karyawan_index",
		DocumentID: fmt.Sprintf("%d", id),
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("gagal koneksi ke elastic: %v", err)
	}
	defer res.Body.Close()

	// Jika errornya adalah 404 (Not Found), kita anggap sukses
	// karena tujuan utamanya adalah memastikan datanya tidak ada.
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("gagal menghapus data di elastic, status code: %d", res.StatusCode)
	}

	return nil
}

// GetAllSortedByNIK mengambil semua ID dari Elastic yang sudah diurutkan berdasarkan NIK ASC
func GetAllSortedByNIK(es *elasticsearch.Client, limit int, offset int) ([]int, int, error) {
	queryBody := fmt.Sprintf(`{
		"from": %d,
        "size": %d,
		"track_total_hits": true,
		"query": { "match_all": {} },
		"sort": [ { "nik.keyword": { "order": "asc" } } ]
	}`, offset, limit)

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("karyawan_index"),
		es.Search.WithBody(bytes.NewReader([]byte(queryBody))),
	)
	if err != nil || res.IsError() {
		return nil, 0, fmt.Errorf("gagal sort NIK di elastic: %s", res.String())
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	var ids []int
	totalData := 0

	if hitsData, ok := result["hits"].(map[string]interface{}); ok {
		// Ambil total data dari elastic
		if totalMap, ok := hitsData["total"].(map[string]interface{}); ok {
			if totalVal, ok := totalMap["value"].(float64); ok {
				totalData = int(totalVal)
			}
		}

		if hitsList, ok := hitsData["hits"].([]interface{}); ok {
			for _, hit := range hitsList {
				docID := hit.(map[string]interface{})["_id"].(string)
				var id int
				fmt.Sscanf(docID, "%d", &id)
				ids = append(ids, id)
			}
		}
	}
	return ids, totalData, nil
}

// GetPhoneProviderStats melakukan GROUP BY (Agregasi) berdasarkan 4 angka awalan telepon
func GetPhoneProviderStats(es *elasticsearch.Client) (map[string]int, error) {
	// mengecek apakah data telepon ada, lalu memotong 4 digit pertamanya
	queryBody := `{
		"size": 0,
		"aggs": {
			"provider_stats": {
				"terms": {
					"script": {
						"source": "if (doc['phone.keyword'].size() != 0) { def p = doc['phone.keyword'].value; if (p.length() >= 4) { return p.substring(0,4); } } return 'Lainnya';"
					}
				}
			}
		}
	}`

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("karyawan_index"),
		es.Search.WithBody(bytes.NewReader([]byte(queryBody))),
	)
	if err != nil || res.IsError() {
		return nil, fmt.Errorf("gagal melakukan agregasi di elastic: %v", res)
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	// Membaca hasil agregasi (buckets) dari JSON Elastic ke dalam Map Golang
	stats := make(map[string]int)
	if aggs, ok := result["aggregations"].(map[string]interface{}); ok {
		if providerStats, ok := aggs["provider_stats"].(map[string]interface{}); ok {
			if buckets, ok := providerStats["buckets"].([]interface{}); ok {
				for _, b := range buckets {
					bucket := b.(map[string]interface{})
					key := bucket["key"].(string)
					count := int(bucket["doc_count"].(float64))
					stats[key] = count
				}
			}
		}
	}
	return stats, nil
}
