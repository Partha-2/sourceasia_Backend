package catalog

import "time"

type Product struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	ImageCount   int       `json:"image_count"`
	VideoCount   int       `json:"video_count"`
	ThumbnailURL string    `json:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type MediaStore struct {
	ImageURLs []string `json:"image_urls"`
	VideoURLs []string `json:"video_urls"`
}

type CreateProductRequest struct {
	Name      string   `json:"name"`
	SKU       string   `json:"sku"`
	ImageURLs []string `json:"image_urls"`
	VideoURLs []string `json:"video_urls"`
}

type AddMediaRequest struct {
	ImageURLs []string `json:"image_urls"`
	VideoURLs []string `json:"video_urls"`
}

type ProductResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	ImageCount   int       `json:"image_count"`
	VideoCount   int       `json:"video_count"`
	ThumbnailURL string    `json:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at"`
	ImageURLs    []string  `json:"image_urls,omitempty"`
	VideoURLs    []string  `json:"video_urls,omitempty"`
}

type ProductListItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	ImageCount   int       `json:"image_count"`
	VideoCount   int       `json:"video_count"`
	ThumbnailURL string    `json:"thumbnail_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProductListResponse struct {
	Products []ProductListItem `json:"products"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

func NewProductResponse(p *Product, m *MediaStore) ProductResponse {
	resp := ProductResponse{
		ID:           p.ID,
		Name:         p.Name,
		SKU:          p.SKU,
		ImageCount:   p.ImageCount,
		VideoCount:   p.VideoCount,
		ThumbnailURL: p.ThumbnailURL,
		CreatedAt:    p.CreatedAt,
	}
	if m != nil {
		resp.ImageURLs = m.ImageURLs
		resp.VideoURLs = m.VideoURLs
	}
	return resp
}

func NewProductListItem(p *Product) ProductListItem {
	return ProductListItem{
		ID:           p.ID,
		Name:         p.Name,
		SKU:          p.SKU,
		ImageCount:   p.ImageCount,
		VideoCount:   p.VideoCount,
		ThumbnailURL: p.ThumbnailURL,
		CreatedAt:    p.CreatedAt,
	}
}
