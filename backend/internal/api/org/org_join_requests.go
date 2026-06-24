package org

import (
	"errors"
	"net/http"
	"software-web-manager/backend/internal/db/schema"
	"strings"
	"time"

	"software-web-manager/backend/internal/middleware"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createOrgJoinRequestRequest struct {
	Reason *string `json:"reason"`
}

type rejectOrgJoinRequestRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type batchDeleteMyOrgJoinRequestsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchWithdrawMyOrgJoinRequestsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type batchDeleteOrgJoinRequestsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

func (h *Handler) CreateOrgJoinRequest(c *gin.Context) {
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org_id required"})
		return
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
		return
	}
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	var req createOrgJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
	}

	var org models.Org
	if err := h.DB.Where("id = ?", orgUUID).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "org not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load org"})
		return
	}
	if strings.ToLower(strings.TrimSpace(org.Status)) != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "org not active"})
		return
	}
	if schema.HasOrgTypeColumn(h.DB) && strings.ToLower(strings.TrimSpace(org.OrgType)) != "enterprise" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only enterprise org can be requested"})
		return
	}

	var member models.OrgMember
	if err := h.DB.Where("scope_id = ? AND user_id = ?", orgUUID, userUUID).First(&member).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already in org"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check membership"})
		return
	}

	var pendingCount int64
	if err := h.DB.Model(&models.OrgJoinRequest{}).
		Where("org_id = ? AND user_id = ? AND status = ?", orgUUID, userUUID, "pending").
		Count(&pendingCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check requests"})
		return
	}
	if pendingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "request already pending"})
		return
	}

	item := models.OrgJoinRequest{
		OrgID:  orgUUID,
		UserID: userUUID,
		Reason: reason,
		Status: "pending",
	}
	if err := h.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create join request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": item})
}

func (h *Handler) ListMyOrgJoinRequests(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if !h.RequirePermission(c, "org_join_request.manage_own") {
		return
	}
	type row struct {
		ID           uuid.UUID  `json:"id"`
		OrgID        uuid.UUID  `json:"org_id"`
		OrgName      string     `json:"org_name"`
		Status       string     `json:"status"`
		Reason       string     `json:"reason"`
		ReviewReason *string    `json:"review_reason"`
		CreatedAt    time.Time  `json:"created_at"`
		ReviewedAt   *time.Time `json:"reviewed_at"`
	}
	var rows []row
	if err := h.DB.Raw(`
		SELECT r.id, r.org_id, o.name AS org_name, r.status, r.reason, r.review_reason, r.created_at, r.reviewed_at
		FROM org_join_requests r
		JOIN orgs o ON o.id = r.org_id
		WHERE r.user_id = ?
		ORDER BY r.created_at DESC
	`, userID).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list join requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) BatchDeleteMyOrgJoinRequests(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if !h.RequirePermission(c, "org_join_request.manage_own") {
		return
	}

	var req batchDeleteMyOrgJoinRequestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var items []models.OrgJoinRequest
	if err := h.DB.Where("id IN ? AND user_id = ?", ids, userID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load join requests"})
		return
	}
	if len(items) == 0 || len(items) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "join requests not found"})
		return
	}

	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Status)) == "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请先等待审批或主动撤回"})
			return
		}
	}

	res := h.DB.Where("id IN ? AND user_id = ?", ids, userID).Delete(&models.OrgJoinRequest{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete join requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

func (h *Handler) BatchWithdrawMyOrgJoinRequests(c *gin.Context) {
	userID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if !h.RequirePermission(c, "org_join_request.manage_own") {
		return
	}

	var req batchWithdrawMyOrgJoinRequestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var items []models.OrgJoinRequest
	if err := h.DB.Where("id IN ? AND user_id = ?", ids, userID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load join requests"})
		return
	}
	if len(items) == 0 || len(items) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "join requests not found"})
		return
	}
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Status)) != "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持撤回待审核申请"})
			return
		}
	}

	now := time.Now()
	res := h.DB.Model(&models.OrgJoinRequest{}).
		Where("id IN ? AND user_id = ? AND status = ?", ids, userID, "pending").
		Updates(map[string]any{
			"status":      "withdrawn",
			"reviewed_at": now,
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to withdraw join requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"withdrawn": res.RowsAffected})
}

