package controller

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"gorm.io/gorm/clause"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ListSiteDocuments(c *gin.Context) {
	docs := []model.SiteDocument{}
	query := model.DB.Model(&model.SiteDocument{}).Omit("content").Order("sort_order, slug")
	if !c.GetBool("site_docs_admin") {
		query = query.Where("published = ?", true)
	}
	if err := query.Find(&docs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	// Client query caches handle navigation; revalidate after a reload or publication.
	c.Header("Cache-Control", "no-store")
	common.ApiSuccess(c, docs)
}

func GetSiteDocument(c *gin.Context) {
	var doc model.SiteDocument
	query := model.DB.Where("slug = ?", c.Param("slug"))
	if !c.GetBool("site_docs_admin") {
		query = query.Where("published = ?", true)
	}
	if err := query.First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Document not found"})
			return
		}
		common.ApiError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	common.ApiSuccess(c, doc)
}

func SaveSiteDocument(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10)
	var doc model.SiteDocument
	if err := c.ShouldBindJSON(&doc); err != nil {
		common.ApiErrorMsg(c, "Invalid document")
		return
	}
	if c.Param("slug") != "" {
		doc.Slug = c.Param("slug")
	}
	if err := doc.Validate(); err != nil {
		common.ApiError(c, err)
		return
	}
	doc.UpdatedAt = time.Now().Unix()
	if c.Request.Method == http.MethodPost {
		var count int64
		if err := model.DB.Model(&model.SiteDocument{}).Where("slug = ?", doc.Slug).Count(&count).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Document slug already exists"})
			return
		}
		doc.Version = 1
		doc.CreatedAt = doc.UpdatedAt
		if err := model.DB.Create(&doc).Error; err != nil {
			common.ApiError(c, err)
			return
		}
	} else {
		previousVersion := doc.Version
		doc.Version++
		result := model.DB.Model(&model.SiteDocument{}).Where("slug = ? AND version = ?", doc.Slug, previousVersion).
			Select("title", "category", "content", "published", "sort_order", "version", "updated_at").Updates(&doc)
		if result.Error != nil {
			common.ApiError(c, result.Error)
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "Document changed. Reload before saving."})
			return
		}
	}
	common.ApiSuccess(c, doc)
}

func UploadSiteDocumentImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1500<<10)
	var payload struct {
		Image string `json:"image"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.ApiErrorMsg(c, "Invalid image")
		return
	}
	if err := validateManagedWebPDataURL(payload.Image, 1024*1024); err != nil {
		common.ApiError(c, err)
		return
	}
	data, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(payload.Image, managedWebPDataPrefix))
	sum := sha256.Sum256(data)
	image := model.SiteDocumentImage{Hash: hex.EncodeToString(sum[:]), Data: data}
	if err := model.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&image).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"url": "/api/site-doc-images/" + image.Hash, "size": len(data)})
}

func GetSiteDocumentImage(c *gin.Context) {
	hash := c.Param("hash")
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != 32 {
		c.Status(http.StatusNotFound)
		return
	}
	var image model.SiteDocumentImage
	if err := model.DB.First(&image, "hash = ?", hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			common.ApiError(c, err)
		}
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/webp", image.Data)
}
