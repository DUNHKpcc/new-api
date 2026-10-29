package model

import (
	"embed"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SiteDocument keeps article bodies out of the public navigation query.
type SiteDocument struct {
	Slug      string `json:"slug" gorm:"primaryKey;type:varchar(160)"`
	Title     string `json:"title" gorm:"type:varchar(255);not null"`
	Category  string `json:"category" gorm:"type:varchar(80)"`
	Content   string `json:"content,omitempty" gorm:"type:text"`
	Published bool   `json:"published" gorm:"index:idx_site_documents_catalog,priority:1"`
	SortOrder int    `json:"sort_order" gorm:"index:idx_site_documents_catalog,priority:2"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

var siteDocumentSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (d *SiteDocument) Validate() error {
	d.Title = strings.TrimSpace(d.Title)
	d.Category = strings.TrimSpace(d.Category)
	if len(d.Slug) > 160 || !siteDocumentSlug.MatchString(d.Slug) || d.Title == "" || utf8.RuneCountInString(d.Title) > 255 || utf8.RuneCountInString(d.Category) > 80 || len(d.Content) > 60000 || strings.TrimSpace(d.Content) == "" {
		return errors.New("invalid document fields (content limit: 60000 bytes)")
	}
	return nil
}

//go:embed site-documents-seed.json
var siteDocumentSeeds embed.FS

// Existing rows are never overwritten, including unpublished administrator edits.
func SeedSiteDocuments(db *gorm.DB) error {
	data, err := siteDocumentSeeds.ReadFile("site-documents-seed.json")
	if err != nil {
		return err
	}
	var docs []SiteDocument
	if err = common.Unmarshal(data, &docs); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, doc := range docs {
			doc.Version = 1
			doc.CreatedAt = time.Now().Unix()
			doc.UpdatedAt = doc.CreatedAt
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&doc).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Content-addressed images are stored separately so article/catalog reads stay small.
type SiteDocumentImage struct {
	Hash string `gorm:"primaryKey;type:varchar(64)"`
	Data []byte
}
