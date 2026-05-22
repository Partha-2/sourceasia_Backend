package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"backend-assignment/internal/catalog"
	"backend-assignment/internal/validation"
)

type ProductHandler struct {
	store *catalog.ProductStore
}

func NewProductHandler(store *catalog.ProductStore) *ProductHandler {
	return &ProductHandler{store: store}
}

func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/products")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodPost:
			h.createProduct(w, r)
		case http.MethodGet:
			h.listProducts(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 && parts[1] == "media" {
		if r.Method == http.MethodPost {
			h.addMedia(w, r, parts[0])
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}

	if len(parts) == 1 && parts[0] != "" {
		if r.Method == http.MethodGet {
			h.getProduct(w, r, parts[0])
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (h *ProductHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var req catalog.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if err := validation.ValidateStringField(req.Name, "name"); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := validation.ValidateStringField(req.SKU, "sku"); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if req.ImageURLs == nil {
		req.ImageURLs = []string{}
	}
	if req.VideoURLs == nil {
		req.VideoURLs = []string{}
	}

	if err := validation.ValidateURLs(req.ImageURLs, validation.ImageURL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validation.ValidateURLs(req.VideoURLs, validation.VideoURL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, media, err := h.store.CreateProduct(req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate SKU") {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	resp := catalog.NewProductResponse(product, media)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *ProductHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 20
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	resp := h.store.ListProducts(limit, offset)
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) getProduct(w http.ResponseWriter, r *http.Request, id string) {
	product, media, err := h.store.GetProduct(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	resp := catalog.NewProductResponse(product, media)
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) addMedia(w http.ResponseWriter, r *http.Request, id string) {
	var req catalog.AddMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if len(req.ImageURLs) == 0 && len(req.VideoURLs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one of image_urls or video_urls is required"})
		return
	}

	if err := validation.ValidateURLs(req.ImageURLs, validation.ImageURL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validation.ValidateURLs(req.VideoURLs, validation.VideoURL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	product, media, err := h.store.AddMedia(id, req)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	resp := catalog.NewProductResponse(product, media)
	writeJSON(w, http.StatusOK, resp)
}
