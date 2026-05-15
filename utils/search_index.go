// Package utils 提供搜索索引相关的工具函数
package utils

import (
	"ginchat/models"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// SearchIndex 全局搜索索引实例
var SearchIndex *InvertedIndex

// SearchResult 搜索结果结构体
type SearchResult struct {
	ID          uint       `json:"id"`           // 文档ID
	Title       string     `json:"title"`        // 标题
	URL         string     `json:"url"`          // URL
	PublishedAt *time.Time `json:"published_at"` // 发布时间
	Relevance   float64    `json:"relevance"`    // 相关性得分
	Snippet     string     `json:"main_match"`   // 摘要片段
}

// InvertedIndex 倒排索引结构体，用于实现全文搜索
type InvertedIndex struct {
	lock      sync.RWMutex              // 读写锁，用于并发安全
	termDocs  map[string]map[uint]int   // 词项到文档ID和词频的映射
	docFreq   map[string]int            // 词项的文档频率
	docNorm   map[uint]float64          // 文档的归一化因子
	docMeta   map[uint]*models.Document // 文档元数据
	totalDocs int                       // 总文档数
}

// NewInvertedIndex 创建一个新的倒排索引实例
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		termDocs: make(map[string]map[uint]int),
		docFreq:  make(map[string]int),
		docNorm:  make(map[uint]float64),
		docMeta:  make(map[uint]*models.Document),
	}
}

// tokenize 将文本按空格分割，返回词项列表
func (idx *InvertedIndex) tokenize(text string) []string {
	text = strings.ToLower(text)
	return strings.Fields(text)
}

// Build 从文档列表构建倒排索引
func (idx *InvertedIndex) Build(documents []models.Document) {
	idx.lock.Lock()
	defer idx.lock.Unlock()

	idx.termDocs = make(map[string]map[uint]int)
	idx.docFreq = make(map[string]int)
	idx.docNorm = make(map[uint]float64)
	idx.docMeta = make(map[uint]*models.Document)
	idx.totalDocs = len(documents)

	docTerms := make(map[uint]map[string]int)

	for i := range documents {
		doc := &documents[i]
		idx.docMeta[doc.ID] = doc

		text := strings.Repeat(doc.Title+" ", 2) + doc.Summary + " " + doc.Content
		tokens := idx.tokenize(text)
		if len(tokens) == 0 {
			continue
		}

		tf := make(map[string]int)
		for _, term := range tokens {
			if term == "" {
				continue
			}
			tf[term]++
		}
		docTerms[doc.ID] = tf
		for term := range tf {
			idx.docFreq[term]++
			if idx.termDocs[term] == nil {
				idx.termDocs[term] = make(map[uint]int)
			}
			idx.termDocs[term][doc.ID] = tf[term]
		}
	}

	for docID, tf := range docTerms {
		norm := 0.0
		for term, count := range tf {
			idf := idx.idf(term)
			weight := float64(count) * idf
			norm += weight * weight
		}
		idx.docNorm[docID] = math.Sqrt(norm)
	}
}

// idf 计算词项的逆文档频率
func (idx *InvertedIndex) idf(term string) float64 {
	if idx.totalDocs == 0 {
		return 0
	}
	df := idx.docFreq[term]
	return math.Log(float64(idx.totalDocs)/float64(df+1)+1.0) + 1.0
}

// BuildFromDB 从数据库加载已发布的文档并构建索引
func (idx *InvertedIndex) BuildFromDB() error {
	var docs []models.Document
	if err := DB.Preload("Tags").Where("status = ?", "published").Find(&docs).Error; err != nil {
		return err
	}
	idx.Build(docs)
	return nil
}

// AddDocument 向索引中添加单个文档
func (idx *InvertedIndex) AddDocument(doc *models.Document) {
	idx.lock.Lock()
	defer idx.lock.Unlock()

	idx.totalDocs++
	idx.docMeta[doc.ID] = doc

	text := strings.Repeat(doc.Title+" ", 2) + doc.Summary + " " + doc.Content
	tokens := idx.tokenize(text)
	if len(tokens) == 0 {
		return
	}

	tf := make(map[string]int)
	for _, term := range tokens {
		if term == "" {
			continue
		}
		tf[term]++
	}

	for term, count := range tf {
		idx.docFreq[term]++
		if idx.termDocs[term] == nil {
			idx.termDocs[term] = make(map[uint]int)
		}
		idx.termDocs[term][doc.ID] = count
	}

	norm := 0.0
	for term, count := range tf {
		weight := float64(count) * idx.idf(term)
		norm += weight * weight
	}
	idx.docNorm[doc.ID] = math.Sqrt(norm)
}

