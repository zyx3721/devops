package handler

import (
	"net/http"
	"strconv"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
)

// BookmarkHandler 书签处理器
type BookmarkHandler struct {
	bookmarkService *logs.BookmarkService
}

// NewBookmarkHandler 创建书签处理器
func NewBookmarkHandler(bookmarkService *logs.BookmarkService) *BookmarkHandler {
	return &BookmarkHandler{
		bookmarkService: bookmarkService,
	}
}

// ListBookmarks 获取书签列表
// @Summary 获取书签列表
// @Tags 日志书签
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=[]dto.BookmarkResponse}
// @Router /api/v1/logs/bookmarks [get]
func (h *BookmarkHandler) ListBookmarks(c *gin.Context) {
	userID := c.GetInt64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	bookmarks, total, err := h.bookmarkService.List(userID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": bookmarks,
		"total": total,
		"page":  page,
	})
}

// CreateBookmark 创建书签
// @Summary 创建书签
// @Tags 日志书签
// @Accept json
// @Produce json
// @Param body body dto.BookmarkRequest true "书签信息"
// @Success 200 {object} response.Response{data=dto.BookmarkResponse}
// @Router /api/v1/logs/bookmarks [post]
func (h *BookmarkHandler) CreateBookmark(c *gin.Context) {
	var req dto.BookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := c.GetInt64("user_id")
	bookmark, err := h.bookmarkService.Create(userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, bookmark)
}

// UpdateBookmark 更新书签
// @Summary 更新书签备注
// @Tags 日志书签
// @Accept json
// @Produce json
// @Param id path int true "书签ID"
// @Param body body object{note string} true "备注"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/bookmarks/{id} [put]
func (h *BookmarkHandler) UpdateBookmark(c *gin.Context) {
	bookmarkID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var req struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.bookmarkService.Update(userID, bookmarkID, req.Note); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// DeleteBookmark 删除书签
// @Summary 删除书签
// @Tags 日志书签
// @Param id path int true "书签ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/bookmarks/{id} [delete]
func (h *BookmarkHandler) DeleteBookmark(c *gin.Context) {
	bookmarkID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	if err := h.bookmarkService.Delete(userID, bookmarkID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ShareBookmark 分享书签
// @Summary 生成分享链接
// @Tags 日志书签
// @Accept json
// @Produce json
// @Param id path int true "书签ID"
// @Param body body dto.BookmarkShareRequest true "分享设置"
// @Success 200 {object} response.Response{data=object{share_url string}}
// @Router /api/v1/logs/bookmarks/{id}/share [post]
func (h *BookmarkHandler) ShareBookmark(c *gin.Context) {
	bookmarkID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var req dto.BookmarkShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.ExpiresInDays = 7 // 默认 7 天
	}

	shareURL, err := h.bookmarkService.Share(userID, bookmarkID, req.ExpiresInDays)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"share_url": shareURL})
}

// GetSharedBookmark 获取分享的书签
// @Summary 通过分享链接获取书签
// @Tags 日志书签
// @Param share_url path string true "分享链接"
// @Success 200 {object} response.Response{data=dto.BookmarkResponse}
// @Router /api/v1/logs/bookmarks/shared/{share_url} [get]
func (h *BookmarkHandler) GetSharedBookmark(c *gin.Context) {
	shareURL := c.Param("share_url")

	bookmark, err := h.bookmarkService.GetByShareURL(shareURL)
	if err != nil {
		response.Error(c, http.StatusNotFound, "书签不存在或已过期")
		return
	}

	response.Success(c, bookmark)
}
