package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/agentscan/agentscan/internal/api"
)

// OrganizationHandlers provides HTTP handlers for organization management
type OrganizationHandlers struct {
	orgService *OrganizationService
}

// NewOrganizationHandlers creates new organization handlers
func NewOrganizationHandlers(orgService *OrganizationService) *OrganizationHandlers {
	return &OrganizationHandlers{
		orgService: orgService,
	}
}

// CreateOrganization creates a new organization
func (h *OrganizationHandlers) CreateOrganization(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	org, err := h.orgService.CreateOrganization(c.Request.Context(), userID, &req)
	if err != nil {
		if err.Error() == "slug '"+req.Slug+"' is already taken" {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to create organization")
		return
	}

	api.CreatedResponse(c, org)
}

// GetOrganization gets an organization by ID
func (h *OrganizationHandlers) GetOrganization(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	org, err := h.orgService.GetOrganization(c.Request.Context(), userID, orgID)
	if err != nil {
		if err.Error() == "insufficient permissions to read organization" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to get organization")
		return
	}

	api.SuccessResponse(c, org)
}

// UpdateOrganization updates an organization
func (h *OrganizationHandlers) UpdateOrganization(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	org, err := h.orgService.UpdateOrganization(c.Request.Context(), userID, orgID, &req)
	if err != nil {
		if err.Error() == "insufficient permissions to update organization" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to update organization")
		return
	}

	api.SuccessResponse(c, org)
}

// DeleteOrganization deletes an organization
func (h *OrganizationHandlers) DeleteOrganization(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	err := h.orgService.DeleteOrganization(c.Request.Context(), userID, orgID)
	if err != nil {
		if err.Error() == "insufficient permissions to delete organization" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to delete organization")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Organization deleted successfully",
	})
}

// ListUserOrganizations lists all organizations a user is a member of
func (h *OrganizationHandlers) ListUserOrganizations(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgs, err := h.orgService.ListUserOrganizations(c.Request.Context(), userID)
	if err != nil {
		api.InternalErrorResponse(c, "Failed to list organizations")
		return
	}

	api.SuccessResponse(c, orgs)
}

// GetOrganizationMembers gets all members of an organization
func (h *OrganizationHandlers) GetOrganizationMembers(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	members, err := h.orgService.GetOrganizationMembers(c.Request.Context(), userID, orgID)
	if err != nil {
		if err.Error() == "insufficient permissions to read organization members" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to get organization members")
		return
	}

	api.SuccessResponse(c, members)
}

// InviteMember invites a user to join an organization
func (h *OrganizationHandlers) InviteMember(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	err := h.orgService.InviteMember(c.Request.Context(), userID, orgID, &req)
	if err != nil {
		if err.Error() == "insufficient permissions to invite members" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		if err.Error() == "user is already a member of this organization" {
			api.BadRequestResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "not found") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to invite member")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Member invited successfully",
	})
}

// RemoveMember removes a member from an organization
func (h *OrganizationHandlers) RemoveMember(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	memberUserIDStr := c.Param("memberUserId")
	if memberUserIDStr == "" {
		api.BadRequestResponse(c, "Member user ID required")
		return
	}

	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		api.BadRequestResponse(c, "Invalid member user ID")
		return
	}

	err = h.orgService.RemoveMember(c.Request.Context(), userID, orgID, memberUserID)
	if err != nil {
		if err.Error() == "insufficient permissions to remove members" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "last owner") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to remove member")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Member removed successfully",
	})
}

// UpdateMemberRole updates a member's role in an organization
func (h *OrganizationHandlers) UpdateMemberRole(c *gin.Context) {
	userID, exists := GetCurrentUserID(c)
	if !exists {
		api.UnauthorizedResponse(c, "User not found in context")
		return
	}

	orgID, exists := GetOrganizationID(c)
	if !exists {
		api.BadRequestResponse(c, "Organization ID not found in context")
		return
	}

	memberUserIDStr := c.Param("memberUserId")
	if memberUserIDStr == "" {
		api.BadRequestResponse(c, "Member user ID required")
		return
	}

	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		api.BadRequestResponse(c, "Invalid member user ID")
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestResponse(c, "Invalid request body")
		return
	}

	err = h.orgService.UpdateMemberRole(c.Request.Context(), userID, orgID, memberUserID, &req)
	if err != nil {
		if err.Error() == "insufficient permissions to update member roles" {
			api.ForbiddenResponse(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "last owner") {
			api.BadRequestResponse(c, err.Error())
			return
		}
		api.InternalErrorResponse(c, "Failed to update member role")
		return
	}

	api.SuccessResponse(c, map[string]string{
		"message": "Member role updated successfully",
	})
}