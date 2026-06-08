package controller

import (
	"fmt"
	"ginchat/models"
	"ginchat/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetUsers(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "List of users",
	})
}

func CreateUser(c *gin.Context) {
	c.JSON(201, gin.H{
		"message": "User created successfully",
	})
}

func GetUser(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "User details",
	})
}

func UpdateUser(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "User updated successfully",
	})
}

func DeleteUser(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "User deleted successfully",
	})
}

func GetIndex(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Welcome to the InfoSearch API",
	})
}

// SearchDocuments 搜索文档接口
func SearchDocuments(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request parameters",
			"details": err.Error(),
		})
		return
	}

	if utils.SearchIndex == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search index is not ready",
		})
		return
	}

	startTime := time.Now()
	results := utils.SearchIndex.Search(req)
	total := len(results)

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize
	end := offset + req.PageSize
	if offset > total {
		results = []utils.SearchResult{}
	} else if end > total {
		results = results[offset:total]
	} else {
		results = results[offset:end]
	}

	duration := time.Since(startTime)

	searchRecord := models.SearchRecord{
		Query:       req.Query,
		UserIP:      c.ClientIP(),
		ResultCount: total,
		DurationMs:  duration.Milliseconds(),
		Filters:     fmt.Sprintf("tags:%v,start:%v,end:%v", req.TagIDs, req.StartAt, req.EndAt),
	}
	utils.DB.Create(&searchRecord)

	c.JSON(http.StatusOK, gin.H{
		"data": results,
		"pagination": gin.H{
			"page":      req.Page,
			"page_size": req.PageSize,
			"total":     total,
			"pages":     (int64(total) + int64(req.PageSize) - 1) / int64(req.PageSize),
		},
		"query": req.Query,
		"took":  duration.Milliseconds(),
	})
}

// CreateDocument 创建文档接口
func CreateDocument(c *gin.Context) {
	var doc models.Document
	if err := c.ShouldBindJSON(&doc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid document data",
			"details": err.Error(),
		})
		return
	}

	// 计算字数
	doc.WordCount = len(strings.Fields(doc.Content))

	// 设置默认状态
	if doc.Status == "" {
		doc.Status = "published"
	}

	if err := utils.DB.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create document",
			"details": err.Error(),
		})
		return
	}

	if utils.SearchIndex != nil {
		utils.SearchIndex.AddDocument(&doc)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Document created successfully",
		"data":    doc,
	})
}

// GetDocument 获取文档详情接口
func GetDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid document ID",
		})
		return
	}

	var doc models.Document
	if err := utils.DB.Preload("Tags").First(&doc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Document not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get document",
				"details": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": doc,
	})
}

// ExtractDocument 根据文档内容调用外部 AI 抽取信息
func ExtractDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid document ID",
		})
		return
	}

	var doc models.Document
	if err := utils.DB.Preload("Tags").First(&doc, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Document not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get document",
				"details": err.Error(),
			})
		}
		return
	}

	var tags []string
	for _, tag := range doc.Tags {
		tags = append(tags, tag.Name)
	}

	prompt := fmt.Sprintf(`请将以下文章内容提取为纯 JSON，返回一个对象。字段包括：
- title：文章标题
- summary：文章摘要
- author：作者
- source：来源
- published_at：发布时间（ISO 8601）
- tags：标签数组
- keywords：关键词数组
- main_points：主要结论或要点数组
- entities：抽取出的实体或专有名词数组
- sentiment：情绪倾向（positive/neutral/negative）

请不要输出任何额外说明文本，只输出 JSON。文章标题：%s
文章摘要：%s
文章标签：%v
文章内容：%s`, doc.Title, doc.Summary, tags, doc.Content)

	result, err := utils.ExtractWithDeepseek(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to extract document information",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// GetTags 获取所有标签接口
func GetTags(c *gin.Context) {
	var tags []models.Tag
	if err := utils.DB.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get tags",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tags,
	})
}

// GetSearchHistory 获取搜索历史接口
func GetSearchHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	offset := (page - 1) * pageSize
	var records []models.SearchRecord
	var total int64

	utils.DB.Model(&models.SearchRecord{}).Count(&total)
	utils.DB.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records)

	c.JSON(http.StatusOK, gin.H{
		"data": records,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
			"pages":     (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
// controller/api.go 追加以下代码

// SubmitEvaluation 提交检索结果的人工评价接口
func SubmitEvaluation(c *gin.Context) {
	var eval models.SearchEvaluation
	if err := c.ShouldBindJSON(&eval); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid parameters",
			"details": err.Error(),
		})
		return
	}

	eval.UserIP = c.ClientIP()

	if err := utils.DB.Create(&eval).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save evaluation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Evaluation submitted successfully",
		"data":    eval,
	})
}

// GetEvaluationMetrics 获取人工评价的准确率统计接口
func GetEvaluationMetrics(c *gin.Context) {
	var total int64
	var relevant int64

	// 统计总评价数
	utils.DB.Model(&models.SearchEvaluation{}).Count(&total)
	// 统计被标记为相关的评价数
	utils.DB.Model(&models.SearchEvaluation{}).Where("is_relevant = ?", true).Count(&relevant)

	precision := 0.0
	if total > 0 {
		precision = float64(relevant) / float64(total)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_evaluated": total,
			"relevant_count":  relevant,
			"precision":       precision, // 准确率
		},
	})
}