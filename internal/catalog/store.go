package catalog

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type ProductStore struct {
	mu       sync.RWMutex
	products map[string]*Product
	media    map[string]*MediaStore
	skuIndex map[string]string
}

func NewProductStore() *ProductStore {
	return &ProductStore{
		products: make(map[string]*Product),
		media:    make(map[string]*MediaStore),
		skuIndex: make(map[string]string),
	}
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate ID: %v", err))
	}
	return fmt.Sprintf("%x", b)
}

func (s *ProductStore) CreateProduct(req CreateProductRequest) (*Product, *MediaStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.skuIndex[req.SKU]; exists {
		return nil, nil, fmt.Errorf("duplicate SKU: %s", req.SKU)
	}

	id := generateID()
	now := time.Now().UTC()

	media := &MediaStore{
		ImageURLs: req.ImageURLs,
		VideoURLs: req.VideoURLs,
	}

	thumbnailURL := ""
	if len(req.ImageURLs) > 0 {
		thumbnailURL = req.ImageURLs[0]
	}

	product := &Product{
		ID:           id,
		Name:         req.Name,
		SKU:          req.SKU,
		ImageCount:   len(req.ImageURLs),
		VideoCount:   len(req.VideoURLs),
		ThumbnailURL: thumbnailURL,
		CreatedAt:    now,
	}

	s.products[id] = product
	s.media[id] = media
	s.skuIndex[req.SKU] = id

	return product, media, nil
}

func (s *ProductStore) ListProducts(limit, offset int) *ProductListResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.products)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	items := make([]ProductListItem, 0, end-start)

	ids := make([]string, 0, total)
	for id := range s.products {
		ids = append(ids, id)
	}

	for i := start; i < end; i++ {
		p := s.products[ids[i]]
		items = append(items, NewProductListItem(p))
	}

	return &ProductListResponse{
		Products: items,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}
}

func (s *ProductStore) GetProduct(id string) (*Product, *MediaStore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.products[id]
	if !exists {
		return nil, nil, fmt.Errorf("product not found: %s", id)
	}

	m := s.media[id]
	return p, m, nil
}

func (s *ProductStore) AddMedia(id string, req AddMediaRequest) (*Product, *MediaStore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, exists := s.products[id]
	if !exists {
		return nil, nil, fmt.Errorf("product not found: %s", id)
	}

	m := s.media[id]
	if m == nil {
		m = &MediaStore{}
		s.media[id] = m
	}

	m.ImageURLs = append(m.ImageURLs, req.ImageURLs...)
	m.VideoURLs = append(m.VideoURLs, req.VideoURLs...)

	p.ImageCount = len(m.ImageURLs)
	p.VideoCount = len(m.VideoURLs)

	if p.ThumbnailURL == "" && len(req.ImageURLs) > 0 {
		p.ThumbnailURL = req.ImageURLs[0]
	}

	return p, m, nil
}