// Search 根据搜索请求执行搜索，返回排序后的结果
func (idx *InvertedIndex) Search(req models.SearchRequest) []SearchResult {
	idx.lock.RLock()
	defer idx.lock.RUnlock()

	queryText := strings.TrimSpace(req.Query)
	queryTerms := idx.tokenize(queryText)

	scores := make(map[uint]float64)
	if len(queryTerms) > 0 {
		qtf := make(map[string]int)
		for _, term := range queryTerms {
			qtf[term]++
		}

		queryNorm := 0.0
		for term, count := range qtf {
			idf := idx.idf(term)
			weight := float64(count) * idf
			queryNorm += weight * weight
			for docID, docCount := range idx.termDocs[term] {
				scores[docID] += weight * float64(docCount) * idf
			}
		}

		if queryNorm > 0 {
			queryNorm = math.Sqrt(queryNorm)
			for docID, score := range scores {
				if idx.docNorm[docID] > 0 {
					scores[docID] = score / (idx.docNorm[docID] * queryNorm)
				}
			}
		}
	}

	results := make([]SearchResult, 0, len(scores))
	for docID, doc := range idx.docMeta {
		if !idx.matchFilters(doc, req) {
			continue
		}

		if len(queryTerms) > 0 {
			score, found := scores[docID]
			if !found || score <= 0 {
				continue
			}
			results = append(results, SearchResult{
				ID:          docID,
				Title:       doc.Title,
				URL:         doc.URL,
				PublishedAt: doc.PublishedAt,
				Relevance:   math.Round(score*10000) / 10000,
				Snippet:     idx.makeSnippet(doc, queryTerms),
			})
		} else {
			results = append(results, SearchResult{
				ID:          docID,
				Title:       doc.Title,
				URL:         doc.URL,
				PublishedAt: doc.PublishedAt,
				Relevance:   0,
				Snippet:     doc.Summary,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Relevance == results[j].Relevance {
			if results[i].PublishedAt == nil || results[j].PublishedAt == nil {
				return results[i].ID < results[j].ID
			}
			return results[i].PublishedAt.After(*results[j].PublishedAt)
		}
		return results[i].Relevance > results[j].Relevance
	})

	return results
}

// matchFilters 检查文档是否匹配搜索请求的过滤条件
func (idx *InvertedIndex) matchFilters(doc *models.Document, req models.SearchRequest) bool {
	if len(req.TagIDs) > 0 {
		match := false
		for _, tagID := range req.TagIDs {
			for _, tag := range doc.Tags {
				if tag.ID == tagID {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		}
	}

	if !req.StartAt.IsZero() {
		if doc.PublishedAt == nil || doc.PublishedAt.Before(req.StartAt) {
			return false
		}
	}
	if !req.EndAt.IsZero() {
		if doc.PublishedAt == nil || doc.PublishedAt.After(req.EndAt) {
			return false
		}
	}
	return true
}

// makeSnippet 为文档生成搜索结果摘要片段
func (idx *InvertedIndex) makeSnippet(doc *models.Document, terms []string) string {
	for _, term := range terms {
		if snippet := idx.excerpt(doc.Title, term); snippet != "" {
			return snippet
		}
		if snippet := idx.excerpt(doc.Summary, term); snippet != "" {
			return snippet
		}
		if snippet := idx.excerpt(doc.Content, term); snippet != "" {
			return snippet
		}
	}
	if doc.Summary != "" {
		return doc.Summary
	}
	if len(doc.Content) > 120 {
		return doc.Content[:120] + "..."
	}
	return doc.Content
}

// excerpt 从文本中提取包含指定词项的片段
func (idx *InvertedIndex) excerpt(text, term string) string {
	lower := strings.ToLower(text)
	pos := strings.Index(lower, term)
	if pos < 0 {
		return ""
	}
	start := pos - 30
	if start < 0 {
		start = 0
	}
	end := pos + len(term) + 30
	if end > len(text) {
		end = len(text)
	}
	excerpt := strings.TrimSpace(text[start:end])
	if start > 0 {
		excerpt = "..." + excerpt
	}
	if end < len(text) {
		excerpt = excerpt + "..."
	}
	return excerpt
}