func (h *Handler) ListOrgJoinRequests(c *gin.Context) {
	if !h.RequirePermission(c, "org_join_request.review") {
		return
	}
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID != strings.TrimSpace(c.GetString(middleware.ContextOrgID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	type row struct {
		ID           uuid.UUID  `json:"id"`
		OrgID        uuid.UUID  `json:"org_id"`
		UserID       uuid.UUID  `json:"user_id"`
		UserEmail    string     `json:"user_email"`
		Status       string     `json:"status"`
		Reason       string     `json:"reason"`
		ReviewReason *string    `json:"review_reason"`
		CreatedAt    time.Time  `json:"created_at"`
		ReviewedAt   *time.Time `json:"reviewed_at"`
	}
	var rows []row
	if err := h.DB.Raw(`
		SELECT r.id, r.org_id, r.user_id, u.email AS user_email, r.status, r.reason, r.review_reason, r.created_at, r.reviewed_at
		FROM org_join_requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.org_id = ? AND r.status <> 'withdrawn'
		ORDER BY r.created_at DESC
	`, orgID).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list join requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) BatchDeleteOrgJoinRequests(c *gin.Context) {
	if !h.RequirePermission(c, "org_join_request.review") {
		return
	}
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID != strings.TrimSpace(c.GetString(middleware.ContextOrgID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req batchDeleteOrgJoinRequestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, err := uuid.Parse(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	var items []models.OrgJoinRequest
	if err := h.DB.Where("id IN ? AND org_id = ?", ids, orgID).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load join requests"})
		return
	}
	if len(items) == 0 || len(items) != len(ids) {
		c.JSON(http.StatusNotFound, gin.H{"error": "join requests not found"})
		return
	}

	for _, item := range items {
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if status != "approved" && status != "rejected" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持删除已驳回或已通过申请记录"})
			return
		}
	}

	res := h.DB.Where("id IN ? AND org_id = ?", ids, orgID).Delete(&models.OrgJoinRequest{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete join requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

func (h *Handler) ApproveOrgJoinRequest(c *gin.Context) {
	if !h.RequirePermission(c, "org_join_request.review") {
		return
	}
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID != strings.TrimSpace(c.GetString(middleware.ContextOrgID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id required"})
		return
	}
	reviewerID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	reviewerUUID, err := uuid.Parse(reviewerID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var item models.OrgJoinRequest
		if err := tx.Where("id = ? AND org_id = ?", requestID, orgID).First(&item).Error; err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSpace(item.Status)) != "pending" {
			return errors.New("request not pending")
		}
		var member models.OrgMember
		memberErr := tx.Where("scope_id = ? AND user_id = ?", item.OrgID, item.UserID).First(&member).Error
		if memberErr != nil {
			if !errors.Is(memberErr, gorm.ErrRecordNotFound) {
				return memberErr
			}
			member = models.OrgMember{OrgID: item.OrgID, UserID: item.UserID, Role: "viewer"}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&models.OrgJoinRequest{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"status":        "approved",
				"review_reason": nil,
				"reviewed_by":   reviewerUUID,
				"reviewed_at":   now,
			}).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "join request not found"})
			return
		}
		if err.Error() == "request not pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request not pending"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve join request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RejectOrgJoinRequest(c *gin.Context) {
	if !h.RequirePermission(c, "org_join_request.review") {
		return
	}
	orgID := strings.TrimSpace(c.Param("id"))
	if orgID != strings.TrimSpace(c.GetString(middleware.ContextOrgID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	requestID := strings.TrimSpace(c.Param("request_id"))
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id required"})
		return
	}
	var req rejectOrgJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason required"})
		return
	}
	reviewerID := strings.TrimSpace(c.GetString(middleware.ContextUserID))
	reviewerUUID, err := uuid.Parse(reviewerID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	now := time.Now()
	res := h.DB.Model(&models.OrgJoinRequest{}).
		Where("id = ? AND org_id = ? AND status = ?", requestID, orgID, "pending").
		Updates(map[string]any{
			"status":        "rejected",
			"review_reason": reason,
			"reviewed_by":   reviewerUUID,
			"reviewed_at":   now,
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject join request"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "join request not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
